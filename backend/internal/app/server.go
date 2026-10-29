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
	FrontendURL     string
	PublicURL       string
	AuthRateLimit   int
	Store           repository.Store
	Transactions    *sqlite.TransactionRepository
	AuthRepository  *sqlite.AuthRepository
	Workspaces      *sqlite.WorkspaceRepository
	PaymentRequests *sqlite.PaymentRequestRepository
	Mailer          service.Mailer
	OAuthProviders  map[string]auth.OAuthProvider
	Auth            *auth.Manager
	Chain           *chain.Client
	Cluster         *cluster.Client
	Hub             *realtime.Hub
}

type MPCClient interface {
	service.WalletKeygen
	service.PaymentSigner
}

// Server wires HTTP handlers and services. Business logic lives in service/.
type Server struct {
	routes       handler.Dependencies
	transactions *service.TransactionService
	approvals    *service.PaymentRequestService
	wallets      *service.WalletService
	settings     *service.SettingsService
}

func NewServer(d Dependencies) *Server {
	transactions := service.NewTransactionService(d.Store, d.Transactions, d.Chain, d.Cluster, d.Hub)
	wallets := service.NewWalletService(d.Store, d.Transactions, d.Workspaces, d.Chain, d.Cluster, d.Hub)
	settings := service.NewSettingsService(d.Chain)
	approvals := service.NewPaymentRequestService(d.Store, d.Workspaces, d.PaymentRequests, transactions, d.Hub, d.Mailer, d.FrontendURL)
	transactions.SetApprovals(approvals)
	routes := handler.Dependencies{
		Auth:                  d.Auth,
		AuthService:           service.NewAuthService(d.AuthRepository, d.OAuthProviders),
		WalletService:         wallets,
		TransactionService:    transactions,
		SwapService:           service.NewSwapService(d.Store, d.Chain, transactions),
		TrustlineService:      service.NewTrustlineService(d.Store, d.Transactions, d.Chain, transactions),
		SettingsService:       settings,
		WorkspaceService:      service.NewWorkspaceService(d.Workspaces, d.AuthRepository, wallets, d.Mailer, d.FrontendURL),
		PaymentRequestService: approvals,
		Cluster:               d.Cluster,
		Hub:                   d.Hub,
		FrontendURL:           d.FrontendURL,
		PublicURL:             d.PublicURL,
		AuthRateLimit:         d.AuthRateLimit,
	}
	return &Server{routes: routes, transactions: transactions, approvals: approvals, wallets: wallets, settings: settings}
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
		if transaction := s.transactions.OnSign(result); transaction != nil {
			go s.wallets.RefreshAfterTransaction(transaction.WalletID, transaction.Counterparty)
		}
	}}
}

func (s *Server) Start() {
	s.wallets.StartBalanceRefresher()
	s.transactions.ReconcilePending()
	s.approvals.ReconcileExecuting()
	s.wallets.ReconcilePending()
}
