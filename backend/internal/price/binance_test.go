package price

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stellar-wallet-backend/internal/config"
)

const usdcIssuer = "GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5"

type binanceStub struct {
	server *httptest.Server
	status int
	body   string
	calls  int
	query  string
}

func newBinanceStub(t *testing.T) *binanceStub {
	t.Helper()
	stub := &binanceStub{status: http.StatusOK, body: `[{"symbol":"XLMUSDT","price":"0.25000000"},{"symbol":"USDCUSDT","price":"1.00010000"}]`}
	stub.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.calls++
		stub.query = r.URL.Query().Get("symbols")
		w.WriteHeader(stub.status)
		fmt.Fprint(w, stub.body)
	}))
	t.Cleanup(stub.server.Close)
	return stub
}

func newTestFeed(stub *binanceStub, clock *time.Time) *Feed {
	feed := NewFeed(stub.server.Client(), config.BinanceConfig{
		Endpoint: stub.server.URL,
		MaxAge:   time.Minute,
		Assets: []config.PricedAsset{
			{Code: "XLM", Symbol: "XLMUSDT"},
			{Code: "USDC", Issuer: usdcIssuer, Symbol: "USDCUSDT"},
		},
	})
	feed.now = func() time.Time { return *clock }
	return feed
}

func TestQuoteMatchesCodeAndIssuer(t *testing.T) {
	cases := []struct {
		name, code, issuer string
		price              string
		err                error
	}{
		{"native by empty code", "", "", "0.25000000", nil},
		{"native by code", "XLM", "", "0.25000000", nil},
		{"mapped issuer", "USDC", usdcIssuer, "1.00010000", nil},
		{"same code, other issuer", "USDC", "GFAKEISSUER", "", ErrNoSource},
		{"unmapped asset", "EURC", "GISSUER", "", ErrNoSource},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clock := time.Unix(1_700_000_000, 0)
			feed := newTestFeed(newBinanceStub(t), &clock)
			quote, err := feed.Quote(tc.code, tc.issuer)
			if !errors.Is(err, tc.err) || quote.Price != tc.price {
				t.Fatalf("quote = %#v, err = %v", quote, err)
			}
		})
	}
}

func TestQuoteFailsClosedWhenPricesAreMissingOrStale(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		elapsed time.Duration
		cached  bool
		ok      bool
	}{
		{"blocked region", http.StatusUnavailableForLegalReasons, `{"code":0}`, 0, false, false},
		{"malformed body", http.StatusOK, `not json`, 0, false, false},
		{"zero price", http.StatusOK, `[{"symbol":"XLMUSDT","price":"0"}]`, 0, false, false},
		{"symbol missing from response", http.StatusOK, `[{"symbol":"USDCUSDT","price":"1"}]`, 0, false, false},
		{"refresh fails within max age", http.StatusInternalServerError, ``, 50 * time.Second, true, true},
		{"refresh fails past max age", http.StatusInternalServerError, ``, 61 * time.Second, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := newBinanceStub(t)
			clock := time.Unix(1_700_000_000, 0)
			feed := newTestFeed(stub, &clock)
			if tc.cached {
				if _, err := feed.Quote("XLM", ""); err != nil {
					t.Fatal(err)
				}
			}
			stub.status, stub.body = tc.status, tc.body
			clock = clock.Add(tc.elapsed)
			_, err := feed.Quote("XLM", "")
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v, want ok = %v", err, tc.ok)
			}
		})
	}
}

func TestQuoteReusesRecentPrices(t *testing.T) {
	stub := newBinanceStub(t)
	clock := time.Unix(1_700_000_000, 0)
	feed := newTestFeed(stub, &clock)
	for range 3 {
		if _, err := feed.Quote("XLM", ""); err != nil {
			t.Fatal(err)
		}
	}
	if stub.calls != 1 {
		t.Fatalf("binance calls = %d, want 1", stub.calls)
	}
	clock = clock.Add(refreshAfter)
	if _, err := feed.Quote("USDC", usdcIssuer); err != nil || stub.calls != 2 {
		t.Fatalf("err = %v, calls = %d", err, stub.calls)
	}
	if stub.query != `["XLMUSDT","USDCUSDT"]` && stub.query != `["USDCUSDT","XLMUSDT"]` {
		t.Fatalf("symbols query = %s", stub.query)
	}
}

func TestPricesKeyedByCode(t *testing.T) {
	clock := time.Unix(1_700_000_000, 0)
	prices, err := newTestFeed(newBinanceStub(t), &clock).Prices()
	if err != nil || prices["XLM"] != 0.25 || prices["USDC"] != 1.0001 {
		t.Fatalf("prices = %v, err = %v", prices, err)
	}
}
