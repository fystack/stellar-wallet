package sqlite

import (
	"errors"
	modernsqlite "modernc.org/sqlite"
	"testing"

	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/repository"
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
	if err := NewPaymentRepository(db).CreateTransaction(transaction); err != nil {
		t.Fatalf("CreateTransaction() error = %v", err)
	}
	stored, ok := NewPaymentRepository(db).TransactionForUser(transaction.ID, user.ID)
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

func TestCreateUserDistinguishesConflictFromDatabaseFailure(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewStore(db)
	user := domain.User{ID: "user-1", Email: "owner@example.com", PasswordHash: "hash"}
	if err := repo.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	user.ID = "user-2"
	err = repo.CreateUser(user)
	var driverError *modernsqlite.Error
	if !errors.Is(err, storecontract.ErrConflict) || !errors.As(err, &driverError) {
		t.Fatalf("conflict lost sentinel or SQL cause: %v", err)
	}
	db.Close()
	err = repo.CreateUser(user)
	if err == nil || errors.Is(err, storecontract.ErrConflict) {
		t.Fatalf("database failure = %v", err)
	}
}
