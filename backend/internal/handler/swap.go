package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// parseAsset turns a "XLM" or "CODE:ISSUER" string into a chain.SwapAsset.
func parseAsset(raw string) chain.SwapAsset {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "XLM") {
		return chain.SwapAsset{Code: "XLM"}
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		return chain.SwapAsset{Code: parts[0], Issuer: parts[1]}
	}
	return chain.SwapAsset{Code: parts[0]}
}

// swapQuote returns an estimate of how much of the destination asset a
// strict-send swap would yield, without signing anything.
func (s *Handler) swapQuote(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "wallet_not_found", "Wallet not found."))
		return
	}
	if wallet.Chain != "stellar" {
		httpapi.Error(c, apperror.New(apperror.Invalid, "unsupported_chain", "Swaps are only supported on Stellar."))
		return
	}
	amount := strings.TrimSpace(c.Query("amount"))
	if value, err := strconv.ParseFloat(amount, 64); err != nil || value <= 0 {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_amount", "Amount must be positive."))
		return
	}
	send := parseAsset(c.Query("from"))
	dest := parseAsset(c.Query("to"))
	if send.Code == dest.Code && send.Issuer == dest.Issuer {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_asset_pair", "Choose two different assets."))
		return
	}
	quote, err := s.chain.SwapQuote(send, amount, dest)
	if err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Invalid, "swap_quote_failed", "Unable to quote this swap. Try another asset pair or amount."))
		return
	}
	httpapi.Success(c, http.StatusOK, quote)
}

type swapBody struct {
	From        string `json:"from"` // "XLM" or "CODE:ISSUER"
	To          string `json:"to"`
	Amount      string `json:"amount"`       // exact amount of `from` to send
	SlippageBps int    `json:"slippage_bps"` // acceptable slippage, default 100 (1%)
}

// createSwap quotes, applies slippage protection, builds a PathPaymentStrictSend,
// and dispatches it for MPC signing — reusing the same broadcast path as a send.
func (s *Handler) createSwap(c *gin.Context) {
	userID := c.GetString("userID")
	wallet, ok := s.store.WalletFor(userID, c.Param("id"))
	if !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "wallet_not_found", "Wallet not found."))
		return
	}
	if wallet.Status != "ready" {
		httpapi.Error(c, apperror.New(apperror.Conflict, "wallet_not_ready", "Wallet is not ready yet."))
		return
	}
	if wallet.Chain != "stellar" {
		httpapi.Error(c, apperror.New(apperror.Invalid, "unsupported_chain", "Swaps are only supported on Stellar."))
		return
	}
	var body swapBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_request", "Invalid request body."))
		return
	}
	amountValue, err := strconv.ParseFloat(strings.TrimSpace(body.Amount), 64)
	if err != nil || amountValue <= 0 {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_amount", "Amount must be positive."))
		return
	}
	send := parseAsset(body.From)
	dest := parseAsset(body.To)
	if send.Code == dest.Code && send.Issuer == dest.Issuer {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_asset_pair", "Choose two different assets."))
		return
	}

	if err := s.walletService.RequireNodes(service.SignQuorum, "signing"); err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Unavailable, "mpc_unavailable", "Not enough MPC nodes are online. Please try again later."))
		return
	}

	quote, err := s.chain.SwapQuote(send, strings.TrimSpace(body.Amount), dest)
	if err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Invalid, "swap_quote_failed", "Unable to quote this swap. Try another asset pair or amount."))
		return
	}
	slippage := body.SlippageBps
	if slippage <= 0 {
		slippage = 100 // 1% default
	}
	destMin := chain.ApplySlippage(quote.DestAmount, slippage)

	envelope, hash, err := s.chain.BuildSwap(
		wallet, send, strings.TrimSpace(body.Amount), dest, destMin, quote.Path,
	)
	if err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Invalid, "swap_build_failed", "Unable to prepare swap. Check the assets, amount and balance."))
		return
	}

	transaction := domain.Transaction{
		ID:           uuid.NewString(),
		WalletID:     wallet.ID,
		UserID:       userID,
		Type:         "swap",
		Counterparty: wallet.Address, // strict-send lands back on the same account
		Amount:       strconv.FormatFloat(amountValue, 'f', 2, 64),
		Symbol:       send.Code,
		RecvAmount:   quote.DestAmount,
		RecvSymbol:   dest.Code,
		Status:       "signing",
		EnvelopeXDR:  envelope,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.payments.CreateTransaction(transaction); err != nil {
		httpapi.Error(c, fmt.Errorf("persist swap: %w", err))
		return
	}
	if err := s.mpc.Sign(
		wallet.ID, transaction.ID, domain.Chains[wallet.Chain].Network, hash,
	); err != nil {
		s.payments.SetTransactionStatus(transaction.ID, "failed")
		httpapi.Error(c, apperror.Wrap(err, apperror.Upstream, "sign_dispatch_failed", "Unable to start transaction signing."))
		return
	}
	s.paymentService.WatchSignTimeout(userID, transaction.ID)
	s.hub.PublishTransaction(userID, transaction)
	httpapi.Success(c, http.StatusCreated, gin.H{"transaction": transaction, "quote": quote, "dest_min": destMin})
}
