package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"stellar-wallet-backend/internal/app"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/mpcium"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/storage/sqlite"
)

func main() {
	dbPath := getenv("DB_PATH", "wallet.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	httpClient := &http.Client{Timeout: 20 * time.Second}
	chainClient := chain.NewClient(chain.DefaultHorizonURL, chain.DefaultSolanaURL, httpClient)
	authManager := auth.New([]byte("dev-secret-change-me"), 24*time.Hour)
	healthBase, _ := strconv.Atoi(getenv("HEALTH_BASE_PORT", "8091"))
	clusterClient := cluster.NewClient(httpClient, getenv("CONSUL_ADDR", "10.10.0.1:8500"), healthBase)
	server := app.NewServer(app.Dependencies{
		Store:   sqlite.NewStore(db),
		Auth:    authManager,
		Chain:   chainClient,
		Cluster: clusterClient,
		Hub:     realtime.NewHub(),
	})

	natsURL := getenv("NATS_URL", "nats://10.10.0.1:4222")
	keyPath := getenv("INITIATOR_KEY", "../mpcium/event_initiator.key")
	mpcClient, err := mpcium.New(natsURL, keyPath, server.MPCCallbacks())
	if err != nil {
		log.Fatalf("connect mpcium: %v", err)
	}
	defer mpcClient.Close()
	server.SetMPC(mpcClient)
	log.Printf("linked to mpcium cluster via %s", natsURL)

	server.ApplyRPCConfig()
	server.StartBalanceRefresher()

	addr := getenv("ADDR", ":8080")
	log.Printf("backend listening on %s (db=%s)", addr, dbPath)
	if err := server.Router(getenv("CORS_ORIGIN", "http://localhost:5173")).Run(addr); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
