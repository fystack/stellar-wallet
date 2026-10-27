package domain

type WorkspaceRole string

const (
	RoleOwner  WorkspaceRole = "owner"
	RoleMember WorkspaceRole = "member"
)

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
	ID             string `json:"id"`
	WorkspaceID    string `json:"workspaceId"`
	WorkspaceName  string `json:"workspaceName,omitempty"`
	Email          string `json:"email"`
	TokenHash      string `json:"-"`
	InvitedBy      string `json:"-"`
	InvitedByEmail string `json:"invitedBy"`
	CreatedAt      string `json:"createdAt"`
	ExpiresAt      int64  `json:"expiresAt"`
	AcceptedAt     string `json:"-"`
	RevokedAt      string `json:"-"`
}
