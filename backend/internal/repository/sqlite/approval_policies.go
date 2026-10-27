package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/repository"
)

// requireApprovers runs inside the change's transaction, so two concurrent changes can't both pass it.
func requireApprovers(tx *sql.Tx, workspaceID string) error {
	var tooFew bool
	err := tx.QueryRow(
		`SELECT required_approvals > (
			SELECT COUNT(1) - 1 FROM workspace_members WHERE workspace_id = ? AND role IN (?, ?))
		 FROM workspace_approval_policies WHERE workspace_id = ?`,
		workspaceID, domain.RoleOwner, domain.RoleOperator, workspaceID,
	).Scan(&tooFew)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check approval policy: %w", err)
	}
	if tooFew {
		return storecontract.ErrTooFewSenders
	}
	return nil
}

// ApprovalPolicy returns ErrNotFound when the workspace has no policy.
func (s *WorkspaceRepository) ApprovalPolicy(workspaceID string) (domain.ApprovalPolicy, error) {
	var policy domain.ApprovalPolicy
	err := s.db.QueryRow(
		`SELECT workspace_id, threshold_xlm, required_approvals, below_threshold, updated_at
		 FROM workspace_approval_policies WHERE workspace_id = ?`, workspaceID,
	).Scan(&policy.WorkspaceID, &policy.ThresholdXLM, &policy.RequiredApprovals, &policy.BelowThreshold, &policy.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return policy, storecontract.ErrNotFound
	}
	if err != nil {
		return policy, fmt.Errorf("read approval policy: %w", err)
	}
	return policy, nil
}

func (s *WorkspaceRepository) SaveApprovalPolicy(policy domain.ApprovalPolicy, updatedBy string) error {
	_, err := s.db.Exec(
		`INSERT INTO workspace_approval_policies (workspace_id, threshold_xlm, required_approvals, below_threshold, updated_by, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(workspace_id) DO UPDATE SET threshold_xlm = excluded.threshold_xlm, required_approvals = excluded.required_approvals,
		 below_threshold = excluded.below_threshold, updated_by = excluded.updated_by, updated_at = excluded.updated_at`,
		policy.WorkspaceID, policy.ThresholdXLM, policy.RequiredApprovals, policy.BelowThreshold, updatedBy, policy.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save approval policy: %w", err)
	}
	return nil
}

func (s *WorkspaceRepository) DeleteApprovalPolicy(workspaceID string) error {
	if _, err := s.db.Exec(`DELETE FROM workspace_approval_policies WHERE workspace_id = ?`, workspaceID); err != nil {
		return fmt.Errorf("delete approval policy: %w", err)
	}
	return nil
}
