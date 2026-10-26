package chain

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBalanceDistinguishesUnfundedAccountFromHorizonOutage(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{name: "funded", status: http.StatusOK, body: `{"balances":[{"asset_type":"native","balance":"42.5000000"}]}`, want: "42.5000000"},
		{name: "unfunded", status: http.StatusNotFound, body: `{"type":"https://stellar.org/horizon-errors/not_found","title":"Resource Missing","status":404}`, want: "0.00"},
		{name: "outage", status: http.StatusServiceUnavailable, body: `{"status":503,"title":"Service Unavailable"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			horizon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/hal+json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer horizon.Close()
			client := NewClient(horizon.URL, horizon.Client())

			balance, err := client.Balance("stellar", "GADDRESS")
			if (err != nil) != tt.wantErr || balance != tt.want {
				t.Fatalf("Balance = %q, %v", balance, err)
			}
			_, err = client.Balances("stellar", "GADDRESS")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Balances err = %v", err)
			}
		})
	}
}
