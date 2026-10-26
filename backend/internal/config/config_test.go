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
