package handler

import (
	"net/http"
	"stellar-wallet-backend/internal/service"
	"strings"
	"time"

	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Handler) syncWallet(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
	if !ok || wallet.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"synced": s.walletService.SyncIncoming(wallet)})
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func (s *Handler) listWallets(c *gin.Context) {
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

func (s *Handler) createWallet(c *gin.Context) {
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

	if err := s.walletService.RequireNodes(service.KeygenQuorum, "keygen"); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
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
	s.walletService.WatchKeygenTimeout(userID, wallet.ID)
	c.JSON(http.StatusOK, wallet)
}

func (s *Handler) getWallet(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

func (s *Handler) walletBalance(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
	if !ok || wallet.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"assets":   s.chain.Balances(wallet.Chain, wallet.Address),
		"symbol":   wallet.Symbol,
		"explorer": chain.ExplorerAddress(wallet.Chain, wallet.Address),
	})
	go s.walletService.RefreshWallet(wallet.ID)
}

func (s *Handler) fundWallet(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
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
	if updated, found := s.store.WalletByID(wallet.ID); found {
		s.hub.PublishWallet(c.GetString("userID"), updated)
	}
	s.walletService.SyncIncoming(wallet)
	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (s *Handler) deleteWallet(c *gin.Context) {
	userID := c.GetString("userID")
	walletID := c.Param("id")
	if _, ok := s.store.WalletFor(userID, walletID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	s.store.DeleteWallet(userID, walletID)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
