// Package config loads backend settings from config.yaml.
//
// Values in the file override the built-in defaults; a missing file just
// uses the defaults. That's the whole contract — no env overrides.
package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const Path = "config.yaml"

type OAuthCredentials struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}
type OAuthConfig struct {
	Google OAuthCredentials `yaml:"google"`
	GitHub OAuthCredentials `yaml:"github"`
}

// SMTPConfig enables invite emails. An empty host disables them; invite links still work.
type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type LogConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // console or json
}

type Config struct {
	Log LogConfig `yaml:"log"`
	// AuthRateLimit caps sign-in, sign-up and OAuth starts per client IP per minute.
	AuthRateLimit  int         `yaml:"auth_rate_limit"`
	SMTP           SMTPConfig  `yaml:"smtp"`
	PublicURL      string      `yaml:"public_url"`
	FrontendURL    string      `yaml:"frontend_url"`
	CookieSecure   bool        `yaml:"cookie_secure"`
	OAuth          OAuthConfig `yaml:"oauth"`
	Addr           string      `yaml:"addr"`
	DBPath         string      `yaml:"db_path"`
	CORSOrigin     string      `yaml:"cors_origin"`
	NATSURL        string      `yaml:"nats_url"`
	ConsulAddr     string      `yaml:"consul_addr"`
	InitiatorKey   string      `yaml:"initiator_key"`
	HealthBasePort int         `yaml:"health_base_port"`
	// NodeHealthURL templates each node's /health URL. Empty = the local
	// default (http://localhost:<health_base_port+index>/health). In Docker,
	// where nodes are separate hosts, set e.g. "http://node{i}:8091/health"
	// ({i} = node index, {port} = health_base_port+index).
	NodeHealthURL string `yaml:"node_health_url"`
	// AssumePeersOnline treats every Consul-registered peer as live without
	// hitting a /health endpoint. The official fystacklabs/mpcium image ships
	// no health server, so set this true there; the timeout watchdog still
	// catches a node that dies mid-keygen/sign.
	AssumePeersOnline bool   `yaml:"assume_peers_online"`
	HorizonURL        string `yaml:"horizon_url"`
}

func defaults() Config {
	return Config{
		Log:            LogConfig{Level: "info", Format: "console"},
		AuthRateLimit:  30,
		Addr:           ":8090",
		DBPath:         "wallet.db",
		CORSOrigin:     "http://localhost:5173",
		PublicURL:      "http://localhost:8090",
		FrontendURL:    "http://localhost:5173",
		NATSURL:        "nats://10.8.0.1:4222",
		ConsulAddr:     "10.8.0.1:8500",
		InitiatorKey:   "../mpcium/event_initiator.key",
		HealthBasePort: 8091,
		HorizonURL:     "https://horizon-testnet.stellar.org",
	}
}

// Load reads config.yaml over the defaults. A missing file is not an error.
func Load() (Config, error) {
	cfg := defaults()
	data, err := os.ReadFile(Path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, cfg.Validate()
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, cfg.Validate()
}

// Validate prevents credentialed auth from being deployed with wildcard origins
// or insecure cookies on public hosts. HTTP is supported for local development.
func (c Config) Validate() error {
	for name, value := range map[string]string{"public_url": c.PublicURL, "frontend_url": c.FrontendURL, "cors_origin": c.CORSOrigin} {
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return fmt.Errorf("%s must be an absolute origin URL", name)
		}
		local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
		if u.Scheme != "https" && !(u.Scheme == "http" && local) {
			return fmt.Errorf("%s requires HTTPS outside localhost", name)
		}
		if u.Scheme == "https" && !c.CookieSecure {
			return fmt.Errorf("cookie_secure must be true for HTTPS")
		}
	}
	if strings.TrimRight(c.CORSOrigin, "/") != strings.TrimRight(c.FrontendURL, "/") {
		return fmt.Errorf("cors_origin must match frontend_url")
	}
	for name, p := range map[string]OAuthCredentials{"google": c.OAuth.Google, "github": c.OAuth.GitHub} {
		if (p.ClientID == "") != (p.ClientSecret == "") {
			return fmt.Errorf("oauth.%s requires both client_id and client_secret", name)
		}
	}
	if c.AuthRateLimit <= 0 {
		return fmt.Errorf("auth_rate_limit must be positive")
	}
	if c.Log.Format != "console" && c.Log.Format != "json" {
		return fmt.Errorf("log.format must be console or json")
	}
	if c.SMTP.Host != "" && (c.SMTP.Port == 0 || c.SMTP.From == "") {
		return fmt.Errorf("smtp requires port and from when host is set")
	}
	if c.SMTP.Host != "" {
		if _, err := mail.ParseAddress(c.SMTP.From); err != nil {
			return fmt.Errorf("smtp.from must be an email address, optionally with a name: %w", err)
		}
	}
	return nil
}

// LocalFrontend reports whether the browser-facing app runs on this machine.
func (c Config) LocalFrontend() bool {
	u, err := url.Parse(c.FrontendURL)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}
