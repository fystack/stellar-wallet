package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouterPreservesPublicRouteContract(t *testing.T) {
	router := NewRouter(Dependencies{AuthRateLimit: 30, Auth: testAuthManager(t)}, "http://localhost:5173")

	want := map[string]bool{
		"GET /api/v1/auth/providers":                      true,
		"GET /api/v1/auth/:provider/start":                true,
		"GET /api/v1/auth/:provider/callback":             true,
		"POST /api/v1/auth/logout":                        true,
		"GET /api/v1/auth/me":                             true,
		"POST /api/v1/auth/reauthenticate":                true,
		"POST /api/v1/auth/logout-all":                    true,
		"POST /api/v1/auth/:provider/link":                true,
		"GET /health":                                     true,
		"GET /api/v1/cluster":                             true,
		"POST /api/v1/auth/register":                      true,
		"POST /api/v1/auth/login":                         true,
		"GET /api/v1/events":                              true,
		"GET /api/v1/prices":                              true,
		"GET /api/v1/wallets":                             true,
		"POST /api/v1/wallets":                            true,
		"GET /api/v1/wallets/:id":                         true,
		"DELETE /api/v1/wallets/:id":                      true,
		"GET /api/v1/wallets/:id/balance":                 true,
		"POST /api/v1/wallets/:id/fund":                   true,
		"POST /api/v1/wallets/:id/trustline":              true,
		"GET /api/v1/wallets/:id/sync":                    true,
		"GET /api/v1/chains":                              true,
		"GET /api/v1/config":                              true,
		"GET /api/v1/workspaces/:id/assets":               true,
		"POST /api/v1/workspaces/:id/assets":              true,
		"DELETE /api/v1/workspaces/:id/assets":            true,
		"GET /api/v1/wallets/:id/transactions":            true,
		"POST /api/v1/transactions":                       true,
		"GET /api/v1/transactions/:id":                    true,
		"GET /api/v1/tx/:hash/chain":                      true,
		"GET /api/v1/resolve":                             true,
		"GET /api/v1/wallets/:id/swap/quote":              true,
		"POST /api/v1/wallets/:id/swap":                   true,
		"GET /api/v1/workspaces":                          true,
		"POST /api/v1/workspaces":                         true,
		"POST /api/v1/workspaces/:id/switch":              true,
		"GET /api/v1/workspaces/:id/members":              true,
		"PATCH /api/v1/workspaces/:id/members/:userId":    true,
		"DELETE /api/v1/workspaces/:id/members/:userId":   true,
		"GET /api/v1/workspaces/:id/approval-policy":      true,
		"PUT /api/v1/workspaces/:id/approval-policy":      true,
		"DELETE /api/v1/workspaces/:id/approval-policy":   true,
		"POST /api/v1/workspaces/:id/invites":             true,
		"DELETE /api/v1/workspaces/:id/invites/:inviteId": true,
		"GET /api/v1/invites/:token":                      true,
		"POST /api/v1/invites/:token/accept":              true,
		"GET /api/v1/invitations":                         true,
		"GET /api/v1/workspaces/:id/payment-requests":     true,
		"GET /api/v1/payment-requests/:id":                true,
		"POST /api/v1/payment-requests/:id/approve":       true,
		"POST /api/v1/payment-requests/:id/reject":        true,
		"POST /api/v1/payment-requests/:id/cancel":        true,
		"POST /api/v1/invitations/:id/accept":             true,
		"POST /api/v1/invitations/:id/decline":            true,
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
	router := NewRouter(Dependencies{AuthRateLimit: 30, Auth: testAuthManager(t)}, "http://localhost:5173")
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "{\"data\":{\"status\":\"ok\"}}" {
		t.Fatalf("health response = %d %q", response.Code, response.Body.String())
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Fatalf("CORS origin = %q", origin)
	}

	preflight := httptest.NewRecorder()
	router.ServeHTTP(preflight, httptest.NewRequest(http.MethodOptions, "/api/v1/wallets", nil))
	if preflight.Code != http.StatusNoContent || preflight.Body.Len() != 0 {
		t.Fatalf("preflight = %d %s", preflight.Code, preflight.Body)
	}
	methods := preflight.Header().Get("Access-Control-Allow-Methods")
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		if !strings.Contains(methods, method) {
			t.Fatalf("preflight allows %q, missing %s", methods, method)
		}
	}
}

func TestRouterFormatsUnknownRoutesMethodsAndPanics(t *testing.T) {
	r := NewRouter(Dependencies{AuthRateLimit: 30, Auth: testAuthManager(t)}, "http://localhost")
	r.GET("/panic", func(c *gin.Context) { panic("private panic details") })
	for _, tt := range []struct {
		method, path string
		status       int
		code         string
	}{
		{"GET", "/api/v1/unknown", 404, "route_not_found"},
		{"POST", "/health", 405, "method_not_allowed"},
		{"GET", "/panic", 500, "internal_error"},
		{"GET", "/api/v1/wallets", 401, "authentication_required"},
		{"GET", "/api/v1/events", 401, "authentication_required"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, tt.path, nil)
		req.Header.Set("X-CSRF-Protection", "1")
		r.ServeHTTP(w, req)
		if w.Code != tt.status || !strings.Contains(w.Body.String(), `"code":"`+tt.code+`"`) || strings.Contains(w.Body.String(), "private") {
			t.Errorf("%s %s: %d %s", tt.method, tt.path, w.Code, w.Body)
		}
	}
}
