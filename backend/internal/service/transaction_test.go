package service

import (
	"errors"
	"testing"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository/sqlite"
)

type paymentChainStub struct {
	resolved          chain.Resolved
	buildErr          error
	submitErr         error
	submittedEnvelope string
	submitCount       int
	builtAmount       string
	builtMemo         string
	builtMemoType     string
}

func (c *paymentChainStub) ResolveRecipient(string) (chain.Resolved, error) { return c.resolved, nil }
func (c *paymentChainStub) BuildPayment(_ domain.Wallet, _ string, amount, _, _, memo, memoType string) (string, []byte, error) {
	c.builtAmount, c.builtMemo, c.builtMemoType = amount, memo, memoType
	return "unsigned-envelope", []byte("payment-hash"), c.buildErr
}
func (c *paymentChainStub) TransactionOnChain(string) (map[string]any, error) { return nil, nil }
func (c *paymentChainStub) Submit(envelope, _ string, _ []byte) (string, error) {
	c.submittedEnvelope = envelope
	c.submitCount++
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

type paymentClusterStub int

func (c paymentClusterStub) OnlineCount() int { return int(c) }

func newTestTransactionService(t *testing.T) (*TransactionService, *paymentChainStub, *paymentSignerStub) {
	t.Helper()
	db, err := sqlite.Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	wallets := sqlite.NewStore(db)
	if err := wallets.CreateWallet(domain.Wallet{ID: "wallet-1", UserID: "owner", Chain: "stellar", Symbol: "XLM", Pubkey: "public-key", Status: "ready"}); err != nil {
		t.Fatal(err)
	}
	c := &paymentChainStub{resolved: chain.Resolved{Address: "resolved-address", Memo: "42", MemoType: "id"}}
	signer := &paymentSignerStub{}
	service := &TransactionService{wallets: wallets, transactions: sqlite.NewTransactionRepository(db), chain: c, mpc: signer, cluster: paymentClusterStub(3), hub: realtime.NewHub()}
	return service, c, signer
}

func TestPaymentCreatePersistsResolvedPaymentAndDispatchesHash(t *testing.T) {
	s, c, signer := newTestTransactionService(t)
	events := s.hub.Subscribe("owner")
	defer s.hub.Unsubscribe("owner", events)
	tx, err := s.Create("owner", PaymentInput{WalletID: "wallet-1", To: "name*example.com", Amount: "12.345"})
	if err != nil {
		t.Fatal(err)
	}
	stored, ok := s.transactions.TransactionForUser(tx.ID, "owner")
	if !ok || stored.Status != "signing" || stored.EnvelopeXDR != "unsigned-envelope" || stored.Counterparty != "resolved-address" || stored.Memo != "42" {
		t.Fatalf("stored payment = %#v, found=%v", stored, ok)
	}
	// Preserve the current two-decimal behavior in this extraction; precision is a separate fix.
	if c.builtAmount != "12.345" || c.builtMemo != "42" || c.builtMemoType != "id" {
		t.Fatalf("unexpected build: %#v", c)
	}
	if signer.walletID != tx.WalletID || signer.transactionID != tx.ID || signer.network != "stellar-testnet" || string(signer.payload) != "payment-hash" {
		t.Fatalf("unexpected signing request: %#v", signer)
	}
	select {
	case <-events:
	default:
		t.Fatal("missing transaction event")
	}
}

func TestPaymentCreateDispatchFailurePersistsFailedStatus(t *testing.T) {
	s, _, signer := newTestTransactionService(t)
	signer.err = errors.New("NATS unavailable")
	_, err := s.Create("owner", PaymentInput{WalletID: "wallet-1", To: "destination", Amount: "1"})
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Kind != apperror.Upstream || failure.Code != "sign_dispatch_failed" || !errors.Is(err, signer.err) {
		t.Fatalf("error = %v", err)
	}
	tx, ok := s.transactions.TransactionByID(signer.transactionID)
	if !ok || tx.Status != "failed" {
		t.Fatalf("transaction = %#v, found=%v", tx, ok)
	}
}

func TestPaymentCreateRejectsOtherUsersWalletBeforeSigning(t *testing.T) {
	s, _, signer := newTestTransactionService(t)
	_, err := s.Create("other-user", PaymentInput{WalletID: "wallet-1", To: "destination", Amount: "1"})
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Kind != apperror.NotFound || failure.Code != "wallet_not_found" {
		t.Fatalf("error = %v", err)
	}
	if signer.transactionID != "" {
		t.Fatal("unauthorized payment dispatched")
	}
}

func TestPaymentSignResultBroadcastsPersistedEnvelope(t *testing.T) {
	s, c, _ := newTestTransactionService(t)
	tx, err := s.Create("owner", PaymentInput{WalletID: "wallet-1", To: "destination", Amount: "1"})
	if err != nil {
		t.Fatal(err)
	}
	completed := s.OnSign(mpc.SignResult{TransactionID: tx.ID, Successful: true, Signature: []byte{1, 2}})
	stored, _ := s.transactions.TransactionByID(tx.ID)
	if completed == nil || stored.Status != "confirmed" || stored.TxHash != "stellar-tx-hash" || stored.Signature != "0102" || c.submittedEnvelope != "unsigned-envelope" {
		t.Fatalf("transaction = %#v, refresh = %#v", stored, completed)
	}
}

func TestPaymentSignResultFailuresPersistReason(t *testing.T) {
	for _, stage := range []string{"signing", "broadcast"} {
		t.Run(stage, func(t *testing.T) {
			s, c, _ := newTestTransactionService(t)
			tx, err := s.Create("owner", PaymentInput{WalletID: "wallet-1", To: "destination", Amount: "1"})
			if err != nil {
				t.Fatal(err)
			}
			result := mpc.SignResult{TransactionID: tx.ID, Successful: stage == "broadcast"}
			want := "MPC signing failed"
			if stage == "broadcast" {
				c.submitErr = errors.New("Horizon unavailable")
				want = "broadcast: Horizon unavailable"
			}
			if completed := s.OnSign(result); completed != nil {
				t.Fatal("failed transaction requested refresh")
			}
			stored, _ := s.transactions.TransactionByID(tx.ID)
			if stored.Status != "failed" || stored.Error != want {
				t.Fatalf("transaction = %#v", stored)
			}
		})
	}
}

func TestPaymentReconcileFailsPendingPaymentsBelowQuorum(t *testing.T) {
	s, _, _ := newTestTransactionService(t)
	tx, err := s.Create("owner", PaymentInput{WalletID: "wallet-1", To: "destination", Amount: "1"})
	if err != nil {
		t.Fatal(err)
	}
	s.cluster = paymentClusterStub(1)
	s.ReconcilePending()
	stored, _ := s.transactions.TransactionByID(tx.ID)
	if stored.Status != "failed" || stored.Error != "signing aborted — MPC cluster below quorum" {
		t.Fatalf("transaction = %#v", stored)
	}
}

func TestPaymentSignResultAlsoBroadcastsSwapAndTrustline(t *testing.T) {
	for _, kind := range []string{"swap", "trustline"} {
		t.Run(kind, func(t *testing.T) {
			s, c, _ := newTestTransactionService(t)
			tx := domain.Transaction{ID: kind, WalletID: "wallet-1", UserID: "owner", Type: "swap", Status: "signing", EnvelopeXDR: kind + "-envelope"}
			if kind == "trustline" {
				tx.Type = "out"
				tx.Memo = "Add trustline"
			}
			if err := s.transactions.CreateTransaction(tx); err != nil {
				t.Fatal(err)
			}
			if s.OnSign(mpc.SignResult{TransactionID: tx.ID, Successful: true, Signature: []byte{1}}) == nil {
				t.Fatal("missing balance refresh")
			}
			stored, _ := s.transactions.TransactionByID(tx.ID)
			if stored.Status != "confirmed" || stored.TxHash != "stellar-tx-hash" || c.submittedEnvelope != tx.EnvelopeXDR {
				t.Fatalf("transaction = %#v", stored)
			}
		})
	}
}

func TestSignResultIsAppliedOnlyWhileSigning(t *testing.T) {
	t.Run("duplicate result broadcasts once", func(t *testing.T) {
		s, c, _ := newTestTransactionService(t)
		transaction, err := s.Create("owner", PaymentInput{WalletID: "wallet-1", To: "destination", Amount: "1"})
		if err != nil {
			t.Fatal(err)
		}
		result := mpc.SignResult{TransactionID: transaction.ID, Successful: true, Signature: []byte{1}}
		if s.OnSign(result) == nil {
			t.Fatal("first result was not broadcast")
		}
		if s.OnSign(result) != nil {
			t.Fatal("duplicate result was broadcast")
		}
		stored, _ := s.transactions.TransactionByID(transaction.ID)
		if c.submitCount != 1 || stored.Status != domain.TransactionConfirmed {
			t.Fatalf("submits = %d, stored = %#v", c.submitCount, stored)
		}
	})
	t.Run("late result after timeout is ignored", func(t *testing.T) {
		s, c, _ := newTestTransactionService(t)
		transaction, err := s.Create("owner", PaymentInput{WalletID: "wallet-1", To: "destination", Amount: "1"})
		if err != nil {
			t.Fatal(err)
		}
		s.failSigning(transaction.UserID, transaction.ID, "signing timed out")
		if s.OnSign(mpc.SignResult{TransactionID: transaction.ID, Successful: true, Signature: []byte{1}}) != nil {
			t.Fatal("late result was broadcast")
		}
		stored, _ := s.transactions.TransactionByID(transaction.ID)
		if c.submitCount != 0 || stored.Status != domain.TransactionFailed || stored.Signature != "" {
			t.Fatalf("submits = %d, stored = %#v", c.submitCount, stored)
		}
	})
}
