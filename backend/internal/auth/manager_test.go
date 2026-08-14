package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestManagerRoundTripsUserID(t *testing.T) {
	manager := New([]byte("test-secret"), time.Hour)

	token, ttl, err := manager.MakeToken("user-123")
	if err != nil {
		t.Fatalf("MakeToken() error = %v", err)
	}
	if ttl != 3600 {
		t.Fatalf("MakeToken() ttl = %d, want 3600", ttl)
	}
	userID, ok := manager.ParseToken(token)
	if !ok || userID != "user-123" {
		t.Fatalf("ParseToken() = %q, %v, want user-123, true", userID, ok)
	}
}

func TestMiddlewareRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := New([]byte("test-secret"), time.Hour)
	router := gin.New()
	router.Use(manager.Middleware())
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if response.Body.String() != "{\"error\":\"missing bearer token\"}" {
		t.Fatalf("body = %q", response.Body.String())
	}
}
