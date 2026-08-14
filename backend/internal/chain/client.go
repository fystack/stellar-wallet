package chain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"stellar-wallet-backend/internal/domain"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

const (
	DefaultHorizonURL = "https://horizon-testnet.stellar.org"
	DefaultSolanaURL  = "https://api.devnet.solana.com"
	friendbotURL      = "https://friendbot.stellar.org"
)

type IncomingPayment struct {
	Hash   string
	From   string
	Amount string
	Symbol string
	At     string
}

type Client struct {
	httpClient *http.Client

	rpcMu      sync.RWMutex
	horizonURL string
	solanaURL  string

	priceMu     sync.Mutex
	priceCache  map[string]float64
	priceExpiry time.Time
}

func NewClient(horizonURL, solanaURL string, httpClient *http.Client) *Client {
	return &Client{
		httpClient: httpClient,
		horizonURL: horizonURL,
		solanaURL:  solanaURL,
	}
}

func (c *Client) SetRPC(horizonURL, solanaURL string) {
	c.rpcMu.Lock()
	defer c.rpcMu.Unlock()
	c.horizonURL = horizonURL
	c.solanaURL = solanaURL
}

func (c *Client) HorizonURL() string {
	c.rpcMu.RLock()
	defer c.rpcMu.RUnlock()
	return c.horizonURL
}

func (c *Client) SolanaURL() string {
	c.rpcMu.RLock()
	defer c.rpcMu.RUnlock()
	return c.solanaURL
}

func (c *Client) horizon() *horizonclient.Client {
	return &horizonclient.Client{HorizonURL: c.HorizonURL(), HTTP: c.httpClient}
}

func (c *Client) Incoming(address string) ([]IncomingPayment, error) {
	page, err := c.horizon().Payments(horizonclient.OperationRequest{
		ForAccount: address,
		Order:      horizonclient.OrderDesc,
		Limit:      20,
	})
	if err != nil {
		return nil, err
	}

	var payments []IncomingPayment
	for _, record := range page.Embedded.Records {
		switch operation := record.(type) {
		case operations.Payment:
			if operation.To != address || operation.From == address {
				continue
			}
			symbol := "XLM"
			if operation.Asset.Type != "native" {
				symbol = operation.Asset.Code
			}
			payments = append(payments, IncomingPayment{
				Hash: operation.TransactionHash, From: operation.From,
				Amount: operation.Amount, Symbol: symbol,
				At: operation.LedgerCloseTime.UTC().Format(time.RFC3339),
			})
		case operations.CreateAccount:
			if operation.Account == address {
				payments = append(payments, IncomingPayment{
					Hash: operation.TransactionHash, From: operation.Funder,
					Amount: operation.StartingBalance, Symbol: "XLM",
					At: operation.LedgerCloseTime.UTC().Format(time.RFC3339),
				})
			}
		}
	}
	return payments, nil
}

func (c *Client) Balances(chainName, address string) []domain.AssetBalance {
	switch chainName {
	case "stellar":
		account, err := c.horizon().AccountDetail(horizonclient.AccountRequest{AccountID: address})
		if err != nil {
			return []domain.AssetBalance{{Symbol: "XLM", Balance: "0"}}
		}
		balances := []domain.AssetBalance{}
		for _, balance := range account.Balances {
			if balance.Asset.Type == "native" {
				balances = append(balances, domain.AssetBalance{Symbol: "XLM", Balance: balance.Balance})
			}
		}
		for _, balance := range account.Balances {
			if balance.Asset.Type != "native" && balance.Asset.Code != "" {
				balances = append(balances, domain.AssetBalance{
					Symbol: balance.Asset.Code, Balance: balance.Balance, Issuer: balance.Asset.Issuer,
				})
			}
		}
		return balances
	case "solana":
		balance, _ := c.Balance("solana", address)
		return []domain.AssetBalance{{Symbol: "SOL", Balance: balance}}
	default:
		return nil
	}
}

func (c *Client) PingHorizon() bool {
	response, err := c.httpClient.Get(c.HorizonURL())
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode < http.StatusBadRequest
}

func (c *Client) PingSolana() bool {
	var response struct {
		Result string `json:"result"`
	}
	return c.solanaRPC("getHealth", []any{}, &response) == nil && response.Result == "ok"
}

func (c *Client) Balance(chainName, address string) (string, error) {
	switch chainName {
	case "stellar":
		account, err := c.horizon().AccountDetail(horizonclient.AccountRequest{AccountID: address})
		if err != nil {
			return "0.00", nil
		}
		for _, balance := range account.Balances {
			if balance.Asset.Type == "native" {
				return balance.Balance, nil
			}
		}
		return "0.00", nil
	case "solana":
		var response struct {
			Result struct {
				Value uint64 `json:"value"`
			} `json:"result"`
		}
		if err := c.solanaRPC("getBalance", []any{address}, &response); err != nil {
			return "0.00", nil
		}
		return fmt.Sprintf("%.4f", float64(response.Result.Value)/1e9), nil
	default:
		return "0.00", nil
	}
}

