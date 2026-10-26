package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"stellar-wallet-backend/internal/apperror"

	"github.com/gin-gonic/gin"
)

func TestSuccessEnvelope(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		data   any
		meta   any
		want   string
	}{
		{"object", 201, gin.H{"id": "wallet-1"}, nil, `{"data":{"id":"wallet-1"}}`},
		{"empty list", 200, []string{}, nil, `{"data":[]}`},
		{"pagination", 200, []string{}, gin.H{"has_more": false}, `{"data":[],"meta":{"has_more":false}}`},
		{"no content", 204, nil, nil, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(c *gin.Context) { SuccessWithMeta(c, tt.status, tt.data, tt.meta) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
			if w.Code != tt.status || w.Body.String() != tt.want {
				t.Fatalf("response = %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestWrappedErrorsMapWithoutExposingCause(t *testing.T) {
	cause := errors.New("database password=private; SQL INSERT INTO users")
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"unknown", fmt.Errorf("create session: %w", cause), 500, `{"error":{"code":"internal_error","message":"An unexpected error occurred."}}`},
		{"conflict", fmt.Errorf("link identity: %w", apperror.Wrap(cause, apperror.Conflict, "identity_already_linked", "This login method belongs to another account.")), 409, `{"error":{"code":"identity_already_linked","message":"This login method belongs to another account."}}`},
		{"upstream", apperror.Wrap(cause, apperror.Upstream, "sign_dispatch_failed", "Unable to start transaction signing."), 502, `{"error":{"code":"sign_dispatch_failed","message":"Unable to start transaction signing."}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, cause) {
				t.Fatal("wrapped error lost its cause")
			}
			r := gin.New()
			called := false
			r.Use(func(c *gin.Context) { Error(c, tt.err) })
			r.GET("/", func(c *gin.Context) { called = true; c.Status(http.StatusOK) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
			if called || w.Code != tt.status || w.Body.String() != tt.want {
				t.Fatalf("response = %d %s; next handler called=%v", w.Code, w.Body, called)
			}
		})
	}
}
