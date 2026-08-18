package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stellar-wallet-backend/internal/auth"
)

func TestRouterPreservesPublicRouteContract(t *testing.T) {
	server := NewServer(Dependencies{Auth: auth.New([]byte("test-secret"), time.Hour)})
	router := server.Router("http://localhost:5173")

	want := map[string]bool{
		"GET /health":                          true,
		"GET /api/v1/cluster":                  true,
		"POST /api/v1/auth/register":           true,
		"POST /api/v1/auth/login":              true,
		"GET /api/v1/events":                   true,
		"GET /api/v1/prices":                   true,
		"GET /api/v1/wallets":                  true,
		"POST /api/v1/wallets":                 true,
		"GET /api/v1/wallets/:id":              true,
		"DELETE /api/v1/wallets/:id":           true,
		"GET /api/v1/wallets/:id/balance":      true,
		"POST /api/v1/wallets/:id/fund":        true,
		"POST /api/v1/wallets/:id/trustline":   true,
		"GET /api/v1/wallets/:id/sync":         true,
		"GET /api/v1/chains":                   true,
		"GET /api/v1/config":                   true,
		"PUT /api/v1/config":                   true,
		"POST /api/v1/assets":                  true,
		"DELETE /api/v1/assets":                true,
		"GET /api/v1/wallets/:id/transactions": true,
		"POST /api/v1/transactions":            true,
		"GET /api/v1/transactions/:id":         true,
		"GET /api/v1/tx/:hash/chain":           true,
		"GET /api/v1/resolve":                  true,
		"GET /api/v1/wallets/:id/swap/quote":   true,
		"POST /api/v1/wallets/:id/swap":        true,
	}

	got := map[string]bool{}
	for _, route := range router.Routes() {
		got[route.Method+" "+route.Path] = true
	}
	if len(got) != len(want) {
		t.Fatalf("route count = %d, want %d; routes = %#v", len(got), len(want), got)
	}
	for route := range want {
		if !got[route] {
			t.Errorf("missing route %s", route)
		}
	}
}

func TestHealthAndCORSContract(t *testing.T) {
	server := NewServer(Dependencies{Auth: auth.New([]byte("test-secret"), time.Hour)})
	router := server.Router("http://localhost:5173")
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "{\"status\":\"ok\"}" {
		t.Fatalf("health response = %d %q", response.Code, response.Body.String())
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Fatalf("CORS origin = %q", origin)
	}
}
