package domain

// PaymentRequest stores payment inputs, not an envelope: envelopes expire 300 seconds after they are built.
type PaymentRequest struct {
	ID                string               `json:"id"`
	WorkspaceID       string               `json:"workspaceId"`
	WalletID          string               `json:"walletId"`
	WalletName        string               `json:"walletName"`
	InitiatorID       string               `json:"initiatorId"`
	InitiatorEmail    string               `json:"initiatorEmail"`
	Destination       string               `json:"destination"`
	Amount            string               `json:"amount"`
	AssetCode         string               `json:"assetCode,omitempty"`
	AssetIssuer       string               `json:"assetIssuer,omitempty"`
	Symbol            string               `json:"symbol"`
	Memo              string               `json:"memo,omitempty"`
	MemoType          string               `json:"memoType,omitempty"`
	RequiredApprovals int                  `json:"requiredApprovals"`
	Status            PaymentRequestStatus `json:"status"`
	TransactionID     string               `json:"transactionId,omitempty"`
	TransactionStatus TransactionStatus    `json:"transactionStatus,omitempty"`
	TxHash            string               `json:"txHash,omitempty"`
	Reason            string               `json:"reason,omitempty"`
	CreatedAt         string               `json:"createdAt"`
	ExpiresAt         int64                `json:"expiresAt"`
	DecidedAt         string               `json:"decidedAt,omitempty"`
	Votes             []PaymentRequestVote `json:"votes"`
}

type PaymentRequestVote struct {
	UserID    string       `json:"userId"`
	Email     string       `json:"email"`
	Decision  VoteDecision `json:"decision"`
	CreatedAt string       `json:"createdAt"`
}

func (r PaymentRequest) Approvals() int {
	approvals := 0
	for _, vote := range r.Votes {
		if vote.Decision == VoteApprove {
			approvals++
		}
	}
	return approvals
}

func (r PaymentRequest) VotedBy(userID string) bool {
	for _, vote := range r.Votes {
		if vote.UserID == userID {
			return true
		}
	}
	return false
}
