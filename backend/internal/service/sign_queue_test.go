package service

import (
	"testing"

	"stellar-wallet-backend/internal/mpc"
)

func (m *paymentSignerStub) dispatched() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestSignRequestsAreLimitedInFlight(t *testing.T) {
	service, _, signer := newTestTransactionService(t)
	ids := []string{}
	for range maxInflightSigns + 2 {
		result, err := service.Create("owner", PaymentInput{WalletID: "wallet-1", To: "resolved-address", Amount: "1"})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, result.Transaction.ID)
	}
	if signer.dispatched() != maxInflightSigns {
		t.Fatalf("dispatched = %d, want %d", signer.dispatched(), maxInflightSigns)
	}

	service.OnSign(mpc.SignResult{TransactionID: ids[0], Successful: false, Error: "boom"})
	if signer.dispatched() != maxInflightSigns+1 {
		t.Fatalf("after failure dispatched = %d", signer.dispatched())
	}
	service.OnSign(mpc.SignResult{TransactionID: "unknown", Successful: false})
	if signer.dispatched() != maxInflightSigns+1 {
		t.Fatal("unknown result released a slot")
	}
	service.OnSign(mpc.SignResult{TransactionID: ids[1], Successful: true, Signature: []byte("sig")})
	if signer.dispatched() != maxInflightSigns+2 {
		t.Fatalf("after success dispatched = %d", signer.dispatched())
	}
	if signer.transactionID != ids[len(ids)-1] {
		t.Fatalf("last dispatched = %s, want %s", signer.transactionID, ids[len(ids)-1])
	}
}
