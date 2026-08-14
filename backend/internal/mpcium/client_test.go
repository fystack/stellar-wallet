package mpcium

import (
	"bytes"
	"testing"

	"github.com/fystack/mpcium/pkg/types"
)

func TestNewSignMessagePreservesRequestFields(t *testing.T) {
	payload := []byte("transaction")
	message := newSignMessage("wallet-1", "tx-1", "stellar-testnet", payload)

	if message.KeyType != types.KeyTypeEd25519 {
		t.Fatalf("KeyType = %v", message.KeyType)
	}
	if message.WalletID != "wallet-1" || message.TxID != "tx-1" {
		t.Fatalf("ids = %q, %q", message.WalletID, message.TxID)
	}
	if message.NetworkInternalCode != "stellar-testnet" {
		t.Fatalf("network = %q", message.NetworkInternalCode)
	}
	if !bytes.Equal(message.Tx, payload) {
		t.Fatalf("payload = %q", message.Tx)
	}
}
