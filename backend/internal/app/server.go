package app

import (
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/handler"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	AuthService  *service.AuthService
	FrontendURL  string
	PublicURL    string
	Store        repository.Store
	Transactions *sqlite.TransactionRepository
	Auth         *auth.Manager
	Chain        *chain.Client
	Cluster      *cluster.Client
	Hub          *realtime.Hub
}

type MPCClient interface {
	service.WalletKeygen
	service.PaymentSigner
}

// Server wires HTTP handlers and services. Business logic lives in service/.
type Server struct {
	routes       handler.Dependencies
	transactions *service.TransactionService
	wallets      *service.WalletService
	settings     *service.SettingsService
}

func NewServer(d Dependencies) *Server {
	transactions := service.NewTransactionService(d.Store, d.Transactions, d.Chain, d.Cluster, d.Hub)
	wallets := service.NewWalletService(d.Store, d.Transactions, d.Chain, d.Cluster, d.Hub)
	settings := service.NewSettingsService(d.Store, d.Chain)
	routes := handler.Dependencies{
		Auth:               d.Auth,
		AuthService:        d.AuthService,
		WalletService:      wallets,
		TransactionService: transactions,
		SwapService:        service.NewSwapService(d.Store, d.Chain, transactions),
		TrustlineService:   service.NewTrustlineService(d.Store, d.Transactions, d.Chain, transactions),
		SettingsService:    settings,
		Cluster:            d.Cluster,
		Hub:                d.Hub,
		FrontendURL:        d.FrontendURL,
		PublicURL:          d.PublicURL,
	}
	return &Server{routes: routes, transactions: transactions, wallets: wallets, settings: settings}
}

func (s *Server) Router(corsOrigin string) *gin.Engine {
	return handler.NewRouter(s.routes, corsOrigin)
}

func (s *Server) SetMPC(client MPCClient) {
	s.wallets.SetMPC(client)
	s.transactions.SetMPC(client)
}

func (s *Server) MPCCallbacks() mpc.Callbacks {
	return mpc.Callbacks{OnKeygen: s.wallets.OnKeygen, OnSign: func(result mpc.SignResult) {
		if tx := s.transactions.OnSign(result); tx != nil {
			go s.wallets.RefreshAfterTransaction(tx.WalletID, tx.Counterparty)
		}
	}}
}

func (s *Server) Start() {
	s.settings.ApplyRPC()
	s.wallets.StartBalanceRefresher()
	s.transactions.ReconcilePending()
	s.wallets.ReconcilePending()
}
