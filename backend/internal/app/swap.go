package app

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"

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
func (s *Server) swapQuote(c *gin.Context) {
	wallet, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	if wallet.Chain != "stellar" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "swaps are Stellar-only"})
		return
	}
	amount := strings.TrimSpace(c.Query("amount"))
	if value, err := strconv.ParseFloat(amount, 64); err != nil || value <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}
	send := parseAsset(c.Query("from"))
	dest := parseAsset(c.Query("to"))
	if send.Code == dest.Code && send.Issuer == dest.Issuer {
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose two different assets"})
		return
	}
	quote, err := s.chain.SwapQuote(send, amount, dest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, quote)
}

type swapBody struct {
	From        string `json:"from"`         // "XLM" or "CODE:ISSUER"
	To          string `json:"to"`
	Amount      string `json:"amount"`       // exact amount of `from` to send
	SlippageBps int    `json:"slippage_bps"` // acceptable slippage, default 100 (1%)
}

// createSwap quotes, applies slippage protection, builds a PathPaymentStrictSend,
// and dispatches it for MPC signing — reusing the same broadcast path as a send.
func (s *Server) createSwap(c *gin.Context) {
	userID := c.GetString("userID")
	wallet, ok := s.walletFor(userID, c.Param("id"))
	if !ok || wallet.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	if wallet.Chain != "stellar" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "swaps are Stellar-only"})
		return
	}
	var body swapBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	amountValue, err := strconv.ParseFloat(strings.TrimSpace(body.Amount), 64)
	if err != nil || amountValue <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}
	send := parseAsset(body.From)
	dest := parseAsset(body.To)
	if send.Code == dest.Code && send.Issuer == dest.Issuer {
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose two different assets"})
		return
	}

	quote, err := s.chain.SwapQuote(send, strings.TrimSpace(body.Amount), dest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if err := s.store.CreateTransaction(transaction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	if err := s.mpc.Sign(
		wallet.ID, transaction.ID, domain.Chains[wallet.Chain].Network, hash,
	); err != nil {
		s.store.SetTransactionStatus(transaction.ID, "failed")
		c.JSON(http.StatusBadGateway, gin.H{"error": "sign dispatch failed: " + err.Error()})
		return
	}
	s.hub.PublishTransaction(userID, transaction)
	c.JSON(http.StatusOK, gin.H{"transaction": transaction, "quote": quote, "dest_min": destMin})
}
