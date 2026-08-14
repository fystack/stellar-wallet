package main

import (
	"sync"
	"time"
)

const (
	balanceInterval    = 30 * time.Second
	balanceConcurrency = 8 // cap simultaneous RPC calls regardless of wallet count
)

// startBalanceRefresher keeps cached balances fresh in the background so the
// wallet list never has to hit chain RPC on load (scales to many wallets).
func (s *server) startBalanceRefresher() {
	go func() {
		for {
			s.refreshAllBalances()
			time.Sleep(balanceInterval)
		}
	}()
}

type walletRow struct{ id, user, chain, addr, bal string }

func (s *server) refreshAllBalances() {
	rows, err := s.db.Query(
		`SELECT id, user_id, chain, address, balance FROM wallets WHERE status = 'ready'`)
	if err != nil {
		return
	}
	var list []walletRow
	for rows.Next() {
		var w walletRow
		if rows.Scan(&w.id, &w.user, &w.chain, &w.addr, &w.bal) == nil {
			list = append(list, w)
		}
	}
	rows.Close()

	sem := make(chan struct{}, balanceConcurrency)
	var wg sync.WaitGroup
	for _, w := range list {
		wg.Add(1)
		sem <- struct{}{}
		go func(w walletRow) {
			defer wg.Done()
			defer func() { <-sem }()
			s.refreshOne(w)
		}(w)
	}
	wg.Wait()
}

// refreshWallet updates a single wallet's cached balance (e.g. after a tx).
func (s *server) refreshWallet(id string) {
	var w walletRow
	err := s.db.QueryRow(
		`SELECT id, user_id, chain, address, balance FROM wallets WHERE id = ?`, id).
		Scan(&w.id, &w.user, &w.chain, &w.addr, &w.bal)
	if err == nil {
		s.refreshOne(w)
	}
}

// refreshWalletByAddress refreshes any local wallet holding this address
// (e.g. the recipient of a transfer between the user's own wallets).
func (s *server) refreshWalletByAddress(addr string) {
	var w walletRow
	err := s.db.QueryRow(
		`SELECT id, user_id, chain, address, balance FROM wallets WHERE address = ?`, addr).
		Scan(&w.id, &w.user, &w.chain, &w.addr, &w.bal)
	if err == nil {
		s.refreshOne(w)
	}
}

// refreshAfterTx updates sender + recipient balances, with one retry to beat
// Horizon's brief indexing lag right after a broadcast.
func (s *server) refreshAfterTx(senderID, destAddr string) {
	s.refreshWallet(senderID)
	s.refreshWalletByAddress(destAddr)
	time.Sleep(3 * time.Second)
	s.refreshWallet(senderID)
	s.refreshWalletByAddress(destAddr)
}

func (s *server) refreshOne(w walletRow) {
	bal, err := getBalance(w.chain, w.addr)
	if err != nil || bal == w.bal {
		return
	}
	s.db.Exec(`UPDATE wallets SET balance = ? WHERE id = ?`, bal, w.id)
	if ww, ok := s.walletByID(w.id); ok {
		s.hub.publishWallet(w.user, ww) // push the new balance to open clients
	}
}
