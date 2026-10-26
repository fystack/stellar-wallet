package sqlite

import (
	"testing"

	"stellar-wallet-backend/internal/domain"
)

func TestTransactionRepositoryOwnershipAndSigningLifecycle(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewTransactionRepository(db)
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
	if err := repo.SetTransactionSignature(tx.ID, "signature"); err != nil {
		t.Fatal(err)
	}
	if err := repo.BroadcastTransaction(tx.ID, "hash"); err != nil {
		t.Fatal(err)
	}
	stored, ok := repo.TransactionByID(tx.ID)
	if !ok || stored.Signature != "signature" || stored.TxHash != "hash" || stored.Status != "broadcast" {
		t.Fatalf("broadcast = %#v", stored)
	}
	if err := repo.SetTransactionStatus(tx.ID, "confirmed"); err != nil {
		t.Fatal(err)
	}
	if list := repo.PendingTransactions(); len(list) != 0 {
		t.Fatalf("pending = %#v", list)
	}
	if err := repo.FailTransaction(tx.ID, "failed reason"); err != nil {
		t.Fatal(err)
	}
	stored, _ = repo.TransactionByID(tx.ID)
	if stored.Status != "failed" || stored.Error != "failed reason" {
		t.Fatalf("failed = %#v", stored)
	}
}

func TestTransactionRepositoryPreservesIncomingAndTrustlineQueries(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewTransactionRepository(db)
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
	if err := repo.FailTransaction("trustline", "timeout"); err != nil {
		t.Fatal(err)
	}
	if repo.TrustlineTransactionExists("wallet", "USDC") {
		t.Fatal("failed trustline blocks retry")
	}
}
