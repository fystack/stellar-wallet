package service

import (
	"errors"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"
)

type ApprovalPolicySettings struct {
	Policy *domain.ApprovalPolicy `json:"policy"`
	// MaxApprovals is how many approvals a payment can require: every sender except the initiator.
	MaxApprovals int `json:"maxApprovals"`
}

type ApprovalPolicyInput struct {
	ThresholdXLM      string                    `json:"thresholdXlm"`
	RequiredApprovals int                       `json:"requiredApprovals"`
	BelowThreshold    domain.BelowThresholdMode `json:"belowThreshold"`
}

func (s *WorkspaceService) ApprovalPolicy(userID, workspaceID string) (ApprovalPolicySettings, error) {
	if _, err := memberWorkspace(s.workspaces, workspaceID, userID); err != nil {
		return ApprovalPolicySettings{}, err
	}
	return s.approvalPolicySettings(workspaceID)
}

func (s *WorkspaceService) SetApprovalPolicy(userID, workspaceID string, input ApprovalPolicyInput) (ApprovalPolicySettings, error) {
	if err := s.requireTeamOwner(userID, workspaceID); err != nil {
		return ApprovalPolicySettings{}, err
	}
	threshold, err := parseAmount(input.ThresholdXLM)
	if err != nil {
		return ApprovalPolicySettings{}, apperror.InvalidApprovalThreshold.New()
	}
	if input.BelowThreshold != domain.BelowThresholdSend && input.BelowThreshold != domain.BelowThresholdApprove {
		return ApprovalPolicySettings{}, apperror.InvalidBelowThreshold.New()
	}
	maxApprovals, err := maxApprovals(s.workspaces, workspaceID)
	if err != nil {
		return ApprovalPolicySettings{}, err
	}
	if input.RequiredApprovals < 1 || input.RequiredApprovals > maxApprovals {
		return ApprovalPolicySettings{}, apperror.InvalidRequiredApprovals.New()
	}
	policy := domain.ApprovalPolicy{
		WorkspaceID:       workspaceID,
		ThresholdXLM:      threshold,
		RequiredApprovals: input.RequiredApprovals,
		BelowThreshold:    input.BelowThreshold,
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.workspaces.SaveApprovalPolicy(policy, userID); err != nil {
		return ApprovalPolicySettings{}, err
	}
	return ApprovalPolicySettings{Policy: &policy, MaxApprovals: maxApprovals}, nil
}

func (s *WorkspaceService) DisableApprovalPolicy(userID, workspaceID string) error {
	if err := s.requireTeamOwner(userID, workspaceID); err != nil {
		return err
	}
	return s.workspaces.DeleteApprovalPolicy(workspaceID)
}

func (s *WorkspaceService) requireTeamOwner(userID, workspaceID string) error {
	workspace, err := requireOwner(s.workspaces, workspaceID, userID)
	if err != nil {
		return err
	}
	if workspace.Personal {
		return apperror.PersonalWorkspacePolicy.New()
	}
	return nil
}

func (s *WorkspaceService) approvalPolicySettings(workspaceID string) (ApprovalPolicySettings, error) {
	maxApprovals, err := maxApprovals(s.workspaces, workspaceID)
	if err != nil {
		return ApprovalPolicySettings{}, err
	}
	settings := ApprovalPolicySettings{MaxApprovals: maxApprovals}
	policy, err := s.workspaces.ApprovalPolicy(workspaceID)
	if errors.Is(err, repository.ErrNotFound) {
		return settings, nil
	}
	if err != nil {
		return ApprovalPolicySettings{}, err
	}
	settings.Policy = &policy
	return settings, nil
}

// maxApprovals is how many people can approve a payment: every owner and operator except the initiator.
func maxApprovals(workspaces *sqlite.WorkspaceRepository, workspaceID string) (int, error) {
	senders, err := workspaces.CountSenders(workspaceID)
	if err != nil {
		return 0, err
	}
	return max(senders-1, 0), nil
}
