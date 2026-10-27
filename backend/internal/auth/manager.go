package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"

	"github.com/gin-gonic/gin"
)

const (
	SessionCookie = "wallet_session"
	OAuthCookie   = "wallet_oauth"
	SessionTTL    = 24 * time.Hour
)

type Manager struct {
	repo   *sqlite.AuthRepository
	secure bool
}

func New(repo *sqlite.AuthRepository, secure bool) *Manager {
	return &Manager{repo: repo, secure: secure}
}

func RandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func CookieToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}
func (m *Manager) Session(r *http.Request) (domain.Session, error) {
	token := CookieToken(r)
	if token == "" {
		return domain.Session{}, repository.ErrNotFound
	}
	return m.repo.Session(HashToken(token))
}
func (m *Manager) SetCookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: m.secure, SameSite: http.SameSiteLaxMode})
}
func (m *Manager) SetSession(c *gin.Context, token string) {
	m.SetCookie(c, SessionCookie, token, int(SessionTTL.Seconds()))
}
func (m *Manager) ClearSession(c *gin.Context) { m.SetCookie(c, SessionCookie, "", -1) }

func (m *Manager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := m.Session(c.Request)
		if errors.Is(err, repository.ErrNotFound) {
			httpapi.Error(c, apperror.AuthenticationRequired.New())
			return
		}
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		c.Set("userID", session.UserID)
		c.Set("workspaceID", session.WorkspaceID)
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}
