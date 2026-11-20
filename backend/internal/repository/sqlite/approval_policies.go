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
		`SELECT workspace_id, threshold_usd, required_approvals, below_threshold, updated_at
		 FROM workspace_approval_policies WHERE workspace_id = ?`, workspaceID,
	).Scan(&policy.WorkspaceID, &policy.ThresholdUSD, &policy.RequiredApprovals, &policy.BelowThreshold, &policy.UpdatedAt)
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
		`INSERT INTO workspace_approval_policies (workspace_id, threshold_xlm, threshold_usd, required_approvals, below_threshold, updated_by, updated_at)
		 VALUES (?, '', ?, ?, ?, ?, ?)
		 ON CONFLICT(workspace_id) DO UPDATE SET threshold_xlm = '', threshold_usd = excluded.threshold_usd,
		 required_approvals = excluded.required_approvals, below_threshold = excluded.below_threshold,
		 updated_by = excluded.updated_by, updated_at = excluded.updated_at`,
		policy.WorkspaceID, policy.ThresholdUSD, policy.RequiredApprovals, policy.BelowThreshold, updatedBy, policy.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save approval policy: %w", err)
	}
	return nil
}

// LegacyXLMThreshold is a policy saved before thresholds moved to USD.
type LegacyXLMThreshold struct {
	WorkspaceID  string
	ThresholdXLM string
}

func (s *WorkspaceRepository) LegacyXLMThresholds() ([]LegacyXLMThreshold, error) {
	rows, err := s.db.Query(`SELECT workspace_id, threshold_xlm FROM workspace_approval_policies WHERE threshold_usd = '' AND threshold_xlm != ''`)
	if err != nil {
		return nil, fmt.Errorf("list legacy approval thresholds: %w", err)
	}
	defer rows.Close()
	var legacy []LegacyXLMThreshold
	for rows.Next() {
		var policy LegacyXLMThreshold
		if err := rows.Scan(&policy.WorkspaceID, &policy.ThresholdXLM); err != nil {
			return nil, fmt.Errorf("read legacy approval threshold: %w", err)
		}
		legacy = append(legacy, policy)
	}
	return legacy, rows.Err()
}

// ConvertXLMThreshold only touches a policy still on its XLM threshold, so an owner's newer save wins.
func (s *WorkspaceRepository) ConvertXLMThreshold(workspaceID, thresholdXLM, thresholdUSD string) error {
	_, err := s.db.Exec(
		`UPDATE workspace_approval_policies SET threshold_usd = ?, threshold_xlm = ''
		 WHERE workspace_id = ? AND threshold_usd = '' AND threshold_xlm = ?`,
		thresholdUSD, workspaceID, thresholdXLM,
	)
	if err != nil {
		return fmt.Errorf("convert approval threshold: %w", err)
	}
	return nil
}

func (s *WorkspaceRepository) DeleteApprovalPolicy(workspaceID string) error {
	if _, err := s.db.Exec(`DELETE FROM workspace_approval_policies WHERE workspace_id = ?`, workspaceID); err != nil {
		return fmt.Errorf("delete approval policy: %w", err)
	}
	return nil
}
