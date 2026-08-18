package chain

import "testing"

func TestDeriveAddressUsesTheChainEncoding(t *testing.T) {
	publicKey := make([]byte, 32)

	if got := DeriveAddress("stellar", publicKey); got != "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF" {
		t.Fatalf("stellar address = %q", got)
	}
}

func TestExplorerAddressMatchesNetwork(t *testing.T) {
	if got := ExplorerAddress("stellar", "GABC"); got != "https://stellar.expert/explorer/testnet/account/GABC" {
		t.Fatalf("stellar explorer = %q", got)
	}
}
