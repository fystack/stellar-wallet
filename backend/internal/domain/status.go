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

type PaymentRequestStatus string

const (
	PaymentRequestPending   PaymentRequestStatus = "pending"
	PaymentRequestExecuting PaymentRequestStatus = "executing"
	PaymentRequestExecuted  PaymentRequestStatus = "executed"
	PaymentRequestFailed    PaymentRequestStatus = "failed"
	PaymentRequestRejected  PaymentRequestStatus = "rejected"
	PaymentRequestCancelled PaymentRequestStatus = "cancelled"
	PaymentRequestExpired   PaymentRequestStatus = "expired"
)

type VoteDecision string

const (
	VoteApprove VoteDecision = "approve"
	VoteReject  VoteDecision = "reject"
)
