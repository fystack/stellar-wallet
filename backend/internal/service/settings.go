package service

import (
	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
)

const horizonURLSetting = "horizon_url"

// SettingsService owns runtime chain settings: the Horizon RPC and custom assets.
type SettingsService struct {
	store repository.Store
	chain *chain.Client
}

func NewSettingsService(store repository.Store, chain *chain.Client) *SettingsService {
	return &SettingsService{store: store, chain: chain}
}

type Settings struct {
	HorizonURL string               `json:"horizonUrl"`
	Assets     []domain.CustomAsset `json:"assets"`
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
	return Settings{HorizonURL: s.horizonURL(), Assets: s.store.CustomAssets()}
}

func (s *SettingsService) SetHorizonURL(url string) (Settings, error) {
	if url != "" {
		if err := s.store.SetSetting(horizonURLSetting, url); err != nil {
			return Settings{}, err
		}
	}
	s.ApplyRPC()
	return s.Get(), nil
}

// ApplyRPC points the chain client at the stored Horizon URL.
func (s *SettingsService) ApplyRPC() {
	s.chain.SetRPC(s.horizonURL())
}

func (s *SettingsService) Chains() []ChainStatus {
	return []ChainStatus{
		{Chain: "stellar", Name: "Stellar", Symbol: "XLM", Network: "testnet", RPC: s.horizonURL(), Online: s.chain.PingHorizon()},
	}
}

func (s *SettingsService) AddAsset(asset domain.CustomAsset) ([]domain.CustomAsset, error) {
	if asset.Code == "" || asset.Issuer == "" {
		return nil, apperror.InvalidAsset.New()
	}
	if err := s.store.AddAsset(asset); err != nil {
		return nil, err
	}
	return s.store.CustomAssets(), nil
}

func (s *SettingsService) RemoveAsset(code, issuer string) ([]domain.CustomAsset, error) {
	if err := s.store.RemoveAsset(code, issuer); err != nil {
		return nil, err
	}
	return s.store.CustomAssets(), nil
}

func (s *SettingsService) Prices() map[string]float64 {
	return s.chain.Prices()
}

func (s *SettingsService) horizonURL() string {
	if value, ok := s.store.Setting(horizonURLSetting); ok {
		return value
	}
	return chain.DefaultHorizonURL
}
