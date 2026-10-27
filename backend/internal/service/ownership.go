package service

import (
	"errors"
	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"

	"go.uber.org/zap"
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

// signableWallet is a ready wallet the user may sign for; members are read-only until payment approvals exist.
func signableWallet(store repository.Store, userID, walletID string) (domain.Wallet, error) {
	wallet, err := readyWallet(store, userID, walletID)
	if err != nil {
		return domain.Wallet{}, err
	}
	role, err := store.WorkspaceRole(wallet.WorkspaceID, userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return domain.Wallet{}, err
	}
	if role != domain.RoleOwner {
		return domain.Wallet{}, apperror.OwnerOnlySigning.New()
	}
	return wallet, nil
}

// logError records a failed write on a background path that has no caller to return it to.
func logError(context string, err error) {
	if err != nil {
		zap.L().Error(context, zap.Error(err))
	}
}
