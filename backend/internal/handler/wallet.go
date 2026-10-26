package handler

import (
	"fmt"
	"net/http"
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

func (s *Handler) syncWallet(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "wallet_not_found", "Wallet not found."))
		return
	}
	if wallet.Status != "ready" {
		httpapi.Error(c, apperror.New(apperror.Conflict, "wallet_not_ready", "Wallet is not ready yet."))
		return
	}
	httpapi.Success(c, http.StatusOK, gin.H{"synced": s.walletService.SyncIncoming(wallet)})
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
		httpapi.Error(c, fmt.Errorf("list wallets: %w", err))
		return
	}
	httpapi.Success(c, http.StatusOK, wallets)
}

type createWalletBody struct {
	Name  string `json:"name"`
	Chain string `json:"chain"`
}

func (s *Handler) createWallet(c *gin.Context) {
	userID := c.GetString("userID")
	var body createWalletBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_request", "Invalid request body."))
		return
	}
	metadata, ok := domain.Chains[body.Chain]
	if !ok {
		httpapi.Error(c, apperror.New(apperror.Invalid, "unsupported_chain", "Unsupported chain."))
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = capitalize(body.Chain) + " Wallet"
	}

	if err := s.walletService.RequireNodes(service.KeygenQuorum, "keygen"); err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Unavailable, "mpc_unavailable", "Not enough MPC nodes are online. Please try again later."))
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
		httpapi.Error(c, fmt.Errorf("persist wallet: %w", err))
		return
	}
	if err := s.mpc.CreateWallet(wallet.ID); err != nil {
		s.store.SetWalletStatus(wallet.ID, "failed")
		httpapi.Error(c, apperror.Wrap(err, apperror.Upstream, "keygen_dispatch_failed", "Unable to start wallet creation."))
		return
	}
	s.walletService.WatchKeygenTimeout(userID, wallet.ID)
	httpapi.Success(c, http.StatusCreated, wallet)
}

func (s *Handler) getWallet(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "wallet_not_found", "Wallet not found."))
		return
	}
	httpapi.Success(c, http.StatusOK, wallet)
}

func (s *Handler) walletBalance(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "wallet_not_found", "Wallet not found."))
		return
	}
	if wallet.Status != "ready" {
		httpapi.Error(c, apperror.New(apperror.Conflict, "wallet_not_ready", "Wallet is not ready yet."))
		return
	}
	httpapi.Success(c, http.StatusOK, gin.H{
		"assets":   s.chain.Balances(wallet.Chain, wallet.Address),
		"symbol":   wallet.Symbol,
		"explorer": chain.ExplorerAddress(wallet.Chain, wallet.Address),
	})
	go s.walletService.RefreshWallet(wallet.ID)
}

func (s *Handler) fundWallet(c *gin.Context) {
	wallet, ok := s.store.WalletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "wallet_not_found", "Wallet not found."))
		return
	}
	if wallet.Status != "ready" {
		httpapi.Error(c, apperror.New(apperror.Conflict, "wallet_not_ready", "Wallet is not ready yet."))
		return
	}
	if err := s.chain.Fund(wallet.Chain, wallet.Address); err != nil {
		httpapi.Error(c, apperror.Wrap(err, apperror.Upstream, "funding_failed", "Unable to fund the wallet. Please try again later."))
		return
	}
	balance, _ := s.chain.Balance(wallet.Chain, wallet.Address)
	s.store.SetWalletBalance(wallet.ID, balance)
	if updated, found := s.store.WalletByID(wallet.ID); found {
		s.hub.PublishWallet(c.GetString("userID"), updated)
	}
	s.walletService.SyncIncoming(wallet)
	httpapi.Success(c, http.StatusOK, gin.H{"balance": balance})
}

func (s *Handler) deleteWallet(c *gin.Context) {
	userID := c.GetString("userID")
	walletID := c.Param("id")
	if _, ok := s.store.WalletFor(userID, walletID); !ok {
		httpapi.Error(c, apperror.New(apperror.NotFound, "wallet_not_found", "Wallet not found."))
		return
	}
	s.store.DeleteWallet(userID, walletID)
	httpapi.Success(c, http.StatusOK, gin.H{"ok": true})
}
