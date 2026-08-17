package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stellar-wallet-backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) listWalletTransactions(c *gin.Context) {
	userID := c.GetString("userID")
	walletID := c.Param("id")
	if _, ok := s.walletFor(userID, walletID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, s.store.TransactionsByWallet(userID, walletID))
}

func (s *Server) transactionOnChain(c *gin.Context) {
	metadata, err := s.chain.TransactionOnChain(c.Param("hash"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found on chain yet"})
		return
	}
	c.JSON(http.StatusOK, metadata)
}

func (s *Server) getTransaction(c *gin.Context) {
	transaction, ok := s.store.TransactionForUser(c.Param("id"), c.GetString("userID"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, transaction)
}

type sendBody struct {
	WalletID    string `json:"wallet_id"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Memo        string `json:"memo"`
	MemoType    string `json:"memo_type"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
}

// resolveRecipient turns a G/M/federation recipient into a concrete address
// (+ any memo the federation server requires) so the UI can pre-fill and warn.
func (s *Server) resolveRecipient(c *gin.Context) {
	resolved, err := s.chain.ResolveRecipient(c.Query("q"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resolved)
}

func (s *Server) createTransaction(c *gin.Context) {
	userID := c.GetString("userID")
	var body sendBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	wallet, ok := s.walletFor(userID, body.WalletID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	if wallet.Status != "ready" {
		c.JSON(http.StatusConflict, gin.H{"error": "wallet is not ready yet"})
		return
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(body.Amount), 64)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}
	if strings.TrimSpace(body.To) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipient required"})
		return
	}

	// Resolve federation / muxed recipients to a concrete address. If the
	// server requires a memo and the user didn't supply one, adopt it.
	memo, memoType := strings.TrimSpace(body.Memo), body.MemoType
	destination := strings.TrimSpace(body.To)
	if wallet.Chain == "stellar" {
		resolved, resolveErr := s.chain.ResolveRecipient(destination)
		if resolveErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": resolveErr.Error()})
			return
		}
		destination = resolved.Address
		if resolved.MemoType != "" {
			memoType = resolved.MemoType
			if memo == "" {
				memo = resolved.Memo
			}
		}
	}

	symbol := body.AssetCode
	if symbol == "" {
		symbol = wallet.Symbol
	}
	transaction := domain.Transaction{
		ID:           uuid.NewString(),
		WalletID:     wallet.ID,
		UserID:       userID,
		Type:         "out",
		Counterparty: destination,
		Amount:       strconv.FormatFloat(amount, 'f', 2, 64),
		Symbol:       symbol,
		Memo:         memo,
		Status:       "signing",
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	var payload []byte
	if wallet.Chain == "stellar" {
		envelope, hash, buildErr := s.chain.BuildPayment(
			wallet, transaction.Counterparty, transaction.Amount,
			body.AssetCode, body.AssetIssuer, transaction.Memo, memoType,
		)
		if buildErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": buildErr.Error()})
			return
		}
		transaction.EnvelopeXDR = envelope
		payload = hash
	} else {
		payload = []byte(fmt.Sprintf(
			"%s|%s|%s|%s", wallet.ID, transaction.Counterparty, transaction.Amount, transaction.Memo,
		))
	}

	err = s.store.CreateTransaction(transaction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}

	if err := s.mpc.Sign(
		wallet.ID, transaction.ID, domain.Chains[wallet.Chain].Network, payload,
	); err != nil {
		s.store.SetTransactionStatus(transaction.ID, "failed")
		c.JSON(http.StatusBadGateway, gin.H{"error": "sign dispatch failed: " + err.Error()})
		return
	}
	s.hub.PublishTransaction(userID, transaction)
	c.JSON(http.StatusOK, transaction)
}

type trustlineBody struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
}

func (s *Server) addTrustline(c *gin.Context) {
	userID := c.GetString("userID")
	wallet, ok := s.walletFor(userID, c.Param("id"))
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

	if s.store.TrustlineTransactionExists(wallet.ID, body.Code) {
		c.JSON(http.StatusConflict, gin.H{"error": "trustline already added or in progress"})
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
	err = s.store.CreateTransaction(transaction)
	if err != nil {
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
	c.JSON(http.StatusOK, transaction)
}
