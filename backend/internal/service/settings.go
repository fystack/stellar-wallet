package service

import "stellar-wallet-backend/internal/chain"

// SettingsService exposes chain status. The Horizon URL comes from config.yaml only.
type SettingsService struct {
	chain *chain.Client
}

func NewSettingsService(chain *chain.Client) *SettingsService {
	return &SettingsService{chain: chain}
}

type Settings struct {
	HorizonURL string `json:"horizonUrl"`
}

type ChainStatus struct {
	Chain   string `json:"chain"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Network string `json:"network"`
	RPC     string `json:"rpc"`
	Online  bool   `json:"online"`
}

func (s *SettingsService) Get() Settings {
	return Settings{HorizonURL: s.chain.HorizonURL()}
}

func (s *SettingsService) Chains() []ChainStatus {
	return []ChainStatus{
		{Chain: "stellar", Name: "Stellar", Symbol: "XLM", Network: "testnet", RPC: s.chain.HorizonURL(), Online: s.chain.PingHorizon()},
	}
}

func (s *SettingsService) Prices() map[string]float64 {
	return s.chain.Prices()
}
