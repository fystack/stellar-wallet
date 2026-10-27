package service

import (
	"sync"
	"time"

	"stellar-wallet-backend/internal/domain"
)

const (
	balanceInterval    = 30 * time.Second
	balanceConcurrency = 8
)

func (s *WalletService) StartBalanceRefresher() {
	go func() {
		for {
			s.refreshAllBalances()
			time.Sleep(balanceInterval)
		}
	}()
}

func (s *WalletService) refreshAllBalances() {
	wallets, err := s.store.ReadyWallets()
	if err != nil {
		logError("refresh balances", err)
		return
	}

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

func (s *WalletService) RefreshWallet(walletID string) {
	if wallet, ok := s.store.WalletByID(walletID); ok {
		s.refreshOne(wallet)
	}
}

func (s *WalletService) refreshWalletByAddress(address string) {
	if wallet, ok := s.store.WalletByAddress(address); ok {
		s.refreshOne(wallet)
	}
}

func (s *WalletService) RefreshAfterTransaction(senderID, destinationAddress string) {
	s.RefreshWallet(senderID)
	s.refreshWalletByAddress(destinationAddress)
	time.Sleep(3 * time.Second)
	s.RefreshWallet(senderID)
	s.refreshWalletByAddress(destinationAddress)
}

func (s *WalletService) refreshOne(wallet domain.Wallet) {
	balance, err := s.chain.Balance(wallet.Chain, wallet.Address)
	if err != nil || balance == wallet.Balance {
		return
	}
	if err := s.store.SetWalletBalance(wallet.ID, balance); err != nil {
		logError("refresh balance wallet="+wallet.ID, err)
		return
	}
	if updated, ok := s.store.WalletByID(wallet.ID); ok {
		publishWallet(s.store, s.hub, updated)
	}
}
