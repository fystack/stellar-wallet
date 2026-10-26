package domain

type WalletStatus string

const (
	WalletGenerating WalletStatus = "generating"
	WalletReady      WalletStatus = "ready"
	WalletFailed     WalletStatus = "failed"
)

type TransactionStatus string

const (
	TransactionSigning   TransactionStatus = "signing"
	TransactionBroadcast TransactionStatus = "broadcast"
	TransactionConfirmed TransactionStatus = "confirmed"
	TransactionFailed    TransactionStatus = "failed"
)

type TransactionType string

const (
	TransactionIn   TransactionType = "in"
	TransactionOut  TransactionType = "out"
	TransactionSwap TransactionType = "swap"
)

// TrustlineMemo marks the outgoing transactions that add a trustline.
const TrustlineMemo = "Add trustline"
