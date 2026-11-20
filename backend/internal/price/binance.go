// Package price reads USD prices for Stellar assets from Binance.
package price

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"stellar-wallet-backend/internal/config"
)

// refreshAfter is how long a fetched price is reused before asking Binance again.
const refreshAfter = 15 * time.Second

// ErrNoSource means the asset has no configured Binance symbol.
var ErrNoSource = errors.New("asset has no price source")

// Quote is the USD price of one asset unit and when Binance reported it.
type Quote struct {
	Symbol    string
	Price     string
	FetchedAt time.Time
}

type Feed struct {
	httpClient *http.Client
	endpoint   string
	maxAge     time.Duration
	symbols    map[string]string // "CODE:ISSUER" -> Binance symbol
	codes      map[string]string // Binance symbol -> asset code, for the display map
	now        func() time.Time

	mu        sync.Mutex
	prices    map[string]string
	fetchedAt time.Time
}

func NewFeed(httpClient *http.Client, cfg config.BinanceConfig) *Feed {
	feed := &Feed{
		httpClient: httpClient,
		endpoint:   cfg.Endpoint,
		maxAge:     cfg.MaxAge,
		symbols:    map[string]string{},
		codes:      map[string]string{},
		now:        time.Now,
	}
	for _, asset := range cfg.Assets {
		feed.symbols[assetKey(asset.Code, asset.Issuer)] = asset.Symbol
		feed.codes[asset.Symbol] = asset.Code
	}
	return feed
}

func assetKey(code, issuer string) string {
	if code == "" {
		code = "XLM"
	}
	if code == "XLM" {
		issuer = ""
	}
	return code + ":" + issuer
}

// Quote returns the USD price of an asset, or an error when it has no source or no fresh price.
func (f *Feed) Quote(code, issuer string) (Quote, error) {
	symbol, ok := f.symbols[assetKey(code, issuer)]
	if !ok {
		return Quote{}, ErrNoSource
	}
	prices, fetchedAt, err := f.fresh()
	if err != nil {
		return Quote{}, err
	}
	value, ok := prices[symbol]
	if !ok {
		return Quote{}, fmt.Errorf("binance returned no price for %s", symbol)
	}
	return Quote{Symbol: symbol, Price: value, FetchedAt: fetchedAt}, nil
}

// Prices returns USD prices keyed by asset code, for showing balances in USD.
func (f *Feed) Prices() (map[string]float64, error) {
	prices, _, err := f.fresh()
	if err != nil {
		return nil, err
	}
	byCode := make(map[string]float64, len(prices))
	for symbol, value := range prices {
		number, _ := new(big.Rat).SetString(value)
		usd, _ := number.Float64()
		byCode[f.codes[symbol]] = usd
	}
	return byCode, nil
}

// fresh refetches after refreshAfter; a failed refetch still serves prices younger than maxAge.
func (f *Feed) fresh() (map[string]string, time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	if f.prices != nil && now.Sub(f.fetchedAt) < refreshAfter {
		return f.prices, f.fetchedAt, nil
	}
	prices, err := f.fetch()
	if err == nil {
		f.prices, f.fetchedAt = prices, now
		return prices, now, nil
	}
	if f.prices != nil && now.Sub(f.fetchedAt) <= f.maxAge {
		return f.prices, f.fetchedAt, nil
	}
	return nil, time.Time{}, err
}

func (f *Feed) fetch() (map[string]string, error) {
	if len(f.codes) == 0 {
		return nil, ErrNoSource
	}
	symbols := make([]string, 0, len(f.codes))
	for symbol := range f.codes {
		symbols = append(symbols, `"`+symbol+`"`)
	}
	query := url.Values{"symbols": {"[" + strings.Join(symbols, ",") + "]"}}
	response, err := f.httpClient.Get(f.endpoint + "?" + query.Encode())
	if err != nil {
		return nil, fmt.Errorf("fetch binance prices: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch binance prices: HTTP %d", response.StatusCode)
	}
	var tickers []struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	if err := json.NewDecoder(response.Body).Decode(&tickers); err != nil {
		return nil, fmt.Errorf("decode binance prices: %w", err)
	}
	prices := make(map[string]string, len(tickers))
	for _, ticker := range tickers {
		value, ok := new(big.Rat).SetString(ticker.Price)
		if !ok || value.Sign() <= 0 {
			return nil, fmt.Errorf("binance price for %s is invalid: %q", ticker.Symbol, ticker.Price)
		}
		prices[ticker.Symbol] = ticker.Price
	}
	return prices, nil
}
