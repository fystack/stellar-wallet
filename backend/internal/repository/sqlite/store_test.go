package sqlite

import (
	"errors"
	modernsqlite "modernc.org/sqlite"
	"slices"
	"strings"
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
	if err := NewAuthRepository(db).CreateUser(user); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	userID, hash, err := NewAuthRepository(db).UserCredentials(user.Email)
	if err != nil || userID != user.ID || hash != user.PasswordHash {
		t.Fatalf("UserCredentials() = %q, %q, %v", userID, hash, err)
	}

	wallet := domain.Wallet{
		ID: "wallet-1", UserID: user.ID, WorkspaceID: domain.PersonalWorkspaceID(user.ID), Name: "Wallet", Chain: "stellar", Symbol: "XLM",
		Balance: "0.00", Status: "ready", CreatedAt: "2026-08-14T00:00:00Z",
	}
	if err := NewWorkspaceRepository(db).EnsurePersonalWorkspace(user.ID, user.CreatedAt); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateWallet(wallet); err != nil {
		t.Fatalf("CreateWallet() error = %v", err)
	}
	wallets, err := store.ListWallets(domain.PersonalWorkspaceID(user.ID))
	if err != nil || len(wallets) != 1 || wallets[0].ID != wallet.ID {
		t.Fatalf("ListWallets() = %#v, %v", wallets, err)
	}

	transaction := domain.Transaction{
		ID: "tx-1", WalletID: wallet.ID, UserID: user.ID, Type: "out", Counterparty: "GDEST",
		Amount: "1.00", Symbol: "XLM", Status: "signing", CreatedAt: "2026-08-14T00:00:00Z",
	}
	if err := NewTransactionRepository(db).CreateTransaction(transaction); err != nil {
		t.Fatalf("CreateTransaction() error = %v", err)
	}
	stored, ok := NewTransactionRepository(db).TransactionForUser(transaction.ID, user.ID)
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

	_, _, err = NewAuthRepository(db).UserCredentials("missing@example.com")
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
	repo := NewAuthRepository(db)
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

func TestReadyWalletsForUsersFollowsWorkspaceMembership(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store, workspaces := NewStore(db), NewWorkspaceRepository(db)
	for _, userID := range []string{"watcher", "idle"} {
		if err := workspaces.EnsurePersonalWorkspace(userID, "2026-10-01T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}
	for _, wallet := range []domain.Wallet{
		{ID: "watched", UserID: "watcher", WorkspaceID: domain.PersonalWorkspaceID("watcher"), Status: domain.WalletReady},
		{ID: "generating", UserID: "watcher", WorkspaceID: domain.PersonalWorkspaceID("watcher"), Status: domain.WalletGenerating},
		{ID: "unwatched", UserID: "idle", WorkspaceID: domain.PersonalWorkspaceID("idle"), Status: domain.WalletReady},
	} {
		if err := store.CreateWallet(wallet); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name    string
		userIDs []string
		want    []string
	}{
		{"nobody connected", nil, nil},
		{"connected user sees their ready wallets", []string{"watcher"}, []string{"watched"}},
		{"several users", []string{"watcher", "idle"}, []string{"unwatched", "watched"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wallets, err := store.ReadyWalletsForUsers(tc.userIDs)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, wallet := range wallets {
				got = append(got, wallet.ID)
			}
			slices.Sort(got)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("wallets = %v, want %v", got, tc.want)
			}
		})
	}
}
