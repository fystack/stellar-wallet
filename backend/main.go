package main

import (
	"log"
	"net/http"
	"time"

	"stellar-wallet-backend/internal/app"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/config"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository/sqlite"
	"stellar-wallet-backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	httpClient := &http.Client{Timeout: 20 * time.Second}
	chainClient := chain.NewClient(cfg.HorizonURL, httpClient)
	authRepo := sqlite.NewAuthRepository(db)
	authManager := auth.New(authRepo, cfg.CookieSecure)
	authService := service.NewAuthService(authRepo, auth.NewProviders(cfg.OAuth, cfg.PublicURL, httpClient))
	clusterClient := cluster.NewClient(httpClient, cfg.ConsulAddr, cfg.HealthBasePort, cfg.NodeHealthURL, cfg.AssumePeersOnline)
	server := app.NewServer(app.Dependencies{
		Store:       sqlite.NewStore(db),
		AuthService: authService, FrontendURL: cfg.FrontendURL, PublicURL: cfg.PublicURL,
		Transactions: sqlite.NewTransactionRepository(db),
		Auth:         authManager,
		Chain:        chainClient,
		Cluster:      clusterClient,
		Hub:          realtime.NewHub(),
	})

	mpcClient, err := mpc.New(cfg.NATSURL, cfg.InitiatorKey, server.MPCCallbacks())
	if err != nil {
		log.Fatalf("connect mpcium: %v", err)
	}
	defer mpcClient.Close()
	server.SetMPC(mpcClient)
	log.Printf("linked to mpcium cluster via %s", cfg.NATSURL)

	server.Start()

	log.Printf("backend listening on %s (db=%s)", cfg.Addr, cfg.DBPath)
	if err := server.Router(cfg.CORSOrigin).Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
