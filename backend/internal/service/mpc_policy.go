package service

import (
	"fmt"
	"log"
	"time"
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

// requireNodes checks the live node count against a quorum. It returns a
// user-facing error when the cluster is short, so handlers can fail fast
// before creating a record that would otherwise get stuck.
func (s *WalletService) RequireNodes(quorum int, action string) error {
	online := s.cluster.OnlineCount()
	if online < quorum {
		return fmt.Errorf("%s needs at least %d MPC nodes online (%d up)", action, quorum, online)
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
		s.store.SetWalletStatus(wallet.ID, "failed")
		if updated, found := s.store.WalletByID(wallet.ID); found {
			s.hub.PublishWallet(wallet.UserID, updated)
		}
		log.Printf("reconcile: failed stuck wallet=%s (cluster below quorum)", wallet.ID)
	}
}

// watchKeygenTimeout fails a wallet still "generating" after the timeout.
func (s *WalletService) WatchKeygenTimeout(userID, walletID string) {
	time.AfterFunc(keygenTimeout, func() {
		wallet, ok := s.store.WalletByID(walletID)
		if !ok || wallet.Status != "generating" {
			return
		}
		s.store.SetWalletStatus(walletID, "failed")
		if updated, found := s.store.WalletByID(walletID); found {
			s.hub.PublishWallet(userID, updated)
		}
	})
}
