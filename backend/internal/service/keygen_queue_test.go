package service

import (
	"sync"
	"testing"
	"time"

	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
)

type keygenStub struct {
	mu    sync.Mutex
	calls []string
}

func (k *keygenStub) CreateWallet(walletID string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.calls = append(k.calls, walletID)
	return nil
}

func (k *keygenStub) dispatched() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.calls)
}

func waitFor(t *testing.T, want int, count func() int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for count() != want {
		if time.Now().After(deadline) {
			t.Fatalf("dispatched = %d, want %d", count(), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestKeygenRequestsAreSentOneAtATime(t *testing.T) {
	f := newWorkspaceFixture(t)
	stub := &keygenStub{}
	wallets := &WalletService{store: f.store, hub: realtime.NewHub(), mpc: stub, keygens: dispatchQueue{limit: maxInflightKeygens}}
	for _, id := range []string{"w1", "w2", "w3"} {
		if err := f.store.CreateWallet(domain.Wallet{ID: id, UserID: "owner", WorkspaceID: "ws", Chain: "stellar", Symbol: "XLM", Status: domain.WalletGenerating}); err != nil {
			t.Fatal(err)
		}
		wallets.enqueueKeygen(id)
	}
	waitFor(t, 1, stub.dispatched)
	time.Sleep(20 * time.Millisecond)
	if stub.dispatched() != 1 {
		t.Fatalf("dispatched %v before the first keygen finished", stub.calls)
	}

	wallets.OnKeygen(mpc.KeygenResult{WalletID: "w1", Successful: false, Error: "boom"})
	waitFor(t, 2, stub.dispatched)
	if wallet, _ := f.store.WalletByID("w1"); wallet.Status != domain.WalletFailed {
		t.Fatalf("w1 status = %s", wallet.Status)
	}

	wallets.OnKeygen(mpc.KeygenResult{WalletID: "w2", Successful: true, PublicKey: make([]byte, 32)})
	waitFor(t, 3, stub.dispatched)
	if wallet, _ := f.store.WalletByID("w2"); wallet.Status != domain.WalletReady {
		t.Fatalf("w2 status = %s", wallet.Status)
	}

	wallets.OnKeygen(mpc.KeygenResult{WalletID: "unknown", Successful: true, PublicKey: make([]byte, 32)})
	if stub.calls[2] != "w3" {
		t.Fatalf("order = %v", stub.calls)
	}
}
