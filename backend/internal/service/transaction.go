package service

import (
	"encoding/hex"
	"fmt"
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
	"go.uber.org/zap"
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

type TransactionService struct {
	wallets      repository.Store
	transactions *sqlite.TransactionRepository
	chain        PaymentChain
	cluster      SigningCluster
	mpc          PaymentSigner
	hub          *realtime.Hub
	approvals    *PaymentRequestService
	signs        signQueue
}

func NewTransactionService(wallets repository.Store, transactions *sqlite.TransactionRepository, chain PaymentChain, cluster SigningCluster, hub *realtime.Hub) *TransactionService {
	return &TransactionService{wallets: wallets, transactions: transactions, chain: chain, cluster: cluster, hub: hub, signs: signQueue{dispatchQueue: dispatchQueue{limit: maxInflightSigns}}}
}

func (s *TransactionService) SetMPC(client PaymentSigner) { s.mpc = client }

// SetApprovals routes payments through the workspace approval policy.
func (s *TransactionService) SetApprovals(approvals *PaymentRequestService) { s.approvals = approvals }

type PaymentInput struct {
	WalletID    string `json:"wallet_id"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Memo        string `json:"memo"`
	MemoType    string `json:"memo_type"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
}

// PaymentResult carries the transaction when the payment was sent, or the request when it awaits approval.
type PaymentResult struct {
	Transaction *domain.Transaction    `json:"transaction,omitempty"`
	Request     *domain.PaymentRequest `json:"request,omitempty"`
}

// preparedPayment is a validated payment with its recipient resolved, ready to build now or after approval.
type preparedPayment struct {
	Destination string
	Amount      string
	AssetCode   string
	AssetIssuer string
	Symbol      string
	Memo        string
	MemoType    string
}

func (p preparedPayment) native() bool {
	return p.AssetCode == "" || (p.AssetCode == "XLM" && p.AssetIssuer == "")
}

func (s *TransactionService) Create(userID string, body PaymentInput) (PaymentResult, error) {
	wallet, err := sendableWallet(s.wallets, userID, body.WalletID)
	if err != nil {
		return PaymentResult{}, err
	}
	payment, err := s.preparePayment(wallet, body)
	if err != nil {
		return PaymentResult{}, err
	}
	if s.approvals != nil {
		request, err := s.approvals.hold(wallet, userID, payment)
		if err != nil || request != nil {
			return PaymentResult{Request: request}, err
		}
	}
	if err := s.requireSigning(); err != nil {
		return PaymentResult{}, err
	}
	transaction, err := s.sendPayment(wallet, userID, payment, uuid.NewString())
	if err != nil {
		return PaymentResult{}, err
	}
	return PaymentResult{Transaction: &transaction}, nil
}

func (s *TransactionService) preparePayment(wallet domain.Wallet, body PaymentInput) (preparedPayment, error) {
	amount, err := parseAmount(body.Amount)
	if err != nil {
		return preparedPayment{}, err
	}
	if strings.TrimSpace(body.To) == "" {
		return preparedPayment{}, apperror.RecipientRequired.New()
	}

	// A federation server's required memo is used when the user gave none.
	memo, memoType := strings.TrimSpace(body.Memo), body.MemoType
	destination := strings.TrimSpace(body.To)
	if wallet.Chain == "stellar" {
		resolved, resolveErr := s.chain.ResolveRecipient(destination)
		if resolveErr != nil {
			return preparedPayment{}, apperror.RecipientResolutionFailed.Wrap(fmt.Errorf("resolve payment recipient: %w", resolveErr))
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
	return preparedPayment{
		Destination: destination,
		Amount:      amount,
		AssetCode:   body.AssetCode,
		AssetIssuer: body.AssetIssuer,
		Symbol:      symbol,
		Memo:        memo,
		MemoType:    memoType,
	}, nil
}

// sendPayment builds the envelope now, so it must run right before signing: envelopes expire after 300 seconds.
func (s *TransactionService) sendPayment(wallet domain.Wallet, initiatorID string, payment preparedPayment, transactionID string) (domain.Transaction, error) {
	transaction := domain.Transaction{
		ID:           transactionID,
		WalletID:     wallet.ID,
		UserID:       initiatorID,
		Type:         domain.TransactionOut,
		Counterparty: payment.Destination,
		Amount:       payment.Amount,
		Symbol:       payment.Symbol,
		Memo:         payment.Memo,
		Status:       domain.TransactionSigning,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	envelope, hash, buildErr := s.chain.BuildPayment(
		wallet, transaction.Counterparty, transaction.Amount,
		payment.AssetCode, payment.AssetIssuer, transaction.Memo, payment.MemoType,
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
	job := signJob{walletID: wallet.ID, network: domain.Chains[wallet.Chain].Network, hash: hash}
	if err := s.enqueueSign(transaction.ID, job); err != nil {
		return apperror.SignDispatchFailed.Wrap(fmt.Errorf("dispatch signing: %w", err))
	}
	publishTransaction(s.wallets, s.hub, transaction)
	return nil
}

func (s *TransactionService) requireSigning() error {
	return requireQuorum(s.cluster, SignQuorum, "signing")
}

func (s *TransactionService) ListForWallet(userID, walletID string, input PageInput) (Page[domain.Transaction], error) {
	before, limit, err := input.resolve()
	if err != nil {
		return Page[domain.Transaction]{}, err
	}
	if _, err := ownedWallet(s.wallets, userID, walletID); err != nil {
		return Page[domain.Transaction]{}, err
	}
	transactions, err := s.transactions.TransactionsByWallet(walletID, before, limit+1)
	if err != nil {
		return Page[domain.Transaction]{}, err
	}
	return pageOf(transactions, limit, func(transaction domain.Transaction) repository.Cursor {
		return repository.Cursor{CreatedAt: transaction.CreatedAt, ID: transaction.ID}
	}), nil
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

// OnSign returns the transaction when balances should be refreshed.
func (s *TransactionService) OnSign(result mpc.SignResult) *domain.Transaction {
	defer s.signFinished(result.TransactionID)
	transaction, ok := s.transactions.TransactionByID(result.TransactionID)
	if !ok {
		return nil
	}
	if !result.Successful {
		reason := result.Error
		if reason == "" {
			reason = "MPC signing failed"
		}
		zap.L().Warn("sign failed", zap.String("transaction", transaction.ID), zap.String("reason", reason))
		s.failSigning(transaction.ID, reason)
		return nil
	}

	claimed, err := s.transactions.ClaimSignature(transaction.ID, hex.EncodeToString(result.Signature))
	if err != nil {
		logError("claim signature tx="+transaction.ID, err)
		return nil
	}
	if !claimed {
		zap.L().Info("ignored sign result: transaction is no longer signing", zap.String("transaction", transaction.ID))
		return nil
	}
	s.publishTransactionByID(transaction.ID)

	wallet, ok := s.wallets.WalletByID(transaction.WalletID)
	if !ok {
		logError("broadcast tx="+transaction.ID, s.transactions.FailTransaction(transaction.ID, "wallet not found"))
		s.publishTransactionByID(transaction.ID)
		return nil
	}
	hash, err := s.chain.Submit(transaction.EnvelopeXDR, wallet.Pubkey, result.Signature)
	if err != nil {
		zap.L().Warn("broadcast failed", zap.String("transaction", transaction.ID), zap.Error(err))
		logError("broadcast failed tx="+transaction.ID, s.transactions.FailTransaction(transaction.ID, "broadcast: "+err.Error()))
		s.publishTransactionByID(transaction.ID)
		return nil
	}
	logError("confirm tx="+transaction.ID, s.transactions.ConfirmTransaction(transaction.ID, hash))
	s.publishTransactionByID(transaction.ID)
	return &transaction
}

func (s *TransactionService) failSigning(transactionID, reason string) {
	defer s.signFinished(transactionID)
	failed, err := s.transactions.FailSigning(transactionID, reason)
	if err != nil {
		logError("fail signing tx="+transactionID, err)
		return
	}
	if failed {
		s.publishTransactionByID(transactionID)
	}
}

func (s *TransactionService) publishTransactionByID(transactionID string) {
	transaction, ok := s.transactions.TransactionByID(transactionID)
	if !ok {
		return
	}
	publishTransaction(s.wallets, s.hub, transaction)
	if s.approvals != nil {
		s.approvals.settle(transaction)
	}
}

func (s *TransactionService) WatchSignTimeout(transactionID string) {
	time.AfterFunc(signTimeout, func() {
		s.failSigning(transactionID, "signing timed out — MPC nodes did not respond")
	})
}

func (s *TransactionService) ReconcilePending() {
	pending, err := s.transactions.PendingTransactions()
	if err != nil {
		logError("reconcile pending transactions", err)
		return
	}
	signOK := s.cluster.OnlineCount() >= SignQuorum
	for _, transaction := range pending {
		if signOK {
			s.WatchSignTimeout(transaction.ID)
			continue
		}
		s.failSigning(transaction.ID, "signing aborted — MPC cluster below quorum")
		zap.L().Warn("reconcile: failed stuck transaction, cluster below quorum", zap.String("transaction", transaction.ID))
	}
}
