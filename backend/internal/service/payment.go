package service

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"
)

// PaymentChain is the Stellar boundary used by payment creation and broadcast.
type PaymentChain interface {
	ResolveRecipient(string) (chain.Resolved, error)
	BuildPayment(domain.Wallet, string, string, string, string, string, string) (string, []byte, error)
	Submit(string, string, []byte) (string, error)
}

type PaymentSigner interface {
	Sign(walletID, transactionID, network string, payload []byte) error
}

type SigningCluster interface{ OnlineCount() int }

// PaymentService owns payment creation and the shared signing/broadcast lifecycle.
// Other transaction types still build their own envelopes, then use this lifecycle.
type PaymentService struct {
	wallets  repository.Store
	payments *sqlite.PaymentRepository
	chain    PaymentChain
	cluster  SigningCluster
	mpc      PaymentSigner
	hub      *realtime.Hub
}

func NewPaymentService(wallets repository.Store, payments *sqlite.PaymentRepository, chain PaymentChain, cluster SigningCluster, hub *realtime.Hub) *PaymentService {
	return &PaymentService{wallets: wallets, payments: payments, chain: chain, cluster: cluster, hub: hub}
}

func (s *PaymentService) SetMPC(client PaymentSigner) { s.mpc = client }

// PaymentError lets the HTTP layer map a business failure to its existing status.
type PaymentError struct{ Kind, Message string }

func (e *PaymentError) Error() string { return e.Message }

type PaymentInput struct {
	WalletID    string `json:"wallet_id"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Memo        string `json:"memo"`
	MemoType    string `json:"memo_type"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
}

