package service

import (
	"errors"
	"testing"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
)

type trustlineChainStub struct{ err error }

func (c trustlineChainStub) BuildTrustline(domain.Wallet, string, string) (string, []byte, error) {
	return "trustline-envelope", []byte("trustline-hash"), c.err
}

func TestTrustlineAddDispatchesOnceAndRejectsDuplicates(t *testing.T) {
	transactions, _, signer := newTestTransactionService(t)
	trustlines := NewTrustlineService(transactions.wallets, transactions.transactions, trustlineChainStub{}, transactions)

	transaction, err := trustlines.Add("owner", "wallet-1", "USDC", "GISSUER")
	if err != nil {
		t.Fatal(err)
	}
	if transaction.Memo != domain.TrustlineMemo || transaction.Counterparty != "GISSUER" || signer.transactionID != transaction.ID {
		t.Fatalf("transaction = %+v, signer = %+v", transaction, signer)
	}

	_, err = trustlines.Add("owner", "wallet-1", "USDC", "GISSUER")
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Code != "trustline_exists" {
		t.Fatalf("duplicate err = %v", err)
	}
}

func TestTrustlineAddValidatesInput(t *testing.T) {
	tests := []struct {
		name, walletID, code, issuer string
		chainErr                     error
		want                         string
	}{
		{name: "missing issuer", walletID: "wallet-1", code: "USDC", want: "invalid_asset"},
		{name: "other wallet", walletID: "missing", code: "USDC", issuer: "GISSUER", want: "wallet_not_found"},
		{name: "build failed", walletID: "wallet-1", code: "USDC", issuer: "GISSUER", chainErr: errors.New("unfunded"), want: "trustline_build_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions, _, signer := newTestTransactionService(t)
			trustlines := NewTrustlineService(transactions.wallets, transactions.transactions, trustlineChainStub{err: tt.chainErr}, transactions)
			_, err := trustlines.Add("owner", tt.walletID, tt.code, tt.issuer)
			var failure *apperror.Error
			if !errors.As(err, &failure) || failure.Code != tt.want {
				t.Fatalf("err = %v, want %s", err, tt.want)
			}
			if signer.transactionID != "" {
				t.Fatal("trustline dispatched for signing")
			}
		})
	}
}
