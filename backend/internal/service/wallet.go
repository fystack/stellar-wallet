package service

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WalletKeygen interface {
	CreateWallet(walletID string) error
}

// WalletService owns wallet lifecycle, balances and incoming payment synchronization.
type WalletService struct {
	store        repository.Store
	transactions *sqlite.TransactionRepository
	chain        *chain.Client
	cluster      *cluster.Client
	hub          *realtime.Hub
	mpc          WalletKeygen
	workspaces   *sqlite.WorkspaceRepository
}

func NewWalletService(store repository.Store, transactions *sqlite.TransactionRepository, workspaces *sqlite.WorkspaceRepository, chain *chain.Client, cluster *cluster.Client, hub *realtime.Hub) *WalletService {
	return &WalletService{store: store, transactions: transactions, workspaces: workspaces, chain: chain, cluster: cluster, hub: hub}
}

func (s *WalletService) SetMPC(client WalletKeygen) { s.mpc = client }

type WalletBalance struct {
	Assets   []domain.AssetBalance `json:"assets"`
	Symbol   string                `json:"symbol"`
	Explorer string                `json:"explorer"`
}

func (s *WalletService) List(userID, sessionWorkspaceID, requestedWorkspaceID string) ([]domain.Wallet, error) {
	workspace, err := requestedWorkspace(s.workspaces, userID, sessionWorkspaceID, requestedWorkspaceID)
	if err != nil {
		return nil, err
	}
	wallets, err := s.store.ListWallets(workspace.ID)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	return wallets, nil
}

func (s *WalletService) Get(userID, walletID string) (domain.Wallet, error) {
	return ownedWallet(s.store, userID, walletID)
}

func (s *WalletService) Create(userID, sessionWorkspaceID, requestedWorkspaceID, name, chainName string) (domain.Wallet, error) {
	workspace, err := requestedWorkspace(s.workspaces, userID, sessionWorkspaceID, requestedWorkspaceID)
	if err != nil {
		return domain.Wallet{}, err
	}
	if workspace.Role != domain.RoleOwner {
		return domain.Wallet{}, apperror.WorkspaceOwnerRequired.New()
	}
	return s.Provision(userID, workspace.ID, name, chainName)
}

