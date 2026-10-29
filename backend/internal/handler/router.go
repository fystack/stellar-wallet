package handler

import (
	"net/http"
	"strings"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Auth               *auth.Manager
	AuthService        *service.AuthService
	WalletService      *service.WalletService
	TransactionService *service.TransactionService
	SwapService        *service.SwapService
	TrustlineService   *service.TrustlineService
	SettingsService    *service.SettingsService
	Cluster            *cluster.Client
	Hub                *realtime.Hub
	FrontendURL        string
	PublicURL          string
}

func NewRouter(d Dependencies, corsOrigin string) *gin.Engine {
	corsOrigin = strings.TrimRight(corsOrigin, "/")
	if d.FrontendURL == "" {
		d.FrontendURL = corsOrigin
	}

	authHandler := &AuthHandler{sessions: d.Auth, accounts: d.AuthService, frontendURL: d.FrontendURL}
	wallets := &WalletHandler{wallets: d.WalletService, trustlines: d.TrustlineService}
	transactions := &TransactionHandler{transactions: d.TransactionService}
	swaps := &SwapHandler{swaps: d.SwapService}
	settings := &SettingsHandler{settings: d.SettingsService, cluster: d.Cluster}
	events := &EventsHandler{sessions: d.Auth, hub: d.Hub}

	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.NoRoute(func(c *gin.Context) { httpapi.Error(c, apperror.RouteNotFound.New()) })
	router.NoMethod(func(c *gin.Context) { httpapi.Error(c, apperror.UnsupportedMethod.New()) })
	router.Use(
		requestLogger(),
		recoverPanics(),
		corsMiddleware(corsOrigin),
		authRateLimit(),
		csrfProtection(d.FrontendURL, d.PublicURL),
	)

	router.GET("/health", func(c *gin.Context) {
		httpapi.Success(c, http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")
	v1.GET("/cluster", settings.clusterStatus)
	v1.POST("/auth/register", authHandler.register)
	v1.POST("/auth/login", authHandler.login)
	v1.GET("/auth/providers", authHandler.providers)
	v1.GET("/auth/:provider/start", authHandler.oauthStart)
	v1.GET("/auth/:provider/callback", authHandler.oauthCallback)
	v1.POST("/auth/logout", authHandler.logout)
	v1.GET("/prices", settings.getPrices)

	protected := v1.Group("")
	protected.Use(d.Auth.Middleware())
	protected.GET("/auth/me", authHandler.me)
	protected.POST("/auth/reauthenticate", authHandler.reauthenticate)
	protected.POST("/auth/:provider/link", authHandler.oauthLink)
	protected.GET("/events", events.stream)

	protected.GET("/wallets", wallets.list)
	protected.POST("/wallets", wallets.create)
	protected.GET("/wallets/:id", wallets.get)
	protected.DELETE("/wallets/:id", wallets.delete)
	protected.GET("/wallets/:id/balance", wallets.balance)
	protected.POST("/wallets/:id/fund", wallets.fund)
	protected.GET("/wallets/:id/sync", wallets.sync)
	protected.POST("/wallets/:id/trustline", wallets.addTrustline)
	protected.GET("/wallets/:id/transactions", transactions.listForWallet)
	protected.GET("/wallets/:id/swap/quote", swaps.quote)
	protected.POST("/wallets/:id/swap", swaps.create)

	protected.POST("/transactions", transactions.create)
	protected.GET("/transactions/:id", transactions.get)
	protected.GET("/tx/:hash/chain", transactions.onChain)
	protected.GET("/resolve", transactions.resolveRecipient)

	protected.GET("/chains", settings.getChains)
	protected.GET("/config", settings.getConfig)
	protected.POST("/assets", settings.addAsset)
	protected.DELETE("/assets", settings.removeAsset)

	return router
}
