package service

import (
	"fmt"
	"log"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
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

	signTimeout   = 60 * time.Second
	keygenTimeout = 90 * time.Second
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
	keygenOK := s.cluster.OnlineCount() >= KeygenQuorum

	for _, wallet := range s.store.GeneratingWallets() {
		if keygenOK {
			s.WatchKeygenTimeout(wallet.UserID, wallet.ID)
			continue
		}
		logError("reconcile wallet="+wallet.ID, s.store.SetWalletStatus(wallet.ID, domain.WalletFailed))
		if updated, found := s.store.WalletByID(wallet.ID); found {
			s.hub.PublishWallet(wallet.UserID, updated)
		}
		log.Printf("reconcile: failed stuck wallet=%s (cluster below quorum)", wallet.ID)
	}
}

// WatchKeygenTimeout fails a wallet still "generating" after the timeout.
func (s *WalletService) WatchKeygenTimeout(userID, walletID string) {
	time.AfterFunc(keygenTimeout, func() {
		wallet, ok := s.store.WalletByID(walletID)
		if !ok || wallet.Status != domain.WalletGenerating {
			return
		}
		logError("keygen timeout wallet="+walletID, s.store.SetWalletStatus(walletID, domain.WalletFailed))
		if updated, found := s.store.WalletByID(walletID); found {
			s.hub.PublishWallet(userID, updated)
		}
	})
}