// Provision skips authorization; callers must have checked userID's access to workspaceID.
func (s *WalletService) Provision(userID, workspaceID, name, chainName string) (domain.Wallet, error) {
	metadata, ok := domain.Chains[chainName]
	if !ok {
		return domain.Wallet{}, apperror.UnsupportedChain.New()
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = capitalize(chainName) + " Wallet"
	}
	if err := requireQuorum(s.cluster, KeygenQuorum, "keygen"); err != nil {
		return domain.Wallet{}, err
	}

	wallet := domain.Wallet{
		ID:          uuid.NewString(),
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        name,
		Chain:       chainName,
		Symbol:      metadata.Symbol,
		Balance:     "0.00",
		Status:      domain.WalletGenerating,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.store.CreateWallet(wallet); err != nil {
		return domain.Wallet{}, fmt.Errorf("persist wallet: %w", err)
	}
	if err := s.mpc.CreateWallet(wallet.ID); err != nil {
		logError("keygen dispatch wallet="+wallet.ID, s.store.SetWalletStatus(wallet.ID, domain.WalletFailed))
		return domain.Wallet{}, apperror.KeygenDispatchFailed.Wrap(err)
	}
	s.WatchKeygenTimeout(wallet.ID)
	return wallet, nil
}

func (s *WalletService) Delete(userID, walletID string) error {
	wallet, err := ownedWallet(s.store, userID, walletID)
	if err != nil {
		return err
	}
	if err := requireOwner(s.workspaces, wallet.WorkspaceID, userID); err != nil {
		return err
	}
	return s.store.DeleteWallet(walletID)
}

// Balance reads live balances and refreshes the cached wallet balance in the background.
func (s *WalletService) Balance(userID, walletID string) (WalletBalance, error) {
	wallet, err := readyWallet(s.store, userID, walletID)
	if err != nil {
		return WalletBalance{}, err
	}
	assets, err := s.chain.Balances(wallet.Chain, wallet.Address)
	if err != nil {
		return WalletBalance{}, apperror.BalanceUnavailable.Wrap(err)
	}
	go s.RefreshWallet(wallet.ID)
	return WalletBalance{
		Assets:   assets,
		Symbol:   wallet.Symbol,
		Explorer: chain.ExplorerAddress(wallet.Chain, wallet.Address),
	}, nil
}

// Fund tops the wallet up from the testnet faucet and returns the new native balance.
func (s *WalletService) Fund(userID, walletID string) (string, error) {
	wallet, err := readyWallet(s.store, userID, walletID)
	if err != nil {
		return "", err
	}
	if err := s.chain.Fund(wallet.Chain, wallet.Address); err != nil {
		return "", apperror.FundingFailed.Wrap(err)
	}
	if _, err := s.SyncIncoming(wallet); err != nil {
		logError("sync after funding wallet="+wallet.ID, err)
	}
	balance, err := s.chain.Balance(wallet.Chain, wallet.Address)
	if err != nil {
		return "", apperror.BalanceUnavailable.Wrap(err)
	}
	if err := s.store.SetWalletBalance(wallet.ID, balance); err != nil {
		return "", err
	}
	if updated, found := s.store.WalletByID(wallet.ID); found {
		publishWallet(s.store, s.hub, updated)
	}
	return balance, nil
}

func (s *WalletService) Sync(userID, walletID string) (int, error) {
	wallet, err := readyWallet(s.store, userID, walletID)
	if err != nil {
		return 0, err
	}
	return s.SyncIncoming(wallet)
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func (s *WalletService) OnKeygen(result mpc.KeygenResult) {
	if !result.Successful {
		zap.L().Warn("keygen failed", zap.String("wallet", result.WalletID), zap.String("reason", result.Error))
		logError("keygen failed wallet="+result.WalletID, s.store.SetWalletStatus(result.WalletID, domain.WalletFailed))
		return
	}

	wallet, ok := s.store.WalletByID(result.WalletID)
	if !ok {
		return
	}
	address := chain.DeriveAddress(wallet.Chain, result.PublicKey)
	publicKeyHex := hex.EncodeToString(result.PublicKey)
	if err := s.store.CompleteWalletKeygen(result.WalletID, address, publicKeyHex); err != nil {
		logError("complete keygen wallet="+result.WalletID, err)
		return
	}
	if updated, found := s.store.WalletByID(result.WalletID); found {
		publishWallet(s.store, s.hub, updated)
	}
}

// SyncIncoming records new incoming payments from Horizon and returns how many were added.
func (s *WalletService) SyncIncoming(wallet domain.Wallet) (int, error) {
	if wallet.Chain != "stellar" {
		return 0, nil
	}
	payments, err := s.chain.Incoming(wallet.Address)
	if err != nil {
		return 0, apperror.SyncFailed.Wrap(err)
	}

	added := 0
	for _, payment := range payments {
		exists, err := s.transactions.IncomingTransactionExists(payment.Hash)
		if err != nil {
			return added, err
		}
		if exists {
			continue
		}
		createdAt := payment.At
		if createdAt == "" {
			createdAt = time.Now().UTC().Format(time.RFC3339)
		}
		transaction := domain.Transaction{
			ID:           uuid.NewString(),
			WalletID:     wallet.ID,
			UserID:       wallet.UserID,
			Type:         domain.TransactionIn,
			Counterparty: payment.From,
			Amount:       trimAmount(payment.Amount),
			Symbol:       payment.Symbol,
			Status:       domain.TransactionConfirmed,
			TxHash:       payment.Hash,
			CreatedAt:    createdAt,
		}
		if err := s.transactions.CreateTransaction(transaction); err != nil {
			logError("record incoming payment hash="+payment.Hash, err)
			continue
		}
		publishTransaction(s.store, s.hub, transaction)
		added++
	}
	if added > 0 {
		s.RefreshWallet(wallet.ID)
	}
	return added, nil
}
