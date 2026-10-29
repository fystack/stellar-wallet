package service

import (
	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
)

// SettingsService exposes chain status and manages custom assets. The Horizon URL comes from config.yaml only.
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

func (s *SettingsService) Get() (Settings, error) {
	assets, err := s.store.CustomAssets()
	if err != nil {
		return Settings{}, err
	}
	return Settings{HorizonURL: s.chain.HorizonURL(), Assets: assets}, nil
}

func (s *SettingsService) Chains() []ChainStatus {
	return []ChainStatus{
		{Chain: "stellar", Name: "Stellar", Symbol: "XLM", Network: "testnet", RPC: s.chain.HorizonURL(), Online: s.chain.PingHorizon()},
	}
}

func (s *SettingsService) AddAsset(asset domain.CustomAsset) ([]domain.CustomAsset, error) {
	if asset.Code == "" || asset.Issuer == "" {
		return nil, apperror.InvalidAsset.New()
	}
	if err := s.store.AddAsset(asset); err != nil {
		return nil, err
	}
	return s.store.CustomAssets()
}

func (s *SettingsService) RemoveAsset(code, issuer string) ([]domain.CustomAsset, error) {
	if err := s.store.RemoveAsset(code, issuer); err != nil {
		return nil, err
	}
	return s.store.CustomAssets()
}

func (s *SettingsService) Prices() map[string]float64 {
	return s.chain.Prices()
}
