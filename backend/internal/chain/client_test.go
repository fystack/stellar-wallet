package chain

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestBalanceReadsSolanaLamports(t *testing.T) {
	httpClient := &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"jsonrpc":"2.0","result":{"value":1250000000},"id":1}`,
				)),
				Header: make(http.Header),
			}, nil
		}),
	}
	client := NewClient(DefaultHorizonURL, "http://solana.test", httpClient)
	balance, err := client.Balance("solana", "address")
	if err != nil {
		t.Fatalf("Balance() error = %v", err)
	}
	if balance != "1.2500" {
		t.Fatalf("Balance() = %q, want 1.2500", balance)
	}
}
