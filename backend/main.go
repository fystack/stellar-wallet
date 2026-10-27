package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"stellar-wallet-backend/internal/app"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/config"
	"stellar-wallet-backend/internal/logger"
	"stellar-wallet-backend/internal/mail"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository/sqlite"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log, err := logger.New(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()
	zap.ReplaceGlobals(log)
	zap.RedirectStdLog(log)
	gin.SetMode(gin.ReleaseMode)

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		log.Fatal("open db", zap.Error(err))
	}
	defer db.Close()

	httpClient := &http.Client{Timeout: 20 * time.Second}
	chainClient := chain.NewClient(cfg.HorizonURL, httpClient)
	authRepo := sqlite.NewAuthRepository(db)
	authManager := auth.New(authRepo, cfg.CookieSecure)
	var mailer service.Mailer
	switch smtpMailer := mail.New(cfg.SMTP); {
	case smtpMailer != nil:
		mailer = smtpMailer
		log.Info("invite emails via SMTP", zap.String("host", cfg.SMTP.Host), zap.Int("port", cfg.SMTP.Port))
	case cfg.LocalFrontend():
		mailer = mail.LogMailer{}
		log.Info("SMTP not configured: invite emails are written to this log (localhost only)")
	default:
		log.Warn("SMTP not configured: invite emails are disabled; owners share invite links manually")
	}
	clusterClient := cluster.NewClient(httpClient, cfg.ConsulAddr, cfg.HealthBasePort, cfg.NodeHealthURL, cfg.AssumePeersOnline)
	server := app.NewServer(app.Dependencies{
		FrontendURL:     cfg.FrontendURL,
		PublicURL:       cfg.PublicURL,
		AuthRateLimit:   cfg.AuthRateLimit,
		Store:           sqlite.NewStore(db),
		Transactions:    sqlite.NewTransactionRepository(db),
		AuthRepository:  authRepo,
		Workspaces:      sqlite.NewWorkspaceRepository(db),
		PaymentRequests: sqlite.NewPaymentRequestRepository(db),
		Mailer:          mailer,
		OAuthProviders:  auth.NewProviders(cfg.OAuth, cfg.PublicURL, httpClient),
		Auth:            authManager,
		Chain:           chainClient,
		Cluster:         clusterClient,
		Hub:             realtime.NewHub(),
	})

	mpcClient, err := mpc.New(cfg.NATSURL, cfg.InitiatorKey, server.MPCCallbacks())
	if err != nil {
		log.Fatal("connect mpcium", zap.Error(err))
	}
	defer mpcClient.Close()
	server.SetMPC(mpcClient)
	log.Info("linked to mpcium cluster", zap.String("nats", natsHost(cfg.NATSURL)))

	server.Start()

	log.Info("backend listening", zap.String("addr", cfg.Addr), zap.String("db", cfg.DBPath))
	if err := server.Router(cfg.CORSOrigin).Run(cfg.Addr); err != nil {
		log.Fatal("serve http", zap.Error(err))
	}
}

// natsHost drops the credentials NATS URLs carry in their user info.
func natsHost(natsURL string) string {
	parsed, err := url.Parse(natsURL)
	if err != nil {
		return "invalid url"
	}
	return parsed.Host
}
