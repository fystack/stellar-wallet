package sqlite

import (
	"testing"

	"stellar-wallet-backend/internal/domain"
)

func newTestTransactionRepository(t *testing.T) *TransactionRepository {
	t.Helper()
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewTransactionRepository(db)
}

func TestTransactionRepositoryOwnershipAndSigningLifecycle(t *testing.T) {
	repo := newTestTransactionRepository(t)
	transaction := domain.Transaction{ID: "payment-1", WalletID: "wallet-1", UserID: "owner", Type: domain.TransactionOut, Amount: "1", Symbol: "XLM", Status: domain.TransactionSigning, EnvelopeXDR: "envelope"}
	if err := repo.CreateTransaction(transaction); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.TransactionForUser(transaction.ID, "other-user"); ok {
		t.Fatal("other user can read transaction")
	}
	if list, err := repo.TransactionsByWallet("other-user", transaction.WalletID); err != nil || len(list) != 0 {
		t.Fatalf("other user's transactions: %#v, %v", list, err)
	}
	if list, err := repo.PendingTransactions(); err != nil || len(list) != 1 || list[0].EnvelopeXDR != "envelope" {
		t.Fatalf("pending = %#v, %v", list, err)
	}

	claimed, err := repo.ClaimSignature(transaction.ID, "signature")
	if err != nil || !claimed {
		t.Fatalf("claim = %v, %v", claimed, err)
	}
	if claimed, err := repo.ClaimSignature(transaction.ID, "duplicate"); err != nil || claimed {
		t.Fatalf("duplicate claim = %v, %v", claimed, err)
	}
	if failed, err := repo.FailSigning(transaction.ID, "late timeout"); err != nil || failed {
		t.Fatalf("timeout after claim = %v, %v", failed, err)
	}
	if err := repo.ConfirmTransaction(transaction.ID, "hash"); err != nil {
		t.Fatal(err)
	}
	stored, ok := repo.TransactionByID(transaction.ID)
	if !ok || stored.Signature != "signature" || stored.TxHash != "hash" || stored.Status != domain.TransactionConfirmed {
		t.Fatalf("confirmed = %#v", stored)
	}
	if list, err := repo.PendingTransactions(); err != nil || len(list) != 0 {
		t.Fatalf("pending = %#v, %v", list, err)
	}
}

func TestTransactionRepositoryRejectsSignatureAfterTimeout(t *testing.T) {
	repo := newTestTransactionRepository(t)
	if err := repo.CreateTransaction(domain.Transaction{ID: "payment-1", Status: domain.TransactionSigning}); err != nil {
		t.Fatal(err)
	}
	if failed, err := repo.FailSigning("payment-1", "timed out"); err != nil || !failed {
		t.Fatalf("fail = %v, %v", failed, err)
	}
	if claimed, err := repo.ClaimSignature("payment-1", "late"); err != nil || claimed {
		t.Fatalf("late claim = %v, %v", claimed, err)
	}
	stored, _ := repo.TransactionByID("payment-1")
	if stored.Status != domain.TransactionFailed || stored.Signature != "" || stored.Error != "timed out" {
		t.Fatalf("stored = %#v", stored)
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
	assertExists := func(want bool, exists bool, err error) {
		t.Helper()
		if err != nil || exists != want {
			t.Fatalf("exists = %v, %v; want %v", exists, err, want)
		}
	}
	exists, err := repo.IncomingTransactionExists("incoming-hash")
	assertExists(true, exists, err)
	exists, err = repo.IncomingTransactionExists("unknown")
	assertExists(false, exists, err)
	exists, err = repo.TrustlineTransactionExists("wallet", "USDC")
	assertExists(true, exists, err)
	exists, err = repo.TrustlineTransactionExists("other-wallet", "USDC")
	assertExists(false, exists, err)
	if err := repo.FailTransaction("trustline", "timeout"); err != nil {
		t.Fatal(err)
	}
	exists, err = repo.TrustlineTransactionExists("wallet", "USDC")
	assertExists(false, exists, err)
}
