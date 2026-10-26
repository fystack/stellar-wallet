package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository/sqlite"
	"stellar-wallet-backend/internal/service"
)

func TestCreatePaymentHTTPContract(t *testing.T) {
	tests := []struct {
		name, body, userID string
		setup              func(*paymentFixture, *paymentChainStub, *paymentSignerStub)
		status             int
		message            string
		code               string
	}{
		{name: "success", body: `{"wallet_id":"wallet-1","to":"destination","amount":"1"}`, status: 200},
		{name: "malformed JSON", body: `{`, status: 400, message: "invalid body", code: "invalid_request"},
		{name: "wrong field type", body: `{"amount":1}`, status: 400, message: "invalid body", code: "invalid_request"},
		{name: "other owner", body: `{"wallet_id":"wallet-1","to":"destination","amount":"1"}`, userID: "other-user", status: 404, message: "wallet not found", code: "wallet_not_found"},
		{name: "wallet generating", body: `{"wallet_id":"wallet-1","to":"destination","amount":"1"}`, setup: func(s *paymentFixture, _ *paymentChainStub, _ *paymentSignerStub) {
			s.wallets.SetWalletStatus("wallet-1", "generating")
		}, status: 409, message: "wallet is not ready yet", code: "wallet_not_ready"},
		{name: "invalid amount", body: `{"wallet_id":"wallet-1","to":"destination","amount":"0"}`, status: 400, message: "amount must be positive", code: "invalid_amount"},
		{name: "missing recipient", body: `{"wallet_id":"wallet-1","amount":"1"}`, status: 400, message: "recipient required", code: "recipient_required"},
		{name: "insufficient quorum", body: `{"wallet_id":"wallet-1","to":"destination","amount":"1"}`, setup: func(s *paymentFixture, _ *paymentChainStub, _ *paymentSignerStub) { s.cluster.online = 1 }, status: 503, message: "signing needs at least 2 MPC nodes online (1 up)", code: "mpc_unavailable"},
		{name: "build failed", body: `{"wallet_id":"wallet-1","to":"destination","amount":"1"}`, setup: func(_ *paymentFixture, c *paymentChainStub, _ *paymentSignerStub) {
			c.buildErr = errors.New("source account not found — fund it first")
		}, status: 400, message: "source account not found — fund it first", code: "payment_build_failed"},
		{name: "dispatch failed", body: `{"wallet_id":"wallet-1","to":"destination","amount":"1"}`, setup: func(_ *paymentFixture, _ *paymentChainStub, m *paymentSignerStub) { m.err = errors.New("offline") }, status: 502, message: "sign dispatch failed: offline", code: "sign_dispatch_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, c, signer := newTestPaymentService(t)
			if tt.setup != nil {
				tt.setup(service, c, signer)
			}
			manager := auth.New([]byte("test-secret"), time.Hour)
			server := New(Dependencies{Store: service.wallets, Payments: service.payments, Auth: manager, Hub: service.hub})
			server.paymentService = service.service
			userID := tt.userID
			if userID == "" {
				userID = "owner"
			}
			token, _, err := manager.MakeToken(userID)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBufferString(tt.body))
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			server.Router("http://localhost").ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("response = %d %s", response.Code, response.Body)
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tt.message != "" {
				if body["error"] != tt.message {
					t.Fatalf("response = %s", response.Body)
				}
				return
			}
			if body["status"] != "signing" || body["walletId"] != "wallet-1" || body["amount"] != "1.00" || body["id"] == "" || body["createdAt"] == "" {
				t.Fatalf("response = %s", response.Body)
			}
			for _, private := range []string{"user_id", "userId", "envelope_xdr", "EnvelopeXDR"} {
				if _, ok := body[private]; ok {
					t.Fatalf("private field %q in response", private)
				}
			}
		})
	}
}

type paymentChainStub struct {
	resolved          chain.Resolved
	buildErr          error
	submitErr         error
	submittedEnvelope string
	builtAmount       string
	builtMemo         string
	builtMemoType     string
}

func (c *paymentChainStub) ResolveRecipient(string) (chain.Resolved, error) { return c.resolved, nil }
func (c *paymentChainStub) BuildPayment(_ domain.Wallet, _ string, amount, _, _, memo, memoType string) (string, []byte, error) {
	c.builtAmount, c.builtMemo, c.builtMemoType = amount, memo, memoType
	return "unsigned-envelope", []byte("payment-hash"), c.buildErr
}
func (c *paymentChainStub) Submit(envelope, _ string, _ []byte) (string, error) {
	c.submittedEnvelope = envelope
	return "stellar-tx-hash", c.submitErr
}

type paymentSignerStub struct {
	err                              error
	walletID, transactionID, network string
	payload                          []byte
}

func (m *paymentSignerStub) Sign(walletID, transactionID, network string, payload []byte) error {
	m.walletID, m.transactionID, m.network, m.payload = walletID, transactionID, network, payload
	return m.err
}

type paymentClusterStub struct{ online int }

func (c *paymentClusterStub) OnlineCount() int { return c.online }

type paymentFixture struct {
	wallets  *sqlite.Store
	payments *sqlite.PaymentRepository
	hub      *realtime.Hub
	cluster  *paymentClusterStub
	service  *service.PaymentService
}

func newTestPaymentService(t *testing.T) (*paymentFixture, *paymentChainStub, *paymentSignerStub) {
	t.Helper()
	db, err := sqlite.Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	wallets := sqlite.NewStore(db)
	if err := wallets.CreateWallet(domain.Wallet{ID: "wallet-1", UserID: "owner", Chain: "stellar", Symbol: "XLM", Status: "ready"}); err != nil {
		t.Fatal(err)
	}
	payments := sqlite.NewPaymentRepository(db)
	hub := realtime.NewHub()
	cluster := &paymentClusterStub{online: 3}
	chain := &paymentChainStub{resolved: chain.Resolved{Address: "resolved-address"}}
	signer := &paymentSignerStub{}
	svc := service.NewPaymentService(wallets, payments, chain, cluster, hub)
	svc.SetMPC(signer)
	return &paymentFixture{wallets: wallets, payments: payments, hub: hub, cluster: cluster, service: svc}, chain, signer
}
