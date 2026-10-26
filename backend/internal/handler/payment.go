package handler

import (
	"fmt"
	"net/http"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Handler) listWalletTransactions(c *gin.Context) {
	userID := c.GetString("userID")
	walletID := c.Param("id")
	if _, ok := s.store.WalletFor(userID, walletID); !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "wallet_not_found", "Wallet not found."))
		return
	}
	httpapi.Success(c, http.StatusOK, s.payments.TransactionsByWallet(userID, walletID))
}

func (s *Handler) transactionOnChain(c *gin.Context) {
	metadata, err := s.chain.TransactionOnChain(c.Param("hash"))
	if err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.NotFound, "transaction_not_found", "Transaction not found on chain yet."))
		return
	}
	httpapi.Success(c, http.StatusOK, metadata)
}

func (s *Handler) getTransaction(c *gin.Context) {
	transaction, ok := s.payments.TransactionForUser(c.Param("id"), c.GetString("userID"))
	if !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "transaction_not_found", "Transaction not found."))
		return
	}
	httpapi.Success(c, http.StatusOK, transaction)
}

// resolveRecipient turns a G/M/federation recipient into a concrete address
// (+ any memo the federation server requires) so the UI can pre-fill and warn.
func (s *Handler) resolveRecipient(c *gin.Context) {
	resolved, err := s.chain.ResolveRecipient(c.Query("q"))
	if err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Invalid, "recipient_resolution_failed", "Unable to resolve recipient. Check the address and try again."))
		return
	}
	httpapi.Success(c, http.StatusOK, resolved)
}

func (s *Handler) createTransaction(c *gin.Context) {
	var body service.PaymentInput
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_request", "Invalid request body."))
		return
	}
	transaction, err := s.paymentService.Create(c.GetString("userID"), body)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusCreated, transaction)
}

type trustlineBody struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
}

func (s *Handler) addTrustline(c *gin.Context) {
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
		httpapi.Error(c, apperror.New(apperror.Invalid, "unsupported_chain", "Trustlines are only supported on Stellar."))
		return
	}
	var body trustlineBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Code == "" || body.Issuer == "" {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_asset", "Asset code and issuer are required."))
		return
	}

	if s.payments.TrustlineTransactionExists(wallet.ID, body.Code) {
		httpapi.Error(c, apperror.New(apperror.Conflict, "trustline_exists", "Trustline already exists or is in progress."))
		return
	}
	if err := s.walletService.RequireNodes(service.SignQuorum, "signing"); err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Unavailable, "mpc_unavailable", "Not enough MPC nodes are online. Please try again later."))
		return
	}

	envelope, hash, err := s.chain.BuildTrustline(wallet, body.Code, body.Issuer)
	if err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Invalid, "trustline_build_failed", "Unable to prepare trustline. Check the asset and wallet balance."))
		return
	}
	transaction := domain.Transaction{
		ID:           uuid.NewString(),
		WalletID:     wallet.ID,
		UserID:       userID,
		Type:         "out",
		Counterparty: body.Issuer,
		Amount:       "0.00",
		Symbol:       body.Code,
		Memo:         "Add trustline",
		Status:       "signing",
		EnvelopeXDR:  envelope,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	err = s.payments.CreateTransaction(transaction)
	if err != nil {
		httpapi.Error(c, fmt.Errorf("persist trustline: %w", err))
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
	httpapi.Success(c, http.StatusCreated, transaction)
}
