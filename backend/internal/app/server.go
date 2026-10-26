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
)

type Dependencies struct {
	Store    repository.Store
	Payments *sqlite.PaymentRepository
	Auth     *auth.Manager
	Chain    *chain.Client
	Cluster  *cluster.Client
	Hub      *realtime.Hub
}

// Server wires HTTP handlers and services. Business logic lives in service/.
type Server struct {
	HTTP     *handler.Handler
	payments *service.PaymentService
	wallets  *service.WalletService
}

func NewServer(d Dependencies) *Server {
	payments := service.NewPaymentService(d.Store, d.Payments, d.Chain, d.Cluster, d.Hub)
	wallets := service.NewWalletService(d.Store, d.Payments, d.Chain, d.Cluster, d.Hub)
	http := handler.New(handler.Dependencies{
		Store: d.Store, Payments: d.Payments, Auth: d.Auth, Chain: d.Chain,
		Cluster: d.Cluster, Hub: d.Hub, PaymentService: payments, WalletService: wallets,
	})
	return &Server{HTTP: http, payments: payments, wallets: wallets}
}

func (s *Server) SetMPC(client handler.MPCClient) {
	s.HTTP.SetMPC(client)
	s.payments.SetMPC(client)
}

func (s *Server) MPCCallbacks() mpc.Callbacks {
	return mpc.Callbacks{OnKeygen: s.wallets.OnKeygen, OnSign: func(result mpc.SignResult) {
		if tx := s.payments.OnSign(result); tx != nil {
			go s.wallets.RefreshAfterTransaction(tx.WalletID, tx.Counterparty)
		}
	}}
}

func (s *Server) Start() {
	s.HTTP.ApplyRPCConfig()
	s.wallets.StartBalanceRefresher()
	s.payments.ReconcilePending()
	s.wallets.ReconcilePending()
}
