package service

import (
	"encoding/hex"
	"log"
	"strconv"
	"time"

	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/cluster"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"

	"github.com/google/uuid"
)

// WalletService handles wallet keygen results and incoming payment synchronization.
type WalletService struct {
	store    repository.Store
	payments *sqlite.PaymentRepository
	chain    *chain.Client
	cluster  *cluster.Client
	hub      *realtime.Hub
}

func NewWalletService(store repository.Store, payments *sqlite.PaymentRepository, chain *chain.Client, cluster *cluster.Client, hub *realtime.Hub) *WalletService {
	return &WalletService{store: store, payments: payments, chain: chain, cluster: cluster, hub: hub}
}

func (s *WalletService) OnKeygen(result mpc.KeygenResult) {
	if !result.Successful {
		log.Printf("keygen failed wallet=%s reason=%s", result.WalletID, result.Error)
		s.store.SetWalletStatus(result.WalletID, "failed")
		return
	}

	wallet, ok := s.store.WalletByID(result.WalletID)
	if !ok {
		return
	}
	address := chain.DeriveAddress(wallet.Chain, result.PublicKey)
	publicKeyHex := hex.EncodeToString(result.PublicKey)
	s.store.CompleteWalletKeygen(result.WalletID, address, publicKeyHex)
	if updated, found := s.store.WalletByID(result.WalletID); found {
		s.hub.PublishWallet(wallet.UserID, updated)
	}
}

func (s *WalletService) SyncIncoming(wallet domain.Wallet) int {
	if wallet.Chain != "stellar" {
		return 0
	}
	payments, err := s.chain.Incoming(wallet.Address)
	if err != nil {
		return 0
	}

	added := 0
	for _, payment := range payments {
		if s.payments.IncomingTransactionExists(payment.Hash) {
			continue
		}
		amount, _ := strconv.ParseFloat(payment.Amount, 64)
		createdAt := payment.At
		if createdAt == "" {
			createdAt = time.Now().UTC().Format(time.RFC3339)
		}
		transaction := domain.Transaction{
			ID:           uuid.NewString(),
			WalletID:     wallet.ID,
			UserID:       wallet.UserID,
			Type:         "in",
			Counterparty: payment.From,
			Amount:       strconv.FormatFloat(amount, 'f', 4, 64),
			Symbol:       payment.Symbol,
			Status:       "confirmed",
			TxHash:       payment.Hash,
			CreatedAt:    createdAt,
		}
		_ = s.payments.CreateTransaction(transaction)
		s.hub.PublishTransaction(wallet.UserID, transaction)
		added++
	}
	if added > 0 {
		s.RefreshWallet(wallet.ID)
	}
	return added
}