func (c *Client) Fund(chainName, address string) error {
	switch chainName {
	case "stellar":
		response, err := c.httpClient.Get(friendbotURL + "?addr=" + address)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode >= http.StatusMultipleChoices {
			body, _ := io.ReadAll(response.Body)
			return fmt.Errorf("friendbot: %s", string(body))
		}
		return nil
	case "solana":
		var response struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := c.solanaRPC("requestAirdrop", []any{address, 1000000000}, &response); err != nil {
			return err
		}
		if response.Error != nil {
			return fmt.Errorf("airdrop: %s", response.Error.Message)
		}
		return nil
	default:
		return fmt.Errorf("unsupported chain")
	}
}

func (c *Client) solanaRPC(method string, params []any, output any) error {
	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
	})
	response, err := c.httpClient.Post(c.SolanaURL(), "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(output)
}

func (c *Client) BuildPayment(
	wallet domain.Wallet,
	to, amount, assetCode, assetIssuer, memo string,
) (string, []byte, error) {
	account, err := c.horizon().AccountDetail(horizonclient.AccountRequest{AccountID: wallet.Address})
	if err != nil {
		return "", nil, fmt.Errorf("source account not found — fund it first")
	}

	native := assetCode == "" || assetCode == "XLM"
	var asset txnbuild.Asset = txnbuild.NativeAsset{}
	if !native {
		asset = txnbuild.CreditAsset{Code: assetCode, Issuer: assetIssuer}
	}

	var operation txnbuild.Operation = &txnbuild.Payment{
		Destination: to, Amount: amount, Asset: asset,
	}
	if _, destinationErr := c.horizon().AccountDetail(horizonclient.AccountRequest{AccountID: to}); destinationErr != nil {
		if !native {
			return "", nil, fmt.Errorf("recipient account does not exist yet — fund it with XLM first")
		}
		if parsedAmount, _ := strconv.ParseFloat(amount, 64); parsedAmount < 1 {
			return "", nil, fmt.Errorf("recipient account is new — send at least 1 XLM to create it")
		}
		operation = &txnbuild.CreateAccount{Destination: to, Amount: amount}
	}

	params := txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations:           []txnbuild.Operation{operation},
	}
	if memo != "" {
		memoText := memo
		if len(memoText) > 28 {
			memoText = memoText[:28]
		}
		params.Memo = txnbuild.MemoText(memoText)
	}
	transaction, err := txnbuild.NewTransaction(params)
	if err != nil {
		return "", nil, err
	}
	hash, err := transaction.Hash(network.TestNetworkPassphrase)
	if err != nil {
		return "", nil, err
	}
	envelope, err := transaction.Base64()
	if err != nil {
		return "", nil, err
	}
	return envelope, hash[:], nil
}

func (c *Client) BuildTrustline(wallet domain.Wallet, code, issuer string) (string, []byte, error) {
	account, err := c.horizon().AccountDetail(horizonclient.AccountRequest{AccountID: wallet.Address})
	if err != nil {
		return "", nil, fmt.Errorf("source account not found — fund it first")
	}
	line := txnbuild.ChangeTrust{
		Line: txnbuild.CreditAsset{Code: code, Issuer: issuer}.MustToChangeTrustAsset(),
	}
	transaction, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations:           []txnbuild.Operation{&line},
	})
	if err != nil {
		return "", nil, err
	}
	hash, err := transaction.Hash(network.TestNetworkPassphrase)
	if err != nil {
		return "", nil, err
	}
	envelope, err := transaction.Base64()
	if err != nil {
		return "", nil, err
	}
	return envelope, hash[:], nil
}

func (c *Client) Submit(envelopeBase64, publicKeyHex string, signature []byte) (string, error) {
	generic, err := txnbuild.TransactionFromXDR(envelopeBase64)
	if err != nil {
		return "", err
	}
	transaction, ok := generic.Transaction()
	if !ok {
		return "", fmt.Errorf("not a simple transaction")
	}
	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil || len(publicKey) != 32 {
		return "", fmt.Errorf("bad pubkey")
	}
	var hint [4]byte
	copy(hint[:], publicKey[28:32])
	signed, err := transaction.AddSignatureDecorated(xdr.DecoratedSignature{
		Hint: xdr.SignatureHint(hint), Signature: xdr.Signature(signature),
	})
	if err != nil {
		return "", err
	}
	response, err := c.horizon().SubmitTransaction(signed)
	if err != nil {
		return "", horizonError(err)
	}
	return response.Hash, nil
}

func (c *Client) TransactionOnChain(hash string) (map[string]any, error) {
	transaction, err := c.horizon().TransactionDetail(hash)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"fee":        fmt.Sprintf("%.5f", float64(transaction.FeeCharged)/1e7),
		"ledger":     transaction.Ledger,
		"operations": transaction.OperationCount,
		"source":     transaction.Account,
		"memo":       transaction.Memo,
		"successful": transaction.Successful,
	}, nil
}

func horizonError(err error) error {
	if problem := horizonclient.GetError(err); problem != nil {
		if resultCodes, resultErr := problem.ResultCodes(); resultErr == nil {
			codes := resultCodes.TransactionCode
			if len(resultCodes.OperationCodes) > 0 {
				codes += " (" + strings.Join(resultCodes.OperationCodes, ", ") + ")"
			}
			return fmt.Errorf("%s", codes)
		}
	}
	return err
}
