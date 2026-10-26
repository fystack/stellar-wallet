package service

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"

	"github.com/google/uuid"
)

// PaymentChain is the Stellar boundary used by payment creation and broadcast.
type PaymentChain interface {
	ResolveRecipient(string) (chain.Resolved, error)
	BuildPayment(domain.Wallet, string, string, string, string, string, string) (string, []byte, error)
	Submit(string, string, []byte) (string, error)
	TransactionOnChain(string) (map[string]any, error)
}

type PaymentSigner interface {
	Sign(walletID, transactionID, network string, payload []byte) error
}

type SigningCluster interface{ OnlineCount() int }

// TransactionService owns payment creation and the shared signing/broadcast lifecycle.
// Other transaction types still build their own envelopes, then use this lifecycle.
type TransactionService struct {
	wallets      repository.Store
	transactions *sqlite.TransactionRepository
	chain        PaymentChain
	cluster      SigningCluster
	mpc          PaymentSigner
	hub          *realtime.Hub
}

func NewTransactionService(wallets repository.Store, transactions *sqlite.TransactionRepository, chain PaymentChain, cluster SigningCluster, hub *realtime.Hub) *TransactionService {
	return &TransactionService{wallets: wallets, transactions: transactions, chain: chain, cluster: cluster, hub: hub}
}

func (s *TransactionService) SetMPC(client PaymentSigner) { s.mpc = client }

type PaymentInput struct {
	WalletID    string `json:"wallet_id"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Memo        string `json:"memo"`
	MemoType    string `json:"memo_type"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
}

