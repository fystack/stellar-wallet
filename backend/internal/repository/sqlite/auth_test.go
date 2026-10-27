package sqlite

import (
	"errors"
	"testing"
	"time"

	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
)

func TestOAuthFlowIsBrowserBoundAndSingleUse(t *testing.T) {
	db, err := Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(db)
	flow := domain.OAuthFlow{StateHash: "state", BrowserHash: "browser", Provider: "google", Verifier: "verifier", Nonce: "nonce", ExpiresAt: time.Now().Unix() + 60}
	if err := repo.CreateFlow(flow); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ConsumeFlow("state", "wrong-browser", "google"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("browser mismatch: %v", err)
	}
	if _, err := repo.ConsumeFlow("state", "browser", "github"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("provider mismatch: %v", err)
	}
	got, err := repo.ConsumeFlow("state", "browser", "google")
	if err != nil || got.Verifier != "verifier" {
		t.Fatalf("consume = %#v %v", got, err)
	}
	if _, err := repo.ConsumeFlow("state", "browser", "google"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("replay: %v", err)
	}
}

func TestSocialIdentityNeverMergesExistingEmail(t *testing.T) {
	db, err := Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(db)
	if err := repo.CreateUser(domain.User{ID: "existing", Email: "owner@example.com", PasswordHash: "hash"}); err != nil {
		t.Fatal(err)
	}
	identity := domain.Identity{Provider: "google", Subject: "google-id", Email: "owner@example.com"}
	if _, err := repo.SocialUser(identity, ""); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("email collision = %v", err)
	}
	user, err := repo.SocialUser(identity, "existing")
	if err != nil || user.ID != "existing" {
		t.Fatalf("link = %#v %v", user, err)
	}
	identity.Email = "changed@example.com"
	user, err = repo.SocialUser(identity, "")
	if err != nil || user.ID != "existing" {
		t.Fatalf("returning identity = %#v %v", user, err)
	}
	if err := repo.CreateUser(domain.User{ID: "other", Email: "other@example.com", PasswordHash: "hash"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SocialUser(identity, "other"); !errors.Is(err, repository.ErrIdentityLinked) {
		t.Fatalf("identity stolen: %v", err)
	}
}

func TestAuthTablesPreserveLegacyUsersAndWalletsOnUpgrade(t *testing.T) {
	path := t.TempDir() + "/legacy.db"
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	users := NewAuthRepository(db)
	if err := users.CreateUser(domain.User{ID: "legacy-user", Email: "legacy@example.com", PasswordHash: "existing-hash"}); err != nil {
		t.Fatal(err)
	}
	if err := NewStore(db).CreateWallet(domain.Wallet{ID: "legacy-wallet", UserID: "legacy-user", Chain: "stellar", Symbol: "XLM", Name: "Legacy wallet"}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"auth_identities", "auth_sessions", "oauth_flows"} {
		if _, err := db.Exec("DROP TABLE " + table); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	upgraded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	account, err := NewAuthRepository(upgraded).Account("legacy-user")
	if err != nil || !account.HasPassword || len(account.Identities) != 0 {
		t.Fatalf("account=%#v %v", account, err)
	}
	wallets, err := NewStore(upgraded).ListWallets(domain.PersonalWorkspaceID("legacy-user"))
	if err != nil || len(wallets) != 1 || wallets[0].ID != "legacy-wallet" {
		t.Fatalf("wallets=%#v %v", wallets, err)
	}
	if _, ok := NewStore(upgraded).WalletFor("legacy-user", "legacy-wallet"); !ok {
		t.Fatal("legacy owner lost access after workspace backfill")
	}
}

func TestExpiredOAuthFlowCannotBeConsumed(t *testing.T) {
	db, err := Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(db)
	if err := repo.CreateFlow(domain.OAuthFlow{StateHash: "state", BrowserHash: "browser", Provider: "github", Verifier: "verifier", Nonce: "nonce", ExpiresAt: time.Now().Add(-time.Second).Unix()}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ConsumeFlow("state", "browser", "github"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expired state accepted: %v", err)
	}
}
