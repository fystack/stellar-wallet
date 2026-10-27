package service

import (
	"fmt"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"

	"go.uber.org/zap"
)

// MPC quorum + timeout policy.
//
// A threshold signature is 2-of-3: signing only needs a majority of nodes,
// but keygen must reach every node so each holds a share. If the cluster is
// short of quorum we reject up front; if it goes quiet mid-flight the watchdog
// fails the pending record so the UI never hangs on "Signing".
const (
	SignQuorum   = 2 // nodes required to produce a signature (2-of-3)
	KeygenQuorum = 3 // nodes required to generate a full key (all shares)

	signTimeout = 60 * time.Second
	// mpcium queues keygens (about 10 s each, a few at a time), so a burst of new wallets waits in line.
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

// ReconcilePending recovers records left mid-flight by a previous run. On boot
// there is no in-flight NATS request behind a "signing"/"generating" row, so if
// the cluster is below quorum they can never complete — fail them immediately.
// If quorum is met, arm a fresh watchdog (mpcium may re-deliver from JetStream).
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

// WatchKeygenTimeout fails a wallet still "generating" after the timeout.
func (s *WalletService) WatchKeygenTimeout(walletID string) {
	time.AfterFunc(keygenTimeout, func() {
		wallet, ok := s.store.WalletByID(walletID)
		if !ok || wallet.Status != domain.WalletGenerating {
			return
		}
		logError("keygen timeout wallet="+walletID, s.store.SetWalletStatus(walletID, domain.WalletFailed))
		if updated, found := s.store.WalletByID(walletID); found {
			publishWallet(s.store, s.hub, updated)
		}
	})
}
