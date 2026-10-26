package sqlite

import (
	"testing"

	"stellar-wallet-backend/internal/domain"
)

func TestPaymentRepositoryOwnershipAndSigningLifecycle(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPaymentRepository(db)
	tx := domain.Transaction{ID: "payment-1", WalletID: "wallet-1", UserID: "owner", Type: "out", Amount: "1.00", Symbol: "XLM", Status: "signing", EnvelopeXDR: "envelope"}
	if err := repo.CreateTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.TransactionForUser(tx.ID, "other-user"); ok {
		t.Fatal("other user can read transaction")
	}
	if list := repo.TransactionsByWallet("other-user", tx.WalletID); len(list) != 0 {
		t.Fatalf("other user's transactions: %#v", list)
	}
	if list := repo.PendingTransactions(); len(list) != 1 || list[0].EnvelopeXDR != "envelope" {
		t.Fatalf("pending = %#v", list)
	}
	repo.SetTransactionSignature(tx.ID, "signature")
	repo.BroadcastTransaction(tx.ID, "hash")
	stored, ok := repo.TransactionByID(tx.ID)
	if !ok || stored.Signature != "signature" || stored.TxHash != "hash" || stored.Status != "broadcast" {
		t.Fatalf("broadcast = %#v", stored)
	}
	repo.SetTransactionStatus(tx.ID, "confirmed")
	if list := repo.PendingTransactions(); len(list) != 0 {
		t.Fatalf("pending = %#v", list)
	}
	repo.FailTransaction(tx.ID, "failed reason")
	stored, _ = repo.TransactionByID(tx.ID)
	if stored.Status != "failed" || stored.Error != "failed reason" {
		t.Fatalf("failed = %#v", stored)
	}
}

func TestPaymentRepositoryPreservesIncomingAndTrustlineQueries(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPaymentRepository(db)
	for _, tx := range []domain.Transaction{
		{ID: "incoming", UserID: "owner", WalletID: "wallet", Type: "in", TxHash: "incoming-hash", Status: "confirmed"},
		{ID: "trustline", UserID: "owner", WalletID: "wallet", Type: "out", Memo: "Add trustline", Symbol: "USDC", Status: "signing"},
	} {
		if err := repo.CreateTransaction(tx); err != nil {
			t.Fatal(err)
		}
	}
	if !repo.IncomingTransactionExists("incoming-hash") || repo.IncomingTransactionExists("unknown") {
		t.Fatal("incorrect incoming deduplication")
	}
	if !repo.TrustlineTransactionExists("wallet", "USDC") || repo.TrustlineTransactionExists("other-wallet", "USDC") {
		t.Fatal("incorrect trustline scope")
	}
	repo.FailTransaction("trustline", "timeout")
	if repo.TrustlineTransactionExists("wallet", "USDC") {
		t.Fatal("failed trustline blocks retry")
	}
}
