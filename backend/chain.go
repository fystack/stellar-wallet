package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

type incomingPayment struct {
	Hash   string
	From   string
	Amount string
	Symbol string
}

// stellarIncoming returns recent payments received by the address.
func stellarIncoming(address string) ([]incomingPayment, error) {
	page, err := horizon().Payments(horizonclient.OperationRequest{
		ForAccount: address,
		Order:      horizonclient.OrderDesc,
		Limit:      20,
	})
	if err != nil {
		return nil, err
	}
	var out []incomingPayment
	for _, rec := range page.Embedded.Records {
		switch op := rec.(type) {
		case operations.Payment:
			if op.To == address && op.From != address {
				sym := "XLM"
				if op.Asset.Type != "native" {
					sym = op.Asset.Code
				}
				out = append(out, incomingPayment{op.TransactionHash, op.From, op.Amount, sym})
			}
		case operations.CreateAccount:
			if op.Account == address {
				out = append(out, incomingPayment{op.TransactionHash, op.Funder, op.StartingBalance, "XLM"})
			}
		}
	}
	return out, nil
}

const (
	horizonTestnet = "https://horizon-testnet.stellar.org"
	friendbotURL   = "https://friendbot.stellar.org"
	solanaDevnet   = "https://api.devnet.solana.com"
)

var httpc = &http.Client{Timeout: 20 * time.Second}

// Active RPC endpoints — overridable from Settings (see config.go).
var (
	activeHorizonURL = horizonTestnet
	activeSolanaURL  = solanaDevnet
)

func horizon() *horizonclient.Client {
	return &horizonclient.Client{HorizonURL: activeHorizonURL, HTTP: httpc}
}

// explorerAddress returns a block-explorer URL for an address.
func explorerAddress(chain, address string) string {
	switch chain {
	case "stellar":
		return "https://stellar.expert/explorer/testnet/account/" + address
	case "solana":
		return "https://explorer.solana.com/address/" + address + "?cluster=devnet"
	}
	return ""
}

type AssetBalance struct {
	Symbol  string `json:"symbol"`
	Balance string `json:"balance"`
	Issuer  string `json:"issuer,omitempty"` // empty for the native asset (XLM)
}

// getBalances returns per-asset balances (native first, then any held tokens).
func getBalances(chain, address string) []AssetBalance {
	switch chain {
	case "stellar":
		acc, err := horizon().AccountDetail(horizonclient.AccountRequest{AccountID: address})
		if err != nil {
			return []AssetBalance{{Symbol: "XLM", Balance: "0"}}
		}
		out := []AssetBalance{}
		for _, b := range acc.Balances {
			if b.Asset.Type == "native" {
				out = append(out, AssetBalance{Symbol: "XLM", Balance: b.Balance})
			}
		}
		for _, b := range acc.Balances {
			if b.Asset.Type != "native" && b.Asset.Code != "" {
				out = append(out, AssetBalance{
					Symbol:  b.Asset.Code,
					Balance: b.Balance,
					Issuer:  b.Asset.Issuer,
				})
			}
		}
		return out
	case "solana":
		bal, _ := getBalance("solana", address)
		return []AssetBalance{{Symbol: "SOL", Balance: bal}}
	}
	return nil
}

// pingHorizon returns true if the Horizon RPC responds.
func pingHorizon() bool {
	resp, err := httpc.Get(activeHorizonURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 400
}

// pingSolana returns true if the Solana RPC reports healthy.
func pingSolana() bool {
	var r struct {
		Result string `json:"result"`
	}
	if err := solanaRPC("getHealth", []any{}, &r); err != nil {
		return false
	}
	return r.Result == "ok"
}

// getBalance reads the native-asset balance from the chain.
func getBalance(chain, address string) (string, error) {
	switch chain {
	case "stellar":
		acc, err := horizon().AccountDetail(horizonclient.AccountRequest{AccountID: address})
		if err != nil {
			return "0.00", nil // unfunded accounts 404 — treat as zero
		}
		for _, b := range acc.Balances {
			if b.Asset.Type == "native" {
				return b.Balance, nil
			}
		}
		return "0.00", nil
	case "solana":
		var res struct {
			Result struct {
				Value uint64 `json:"value"`
			} `json:"result"`
		}
		if err := solanaRPC("getBalance", []any{address}, &res); err != nil {
			return "0.00", nil
		}
		return fmt.Sprintf("%.4f", float64(res.Result.Value)/1e9), nil
	}
	return "0.00", nil
}

// fund tops up a testnet account (Friendbot for Stellar, airdrop for Solana).
func fund(chain, address string) error {
	switch chain {
	case "stellar":
		resp, err := httpc.Get(friendbotURL + "?addr=" + address)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("friendbot: %s", string(body))
		}
		return nil
	case "solana":
		var res struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		// 1 SOL = 1e9 lamports
		if err := solanaRPC("requestAirdrop", []any{address, 1000000000}, &res); err != nil {
			return err
		}
		if res.Error != nil {
			return fmt.Errorf("airdrop: %s", res.Error.Message)
		}
		return nil
	}
	return fmt.Errorf("unsupported chain")
}

