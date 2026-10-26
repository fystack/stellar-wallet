package apperror

import (
	"errors"
	"testing"
)

func TestCatalogCreatesIndependentErrorsAndPreservesCause(t *testing.T) {
	first := WalletNotFound.New()
	second := WalletNotFound.New()
	if first == second {
		t.Fatal("catalog returned a shared mutable error")
	}
	if first.Kind != NotFound || first.Code != "wallet_not_found" || first.Message != "Wallet not found." {
		t.Fatalf("unexpected catalog error: %#v", first)
	}

	cause := errors.New("node offline")
	wrapped := MPCUnavailable.Wrap(cause)
	if !errors.Is(wrapped, cause) {
		t.Fatal("wrapped error lost its technical cause")
	}
}
