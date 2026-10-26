package handler

import (
	"errors"
	"net/http"
	"stellar-wallet-backend/internal/service"
	"time"

	"stellar-wallet-backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Handler) listWalletTransactions(c *gin.Context) {
	userID := c.GetString("userID")
	walletID := c.Param("id")
	if _, ok := s.store.WalletFor(userID, walletID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, s.payments.TransactionsByWallet(userID, walletID))
}

func (s *Handler) transactionOnChain(c *gin.Context) {
	metadata, err := s.chain.TransactionOnChain(c.Param("hash"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found on chain yet"})
		return
	}
	c.JSON(http.StatusOK, metadata)
}

func (s *Handler) getTransaction(c *gin.Context) {
	transaction, ok := s.payments.TransactionForUser(c.Param("id"), c.GetString("userID"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, transaction)
}

// resolveRecipient turns a G/M/federation recipient into a concrete address
// (+ any memo the federation server requires) so the UI can pre-fill and warn.
func (s *Handler) resolveRecipient(c *gin.Context) {
	resolved, err := s.chain.ResolveRecipient(c.Query("q"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resolved)
}

func (s *Handler) createTransaction(c *gin.Context) {
	var body service.PaymentInput
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	transaction, err := s.paymentService.Create(c.GetString("userID"), body)
	if err != nil {
		status := http.StatusInternalServerError
		var failure *service.PaymentError
		if errors.As(err, &failure) {
			switch failure.Kind {
			case "not_found":
				status = http.StatusNotFound
			case "not_ready":
				status = http.StatusConflict
			case "invalid":
				status = http.StatusBadRequest
			case "unavailable":
				status = http.StatusServiceUnavailable
			case "dispatch":
				status = http.StatusBadGateway
			}
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transaction)
}

type trustlineBody struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
}

func (s *Handler) addTrustline(c *gin.Context) {
	userID := c.GetString("userID")
	wallet, ok := s.store.WalletFor(userID, c.Param("id"))
	if !ok || wallet.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	if wallet.Chain != "stellar" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trustlines are Stellar-only"})
		return
	}
	var body trustlineBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Code == "" || body.Issuer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and issuer required"})
		return
	}

	if s.payments.TrustlineTransactionExists(wallet.ID, body.Code) {
		c.JSON(http.StatusConflict, gin.H{"error": "trustline already added or in progress"})
		return
	}
	if err := s.walletService.RequireNodes(service.SignQuorum, "signing"); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	envelope, hash, err := s.chain.BuildTrustline(wallet, body.Code, body.Issuer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	if err := s.mpc.Sign(
		wallet.ID, transaction.ID, domain.Chains[wallet.Chain].Network, hash,
	); err != nil {
		s.payments.SetTransactionStatus(transaction.ID, "failed")
		c.JSON(http.StatusBadGateway, gin.H{"error": "sign dispatch failed: " + err.Error()})
		return
	}
	s.paymentService.WatchSignTimeout(userID, transaction.ID)
	s.hub.PublishTransaction(userID, transaction)
	c.JSON(http.StatusOK, transaction)
}
