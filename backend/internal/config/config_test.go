package config

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestAuthConfigurationRejectsUnsafeOrigins(t *testing.T) {
	for _, scenario := range []string{"local", "wildcard", "public HTTP", "HTTPS insecure cookies", "partial credentials", "HTTPS secure"} {
		t.Run(scenario, func(t *testing.T) {
			cfg := defaults()
			valid := false
			switch scenario {
			case "local":
				valid = true
			case "wildcard":
				cfg.CORSOrigin = "*"
			case "public HTTP":
				cfg.PublicURL = "http://wallet.example.com"
			case "HTTPS insecure cookies":
				cfg.PublicURL = "https://wallet.example.com"
			case "partial credentials":
				cfg.OAuth.Google.ClientID = "id"
			case "HTTPS secure":
				cfg.PublicURL = "https://wallet.example.com"
				cfg.FrontendURL = cfg.PublicURL
				cfg.CORSOrigin = cfg.PublicURL
				cfg.CookieSecure = true
				valid = true
			}
			if err := cfg.Validate(); (err == nil) != valid {
				t.Fatalf("validation=%v, want valid=%v", err, valid)
			}
		})
	}
}

func TestSMTPAndRateLimitValidation(t *testing.T) {
	cases := []struct {
		name  string
		apply func(cfg *Config)
		valid bool
	}{
		{"smtp disabled", func(*Config) {}, true},
		{"named from", func(cfg *Config) {
			cfg.SMTP = SMTPConfig{Host: "smtp.example.com", Port: 587, From: "Stellar MPC Wallet <no-reply@example.com>"}
		}, true},
		{"bare from", func(cfg *Config) {
			cfg.SMTP = SMTPConfig{Host: "smtp.example.com", Port: 587, From: "no-reply@example.com"}
		}, true},
		{"from without address", func(cfg *Config) {
			cfg.SMTP = SMTPConfig{Host: "smtp.example.com", Port: 587, From: "Stellar MPC Wallet"}
		}, false},
		{"missing port", func(cfg *Config) { cfg.SMTP = SMTPConfig{Host: "smtp.example.com", From: "no-reply@example.com"} }, false},
		{"zero rate limit", func(cfg *Config) { cfg.AuthRateLimit = 0 }, false},
		{"unknown log format", func(cfg *Config) { cfg.Log.Format = "xml" }, false},
		{"plain http price endpoint", func(cfg *Config) { cfg.Binance.Endpoint = "http://api.binance.com/api/v3/ticker/price" }, false},
		{"zero price max age", func(cfg *Config) { cfg.Binance.MaxAge = 0 }, false},
		{"priced asset without issuer", func(cfg *Config) {
			cfg.Binance.Assets = append(cfg.Binance.Assets, PricedAsset{Code: "EURC", Symbol: "EURCUSDT"})
		}, false},
		{"priced asset without symbol", func(cfg *Config) {
			cfg.Binance.Assets = append(cfg.Binance.Assets, PricedAsset{Code: "EURC", Issuer: "GISSUER"})
		}, false},
		{"duplicate priced asset", func(cfg *Config) { cfg.Binance.Assets = append(cfg.Binance.Assets, cfg.Binance.Assets[0]) }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaults()
			tc.apply(&cfg)
			if err := cfg.Validate(); (err == nil) != tc.valid {
				t.Fatalf("validation=%v, want valid=%v", err, tc.valid)
			}
		})
	}
}

func TestBinanceConfigFromYAML(t *testing.T) {
	cfg := defaults()
	raw := `
binance:
  endpoint: "https://data-api.binance.vision/api/v3/ticker/price"
  max_age: 90s
  assets:
    - code: XLM
      symbol: XLMUSDT
`
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Binance.MaxAge != 90*time.Second || len(cfg.Binance.Assets) != 1 || cfg.Binance.Endpoint != "https://data-api.binance.vision/api/v3/ticker/price" {
		t.Fatalf("binance = %#v", cfg.Binance)
	}
}