func (s *TransactionService) Create(userID string, body PaymentInput) (domain.Transaction, error) {
	wallet, err := readyWallet(s.wallets, userID, body.WalletID)
	if err != nil {
		return domain.Transaction{}, err
	}
	amount, err := parseAmount(body.Amount)
	if err != nil {
		return domain.Transaction{}, err
	}
	if strings.TrimSpace(body.To) == "" {
		return domain.Transaction{}, apperror.RecipientRequired.New()
	}
	if err := s.requireSigning(); err != nil {
		return domain.Transaction{}, err
	}

	// Resolve federation / muxed recipients to a concrete address. If the
	// server requires a memo and the user didn't supply one, adopt it.
	memo, memoType := strings.TrimSpace(body.Memo), body.MemoType
	destination := strings.TrimSpace(body.To)
	if wallet.Chain == "stellar" {
		resolved, resolveErr := s.chain.ResolveRecipient(destination)
		if resolveErr != nil {
			return domain.Transaction{}, apperror.RecipientResolutionFailed.Wrap(fmt.Errorf("resolve payment recipient: %w", resolveErr))
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
		Type:         domain.TransactionOut,
		Counterparty: destination,
		Amount:       amount,
		Symbol:       symbol,
		Memo:         memo,
		Status:       domain.TransactionSigning,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	envelope, hash, buildErr := s.chain.BuildPayment(
		wallet, transaction.Counterparty, transaction.Amount,
		body.AssetCode, body.AssetIssuer, transaction.Memo, memoType,
	)
	if buildErr != nil {
		return domain.Transaction{}, apperror.PaymentBuildFailed.Wrap(fmt.Errorf("build payment: %w", buildErr))
	}
	transaction.EnvelopeXDR = envelope
	if err := s.dispatch(wallet, transaction, hash); err != nil {
		return domain.Transaction{}, err
	}
	return transaction, nil
}

// dispatch persists a built transaction and hands its hash to the cluster for signing.
func (s *TransactionService) dispatch(wallet domain.Wallet, transaction domain.Transaction, hash []byte) error {
	if err := s.transactions.CreateTransaction(transaction); err != nil {
		return fmt.Errorf("persist %s transaction: %w", transaction.Type, err)
	}
	if err := s.mpc.Sign(wallet.ID, transaction.ID, domain.Chains[wallet.Chain].Network, hash); err != nil {
		logError("dispatch tx="+transaction.ID, s.transactions.SetTransactionStatus(transaction.ID, domain.TransactionFailed))
		return apperror.SignDispatchFailed.Wrap(fmt.Errorf("dispatch signing: %w", err))
	}
	s.WatchSignTimeout(transaction.UserID, transaction.ID)
	s.hub.PublishTransaction(transaction.UserID, transaction)
	return nil
}

func (s *TransactionService) requireSigning() error {
	return requireQuorum(s.cluster, SignQuorum, "signing")
}

func (s *TransactionService) ListForWallet(userID, walletID string) ([]domain.Transaction, error) {
	if _, err := ownedWallet(s.wallets, userID, walletID); err != nil {
		return nil, err
	}
	return s.transactions.TransactionsByWallet(userID, walletID), nil
}

func (s *TransactionService) Get(userID, transactionID string) (domain.Transaction, error) {
	transaction, ok := s.transactions.TransactionForUser(transactionID, userID)
	if !ok {
		return domain.Transaction{}, apperror.TransactionNotFound.New()
	}
	return transaction, nil
}

// ResolveRecipient turns a G/M/federation recipient into a concrete address plus any required memo.
func (s *TransactionService) ResolveRecipient(query string) (chain.Resolved, error) {
	resolved, err := s.chain.ResolveRecipient(query)
	if err != nil {
		return chain.Resolved{}, apperror.RecipientResolutionFailed.Wrap(err)
	}
	return resolved, nil
}

func (s *TransactionService) OnChain(hash string) (map[string]any, error) {
	metadata, err := s.chain.TransactionOnChain(hash)
	if err != nil {
		return nil, apperror.TransactionNotOnChain.Wrap(err)
	}
	return metadata, nil
}

// OnSign returns a broadcast transaction when balances should be refreshed.
func (s *TransactionService) OnSign(result mpc.SignResult) *domain.Transaction {
	transaction, ok := s.transactions.TransactionByID(result.TransactionID)
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
		logError("sign failed tx="+result.TransactionID, s.transactions.FailTransaction(result.TransactionID, reason))
		s.publishTransactionByID(userID, result.TransactionID)
		return nil
	}

	signature := hex.EncodeToString(result.Signature)
	logError("store signature tx="+result.TransactionID, s.transactions.SetTransactionSignature(result.TransactionID, signature))
	if transaction.EnvelopeXDR != "" {
		wallet, _ := s.wallets.WalletByID(transaction.WalletID)
		hash, err := s.chain.Submit(transaction.EnvelopeXDR, wallet.Pubkey, result.Signature)
		if err != nil {
			log.Printf("broadcast failed tx=%s: %v", result.TransactionID, err)
			logError("broadcast failed tx="+result.TransactionID, s.transactions.FailTransaction(result.TransactionID, "broadcast: "+err.Error()))
			s.publishTransactionByID(userID, result.TransactionID)
			return nil
		}
		logError("broadcast tx="+result.TransactionID, s.transactions.BroadcastTransaction(result.TransactionID, hash))
		s.publishTransactionByID(userID, result.TransactionID)
		logError("confirm tx="+result.TransactionID, s.transactions.SetTransactionStatus(result.TransactionID, domain.TransactionConfirmed))
		s.publishTransactionByID(userID, result.TransactionID)
		return &transaction
	}

	logError("confirm tx="+result.TransactionID, s.transactions.SetTransactionStatus(result.TransactionID, domain.TransactionConfirmed))
	s.publishTransactionByID(userID, result.TransactionID)
	return nil
}

func (s *TransactionService) publishTransactionByID(userID, transactionID string) {
	if transaction, ok := s.transactions.TransactionByID(transactionID); ok {
		s.hub.PublishTransaction(userID, transaction)
	}
}

// WatchSignTimeout fails a transaction still stuck in "signing" after the
// timeout — covers nodes that die (or never respond) after dispatch.
func (s *TransactionService) WatchSignTimeout(userID, transactionID string) {
	time.AfterFunc(signTimeout, func() {
		transaction, ok := s.transactions.TransactionByID(transactionID)
		if !ok || transaction.Status != domain.TransactionSigning {
			return
		}
		logError("sign timeout tx="+transactionID, s.transactions.FailTransaction(transactionID, "signing timed out — MPC nodes did not respond"))
		s.publishTransactionByID(userID, transactionID)
	})
}

func (s *TransactionService) ReconcilePending() {
	signOK := s.cluster.OnlineCount() >= SignQuorum
	for _, transaction := range s.transactions.PendingTransactions() {
		if signOK {
			s.WatchSignTimeout(transaction.UserID, transaction.ID)
			continue
		}
		logError("reconcile tx="+transaction.ID, s.transactions.FailTransaction(transaction.ID, "signing aborted — MPC cluster below quorum"))
		s.publishTransactionByID(transaction.UserID, transaction.ID)
		log.Printf("reconcile: failed stuck tx=%s (cluster below quorum)", transaction.ID)
	}
}
