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

func (s *Server) syncIncoming(wallet domain.Wallet) int {
	if wallet.Chain != "stellar" {
		return 0
	}
	payments, err := s.chain.Incoming(wallet.Address)
	if err != nil {
		return 0
	}

	added := 0
	for _, payment := range payments {
		if s.store.IncomingTransactionExists(payment.Hash) {
			continue
		}
		amount, _ := strconv.ParseFloat(payment.Amount, 64)
		createdAt := payment.At
		if createdAt == "" {
			createdAt = time.Now().UTC().Format(time.RFC3339)
		}
		transaction := domain.Transaction{
			ID:           uuid.NewString(),
			WalletID:     wallet.ID,
			UserID:       wallet.UserID,
			Type:         "in",
			Counterparty: payment.From,
			Amount:       strconv.FormatFloat(amount, 'f', 4, 64),
			Symbol:       payment.Symbol,
			Status:       "confirmed",
			TxHash:       payment.Hash,
			CreatedAt:    createdAt,
		}
		_ = s.store.CreateTransaction(transaction)
		s.hub.PublishTransaction(wallet.UserID, transaction)
		added++
	}
	if added > 0 {
		s.refreshWallet(wallet.ID)
	}
	return added
}

func (s *Server) syncWallet(c *gin.Context) {
	wallet, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok || wallet.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"synced": s.syncIncoming(wallet)})
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func (s *Server) listWallets(c *gin.Context) {
	userID := c.GetString("userID")
	wallets, err := s.store.ListWallets(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, wallets)
}

type createWalletBody struct {
	Name  string `json:"name"`
	Chain string `json:"chain"`
}

func (s *Server) createWallet(c *gin.Context) {
	userID := c.GetString("userID")
	var body createWalletBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	metadata, ok := domain.Chains[body.Chain]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = capitalize(body.Chain) + " Wallet"
	}

	wallet := domain.Wallet{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      name,
		Chain:     body.Chain,
		Symbol:    metadata.Symbol,
		Address:   "",
		Balance:   "0.00",
		Status:    "generating",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	err := s.store.CreateWallet(wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	if err := s.mpc.CreateWallet(wallet.ID); err != nil {
		s.store.SetWalletStatus(wallet.ID, "failed")
		c.JSON(http.StatusBadGateway, gin.H{"error": "keygen dispatch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

func (s *Server) getWallet(c *gin.Context) {
	wallet, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

func (s *Server) walletFor(userID, walletID string) (domain.Wallet, bool) {
	return s.store.WalletFor(userID, walletID)
}

func (s *Server) walletBalance(c *gin.Context) {
	wallet, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok || wallet.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"assets":   s.chain.Balances(wallet.Chain, wallet.Address),
		"symbol":   wallet.Symbol,
		"explorer": chain.ExplorerAddress(wallet.Chain, wallet.Address),
	})
	go s.refreshWallet(wallet.ID)
}

func (s *Server) fundWallet(c *gin.Context) {
	wallet, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok || wallet.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	if err := s.chain.Fund(wallet.Chain, wallet.Address); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	balance, _ := s.chain.Balance(wallet.Chain, wallet.Address)
	s.store.SetWalletBalance(wallet.ID, balance)
	if updated, found := s.walletByID(wallet.ID); found {
		s.hub.PublishWallet(c.GetString("userID"), updated)
	}
	s.syncIncoming(wallet)
	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (s *Server) deleteWallet(c *gin.Context) {
	userID := c.GetString("userID")
	walletID := c.Param("id")
	if _, ok := s.walletFor(userID, walletID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	s.store.DeleteWallet(userID, walletID)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) walletByID(walletID string) (domain.Wallet, bool) {
	return s.store.WalletByID(walletID)
}
