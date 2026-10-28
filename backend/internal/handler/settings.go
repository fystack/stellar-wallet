package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/cluster"
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

func (h *SettingsHandler) getPrices(c *gin.Context) {
	httpapi.Success(c, http.StatusOK, h.settings.Prices())
}

func (h *SettingsHandler) clusterStatus(c *gin.Context) {
	httpapi.Success(c, http.StatusOK, gin.H{"threshold": "2-of-3", "nodes": h.cluster.Nodes()})
}
