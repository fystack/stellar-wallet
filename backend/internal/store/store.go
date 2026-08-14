package store

import (
	"errors"

	"stellar-wallet-backend/internal/domain"
)

var ErrNotFound = errors.New("store: not found")

// Store is the persistence port consumed by the application layer.
// Database adapters implement this contract without leaking driver-specific
// types, queries, or transaction details into handlers and use cases.
type Store interface {
	CreateUser(domain.User) error
	UserCredentials(email string) (userID, passwordHash string, err error)
	Setting(key string) (string, bool)
	SetSetting(key, value string)
	CustomAssets() []domain.CustomAsset
	AddAsset(domain.CustomAsset)
	RemoveAsset(code, issuer string)
	ListWallets(userID string) ([]domain.Wallet, error)
	CreateWallet(domain.Wallet) error
	WalletFor(userID, walletID string) (domain.Wallet, bool)
	WalletByID(walletID string) (domain.Wallet, bool)
	WalletByAddress(address string) (domain.Wallet, bool)
	ReadyWallets() []domain.Wallet
	SetWalletStatus(walletID, status string)
	CompleteWalletKeygen(walletID, address, publicKey string)
	SetWalletBalance(walletID, balance string)
	DeleteWallet(userID, walletID string)
	CreateTransaction(domain.Transaction) error
	IncomingTransactionExists(hash string) bool
	TrustlineTransactionExists(walletID, symbol string) bool
	TransactionsByWallet(userID, walletID string) []domain.Transaction
	TransactionForUser(transactionID, userID string) (domain.Transaction, bool)
	TransactionByID(transactionID string) (domain.Transaction, bool)
	SetTransactionStatus(transactionID, status string)
	FailTransaction(transactionID, reason string)
	SetTransactionSignature(transactionID, signature string)
	BroadcastTransaction(transactionID, hash string)
}
