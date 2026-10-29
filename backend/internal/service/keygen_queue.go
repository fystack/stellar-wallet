package service

import (
	"stellar-wallet-backend/internal/domain"

	"go.uber.org/zap"
)

// mpcium runs keygens back to back; sending them one at a time keeps it from being flooded.
const maxInflightKeygens = 1

func (s *WalletService) enqueueKeygen(walletID string) {
	s.dispatchKeygens(s.keygens.push(walletID))
}

func (s *WalletService) keygenFinished(walletID string) {
	s.dispatchKeygens(s.keygens.done(walletID))
}

func (s *WalletService) dispatchKeygens(walletIDs []string) {
	for _, walletID := range walletIDs {
		s.dispatchKeygen(walletID)
	}
}

func (s *WalletService) dispatchKeygen(walletID string) {
	wallet, ok := s.store.WalletByID(walletID)
	if !ok || wallet.Status != domain.WalletGenerating {
		s.keygenFinished(walletID)
		return
	}
	if err := s.mpc.CreateWallet(walletID); err != nil {
		zap.L().Warn("keygen dispatch failed", zap.String("wallet", walletID), zap.Error(err))
		s.failKeygen(walletID)
		return
	}
	s.WatchKeygenTimeout(walletID)
}

func (s *WalletService) failKeygen(walletID string) {
	logError("fail keygen wallet="+walletID, s.store.SetWalletStatus(walletID, domain.WalletFailed))
	if updated, found := s.store.WalletByID(walletID); found {
		publishWallet(s.store, s.hub, updated)
	}
	s.keygenFinished(walletID)
}
