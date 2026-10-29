package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/service"
)

type oauthProviderStub struct{}

func (oauthProviderStub) AuthorizationURL(state, verifier, nonce string) string {
	return "https://provider.test/authorize?state=" + state
}
func (oauthProviderStub) Identity(context.Context, string, string, string) (domain.Identity, error) {
	return domain.Identity{Provider: "google", Subject: "google-user", Email: "social@example.com"}, nil
}

func TestOAuthRedirectBindsBrowserAndReturnsOnlySessionCookie(t *testing.T) {
	manager, _, repo, _ := newAuthFixture(t)
	svc := service.NewAuthService(repo, map[string]auth.OAuthProvider{"google": oauthProviderStub{}})
	r := NewRouter(Dependencies{AuthRateLimit: 30, Auth: manager, AuthService: svc, FrontendURL: "https://wallet.test"}, "https://wallet.test")
	start := httptest.NewRecorder()
	r.ServeHTTP(start, httptest.NewRequest("GET", "/api/v1/auth/google/start", nil))
	if start.Code != 302 {
		t.Fatalf("start=%d %s", start.Code, start.Body)
	}
	location, _ := url.Parse(start.Header().Get("Location"))
	state := location.Query().Get("state")
	browserCookie := start.Result().Cookies()[0]
	if state == "" || !browserCookie.HttpOnly || browserCookie.Name != auth.OAuthCookie {
		t.Fatalf("invalid start: %s", start.Header())
	}
	callback := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/api/v1/auth/google/callback?state="+url.QueryEscape(state)+"&code=test-code", nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	rejected := callback(nil)
	if !strings.Contains(rejected.Header().Get("Location"), "auth_error=invalid_oauth_state") {
		t.Fatal("unbound callback accepted")
	}
	response := callback(browserCookie)
	if response.Code != 302 || response.Header().Get("Location") != "https://wallet.test#/wallets" {
		t.Fatalf("callback=%d %#v", response.Code, response.Header())
	}
	var session *http.Cookie
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == auth.SessionCookie {
			session = cookie
		}
	}
	if session == nil || !session.HttpOnly {
		t.Fatal("missing HttpOnly session")
	}
	if strings.Contains(response.Header().Get("Location"), session.Value) {
		t.Fatal("session token in redirect")
	}
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.AddCookie(session)
	me := httptest.NewRecorder()
	r.ServeHTTP(me, req)
	var body struct {
		Data domain.Account `json:"data"`
	}
	json.Unmarshal(me.Body.Bytes(), &body)
	if me.Code != 200 || body.Data.Email != "social@example.com" || len(body.Data.Identities) != 1 || body.Data.HasPassword {
		t.Fatalf("account=%d %s", me.Code, me.Body)
	}
	if w := callback(browserCookie); !strings.Contains(w.Header().Get("Location"), "auth_error=invalid_oauth_state") {
		t.Fatal("callback replay accepted")
	}
}

func TestAuthLimiterUsesCodedError(t *testing.T) {
	manager, svc, _, _ := newAuthFixture(t)
	r := NewRouter(Dependencies{AuthRateLimit: 30, Auth: manager, AuthService: svc}, "http://localhost")
	for i := 0; i < 31; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/auth/google/start", nil))
		if i == 30 && (w.Code != 429 || !strings.Contains(w.Body.String(), `"code":"rate_limited"`)) {
			t.Fatalf("limit=%d %s", w.Code, w.Body)
		}
	}
}
