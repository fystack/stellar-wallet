package chain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"stellar-wallet-backend/internal/domain"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
)

// Asset is a wire-friendly asset reference. Native XLM has an empty Issuer
// (Code "XLM" or ""); credit assets carry both Code and Issuer.
type SwapAsset struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
}

func (a SwapAsset) native() bool {
	return a.Issuer == "" || strings.EqualFold(a.Code, "XLM")
}

func (a SwapAsset) toTxnbuild() txnbuild.Asset {
	if a.native() {
		return txnbuild.NativeAsset{}
	}
	return txnbuild.CreditAsset{Code: a.Code, Issuer: a.Issuer}
}

// canonical renders the asset the way Horizon's paths endpoint expects it in a
// comma-separated destination_assets / source query.
func (a SwapAsset) canonical() string {
	if a.native() {
		return "native"
	}
	return a.Code + ":" + a.Issuer
}

// SwapQuote asks Horizon for the best strict-send path: given an exact amount
// of the source asset, how much of the destination asset would arrive, and
// through which intermediate hops.
type SwapQuote struct {
	SendAsset      SwapAsset   `json:"send_asset"`
	SendAmount     string      `json:"send_amount"`
	DestAsset      SwapAsset   `json:"dest_asset"`
	DestAmount     string      `json:"dest_amount"`
	Path           []SwapAsset `json:"path"`
}

func (c *Client) SwapQuote(send SwapAsset, sendAmount string, dest SwapAsset) (SwapQuote, error) {
	query := url.Values{}
	query.Set("source_amount", sendAmount)
	query.Set("destination_assets", dest.canonical())
	if send.native() {
		query.Set("source_asset_type", "native")
	} else {
		query.Set("source_asset_type", assetType(send.Code))
		query.Set("source_asset_code", send.Code)
		query.Set("source_asset_issuer", send.Issuer)
	}

	endpoint := strings.TrimRight(c.HorizonURL(), "/") + "/paths/strict-send?" + query.Encode()
	response, err := c.httpClient.Get(endpoint)
	if err != nil {
		return SwapQuote{}, err
	}
	defer response.Body.Close()

	var page struct {
		Embedded struct {
			Records []struct {
				DestinationAmount string `json:"destination_amount"`
				Path              []struct {
					AssetType   string `json:"asset_type"`
					AssetCode   string `json:"asset_code"`
					AssetIssuer string `json:"asset_issuer"`
				} `json:"path"`
			} `json:"records"`
		} `json:"_embedded"`
	}
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		return SwapQuote{}, fmt.Errorf("could not read path response")
	}
	if len(page.Embedded.Records) == 0 {
		return SwapQuote{}, fmt.Errorf("no path found for this pair — try a smaller amount or a different asset")
	}

	// Records are ordered best-first by Horizon.
	best := page.Embedded.Records[0]
	path := make([]SwapAsset, 0, len(best.Path))
	for _, hop := range best.Path {
		if hop.AssetType == "native" {
			path = append(path, SwapAsset{Code: "XLM"})
			continue
		}
		path = append(path, SwapAsset{Code: hop.AssetCode, Issuer: hop.AssetIssuer})
	}

	return SwapQuote{
		SendAsset:  send,
		SendAmount: sendAmount,
		DestAsset:  dest,
		DestAmount: best.DestinationAmount,
		Path:       path,
	}, nil
}

func assetType(code string) string {
	if len(code) <= 4 {
		return "credit_alphanum4"
	}
	return "credit_alphanum12"
}

// BuildSwap constructs a PathPaymentStrictSend that sends an exact amount of
// one asset from the wallet to itself and receives at least destMin of another.
func (c *Client) BuildSwap(
	wallet domain.Wallet,
	send SwapAsset, sendAmount string,
	dest SwapAsset, destMin string,
	path []SwapAsset,
) (string, []byte, error) {
	account, err := c.horizon().AccountDetail(horizonclient.AccountRequest{AccountID: wallet.Address})
	if err != nil {
		return "", nil, fmt.Errorf("source account not found — fund it first")
	}

	hops := make([]txnbuild.Asset, 0, len(path))
	for _, hop := range path {
		hops = append(hops, hop.toTxnbuild())
	}

	operation := &txnbuild.PathPaymentStrictSend{
		SendAsset:   send.toTxnbuild(),
		SendAmount:  sendAmount,
		Destination: wallet.Address,
		DestAsset:   dest.toTxnbuild(),
		DestMin:     destMin,
		Path:        hops,
	}

	transaction, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations:           []txnbuild.Operation{operation},
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

// applySlippage returns destAmount reduced by the given bips (basis points),
// formatted to 7 decimals — the minimum acceptable output for a swap.
func ApplySlippage(destAmount string, bips int) string {
	amount, err := strconv.ParseFloat(destAmount, 64)
	if err != nil {
		return destAmount
	}
	minimum := amount * (1 - float64(bips)/10000)
	if minimum < 0 {
		minimum = 0
	}
	return strconv.FormatFloat(minimum, 'f', 7, 64)
}
