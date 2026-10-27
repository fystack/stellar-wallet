package repository

import (
	"errors"

	"stellar-wallet-backend/internal/domain"
)

var (
	ErrNotFound       = errors.New("repository: not found")
	ErrConflict       = errors.New("repository: conflict")
	ErrIdentityLinked = errors.New("repository: identity already linked")
	ErrDuplicate      = errors.New("repository: duplicate")
	ErrTooFewSenders  = errors.New("repository: approval policy needs more senders")
)

// Store is the persistence port consumed by the application layer.
// Database adapters implement this contract without leaking driver-specific
// types, queries, or transaction details into handlers and use cases.
type Store interface {
	CustomAssets() ([]domain.CustomAsset, error)
	AddAsset(domain.CustomAsset) error
	RemoveAsset(code, issuer string) error
	ListWallets(workspaceID string) ([]domain.Wallet, error)
	CreateWallet(domain.Wallet) error
	WalletFor(userID, walletID string) (domain.Wallet, bool)
	WorkspaceMemberIDs(workspaceID string) ([]string, error)
	WorkspaceRole(workspaceID, userID string) (domain.WorkspaceRole, error)
	WalletByID(walletID string) (domain.Wallet, bool)
	WalletByAddress(address string) (domain.Wallet, bool)
	ReadyWalletsForUsers(userIDs []string) ([]domain.Wallet, error)
	GeneratingWallets() ([]domain.Wallet, error)
	SetWalletStatus(walletID string, status domain.WalletStatus) error
	CompleteWalletKeygen(walletID, address, publicKey string) error
	SetWalletBalance(walletID, balance string) error
	DeleteWallet(walletID string) error
}
