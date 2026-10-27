package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/repository/sqlite"
	"stellar-wallet-backend/internal/service"
)

func newAuthFixture(t *testing.T) (*auth.Manager, *service.AuthService, *sqlite.AuthRepository, *sql.DB) {
	t.Helper()
	db, err := sqlite.Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo := sqlite.NewAuthRepository(db)
	return auth.New(repo, false), service.NewAuthService(repo, nil), repo, db
}
func testAuthManager(t *testing.T) *auth.Manager { m, _, _, _ := newAuthFixture(t); return m }

func TestAuthCookieContractLogoutAndDatabaseFailures(t *testing.T) {
	manager, svc, _, db := newAuthFixture(t)
	r := NewRouter(Dependencies{AuthRateLimit: 30, Auth: manager, AuthService: svc}, "http://localhost")
	request := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/v1/auth/"+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Protection", "1")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	credentials := `{"email":"owner@example.com","password":"password123"}`
	response := request("POST", "register", credentials, nil)
	if response.Code != 201 {
		t.Fatalf("register=%d %s", response.Code, response.Body)
	}
	var body map[string]any
	json.Unmarshal(response.Body.Bytes(), &body)
	data := body["data"].(map[string]any)
	if data["email"] != "owner@example.com" || data["access_token"] != nil {
		t.Fatalf("response leaked token: %#v", data)
	}
	cookie := response.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.Name != auth.SessionCookie {
		t.Fatalf("session cookie=%#v", cookie)
	}
	if w := request("GET", "me", "", cookie); w.Code != 200 {
		t.Fatalf("me=%d %s", w.Code, w.Body)
	}
	if w := request("POST", "logout", "", cookie); w.Code != 204 || w.Body.Len() != 0 {
		t.Fatalf("logout=%d %s", w.Code, w.Body)
	}
	if w := request("GET", "me", "", cookie); w.Code != 401 {
		t.Fatal("revoked cookie still works")
	}
	first := request("POST", "login", credentials, nil).Result().Cookies()[0]
	second := request("POST", "login", credentials, nil).Result().Cookies()[0]
	if w := request("POST", "logout-all", "", first); w.Code != 204 || w.Body.Len() != 0 {
		t.Fatalf("logout-all=%d %s", w.Code, w.Body)
	}
	for _, device := range []*http.Cookie{first, second} {
		if w := request("GET", "me", "", device); w.Code != 401 {
			t.Fatal("logout-all left a session active")
		}
	}
	if w := request("POST", "logout-all", "", nil); w.Code != 401 {
		t.Fatalf("anonymous logout-all=%d", w.Code)
	}
	for _, tt := range []struct {
		path, body string
		status     int
		code       string
	}{
		{"register", credentials, 409, "email_already_registered"},
		{"login", credentials, 200, ""},
		{"login", `{"email":"unknown@example.com","password":"password123"}`, 401, "invalid_credentials"},
		{"login", `{"email":"owner@example.com","password":"wrong"}`, 401, "invalid_credentials"},
		{"login", "{", 400, "invalid_request"},
	} {
		w := request("POST", tt.path, tt.body, nil)
		if w.Code != tt.status || (tt.code != "" && !bytes.Contains(w.Body.Bytes(), []byte(`"code":"`+tt.code+`"`))) {
			t.Fatalf("%s=%d %s", tt.path, w.Code, w.Body)
		}
	}
	db.Close()
	for _, path := range []string{"register", "login"} {
		w := request("POST", path, credentials, nil)
		if w.Code != 500 || !bytes.Contains(w.Body.Bytes(), []byte(`"code":"internal_error"`)) {
			t.Fatalf("DB failure=%d %s", w.Code, w.Body)
		}
	}
}

func TestUnsafeRequestsRequireCSRFHeaderAndAllowedOrigin(t *testing.T) {
	manager, svc, _, _ := newAuthFixture(t)
	r := NewRouter(Dependencies{AuthRateLimit: 30, Auth: manager, AuthService: svc}, "http://localhost:5173")
	for _, tt := range []struct {
		origin, header string
		status         int
	}{
		{"http://evil.test", "1", 403}, {"http://localhost:5173", "", 403}, {"null", "1", 403}, {"http://localhost:5173", "1", 400},
	} {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("{"))
		req.Header.Set("Origin", tt.origin)
		req.Header.Set("X-CSRF-Protection", tt.header)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tt.status {
			t.Fatalf("origin=%s header=%s: %d %s", tt.origin, tt.header, w.Code, w.Body)
		}
	}
}
