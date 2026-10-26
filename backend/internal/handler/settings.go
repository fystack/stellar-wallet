package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settings *service.SettingsService
	cluster  *cluster.Client
}

func (h *SettingsHandler) getChains(c *gin.Context) {
	httpapi.Success(c, http.StatusOK, h.settings.Chains())
}

func (h *SettingsHandler) getConfig(c *gin.Context) {
	httpapi.Success(c, http.StatusOK, h.settings.Get())
}

type configBody struct {
	HorizonURL string `json:"horizonUrl"`
}

func (h *SettingsHandler) putConfig(c *gin.Context) {
	var body configBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	settings, err := h.settings.SetHorizonURL(body.HorizonURL)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, settings)
}

func (h *SettingsHandler) addAsset(c *gin.Context) {
	var asset domain.CustomAsset
	if err := c.ShouldBindJSON(&asset); err != nil {
		httpapi.Error(c, apperror.InvalidAsset.New())
		return
	}
	assets, err := h.settings.AddAsset(asset)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, gin.H{"assets": assets})
}

func (h *SettingsHandler) removeAsset(c *gin.Context) {
	assets, err := h.settings.RemoveAsset(c.Query("code"), c.Query("issuer"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, gin.H{"assets": assets})
}

func (h *SettingsHandler) getPrices(c *gin.Context) {
	httpapi.Success(c, http.StatusOK, h.settings.Prices())
}

func (h *SettingsHandler) clusterStatus(c *gin.Context) {
	httpapi.Success(c, http.StatusOK, gin.H{"threshold": "2-of-3", "nodes": h.cluster.Nodes()})
}
