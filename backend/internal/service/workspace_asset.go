package service

import (
	"strings"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
)

func (s *WorkspaceService) Assets(userID, workspaceID string) ([]domain.CustomAsset, error) {
	if _, err := memberWorkspace(s.workspaces, workspaceID, userID); err != nil {
		return nil, err
	}
	return s.workspaces.WorkspaceAssets(workspaceID)
}

func (s *WorkspaceService) AddAsset(userID, workspaceID string, asset domain.CustomAsset) ([]domain.CustomAsset, error) {
	if _, err := requireOwner(s.workspaces, workspaceID, userID); err != nil {
		return nil, err
	}
	asset.Code, asset.Issuer = strings.TrimSpace(asset.Code), strings.TrimSpace(asset.Issuer)
	if asset.Code == "" || asset.Issuer == "" {
		return nil, apperror.InvalidAsset.New()
	}
	if err := s.workspaces.AddWorkspaceAsset(workspaceID, asset); err != nil {
		return nil, err
	}
	return s.workspaces.WorkspaceAssets(workspaceID)
}

func (s *WorkspaceService) RemoveAsset(userID, workspaceID, code, issuer string) ([]domain.CustomAsset, error) {
	if _, err := requireOwner(s.workspaces, workspaceID, userID); err != nil {
		return nil, err
	}
	if err := s.workspaces.RemoveWorkspaceAsset(workspaceID, code, issuer); err != nil {
		return nil, err
	}
	return s.workspaces.WorkspaceAssets(workspaceID)
}
