package domain

// Identity is returned only by a verified provider response.
type Identity struct {
	Provider string `json:"provider"`
	Subject  string `json:"-"`
	Email    string `json:"email"`
}

type Account struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	HasPassword bool       `json:"has_password"`
	Identities  []Identity `json:"identities"`
}

type Session struct {
	Hash      string
	UserID    string
	CreatedAt int64
	ExpiresAt int64
}

type OAuthFlow struct {
	StateHash   string
	BrowserHash string
	Provider    string
	Verifier    string
	Nonce       string
	SessionHash string
	UserID      string
	ExpiresAt   int64
}
