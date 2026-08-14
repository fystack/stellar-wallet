package domain

// ChainMetadata describes an ed25519 chain supported by the MPC cluster.
type ChainMetadata struct {
	Symbol  string
	Network string
}

var Chains = map[string]ChainMetadata{
	"stellar": {Symbol: "XLM", Network: "stellar-testnet"},
	"solana":  {Symbol: "SOL", Network: "solana-devnet"},
}

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

type Wallet struct {
	ID        string `json:"id"`
	UserID    string `json:"-"`
	Name      string `json:"name"`
	Chain     string `json:"chain"`
	Symbol    string `json:"symbol"`
	Address   string `json:"address"`
	Pubkey    string `json:"pubkey"`
	Balance   string `json:"balance"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type Transaction struct {
	ID           string `json:"id"`
	WalletID     string `json:"walletId"`
	UserID       string `json:"-"`
	Type         string `json:"type"`
	Counterparty string `json:"counterparty"`
	Amount       string `json:"amount"`
	Symbol       string `json:"symbol"`
	Memo         string `json:"memo,omitempty"`
	Status       string `json:"status"`
	Signature    string `json:"signature,omitempty"`
	EnvelopeXDR  string `json:"-"`
	TxHash       string `json:"txHash,omitempty"`
	Error        string `json:"error,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type AssetBalance struct {
	Symbol  string `json:"symbol"`
	Balance string `json:"balance"`
	Issuer  string `json:"issuer,omitempty"`
}

type CustomAsset struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
}

type ClusterNode struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	Online bool   `json:"online"`
}