func solanaRPC(method string, params []any, out any) error {
	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
	})
	resp, err := httpc.Post(activeSolanaURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// stellarBuildPayment builds an unsigned payment tx and returns its base64
// envelope plus the 32-byte hash the MPC cluster needs to sign.
func stellarBuildPayment(w Wallet, to, amount, assetCode, assetIssuer, memo string) (string, []byte, error) {
	acc, err := horizon().AccountDetail(horizonclient.AccountRequest{AccountID: w.Address})
	if err != nil {
		return "", nil, fmt.Errorf("source account not found — fund it first")
	}

	native := assetCode == "" || assetCode == "XLM"
	var asset txnbuild.Asset = txnbuild.NativeAsset{}
	if !native {
		asset = txnbuild.CreditAsset{Code: assetCode, Issuer: assetIssuer}
	}

	var op txnbuild.Operation = &txnbuild.Payment{
		Destination: to, Amount: amount, Asset: asset,
	}
	// Only native XLM can create a brand-new account (base reserve). Non-native
	// payments require the recipient to already hold a trustline for the asset.
	if _, derr := horizon().AccountDetail(horizonclient.AccountRequest{AccountID: to}); derr != nil {
		if !native {
			return "", nil, fmt.Errorf("recipient account does not exist yet — fund it with XLM first")
		}
		if amt, _ := strconv.ParseFloat(amount, 64); amt < 1 {
			return "", nil, fmt.Errorf("recipient account is new — send at least 1 XLM to create it")
		}
		op = &txnbuild.CreateAccount{Destination: to, Amount: amount}
	}

	params := txnbuild.TransactionParams{
		SourceAccount:        &acc,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations:           []txnbuild.Operation{op},
	}
	if memo != "" {
		m := memo
		if len(m) > 28 { // Stellar MemoText limit
			m = m[:28]
		}
		params.Memo = txnbuild.MemoText(m)
	}
	tx, err := txnbuild.NewTransaction(params)
	if err != nil {
		return "", nil, err
	}
	hash, err := tx.Hash(network.TestNetworkPassphrase)
	if err != nil {
		return "", nil, err
	}
	b64, err := tx.Base64()
	if err != nil {
		return "", nil, err
	}
	return b64, hash[:], nil
}

// stellarBuildTrustline builds an unsigned changeTrust op (lets the wallet hold
// a custom asset) and returns the envelope + hash to sign.
func stellarBuildTrustline(w Wallet, code, issuer string) (string, []byte, error) {
	acc, err := horizon().AccountDetail(horizonclient.AccountRequest{AccountID: w.Address})
	if err != nil {
		return "", nil, fmt.Errorf("source account not found — fund it first")
	}
	line := txnbuild.ChangeTrust{
		Line: txnbuild.CreditAsset{Code: code, Issuer: issuer}.MustToChangeTrustAsset(),
	}
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &acc,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations:           []txnbuild.Operation{&line},
	})
	if err != nil {
		return "", nil, err
	}
	hash, err := tx.Hash(network.TestNetworkPassphrase)
	if err != nil {
		return "", nil, err
	}
	b64, err := tx.Base64()
	if err != nil {
		return "", nil, err
	}
	return b64, hash[:], nil
}

// stellarSubmit attaches the MPC signature to the stored envelope and broadcasts.
func stellarSubmit(envelopeB64, pubkeyHex string, sig []byte) (string, error) {
	generic, err := txnbuild.TransactionFromXDR(envelopeB64)
	if err != nil {
		return "", err
	}
	tx, ok := generic.Transaction()
	if !ok {
		return "", fmt.Errorf("not a simple transaction")
	}
	pub, err := hex.DecodeString(pubkeyHex)
	if err != nil || len(pub) != 32 {
		return "", fmt.Errorf("bad pubkey")
	}
	var hint [4]byte
	copy(hint[:], pub[28:32]) // signature hint = last 4 bytes of the public key

	decorated := xdr.DecoratedSignature{
		Hint:      xdr.SignatureHint(hint),
		Signature: xdr.Signature(sig),
	}
	signed, err := tx.AddSignatureDecorated(decorated)
	if err != nil {
		return "", err
	}
	resp, err := horizon().SubmitTransaction(signed)
	if err != nil {
		return "", horizonError(err)
	}
	return resp.Hash, nil
}

// horizonError turns a Horizon submission error into a readable result-code message.
func horizonError(err error) error {
	if p := horizonclient.GetError(err); p != nil {
		if rc, e := p.ResultCodes(); e == nil {
			codes := rc.TransactionCode
			if len(rc.OperationCodes) > 0 {
				codes += " (" + strings.Join(rc.OperationCodes, ", ") + ")"
			}
			return fmt.Errorf("%s", codes)
		}
	}
	return err
}
