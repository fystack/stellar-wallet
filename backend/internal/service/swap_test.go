package service

import (
	"errors"
	"testing"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
)

func TestParseSwap(t *testing.T) {
	usdc := chain.SwapAsset{Code: "USDC", Issuer: "GISSUER"}
	tests := []struct {
		name       string
		input      SwapInput
		amount     string
		send, dest chain.SwapAsset
		code       string
	}{
		{name: "native default", input: SwapInput{To: "USDC:GISSUER", Amount: " 5 "}, amount: "5", send: chain.SwapAsset{Code: "XLM"}, dest: usdc},
		{name: "issued to native", input: SwapInput{From: "USDC:GISSUER", To: "xlm", Amount: "1.5"}, amount: "1.5", send: usdc, dest: chain.SwapAsset{Code: "XLM"}},
		{name: "zero amount", input: SwapInput{To: "USDC:GISSUER", Amount: "0"}, code: "invalid_amount"},
		{name: "same asset", input: SwapInput{From: "XLM", To: "", Amount: "1"}, code: "invalid_asset_pair"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, send, dest, err := parseSwap(tt.input)
			if tt.code != "" {
				var failure *apperror.Error
				if !errors.As(err, &failure) || failure.Code != tt.code {
					t.Fatalf("err = %v, want %s", err, tt.code)
				}
				return
			}
			if err != nil || amount != tt.amount || send != tt.send || dest != tt.dest {
				t.Fatalf("got %q %+v %+v %v", amount, send, dest, err)
			}
		})
	}
}

type swapChainStub struct {
	quote    chain.SwapQuote
	quoteErr error
	destMin  string
}

func (c *swapChainStub) SwapQuote(chain.SwapAsset, string, chain.SwapAsset) (chain.SwapQuote, error) {
	return c.quote, c.quoteErr
}

func (c *swapChainStub) BuildSwap(_ domain.Wallet, _ chain.SwapAsset, _ string, _ chain.SwapAsset, destMin string, _ []chain.SwapAsset) (string, []byte, error) {
	c.destMin = destMin
	return "swap-envelope", []byte("swap-hash"), nil
}

func TestSwapCreateAppliesDefaultSlippageAndDispatches(t *testing.T) {
	transactions, _, signer := newTestTransactionService(t)
	stub := &swapChainStub{quote: chain.SwapQuote{DestAmount: "100.0000000"}}
	swaps := NewSwapService(transactions.wallets, stub, transactions)

	result, err := swaps.Create("owner", "wallet-1", SwapInput{To: "USDC:GISSUER", Amount: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if result.DestMin != "99.0000000" || stub.destMin != result.DestMin {
		t.Fatalf("dest min = %q, built with %q", result.DestMin, stub.destMin)
	}
	stored, ok := transactions.transactions.TransactionByID(result.Transaction.ID)
	if !ok || stored.Type != domain.TransactionSwap || stored.Status != domain.TransactionSigning || stored.Amount != "10" || stored.RecvSymbol != "USDC" || stored.EnvelopeXDR != "swap-envelope" {
		t.Fatalf("stored = %+v", stored)
	}
	if signer.transactionID != result.Transaction.ID || string(signer.payload) != "swap-hash" {
		t.Fatalf("signer = %+v", signer)
	}
}

func TestSwapCreateRejectsBeforeSigning(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*TransactionService, *swapChainStub)
		code  string
	}{
		{name: "quote failed", setup: func(_ *TransactionService, c *swapChainStub) { c.quoteErr = errors.New("no path") }, code: "swap_quote_failed"},
		{name: "wallet not ready", setup: func(s *TransactionService, _ *swapChainStub) {
			if err := s.wallets.SetWalletStatus("wallet-1", domain.WalletGenerating); err != nil {
				t.Fatal(err)
			}
		}, code: "wallet_not_ready"},
		{name: "below quorum", setup: func(s *TransactionService, _ *swapChainStub) { s.cluster = paymentClusterStub(1) }, code: "mpc_unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions, _, signer := newTestTransactionService(t)
			stub := &swapChainStub{quote: chain.SwapQuote{DestAmount: "1"}}
			tt.setup(transactions, stub)
			_, err := NewSwapService(transactions.wallets, stub, transactions).Create("owner", "wallet-1", SwapInput{To: "USDC:GISSUER", Amount: "1"})
			var failure *apperror.Error
			if !errors.As(err, &failure) || failure.Code != tt.code {
				t.Fatalf("err = %v, want %s", err, tt.code)
			}
			if signer.transactionID != "" {
				t.Fatal("swap dispatched for signing")
			}
		})
	}
}
