package app

import (
	"encoding/hex"
	"log"

	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/mpcium"
)

func (s *Server) onKeygen(result mpcium.KeygenResult) {
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
	if updated, found := s.walletByID(result.WalletID); found {
		s.hub.PublishWallet(wallet.UserID, updated)
	}
}

func (s *Server) onSign(result mpcium.SignResult) {
	transaction, ok := s.store.TransactionByID(result.TransactionID)
	if !ok {
		return
	}
	userID := transaction.UserID

	if !result.Successful {
		log.Printf("sign failed tx=%s reason=%s", result.TransactionID, result.Error)
		reason := result.Error
		if reason == "" {
			reason = "MPC signing failed"
		}
		s.store.FailTransaction(result.TransactionID, reason)
		s.publishTransactionByID(userID, result.TransactionID)
		return
	}

	signature := hex.EncodeToString(result.Signature)
	s.store.SetTransactionSignature(result.TransactionID, signature)
	if transaction.EnvelopeXDR != "" {
		wallet, _ := s.walletByID(transaction.WalletID)
		hash, err := s.chain.Submit(transaction.EnvelopeXDR, wallet.Pubkey, result.Signature)
		if err != nil {
			log.Printf("broadcast failed tx=%s: %v", result.TransactionID, err)
			s.store.FailTransaction(result.TransactionID, "broadcast: "+err.Error())
			s.publishTransactionByID(userID, result.TransactionID)
			return
		}
		s.store.BroadcastTransaction(result.TransactionID, hash)
		s.publishTransactionByID(userID, result.TransactionID)
		s.store.SetTransactionStatus(result.TransactionID, "confirmed")
		s.publishTransactionByID(userID, result.TransactionID)
		go s.refreshAfterTransaction(transaction.WalletID, transaction.Counterparty)
		return
	}

	s.store.SetTransactionStatus(result.TransactionID, "confirmed")
	s.publishTransactionByID(userID, result.TransactionID)
}

func (s *Server) publishTransactionByID(userID, transactionID string) {
	if transaction, ok := s.store.TransactionByID(transactionID); ok {
		s.hub.PublishTransaction(userID, transaction)
	}
}
