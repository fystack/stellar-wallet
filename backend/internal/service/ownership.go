package service

import (
	"log"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
)

func ownedWallet(store repository.Store, userID, walletID string) (domain.Wallet, error) {
	wallet, ok := store.WalletFor(userID, walletID)
	if !ok {
		return domain.Wallet{}, apperror.WalletNotFound.New()
	}
	return wallet, nil
}

func readyWallet(store repository.Store, userID, walletID string) (domain.Wallet, error) {
	wallet, err := ownedWallet(store, userID, walletID)
	if err != nil {
		return domain.Wallet{}, err
	}
	if wallet.Status != domain.WalletReady {
		return domain.Wallet{}, apperror.WalletNotReady.New()
	}
	return wallet, nil
}

// logError records a failed write on a background path that has no caller to return it to.
func logError(context string, err error) {
	if err != nil {
		log.Printf("%s: %v", context, err)
	}
}
