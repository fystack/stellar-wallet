package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/repository"
)

type WorkspaceRepository struct{ db *sql.DB }

func NewWorkspaceRepository(db *sql.DB) *WorkspaceRepository { return &WorkspaceRepository{db: db} }

func (s *WorkspaceRepository) EnsurePersonalWorkspace(userID, createdAt string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ensure personal workspace: %w", err)
	}
	defer tx.Rollback()
	if err := insertPersonalWorkspace(tx, userID, createdAt); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ensure personal workspace: %w", err)
	}
	return nil
}

func insertPersonalWorkspace(tx *sql.Tx, userID, createdAt string) error {
	workspaceID := domain.PersonalWorkspaceID(userID)
	if _, err := tx.Exec(`INSERT OR IGNORE INTO workspaces (id, name, created_by, created_at) VALUES (?, ?, ?, ?)`,
		workspaceID, personalName, userID, createdAt); err != nil {
		return fmt.Errorf("create personal workspace: %w", err)
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		workspaceID, userID, domain.RoleOwner, createdAt); err != nil {
		return fmt.Errorf("add personal workspace owner: %w", err)
	}
	return nil
}

func (s *WorkspaceRepository) CreateWorkspace(workspace domain.Workspace, ownerID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO workspaces (id, name, created_by, created_at) VALUES (?, ?, ?, ?)`,
		workspace.ID, workspace.Name, ownerID, workspace.CreatedAt); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		workspace.ID, ownerID, domain.RoleOwner, workspace.CreatedAt); err != nil {
		return fmt.Errorf("add workspace owner: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	return nil
}

const memberWorkspaceColumns = `workspaces.id, workspaces.name, workspace_members.role, workspaces.created_at`

func scanMemberWorkspace(scanner interface{ Scan(...any) error }) (domain.Workspace, error) {
	var workspace domain.Workspace
	err := scanner.Scan(&workspace.ID, &workspace.Name, &workspace.Role, &workspace.CreatedAt)
	return workspace, err
}

func (s *WorkspaceRepository) WorkspaceName(workspaceID string) (string, error) {
	var name string
	err := s.db.QueryRow(`SELECT name FROM workspaces WHERE id = ?`, workspaceID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storecontract.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read workspace name: %w", err)
	}
	return name, nil
}

// MemberWorkspace returns the workspace with the user's role, or ErrNotFound when they are not a member.
func (s *WorkspaceRepository) MemberWorkspace(workspaceID, userID string) (domain.Workspace, error) {
	workspace, err := scanMemberWorkspace(s.db.QueryRow(
		`SELECT `+memberWorkspaceColumns+` FROM workspaces
		 JOIN workspace_members ON workspace_members.workspace_id = workspaces.id
		 WHERE workspaces.id = ? AND workspace_members.user_id = ?`, workspaceID, userID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return workspace, storecontract.ErrNotFound
	}
	if err != nil {
		return workspace, fmt.Errorf("read workspace membership: %w", err)
	}
	return workspace, nil
}

func (s *WorkspaceRepository) WorkspacesForUser(userID string) ([]domain.Workspace, error) {
	rows, err := s.db.Query(
		`SELECT `+memberWorkspaceColumns+` FROM workspaces
		 JOIN workspace_members ON workspace_members.workspace_id = workspaces.id
		 WHERE workspace_members.user_id = ? ORDER BY workspaces.created_at`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()
	workspaces := []domain.Workspace{}
	for rows.Next() {
		workspace, err := scanMemberWorkspace(rows)
		if err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		workspaces = append(workspaces, workspace)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspaces: %w", err)
	}
	return workspaces, nil
}

func (s *WorkspaceRepository) Members(workspaceID string) ([]domain.WorkspaceMember, error) {
	rows, err := s.db.Query(
		`SELECT workspace_members.user_id, users.email, workspace_members.role, workspace_members.joined_at
		 FROM workspace_members JOIN users ON users.id = workspace_members.user_id
		 WHERE workspace_members.workspace_id = ? ORDER BY workspace_members.joined_at`, workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	members := []domain.WorkspaceMember{}
	for rows.Next() {
		var member domain.WorkspaceMember
		if err := rows.Scan(&member.UserID, &member.Email, &member.Role, &member.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

func (s *WorkspaceRepository) IsMemberEmail(workspaceID, email string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(1) FROM workspace_members JOIN users ON users.id = workspace_members.user_id
		 WHERE workspace_members.workspace_id = ? AND users.email = ?`, workspaceID, email,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check member email: %w", err)
	}
	return count > 0, nil
}

