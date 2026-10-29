package domain

type WorkspaceRole string

const (
	RoleOwner    WorkspaceRole = "owner"
	RoleOperator WorkspaceRole = "operator"
	RoleMember   WorkspaceRole = "member"
)

func (r WorkspaceRole) Valid() bool {
	return r == RoleOwner || r == RoleOperator || r == RoleMember
}

// CanSend reports whether the role may start payments and approve other members' payments.
func (r WorkspaceRole) CanSend() bool { return r == RoleOwner || r == RoleOperator }

type InviteStatus string

const (
	InvitePending  InviteStatus = "pending"
	InviteAccepted InviteStatus = "accepted"
	InviteRevoked  InviteStatus = "revoked"
	InviteExpired  InviteStatus = "expired"
)

// PersonalWorkspaceID is deterministic so startup backfill and lazy creation agree.
func PersonalWorkspaceID(userID string) string { return "personal-" + userID }

type Workspace struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Role      WorkspaceRole `json:"role"`
	Personal  bool          `json:"personal"`
	CreatedAt string        `json:"createdAt"`
}

type WorkspaceMember struct {
	UserID   string        `json:"userId"`
	Email    string        `json:"email"`
	Role     WorkspaceRole `json:"role"`
	JoinedAt string        `json:"joinedAt"`
}

type WorkspaceInvite struct {
	ID             string        `json:"id"`
	WorkspaceID    string        `json:"workspaceId"`
	WorkspaceName  string        `json:"workspaceName,omitempty"`
	Email          string        `json:"email"`
	TokenHash      string        `json:"-"`
	InvitedBy      string        `json:"-"`
	InvitedByEmail string        `json:"invitedBy"`
	Role           WorkspaceRole `json:"role"`
	CreatedAt      string        `json:"createdAt"`
	ExpiresAt      int64         `json:"expiresAt"`
	AcceptedAt     string        `json:"-"`
	RevokedAt      string        `json:"-"`
}

type BelowThresholdMode string

const (
	BelowThresholdSend    BelowThresholdMode = "send"
	BelowThresholdApprove BelowThresholdMode = "approve"
)

// ApprovalPolicy also holds every non-XLM payment, since the threshold is in XLM.
type ApprovalPolicy struct {
	WorkspaceID       string             `json:"workspaceId"`
	ThresholdXLM      string             `json:"thresholdXlm"`
	RequiredApprovals int                `json:"requiredApprovals"`
	BelowThreshold    BelowThresholdMode `json:"belowThreshold"`
	UpdatedAt         string             `json:"updatedAt"`
}
