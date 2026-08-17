// Package config loads backend settings from config.yaml.
//
// Values in the file override the built-in defaults; a missing file just
// uses the defaults. That's the whole contract — no env overrides.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const Path = "config.yaml"

type Config struct {
	Addr           string `yaml:"addr"`
	DBPath         string `yaml:"db_path"`
	CORSOrigin     string `yaml:"cors_origin"`
	AuthSecret     string `yaml:"auth_secret"`
	NATSURL        string `yaml:"nats_url"`
	ConsulAddr     string `yaml:"consul_addr"`
	InitiatorKey   string `yaml:"initiator_key"`
	HealthBasePort int    `yaml:"health_base_port"`
	HorizonURL     string `yaml:"horizon_url"`
	SolanaURL      string `yaml:"solana_url"`
}

func defaults() Config {
	return Config{
		Addr:           ":8090",
		DBPath:         "wallet.db",
		CORSOrigin:     "http://localhost:5173",
		AuthSecret:     "dev-secret-change-me",
		NATSURL:        "nats://10.10.0.1:4222",
		ConsulAddr:     "10.10.0.1:8500",
		InitiatorKey:   "../mpcium/event_initiator.key",
		HealthBasePort: 8091,
		HorizonURL:     "https://horizon-testnet.stellar.org",
		SolanaURL:      "https://api.devnet.solana.com",
	}
}

// Load reads config.yaml over the defaults. A missing file is not an error.
func Load() (Config, error) {
	cfg := defaults()
	data, err := os.ReadFile(Path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
