package chain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/stellar/go/clients/stellartoml"
	"github.com/stellar/go/strkey"
)

// Resolved is the outcome of turning a user-typed recipient into a concrete
// Stellar destination. MemoType/Memo are populated only for federated
// addresses whose server tells us a memo is required (e.g. exchange deposits).
type Resolved struct {
	Address  string `json:"address"`            // G... or M... — usable as a payment destination
	MemoType string `json:"memo_type,omitempty"` // "text" | "id" | "hash"
	Memo     string `json:"memo,omitempty"`
	Federal  string `json:"federation,omitempty"` // the original name*domain, echoed back
}

// ResolveRecipient accepts a raw G-address, an M-muxed address, or a
// federation address (name*domain.com) and returns a concrete destination.
func (c *Client) ResolveRecipient(input string) (Resolved, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return Resolved{}, fmt.Errorf("recipient required")
	}

	if strings.Contains(value, "*") {
		return c.resolveFederation(value)
	}

	switch value[0] {
	case 'G':
		if strkey.IsValidEd25519PublicKey(value) {
			return Resolved{Address: value}, nil
		}
	case 'M':
		if _, err := strkey.Decode(strkey.VersionByteMuxedAccount, value); err == nil {
			return Resolved{Address: value}, nil
		}
	}
	return Resolved{}, fmt.Errorf("not a valid Stellar address or federation name")
}

type federationRecord struct {
	AccountID string `json:"account_id"`
	MemoType  string `json:"memo_type"`
	Memo      string `json:"memo"`
	Detail    string `json:"detail"`
}

func (c *Client) resolveFederation(address string) (Resolved, error) {
	parts := strings.SplitN(address, "*", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Resolved{}, fmt.Errorf("bad federation address — expected name*domain")
	}
	domain := parts[1]

	toml, err := stellartoml.GetStellarToml(domain)
	if err != nil || toml.FederationServer == "" {
		return Resolved{}, fmt.Errorf("no federation server for %s", domain)
	}

	endpoint := fmt.Sprintf("%s?q=%s&type=name", toml.FederationServer, url.QueryEscape(address))
	response, err := c.httpClient.Get(endpoint)
	if err != nil {
		return Resolved{}, fmt.Errorf("federation lookup failed: %w", err)
	}
	defer response.Body.Close()

	var record federationRecord
	if err := json.NewDecoder(response.Body).Decode(&record); err != nil {
		return Resolved{}, fmt.Errorf("federation server returned an unreadable response")
	}
	if record.AccountID == "" {
		detail := record.Detail
		if detail == "" {
			detail = "address not found"
		}
		return Resolved{}, fmt.Errorf("federation: %s", detail)
	}

	return Resolved{
		Address:  record.AccountID,
		MemoType: record.MemoType,
		Memo:     record.Memo,
		Federal:  address,
	}, nil
}