// RemoveMember returns ErrNotFound for a non-member and ErrConflict for the last owner.
func (s *WorkspaceRepository) RemoveMember(workspaceID, userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	defer tx.Rollback()
	var role domain.WorkspaceRole
	err = tx.QueryRow(`SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?`, workspaceID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return storecontract.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read member: %w", err)
	}
	if role == domain.RoleOwner {
		var owners int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM workspace_members WHERE workspace_id = ? AND role = ?`, workspaceID, domain.RoleOwner).Scan(&owners); err != nil {
			return fmt.Errorf("count owners: %w", err)
		}
		if owners <= 1 {
			return storecontract.ErrConflict
		}
	}
	if _, err := tx.Exec(`DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?`, workspaceID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// CreateInvite revokes any pending invite for the same email, so re-inviting replaces the old link.
func (s *WorkspaceRepository) CreateInvite(invite domain.WorkspaceInvite, now int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE workspace_invites SET revoked_at = ?
		 WHERE workspace_id = ? AND email = ? AND accepted_at = '' AND revoked_at = '' AND expires_at > ?`,
		invite.CreatedAt, invite.WorkspaceID, invite.Email, now,
	); err != nil {
		return fmt.Errorf("replace pending invite: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO workspace_invites (id, workspace_id, email, token_hash, invited_by, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		invite.ID, invite.WorkspaceID, invite.Email, invite.TokenHash, invite.InvitedBy, invite.CreatedAt, invite.ExpiresAt,
	); err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	return nil
}

const inviteColumns = `workspace_invites.id, workspace_invites.workspace_id, workspace_invites.email, workspace_invites.token_hash,
	workspace_invites.invited_by, COALESCE(users.email, ''), workspace_invites.created_at, workspace_invites.expires_at,
	workspace_invites.accepted_at, workspace_invites.revoked_at`

const inviteFrom = ` FROM workspace_invites LEFT JOIN users ON users.id = workspace_invites.invited_by `

// scanInvite reads inviteColumns, then any extra selected columns into extra.
func scanInvite(scanner interface{ Scan(...any) error }, extra ...any) (domain.WorkspaceInvite, error) {
	var invite domain.WorkspaceInvite
	destinations := []any{&invite.ID, &invite.WorkspaceID, &invite.Email, &invite.TokenHash, &invite.InvitedBy,
		&invite.InvitedByEmail, &invite.CreatedAt, &invite.ExpiresAt, &invite.AcceptedAt, &invite.RevokedAt}
	err := scanner.Scan(append(destinations, extra...)...)
	return invite, err
}

func (s *WorkspaceRepository) InviteByID(inviteID string) (domain.WorkspaceInvite, error) {
	invite, err := scanInvite(s.db.QueryRow(`SELECT `+inviteColumns+inviteFrom+`WHERE workspace_invites.id = ?`, inviteID))
	if errors.Is(err, sql.ErrNoRows) {
		return invite, storecontract.ErrNotFound
	}
	if err != nil {
		return invite, fmt.Errorf("read invite: %w", err)
	}
	return invite, nil
}

// PendingInvitesForUser skips workspaces userID already belongs to.
func (s *WorkspaceRepository) PendingInvitesForUser(userID string, emails []string, now int64) ([]domain.WorkspaceInvite, error) {
	if len(emails) == 0 {
		return []domain.WorkspaceInvite{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(emails)), ",")
	args := []any{now, userID}
	for _, email := range emails {
		args = append(args, email)
	}
	rows, err := s.db.Query(
		`SELECT `+inviteColumns+`, workspaces.name`+inviteFrom+`JOIN workspaces ON workspaces.id = workspace_invites.workspace_id
		 WHERE workspace_invites.accepted_at = '' AND workspace_invites.revoked_at = '' AND workspace_invites.expires_at > ?
		 AND NOT EXISTS (SELECT 1 FROM workspace_members WHERE workspace_members.workspace_id = workspace_invites.workspace_id AND workspace_members.user_id = ?)
		 AND workspace_invites.email IN (`+placeholders+`)
		 ORDER BY workspace_invites.created_at DESC`, args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list user invites: %w", err)
	}
	defer rows.Close()
	invites := []domain.WorkspaceInvite{}
	for rows.Next() {
		var workspaceName string
		invite, err := scanInvite(rows, &workspaceName)
		if err != nil {
			return nil, fmt.Errorf("scan user invite: %w", err)
		}
		invite.WorkspaceName = workspaceName
		invites = append(invites, invite)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user invites: %w", err)
	}
	return invites, nil
}

func (s *WorkspaceRepository) InviteByToken(tokenHash string) (domain.WorkspaceInvite, error) {
	invite, err := scanInvite(s.db.QueryRow(`SELECT `+inviteColumns+inviteFrom+`WHERE workspace_invites.token_hash = ?`, tokenHash))
	if errors.Is(err, sql.ErrNoRows) {
		return invite, storecontract.ErrNotFound
	}
	if err != nil {
		return invite, fmt.Errorf("read invite: %w", err)
	}
	return invite, nil
}

func (s *WorkspaceRepository) PendingInvites(workspaceID string, now int64) ([]domain.WorkspaceInvite, error) {
	rows, err := s.db.Query(
		`SELECT `+inviteColumns+inviteFrom+`WHERE workspace_invites.workspace_id = ? AND workspace_invites.accepted_at = ''
		 AND workspace_invites.revoked_at = '' AND workspace_invites.expires_at > ? ORDER BY workspace_invites.created_at DESC`,
		workspaceID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()
	invites := []domain.WorkspaceInvite{}
	for rows.Next() {
		invite, err := scanInvite(rows)
		if err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		invites = append(invites, invite)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invites: %w", err)
	}
	return invites, nil
}

// RevokeInvite returns false when no pending invite with that id exists in the workspace.
func (s *WorkspaceRepository) RevokeInvite(workspaceID, inviteID, revokedAt string) (bool, error) {
	result, err := s.db.Exec(
		`UPDATE workspace_invites SET revoked_at = ? WHERE id = ? AND workspace_id = ? AND accepted_at = '' AND revoked_at = ''`,
		revokedAt, inviteID, workspaceID,
	)
	if err != nil {
		return false, fmt.Errorf("revoke invite: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("revoke invite: %w", err)
	}
	return changed == 1, nil
}

// AcceptInvite returns ErrConflict when the invite is no longer pending, and false when already a member.
func (s *WorkspaceRepository) AcceptInvite(tokenHash, userID, acceptedAt string, now int64) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("accept invite: %w", err)
	}
	defer tx.Rollback()
	var workspaceID string
	err = tx.QueryRow(
		`UPDATE workspace_invites SET accepted_at = ?
		 WHERE token_hash = ? AND accepted_at = '' AND revoked_at = '' AND expires_at > ?
		 RETURNING workspace_id`, acceptedAt, tokenHash, now,
	).Scan(&workspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, storecontract.ErrConflict
	}
	if err != nil {
		return false, fmt.Errorf("consume invite: %w", err)
	}
	result, err := tx.Exec(
		`INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		workspaceID, userID, domain.RoleMember, acceptedAt,
	)
	if err != nil {
		return false, fmt.Errorf("add member: %w", err)
	}
	added, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("add member: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("accept invite: %w", err)
	}
	return added == 1, nil
}
