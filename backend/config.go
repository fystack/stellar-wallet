package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setting reads a settings value, falling back to def.
func (s *server) setting(key, def string) string {
	var v string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v); err != nil || v == "" {
		return def
	}
	return v
}

func (s *server) setSetting(key, value string) {
	s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
}

func (s *server) horizonURL() string { return s.setting("horizon_url", horizonTestnet) }
func (s *server) solanaURL() string  { return s.setting("solana_url", solanaDevnet) }

type CustomAsset struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
}

func (s *server) customAssets() []CustomAsset {
	rows, err := s.db.Query(`SELECT code, issuer FROM custom_assets ORDER BY code`)
	if err != nil {
		return []CustomAsset{}
	}
	defer rows.Close()
	out := []CustomAsset{}
	for rows.Next() {
		var a CustomAsset
		if rows.Scan(&a.Code, &a.Issuer) == nil {
			out = append(out, a)
		}
	}
	return out
}

// --- handlers ---

type ChainStatus struct {
	Chain   string `json:"chain"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Network string `json:"network"`
	RPC     string `json:"rpc"`
	Online  bool   `json:"online"`
}

func (s *server) getChains(c *gin.Context) {
	c.JSON(http.StatusOK, []ChainStatus{
		{"stellar", "Stellar", "XLM", "testnet", s.horizonURL(), pingHorizon()},
		{"solana", "Solana", "SOL", "devnet", s.solanaURL(), pingSolana()},
	})
}

func (s *server) getConfig(c *gin.Context) {
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

func (s *server) putConfig(c *gin.Context) {
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
	s.applyRPCConfig()
	s.getConfig(c)
}

// applyRPCConfig pushes the stored RPC endpoints into the active chain clients.
func (s *server) applyRPCConfig() {
	activeHorizonURL = s.horizonURL()
	activeSolanaURL = s.solanaURL()
}

func (s *server) addAsset(c *gin.Context) {
	var a CustomAsset
	if err := c.ShouldBindJSON(&a); err != nil || a.Code == "" || a.Issuer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and issuer required"})
		return
	}
	s.db.Exec(`INSERT OR IGNORE INTO custom_assets (code, issuer) VALUES (?, ?)`, a.Code, a.Issuer)
	c.JSON(http.StatusOK, gin.H{"assets": s.customAssets()})
}

func (s *server) removeAsset(c *gin.Context) {
	s.db.Exec(`DELETE FROM custom_assets WHERE code = ? AND issuer = ?`,
		c.Query("code"), c.Query("issuer"))
	c.JSON(http.StatusOK, gin.H{"assets": s.customAssets()})
}
