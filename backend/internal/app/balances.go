package app

import (
	"sync"
	"time"

	"stellar-wallet-backend/internal/domain"
)

const (
	balanceInterval    = 30 * time.Second
	balanceConcurrency = 8
)

func (s *Server) StartBalanceRefresher() {
	go func() {
		for {
			s.refreshAllBalances()
			time.Sleep(balanceInterval)
		}
	}()
}

func (s *Server) refreshAllBalances() {
	wallets := s.store.ReadyWallets()

	semaphore := make(chan struct{}, balanceConcurrency)
	var waitGroup sync.WaitGroup
	for _, wallet := range wallets {
		waitGroup.Add(1)
		semaphore <- struct{}{}
		go func(wallet domain.Wallet) {
			defer waitGroup.Done()
			defer func() { <-semaphore }()
			s.refreshOne(wallet)
		}(wallet)
	}
	waitGroup.Wait()
}

func (s *Server) refreshWallet(walletID string) {
	if wallet, ok := s.store.WalletByID(walletID); ok {
		s.refreshOne(wallet)
	}
}

func (s *Server) refreshWalletByAddress(address string) {
	if wallet, ok := s.store.WalletByAddress(address); ok {
		s.refreshOne(wallet)
	}
}

func (s *Server) refreshAfterTransaction(senderID, destinationAddress string) {
	s.refreshWallet(senderID)
	s.refreshWalletByAddress(destinationAddress)
	time.Sleep(3 * time.Second)
	s.refreshWallet(senderID)
	s.refreshWalletByAddress(destinationAddress)
}

func (s *Server) refreshOne(wallet domain.Wallet) {
	balance, err := s.chain.Balance(wallet.Chain, wallet.Address)
	if err != nil || balance == wallet.Balance {
		return
	}
	s.store.SetWalletBalance(wallet.ID, balance)
	if updated, ok := s.walletByID(wallet.ID); ok {
		s.hub.PublishWallet(wallet.UserID, updated)
	}
}
