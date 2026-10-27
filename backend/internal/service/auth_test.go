package service

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"
)

type socialProviderStub struct {
	identity domain.Identity
	err      error
}

func (p *socialProviderStub) AuthorizationURL(state, verifier, nonce string) string {
	return "https://provider.test/authorize?state=" + state + "&nonce=" + nonce
}
func (p *socialProviderStub) Identity(context.Context, string, string, string) (domain.Identity, error) {
	return p.identity, p.err
}

func TestSocialLoginAndExplicitLinkKeepExistingWallet(t *testing.T) {
	db, err := sqlite.Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := sqlite.NewAuthRepository(db)
	provider := &socialProviderStub{identity: domain.Identity{Provider: "google", Subject: "google-id", Email: "owner@example.com"}}
	s := NewAuthService(repo, map[string]auth.OAuthProvider{"google": provider})
	user, token, err := s.Register("owner@example.com", "password123", "")
	if err != nil {
		t.Fatal(err)
	}
	wallets := sqlite.NewStore(db)
	if err := wallets.CreateWallet(domain.Wallet{ID: "wallet", UserID: user.ID, Name: "Existing", Chain: "stellar", Symbol: "XLM", Status: "ready"}); err != nil {
		t.Fatal(err)
	}
	start, err := s.StartOAuth("google", "", false)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(start.URL)
	_, err = s.CompleteOAuth(context.Background(), "google", u.Query().Get("state"), start.BrowserToken, "code", "", false)
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Code != "account_link_required" {
		t.Fatalf("email collision: %v", err)
	}
	start, err = s.StartOAuth("google", token, true)
	if err != nil {
		t.Fatal(err)
	}
	u, _ = url.Parse(start.URL)
	result, err := s.CompleteOAuth(context.Background(), "google", u.Query().Get("state"), start.BrowserToken, "code", token, false)
	if err != nil || result.User.ID != user.ID || !result.Linked {
		t.Fatalf("link result = %#v %v", result, err)
	}
	if wallet, ok := wallets.WalletByID("wallet"); !ok || wallet.UserID != user.ID {
		t.Fatalf("wallet ownership changed: %#v %v", wallet, ok)
	}
	if _, err := s.CompleteOAuth(context.Background(), "google", u.Query().Get("state"), start.BrowserToken, "code", token, false); err == nil {
		t.Fatal("callback replay accepted")
	}
	if err := s.Logout(token); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Session(auth.HashToken(token)); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("session still active: %v", err)
	}
	start, err = s.StartOAuth("google", "", false)
	if err != nil {
		t.Fatal(err)
	}
	u, _ = url.Parse(start.URL)
	result, err = s.CompleteOAuth(context.Background(), "google", u.Query().Get("state"), start.BrowserToken, "code", "", false)
	if err != nil || result.User.ID != user.ID || result.Token == "" {
		t.Fatalf("social return = %#v %v", result, err)
	}
}

func TestLinkCallbackRequiresOriginalActiveRecentSession(t *testing.T) {
	db, err := sqlite.Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := sqlite.NewAuthRepository(db)
	s := NewAuthService(repo, map[string]auth.OAuthProvider{"google": &socialProviderStub{identity: domain.Identity{Provider: "google", Subject: "id", Email: "social@example.com"}}})
	user, token, err := s.Register("owner@example.com", "password123", "")
	if err != nil {
		t.Fatal(err)
	}
	start, err := s.StartOAuth("google", token, true)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(start.URL)
	s.Logout(token)
	if _, err := s.CompleteOAuth(context.Background(), "google", u.Query().Get("state"), start.BrowserToken, "code", token, false); err == nil {
		t.Fatal("link accepted after logout")
	}
	stale := "stale-token"
	if err := repo.CreateSession(domain.Session{Hash: auth.HashToken(stale), UserID: user.ID, CreatedAt: time.Now().Add(-time.Hour).Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartOAuth("google", stale, true); err == nil {
		t.Fatal("old session linked without reauthentication")
	}
}

func TestRegisterRejectsPasswordOverBcryptByteLimit(t *testing.T) {
	db, err := sqlite.Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewAuthService(sqlite.NewAuthRepository(db), nil)

	_, _, err = s.Register("owner@example.com", "🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐🔐", "")
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Code != "invalid_request" {
		t.Fatalf("expected validation error, got %v", err)
	}
}
