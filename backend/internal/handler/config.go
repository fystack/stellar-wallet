package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"

	"github.com/gin-gonic/gin"
)

func (s *Handler) setting(key, fallback string) string {
	if value, ok := s.store.Setting(key); ok {
		return value
	}
	return fallback
}

func (s *Handler) setSetting(key, value string) {
	s.store.SetSetting(key, value)
}

func (s *Handler) horizonURL() string {
	return s.setting("horizon_url", chain.DefaultHorizonURL)
}

func (s *Handler) customAssets() []domain.CustomAsset {
	return s.store.CustomAssets()
}

type chainStatus struct {
	Chain   string `json:"chain"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Network string `json:"network"`
	RPC     string `json:"rpc"`
	Online  bool   `json:"online"`
}

func (s *Handler) getChains(c *gin.Context) {
	c.JSON(http.StatusOK, []chainStatus{
		{"stellar", "Stellar", "XLM", "testnet", s.horizonURL(), s.chain.PingHorizon()},
	})
}

func (s *Handler) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"horizonUrl": s.horizonURL(),
		"assets":     s.customAssets(),
	})
}

type configBody struct {
	HorizonURL string `json:"horizonUrl"`
}

func (s *Handler) putConfig(c *gin.Context) {
	var body configBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.HorizonURL != "" {
		s.setSetting("horizon_url", body.HorizonURL)
	}
	s.ApplyRPCConfig()
	s.getConfig(c)
}

func (s *Handler) ApplyRPCConfig() {
	s.chain.SetRPC(s.horizonURL())
}

func (s *Handler) addAsset(c *gin.Context) {
	var asset domain.CustomAsset
	if err := c.ShouldBindJSON(&asset); err != nil || asset.Code == "" || asset.Issuer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and issuer required"})
		return
	}
	s.store.AddAsset(asset)
	c.JSON(http.StatusOK, gin.H{"assets": s.customAssets()})
}

func (s *Handler) removeAsset(c *gin.Context) {
	s.store.RemoveAsset(c.Query("code"), c.Query("issuer"))
	c.JSON(http.StatusOK, gin.H{"assets": s.customAssets()})
}

func (s *Handler) getPrices(c *gin.Context) {
	c.JSON(http.StatusOK, s.chain.Prices())
}
