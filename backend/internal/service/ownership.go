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

// sendableWallet is a ready wallet the user may send payments from.
func sendableWallet(store repository.Store, userID, walletID string) (domain.Wallet, error) {
	wallet, role, err := walletWithRole(store, userID, walletID)
	if err != nil {
		return domain.Wallet{}, err
	}
	if !role.CanSend() {
		return domain.Wallet{}, apperror.SendRoleRequired.New()
	}
	return wallet, nil
}

// ownerWallet is a ready wallet only owners may sign for (swaps, trustlines).
func ownerWallet(store repository.Store, userID, walletID string) (domain.Wallet, error) {
	wallet, role, err := walletWithRole(store, userID, walletID)
	if err != nil {
		return domain.Wallet{}, err
	}
	if role != domain.RoleOwner {
		return domain.Wallet{}, apperror.OwnerOnlySigning.New()
	}
	return wallet, nil
}

func walletWithRole(store repository.Store, userID, walletID string) (domain.Wallet, domain.WorkspaceRole, error) {
	wallet, err := readyWallet(store, userID, walletID)
	if err != nil {
		return domain.Wallet{}, "", err
	}
	role, err := store.WorkspaceRole(wallet.WorkspaceID, userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return domain.Wallet{}, "", err
	}
	return wallet, role, nil
}

// logError records a failed write on a background path that has no caller to return it to.
func logError(context string, err error) {
	if err != nil {
		zap.L().Error(context, zap.Error(err))
	}
}
