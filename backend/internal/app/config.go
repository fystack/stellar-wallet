package app

import (
	"net/http"

	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"

	"github.com/gin-gonic/gin"
)

func (s *Server) setting(key, fallback string) string {
	if value, ok := s.store.Setting(key); ok {
		return value
	}
	return fallback
}

func (s *Server) setSetting(key, value string) {
	s.store.SetSetting(key, value)
}

func (s *Server) horizonURL() string {
	return s.setting("horizon_url", chain.DefaultHorizonURL)
}

func (s *Server) solanaURL() string {
	return s.setting("solana_url", chain.DefaultSolanaURL)
}

func (s *Server) customAssets() []domain.CustomAsset {
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

func (s *Server) getChains(c *gin.Context) {
	c.JSON(http.StatusOK, []chainStatus{
		{"stellar", "Stellar", "XLM", "testnet", s.horizonURL(), s.chain.PingHorizon()},
		{"solana", "Solana", "SOL", "devnet", s.solanaURL(), s.chain.PingSolana()},
	})
}

func (s *Server) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"horizonUrl": s.horizonURL(),
		"solanaUrl":  s.solanaURL(),
		"assets":     s.customAssets(),
	})
}

type configBody struct {
	HorizonURL string `json:"horizonUrl"`
	SolanaURL  string `json:"solanaUrl"`
}

func (s *Server) putConfig(c *gin.Context) {
	var body configBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.HorizonURL != "" {
		s.setSetting("horizon_url", body.HorizonURL)
	}
	if body.SolanaURL != "" {
		s.setSetting("solana_url", body.SolanaURL)
	}
	s.ApplyRPCConfig()
	s.getConfig(c)
}

func (s *Server) ApplyRPCConfig() {
	s.chain.SetRPC(s.horizonURL(), s.solanaURL())
}

func (s *Server) addAsset(c *gin.Context) {
	var asset domain.CustomAsset
	if err := c.ShouldBindJSON(&asset); err != nil || asset.Code == "" || asset.Issuer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and issuer required"})
		return
	}
	s.store.AddAsset(asset)
	c.JSON(http.StatusOK, gin.H{"assets": s.customAssets()})
}

func (s *Server) removeAsset(c *gin.Context) {
	s.store.RemoveAsset(c.Query("code"), c.Query("issuer"))
	c.JSON(http.StatusOK, gin.H{"assets": s.customAssets()})
}

func (s *Server) getPrices(c *gin.Context) {
	c.JSON(http.StatusOK, s.chain.Prices())
}
