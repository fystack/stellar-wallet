package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CoinGecko id -> our asset symbol.
var priceIDs = map[string]string{
	"stellar":  "XLM",
	"solana":   "SOL",
	"usd-coin": "USDC",
	"tether":   "USDT",
}

var (
	priceMu     sync.Mutex
	priceCache  map[string]float64
	priceExpiry time.Time
)

func fetchPrices() map[string]float64 {
	priceMu.Lock()
	defer priceMu.Unlock()
	if priceCache != nil && time.Now().Before(priceExpiry) {
		return priceCache
	}

	url := "https://api.coingecko.com/api/v3/simple/price?ids=stellar,solana,usd-coin,tether&vs_currencies=usd"
	out := map[string]float64{}
	resp, err := httpc.Get(url)
	if err == nil {
		defer resp.Body.Close()
		var raw map[string]map[string]float64
		if json.NewDecoder(resp.Body).Decode(&raw) == nil {
			for id, sym := range priceIDs {
				if v, ok := raw[id]["usd"]; ok {
					out[sym] = v
				}
			}
		}
	}
	// Stablecoins fall back to $1 if the API is unavailable.
	if _, ok := out["USDC"]; !ok {
		out["USDC"] = 1
	}
	if _, ok := out["USDT"]; !ok {
		out["USDT"] = 1
	}

	priceCache = out
	priceExpiry = time.Now().Add(60 * time.Second)
	return out
}

func (s *server) getPrices(c *gin.Context) {
	c.JSON(http.StatusOK, fetchPrices())
}
