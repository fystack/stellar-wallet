package app

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
	signQuorum   = 2 // nodes required to produce a signature (2-of-3)
	keygenQuorum = 3 // nodes required to generate a full key (all shares)

	signTimeout   = 60 * time.Second
	keygenTimeout = 90 * time.Second
)

// requireNodes checks the live node count against a quorum. It returns a
// user-facing error when the cluster is short, so handlers can fail fast
// before creating a record that would otherwise get stuck.
func (s *Server) requireNodes(quorum int, action string) error {
	online := s.cluster.OnlineCount()
	if online < quorum {
		return fmt.Errorf("%s needs at least %d MPC nodes online (%d up)", action, quorum, online)
	}
	return nil
}

// watchSignTimeout fails a transaction still stuck in "signing" after the
// timeout — covers nodes that die (or never respond) after dispatch.
func (s *Server) watchSignTimeout(userID, transactionID string) {
	time.AfterFunc(signTimeout, func() {
		transaction, ok := s.store.TransactionByID(transactionID)
		if !ok || transaction.Status != "signing" {
			return
		}
		s.store.FailTransaction(transactionID, "signing timed out — MPC nodes did not respond")
		s.publishTransactionByID(userID, transactionID)
	})
}

// ReconcilePending recovers records left mid-flight by a previous run. On boot
// there is no in-flight NATS request behind a "signing"/"generating" row, so if
// the cluster is below quorum they can never complete — fail them immediately.
// If quorum is met, arm a fresh watchdog (mpcium may re-deliver from JetStream).
func (s *Server) ReconcilePending() {
	signOK := s.cluster.OnlineCount() >= signQuorum
	keygenOK := s.cluster.OnlineCount() >= keygenQuorum

	for _, transaction := range s.store.PendingTransactions() {
		if signOK {
			s.watchSignTimeout(transaction.UserID, transaction.ID)
			continue
		}
		s.store.FailTransaction(transaction.ID, "signing aborted — MPC cluster below quorum")
		s.publishTransactionByID(transaction.UserID, transaction.ID)
		log.Printf("reconcile: failed stuck tx=%s (cluster below quorum)", transaction.ID)
	}

	for _, wallet := range s.store.GeneratingWallets() {
		if keygenOK {
			s.watchKeygenTimeout(wallet.UserID, wallet.ID)
			continue
		}
		s.store.SetWalletStatus(wallet.ID, "failed")
		if updated, found := s.walletByID(wallet.ID); found {
			s.hub.PublishWallet(wallet.UserID, updated)
		}
		log.Printf("reconcile: failed stuck wallet=%s (cluster below quorum)", wallet.ID)
	}
}

// watchKeygenTimeout fails a wallet still "generating" after the timeout.
func (s *Server) watchKeygenTimeout(userID, walletID string) {
	time.AfterFunc(keygenTimeout, func() {
		wallet, ok := s.walletByID(walletID)
		if !ok || wallet.Status != "generating" {
			return
		}
		s.store.SetWalletStatus(walletID, "failed")
		if updated, found := s.walletByID(walletID); found {
			s.hub.PublishWallet(userID, updated)
		}
	})
}
