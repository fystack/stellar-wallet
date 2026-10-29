package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

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
	result, err := tx.Exec(`INSERT OR IGNORE INTO workspaces (id, name, created_by, created_at) VALUES (?, ?, ?, ?)`,
		workspaceID, personalName, userID, createdAt)
	if err != nil {
		return fmt.Errorf("create personal workspace: %w", err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("create personal workspace: %w", err)
	}
	if created == 1 {
		if err := seedWorkspaceAssets(tx, workspaceID); err != nil {
			return err
		}
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
	if err := seedWorkspaceAssets(tx, workspace.ID); err != nil {
		return err
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

// RemoveMember returns ErrNotFound for a non-member, ErrConflict for the last owner and ErrTooFewSenders when the approval policy could no longer be met.
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
		if err := requireAnotherOwner(tx, workspaceID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?`, workspaceID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if err := requireApprovers(tx, workspaceID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// UpdateMemberRole returns the same errors as RemoveMember.
func (s *WorkspaceRepository) UpdateMemberRole(workspaceID, userID string, role domain.WorkspaceRole) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	defer tx.Rollback()
	var current domain.WorkspaceRole
	err = tx.QueryRow(`SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?`, workspaceID, userID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return storecontract.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read member: %w", err)
	}
	if current == domain.RoleOwner && role != domain.RoleOwner {
		if err := requireAnotherOwner(tx, workspaceID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND user_id = ?`, role, workspaceID, userID); err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if err := requireApprovers(tx, workspaceID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	return nil
}

func requireAnotherOwner(tx *sql.Tx, workspaceID string) error {
	var owners int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM workspace_members WHERE workspace_id = ? AND role = ?`, workspaceID, domain.RoleOwner).Scan(&owners); err != nil {
		return fmt.Errorf("count owners: %w", err)
	}
	if owners <= 1 {
		return storecontract.ErrConflict
	}
	return nil
}

// CountSenders counts owners and operators.
func (s *WorkspaceRepository) CountSenders(workspaceID string) (int, error) {
	var senders int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM workspace_members WHERE workspace_id = ? AND role IN (?, ?)`,
		workspaceID, domain.RoleOwner, domain.RoleOperator).Scan(&senders)
	if err != nil {
		return 0, fmt.Errorf("count senders: %w", err)
	}
	return senders, nil
}
