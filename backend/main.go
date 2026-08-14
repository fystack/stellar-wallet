package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

type server struct {
	db  *sql.DB
	hub *hub
	mpc *mpc
}

func main() {
	dbPath := getenv("DB_PATH", "wallet.db")
	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	s := &server{db: db, hub: newHub()}

	natsURL := getenv("NATS_URL", "nats://10.10.0.1:4222")
	keyPath := getenv("INITIATOR_KEY", "../mpcium/event_initiator.key")
	m, err := newMPC(s, natsURL, keyPath)
	if err != nil {
		log.Fatalf("connect mpcium: %v", err)
	}
	s.mpc = m
	log.Printf("linked to mpcium cluster via %s", natsURL)

	s.applyRPCConfig() // load any custom RPC endpoints from settings
	s.startBalanceRefresher()

	r := gin.Default()
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// MPC cluster status — real peer list + readiness from Consul.
	r.GET("/api/v1/cluster", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"threshold": "2-of-3",
			"nodes":     clusterNodes(),
		})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/register", s.register)
		v1.POST("/auth/login", s.login)
		v1.GET("/events", s.events) // SSE; token via ?token=

		v1.GET("/prices", s.getPrices) // public price feed

		auth := v1.Group("")
		auth.Use(authMiddleware())
		{
			auth.GET("/wallets", s.listWallets)
			auth.POST("/wallets", s.createWallet)
			auth.GET("/wallets/:id", s.getWallet)
			auth.DELETE("/wallets/:id", s.deleteWallet)
			auth.GET("/wallets/:id/balance", s.walletBalance)
			auth.POST("/wallets/:id/fund", s.fundWallet)
			auth.POST("/wallets/:id/trustline", s.addTrustline)
			auth.GET("/wallets/:id/sync", s.syncWallet)
			auth.GET("/chains", s.getChains)
			auth.GET("/config", s.getConfig)
			auth.PUT("/config", s.putConfig)
			auth.POST("/assets", s.addAsset)
			auth.DELETE("/assets", s.removeAsset)
			auth.GET("/wallets/:id/transactions", s.listWalletTxns)
			auth.POST("/transactions", s.createTxn)
			auth.GET("/transactions/:id", s.getTxn)
			auth.GET("/tx/:hash/chain", s.txOnChain)
		}
	}

	addr := getenv("ADDR", ":8080")
	log.Printf("backend listening on %s (db=%s)", addr, dbPath)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	origin := getenv("CORS_ORIGIN", "http://localhost:5173")
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
