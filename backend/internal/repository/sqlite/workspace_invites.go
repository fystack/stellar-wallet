package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/repository"
)

const inviteColumns = `workspace_invites.id, workspace_invites.workspace_id, workspace_invites.email, workspace_invites.token_hash,
	workspace_invites.invited_by, COALESCE(users.email, ''), workspace_invites.role, workspace_invites.created_at, workspace_invites.expires_at,
	workspace_invites.accepted_at, workspace_invites.revoked_at`

const inviteFrom = ` FROM workspace_invites LEFT JOIN users ON users.id = workspace_invites.invited_by `

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
		`INSERT INTO workspace_invites (id, workspace_id, email, token_hash, invited_by, role, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		invite.ID, invite.WorkspaceID, invite.Email, invite.TokenHash, invite.InvitedBy, invite.Role, invite.CreatedAt, invite.ExpiresAt,
	); err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	return nil
}

// scanInvite reads inviteColumns, then any extra selected columns into extra.
func scanInvite(scanner interface{ Scan(...any) error }, extra ...any) (domain.WorkspaceInvite, error) {
	var invite domain.WorkspaceInvite
	destinations := []any{&invite.ID, &invite.WorkspaceID, &invite.Email, &invite.TokenHash, &invite.InvitedBy,
		&invite.InvitedByEmail, &invite.Role, &invite.CreatedAt, &invite.ExpiresAt, &invite.AcceptedAt, &invite.RevokedAt}
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

// AcceptInvite returns ErrConflict when the invite is no longer pending; joining twice is a no-op.
func (s *WorkspaceRepository) AcceptInvite(tokenHash, userID, acceptedAt string, now int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("accept invite: %w", err)
	}
	defer tx.Rollback()
	var workspaceID string
	var role domain.WorkspaceRole
	err = tx.QueryRow(
		`UPDATE workspace_invites SET accepted_at = ?
		 WHERE token_hash = ? AND accepted_at = '' AND revoked_at = '' AND expires_at > ?
		 RETURNING workspace_id, role`, acceptedAt, tokenHash, now,
	).Scan(&workspaceID, &role)
	if errors.Is(err, sql.ErrNoRows) {
		return storecontract.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("consume invite: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		workspaceID, userID, role, acceptedAt,
	); err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accept invite: %w", err)
	}
	return nil
}
