package service

import (
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"

	"github.com/google/uuid"
)

type TrustlineChain interface {
	BuildTrustline(wallet domain.Wallet, code, issuer string) (string, []byte, error)
}

type TrustlineService struct {
	wallets            repository.Store
	transactions       *sqlite.TransactionRepository
	chain              TrustlineChain
	transactionService *TransactionService
}

func NewTrustlineService(wallets repository.Store, transactions *sqlite.TransactionRepository, chain TrustlineChain, transactionService *TransactionService) *TrustlineService {
	return &TrustlineService{wallets: wallets, transactions: transactions, chain: chain, transactionService: transactionService}
}

func (s *TrustlineService) Add(userID, walletID, code, issuer string) (domain.Transaction, error) {
	wallet, err := signableWallet(s.wallets, userID, walletID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if wallet.Chain != "stellar" {
		return domain.Transaction{}, apperror.TrustlineUnsupportedChain.New()
	}
	if code == "" || issuer == "" {
		return domain.Transaction{}, apperror.InvalidAsset.New()
	}
	exists, err := s.transactions.TrustlineTransactionExists(wallet.ID, code)
	if err != nil {
		return domain.Transaction{}, err
	}
	if exists {
		return domain.Transaction{}, apperror.TrustlineExists.New()
	}
	if err := s.transactionService.requireSigning(); err != nil {
		return domain.Transaction{}, err
	}

	envelope, hash, err := s.chain.BuildTrustline(wallet, code, issuer)
	if err != nil {
		return domain.Transaction{}, apperror.TrustlineBuildFailed.Wrap(err)
	}
	transaction := domain.Transaction{
		ID:           uuid.NewString(),
		WalletID:     wallet.ID,
		UserID:       userID,
		Type:         domain.TransactionOut,
		Counterparty: issuer,
		Amount:       "0.00",
		Symbol:       code,
		Memo:         domain.TrustlineMemo,
		Status:       domain.TransactionSigning,
		EnvelopeXDR:  envelope,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.transactionService.dispatch(wallet, transaction, hash); err != nil {
		return domain.Transaction{}, err
	}
	return transaction, nil
}
