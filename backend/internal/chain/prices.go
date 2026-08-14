package chain

import (
	"encoding/json"
	"time"
)

var priceIDs = map[string]string{
	"stellar":  "XLM",
	"solana":   "SOL",
	"usd-coin": "USDC",
	"tether":   "USDT",
}

func (c *Client) Prices() map[string]float64 {
	c.priceMu.Lock()
	defer c.priceMu.Unlock()
	if c.priceCache != nil && time.Now().Before(c.priceExpiry) {
		return c.priceCache
	}

	const url = "https://api.coingecko.com/api/v3/simple/price?ids=stellar,solana,usd-coin,tether&vs_currencies=usd"
	prices := map[string]float64{}
	response, err := c.httpClient.Get(url)
	if err == nil {
		defer response.Body.Close()
		var raw map[string]map[string]float64
		if json.NewDecoder(response.Body).Decode(&raw) == nil {
			for id, symbol := range priceIDs {
				if value, ok := raw[id]["usd"]; ok {
					prices[symbol] = value
				}
			}
		}
	}
	if _, ok := prices["USDC"]; !ok {
		prices["USDC"] = 1
	}
	if _, ok := prices["USDT"]; !ok {
		prices["USDT"] = 1
	}

	c.priceCache = prices
	c.priceExpiry = time.Now().Add(time.Minute)
	return prices
}
