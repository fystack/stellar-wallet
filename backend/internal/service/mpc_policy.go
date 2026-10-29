package service

import (
	"fmt"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"

	"go.uber.org/zap"
)

const (
	SignQuorum   = 2 // nodes required to produce a signature (2-of-3)
	KeygenQuorum = 3 // nodes required to generate a full key (all shares)

	signTimeout = 60 * time.Second
	// Measured from dispatch; queued wallets wait their turn before the clock starts.
	keygenTimeout = 3 * time.Minute
)

// requireQuorum fails fast before creating a record that would otherwise get stuck.
func requireQuorum(cluster SigningCluster, quorum int, action string) error {
	online := cluster.OnlineCount()
	if online < quorum {
		return apperror.MPCUnavailable.Wrap(fmt.Errorf("%s needs at least %d MPC nodes online (%d up)", action, quorum, online))
	}
	return nil
}

// ReconcilePending fails rows left mid-flight when below quorum; otherwise mpcium may still re-deliver, so it re-arms the watchdog.
func (s *WalletService) ReconcilePending() {
	generating, err := s.store.GeneratingWallets()
	if err != nil {
		logError("reconcile generating wallets", err)
		return
	}
	keygenOK := s.cluster.OnlineCount() >= KeygenQuorum
	for _, wallet := range generating {
		if keygenOK {
			s.WatchKeygenTimeout(wallet.ID)
			continue
		}
		logError("reconcile wallet="+wallet.ID, s.store.SetWalletStatus(wallet.ID, domain.WalletFailed))
		if updated, found := s.store.WalletByID(wallet.ID); found {
			publishWallet(s.store, s.hub, updated)
		}
		zap.L().Warn("reconcile: failed stuck wallet, cluster below quorum", zap.String("wallet", wallet.ID))
	}
}

// WatchKeygenTimeout fails a wallet still "generating" after the timeout and frees its queue slot.
func (s *WalletService) WatchKeygenTimeout(walletID string) {
	time.AfterFunc(keygenTimeout, func() {
		wallet, ok := s.store.WalletByID(walletID)
		if !ok || wallet.Status != domain.WalletGenerating {
			s.keygenFinished(walletID)
			return
		}
		zap.L().Warn("keygen timeout", zap.String("wallet", walletID))
		s.failKeygen(walletID)
	})
}
