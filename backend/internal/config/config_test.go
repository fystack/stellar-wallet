package config

import "testing"

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
