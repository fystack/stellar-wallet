package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type WalletHandler struct {
	wallets    *service.WalletService
	trustlines *service.TrustlineService
}

func (h *WalletHandler) list(c *gin.Context) {
	wallets, err := h.wallets.List(c.GetString("userID"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, wallets)
}

type createWalletBody struct {
	Name  string `json:"name"`
	Chain string `json:"chain"`
}

func (h *WalletHandler) create(c *gin.Context) {
	var body createWalletBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	wallet, err := h.wallets.Create(c.GetString("userID"), body.Name, body.Chain)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusCreated, wallet)
}

func (h *WalletHandler) get(c *gin.Context) {
	wallet, err := h.wallets.Get(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, wallet)
}

func (h *WalletHandler) delete(c *gin.Context) {
	if err := h.wallets.Delete(c.GetString("userID"), c.Param("id")); err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *WalletHandler) balance(c *gin.Context) {
	balance, err := h.wallets.Balance(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, balance)
}

func (h *WalletHandler) fund(c *gin.Context) {
	balance, err := h.wallets.Fund(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, gin.H{"balance": balance})
}

func (h *WalletHandler) sync(c *gin.Context) {
	synced, err := h.wallets.Sync(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, gin.H{"synced": synced})
}

type trustlineBody struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
}

func (h *WalletHandler) addTrustline(c *gin.Context) {
	var body trustlineBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidAsset.New())
		return
	}
	transaction, err := h.trustlines.Add(c.GetString("userID"), c.Param("id"), body.Code, body.Issuer)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusCreated, transaction)
}
