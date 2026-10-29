package service

import (
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository"
)

// walletAudience is everyone who can see the wallet: the members of its workspace.
func walletAudience(store repository.Store, wallet domain.Wallet) []string {
	if wallet.WorkspaceID == "" {
		return []string{wallet.UserID}
	}
	userIDs, err := store.WorkspaceMemberIDs(wallet.WorkspaceID)
	if err != nil {
		logError("list event audience wallet="+wallet.ID, err)
		return nil
	}
	return userIDs
}

func publishWallet(store repository.Store, hub *realtime.Hub, wallet domain.Wallet) {
	for _, userID := range walletAudience(store, wallet) {
		hub.PublishWallet(userID, wallet)
	}
}

func publishTransaction(store repository.Store, hub *realtime.Hub, transaction domain.Transaction) {
	wallet, ok := store.WalletByID(transaction.WalletID)
	if !ok {
		hub.PublishTransaction(transaction.UserID, transaction)
		return
	}
	for _, userID := range walletAudience(store, wallet) {
		hub.PublishTransaction(userID, transaction)
	}
}
