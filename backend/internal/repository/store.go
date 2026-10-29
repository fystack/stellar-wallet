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

// Cursor marks the last row of a page ordered by created_at DESC, id DESC; the zero value starts at the newest row.
type Cursor struct {
	CreatedAt string
	ID        string
}

type Store interface {
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
