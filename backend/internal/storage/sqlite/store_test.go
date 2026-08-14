package sqlite

import (
	"errors"
	"testing"

	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/store"
)

func TestStorePersistsUsersWalletsAndTransactions(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()
	store := NewStore(db)

	user := domain.User{ID: "user-1", Email: "user@example.com", PasswordHash: "hash", CreatedAt: "2026-08-14T00:00:00Z"}
	if err := store.CreateUser(user); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	userID, hash, err := store.UserCredentials(user.Email)
	if err != nil || userID != user.ID || hash != user.PasswordHash {
		t.Fatalf("UserCredentials() = %q, %q, %v", userID, hash, err)
	}

	wallet := domain.Wallet{
		ID: "wallet-1", UserID: user.ID, Name: "Wallet", Chain: "stellar", Symbol: "XLM",
		Balance: "0.00", Status: "ready", CreatedAt: "2026-08-14T00:00:00Z",
	}
	if err := store.CreateWallet(wallet); err != nil {
		t.Fatalf("CreateWallet() error = %v", err)
	}
	wallets, err := store.ListWallets(user.ID)
	if err != nil || len(wallets) != 1 || wallets[0].ID != wallet.ID {
		t.Fatalf("ListWallets() = %#v, %v", wallets, err)
	}

	transaction := domain.Transaction{
		ID: "tx-1", WalletID: wallet.ID, UserID: user.ID, Type: "out", Counterparty: "GDEST",
		Amount: "1.00", Symbol: "XLM", Status: "signing", CreatedAt: "2026-08-14T00:00:00Z",
	}
	if err := store.CreateTransaction(transaction); err != nil {
		t.Fatalf("CreateTransaction() error = %v", err)
	}
	stored, ok := store.TransactionForUser(transaction.ID, user.ID)
	if !ok || stored.ID != transaction.ID || stored.UserID != user.ID {
		t.Fatalf("TransactionForUser() = %#v, %v", stored, ok)
	}
}

func TestStoreTranslatesMissingUserToPortableError(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	_, _, err = NewStore(db).UserCredentials("missing@example.com")
	if !errors.Is(err, storecontract.ErrNotFound) {
		t.Fatalf("UserCredentials() error = %v, want store.ErrNotFound", err)
	}
}
