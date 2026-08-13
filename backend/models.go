package main

// Chain metadata. Only ed25519 chains — that's the EdDSA key mpcium returns.
var chainMeta = map[string]struct {
	Symbol  string
	Network string // NetworkInternalCode passed to mpcium when signing
}{
	"stellar": {"XLM", "stellar-testnet"},
	"solana":  {"SOL", "solana-devnet"},
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
	Status    string `json:"status"` // generating | ready | failed
	CreatedAt string `json:"created_at"`
}

type Transaction struct {
	ID           string `json:"id"`
	WalletID     string `json:"walletId"`
	UserID       string `json:"-"`
	Type         string `json:"type"` // "in" | "out"
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
