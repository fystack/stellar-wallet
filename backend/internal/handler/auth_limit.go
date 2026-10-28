package handler

import (
	"sync"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/httpapi"

	"github.com/gin-gonic/gin"
)

// Keyed on the direct peer, not forwarded headers, which clients can forge.
func authRateLimit(perMinute int) gin.HandlerFunc {
	type window struct {
		attempts int
		until    time.Time
	}
	var mu sync.Mutex
	windows := map[string]window{}
	return func(c *gin.Context) {
		switch c.FullPath() {
		case "/api/v1/auth/login", "/api/v1/auth/register", "/api/v1/auth/:provider/start", "/api/v1/auth/:provider/link", "/api/v1/auth/reauthenticate":
		default:
			c.Next()
			return
		}
		now := time.Now()
		peer := c.RemoteIP()
		mu.Lock()
		for key, entry := range windows {
			if !entry.until.After(now) {
				delete(windows, key)
			}
		}
		entry := windows[peer]
		if entry.until.IsZero() {
			entry.until = now.Add(time.Minute)
		}
		entry.attempts++
		windows[peer] = entry
		mu.Unlock()
		if entry.attempts > perMinute {
			c.Header("Retry-After", "60")
			httpapi.Error(c, apperror.RateLimited.New())
			return
		}
		c.Next()
	}
}