func (s *PaymentService) Create(userID string, body PaymentInput) (domain.Transaction, error) {
	wallet, ok := s.wallets.WalletFor(userID, body.WalletID)
	if !ok {
		return domain.Transaction{}, &PaymentError{Kind: "not_found", Message: "wallet not found"}
	}
	if wallet.Status != "ready" {
		return domain.Transaction{}, &PaymentError{Kind: "not_ready", Message: "wallet is not ready yet"}
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(body.Amount), 64)
	if err != nil || amount <= 0 {
		return domain.Transaction{}, &PaymentError{Kind: "invalid", Message: "amount must be positive"}
	}
	if strings.TrimSpace(body.To) == "" {
		return domain.Transaction{}, &PaymentError{Kind: "invalid", Message: "recipient required"}
	}
	if err := s.requireSigning(); err != nil {
		return domain.Transaction{}, &PaymentError{Kind: "unavailable", Message: err.Error()}
	}

	// Resolve federation / muxed recipients to a concrete address. If the
	// server requires a memo and the user didn't supply one, adopt it.
	memo, memoType := strings.TrimSpace(body.Memo), body.MemoType
	destination := strings.TrimSpace(body.To)
	if wallet.Chain == "stellar" {
		resolved, resolveErr := s.chain.ResolveRecipient(destination)
		if resolveErr != nil {
			return domain.Transaction{}, &PaymentError{Kind: "invalid", Message: resolveErr.Error()}
		}
		destination = resolved.Address
		if resolved.MemoType != "" {
			memoType = resolved.MemoType
			if memo == "" {
				memo = resolved.Memo
			}
		}
	}

	symbol := body.AssetCode
	if symbol == "" {
		symbol = wallet.Symbol
	}
	transaction := domain.Transaction{
		ID:           uuid.NewString(),
		WalletID:     wallet.ID,
		UserID:       userID,
		Type:         "out",
		Counterparty: destination,
		Amount:       strconv.FormatFloat(amount, 'f', 2, 64),
		Symbol:       symbol,
		Memo:         memo,
		Status:       "signing",
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	envelope, hash, buildErr := s.chain.BuildPayment(
		wallet, transaction.Counterparty, transaction.Amount,
		body.AssetCode, body.AssetIssuer, transaction.Memo, memoType,
	)
	if buildErr != nil {
		return domain.Transaction{}, &PaymentError{Kind: "invalid", Message: buildErr.Error()}
	}
	transaction.EnvelopeXDR = envelope
	payload := hash

	err = s.payments.CreateTransaction(transaction)
	if err != nil {
		return domain.Transaction{}, &PaymentError{Kind: "persistence", Message: "insert failed"}
	}

	if err := s.mpc.Sign(
		wallet.ID, transaction.ID, domain.Chains[wallet.Chain].Network, payload,
	); err != nil {
		s.payments.SetTransactionStatus(transaction.ID, "failed")
		return domain.Transaction{}, &PaymentError{Kind: "dispatch", Message: "sign dispatch failed: " + err.Error()}
	}
	s.WatchSignTimeout(userID, transaction.ID)
	s.hub.PublishTransaction(userID, transaction)
	return transaction, nil
}

func (s *PaymentService) requireSigning() error {
	online := s.cluster.OnlineCount()
	if online < SignQuorum {
		return fmt.Errorf("signing needs at least %d MPC nodes online (%d up)", SignQuorum, online)
	}
	return nil
}

// OnSign returns a broadcast transaction when balances should be refreshed.
func (s *PaymentService) OnSign(result mpc.SignResult) *domain.Transaction {
	transaction, ok := s.payments.TransactionByID(result.TransactionID)
	if !ok {
		return nil
	}
	userID := transaction.UserID

	if !result.Successful {
		log.Printf("sign failed tx=%s reason=%s", result.TransactionID, result.Error)
		reason := result.Error
		if reason == "" {
			reason = "MPC signing failed"
		}
		s.payments.FailTransaction(result.TransactionID, reason)
		s.publishTransactionByID(userID, result.TransactionID)
		return nil
	}

	signature := hex.EncodeToString(result.Signature)
	s.payments.SetTransactionSignature(result.TransactionID, signature)
	if transaction.EnvelopeXDR != "" {
		wallet, _ := s.wallets.WalletByID(transaction.WalletID)
		hash, err := s.chain.Submit(transaction.EnvelopeXDR, wallet.Pubkey, result.Signature)
		if err != nil {
			log.Printf("broadcast failed tx=%s: %v", result.TransactionID, err)
			s.payments.FailTransaction(result.TransactionID, "broadcast: "+err.Error())
			s.publishTransactionByID(userID, result.TransactionID)
			return nil
		}
		s.payments.BroadcastTransaction(result.TransactionID, hash)
		s.publishTransactionByID(userID, result.TransactionID)
		s.payments.SetTransactionStatus(result.TransactionID, "confirmed")
		s.publishTransactionByID(userID, result.TransactionID)
		return &transaction
	}

	s.payments.SetTransactionStatus(result.TransactionID, "confirmed")
	s.publishTransactionByID(userID, result.TransactionID)
	return nil
}

func (s *PaymentService) publishTransactionByID(userID, transactionID string) {
	if transaction, ok := s.payments.TransactionByID(transactionID); ok {
		s.hub.PublishTransaction(userID, transaction)
	}
}

// WatchSignTimeout fails a transaction still stuck in "signing" after the
// timeout — covers nodes that die (or never respond) after dispatch.
func (s *PaymentService) WatchSignTimeout(userID, transactionID string) {
	time.AfterFunc(signTimeout, func() {
		transaction, ok := s.payments.TransactionByID(transactionID)
		if !ok || transaction.Status != "signing" {
			return
		}
		s.payments.FailTransaction(transactionID, "signing timed out — MPC nodes did not respond")
		s.publishTransactionByID(userID, transactionID)
	})
}

func (s *PaymentService) ReconcilePending() {
	signOK := s.cluster.OnlineCount() >= SignQuorum
	for _, transaction := range s.payments.PendingTransactions() {
		if signOK {
			s.WatchSignTimeout(transaction.UserID, transaction.ID)
			continue
		}
		s.payments.FailTransaction(transaction.ID, "signing aborted — MPC cluster below quorum")
		s.publishTransactionByID(transaction.UserID, transaction.ID)
		log.Printf("reconcile: failed stuck tx=%s (cluster below quorum)", transaction.ID)
	}
}
