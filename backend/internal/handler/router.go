package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type MPCClient interface {
	CreateWallet(walletID string) error
	Sign(walletID, transactionID, network string, payload []byte) error
}

type Dependencies struct {
	WalletService  *service.WalletService
	PaymentService *service.PaymentService
	Payments       *sqlite.PaymentRepository
	Store          repository.Store
	Auth           *auth.Manager
	Chain          *chain.Client
	Cluster        *cluster.Client
	Hub            *realtime.Hub
}

type Handler struct {
	payments       *sqlite.PaymentRepository
	walletService  *service.WalletService
	paymentService *service.PaymentService
	store          repository.Store
	auth           *auth.Manager
	chain          *chain.Client
	cluster        *cluster.Client
	hub            *realtime.Hub
	mpc            MPCClient
}

func New(dependencies Dependencies) *Handler {
	return &Handler{
		store:          dependencies.Store,
		payments:       dependencies.Payments,
		auth:           dependencies.Auth,
		chain:          dependencies.Chain,
		cluster:        dependencies.Cluster,
		hub:            dependencies.Hub,
		paymentService: dependencies.PaymentService,
		walletService:  dependencies.WalletService,
	}
}

// SetMPC supplies the signing client to the remaining wallet/swap handlers.
func (s *Handler) SetMPC(client MPCClient) { s.mpc = client }

func (s *Handler) Router(corsOrigin string) *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware(corsOrigin))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/api/v1/cluster", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"threshold": "2-of-3",
			"nodes":     s.cluster.Nodes(),
		})
	})

	v1 := router.Group("/api/v1")
	v1.POST("/auth/register", s.register)
	v1.POST("/auth/login", s.login)
	v1.GET("/events", s.events)
	v1.GET("/prices", s.getPrices)

	protected := v1.Group("")
	protected.Use(s.auth.Middleware())
	protected.GET("/wallets", s.listWallets)
	protected.POST("/wallets", s.createWallet)
	protected.GET("/wallets/:id", s.getWallet)
	protected.DELETE("/wallets/:id", s.deleteWallet)
	protected.GET("/wallets/:id/balance", s.walletBalance)
	protected.POST("/wallets/:id/fund", s.fundWallet)
	protected.POST("/wallets/:id/trustline", s.addTrustline)
	protected.GET("/wallets/:id/sync", s.syncWallet)
	protected.GET("/chains", s.getChains)
	protected.GET("/config", s.getConfig)
	protected.PUT("/config", s.putConfig)
	protected.POST("/assets", s.addAsset)
	protected.DELETE("/assets", s.removeAsset)
	protected.GET("/wallets/:id/transactions", s.listWalletTransactions)
	protected.POST("/transactions", s.createTransaction)
	protected.GET("/transactions/:id", s.getTransaction)
	protected.GET("/tx/:hash/chain", s.transactionOnChain)
	protected.GET("/resolve", s.resolveRecipient)
	protected.GET("/wallets/:id/swap/quote", s.swapQuote)
	protected.POST("/wallets/:id/swap", s.createSwap)

	return router
}

func corsMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
