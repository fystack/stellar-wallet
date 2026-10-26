package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository/sqlite"
)

func TestSessionsAreOpaqueRevocableAndExpire(t *testing.T) {
	db, err := sqlite.Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := sqlite.NewAuthRepository(db)
	manager := New(repo, true)
	if err := repo.CreateUser(domain.User{ID: "user", Email: "user@example.com", PasswordHash: "hash"}); err != nil {
		t.Fatal(err)
	}
	token, err := RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(domain.Session{Hash: HashToken(token), UserID: "user", CreatedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookie, Value: token})
	got, err := manager.Session(request)
	if err != nil || got.UserID != "user" {
		t.Fatalf("session = %#v %v", got, err)
	}
	repo.RevokeSession(HashToken(token))
	if _, err := manager.Session(request); err == nil {
		t.Fatal("revoked cookie accepted")
	}
	repo.CreateSession(domain.Session{Hash: HashToken(token), UserID: "user", ExpiresAt: time.Now().Add(-time.Second).Unix()})
	if _, err := manager.Session(request); err == nil {
		t.Fatal("expired cookie accepted")
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	manager.SetSession(c, token)
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("cookie flags = %#v", cookie)
	}
}
func TestMiddlewareRejectsMissingCookieAndBearerTokens(t *testing.T) {
	manager := New(nil, false)
	router := gin.New()
	router.Use(manager.Middleware())
	router.GET("/", func(c *gin.Context) { t.Fatal("unauthenticated handler executed") })
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Bearer old-jwt")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 401 || response.Body.String() != `{"error":{"code":"authentication_required","message":"Sign in to continue."}}` {
		t.Fatalf("response = %d %s", response.Code, response.Body)
	}
}
