package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"

	"github.com/google/uuid"
)

const inviteTTL = 7 * 24 * time.Hour

// Mailer delivers invite emails. A nil Mailer means email is not configured.
type Mailer interface {
	Send(to, subject, body string) error
}

// WalletProvisioner creates the first wallet of a new team workspace.
type WalletProvisioner interface {
	Provision(userID, workspaceID, name, chainName string) (domain.Wallet, error)
}

type WorkspaceService struct {
	workspaces  *sqlite.WorkspaceRepository
	accounts    *sqlite.AuthRepository
	wallets     WalletProvisioner
	mailer      Mailer
	frontendURL string
}

func NewWorkspaceService(workspaces *sqlite.WorkspaceRepository, accounts *sqlite.AuthRepository, wallets WalletProvisioner, mailer Mailer, frontendURL string) *WorkspaceService {
	return &WorkspaceService{workspaces: workspaces, accounts: accounts, wallets: wallets, mailer: mailer, frontendURL: strings.TrimRight(frontendURL, "/")}
}

type WorkspaceList struct {
	Workspaces []domain.Workspace `json:"workspaces"`
	ActiveID   string             `json:"activeId"`
	// Selected is false when the session has not chosen a workspace yet (fresh sign-in).
	Selected bool `json:"selected"`
}

type WorkspaceMembers struct {
	Members []domain.WorkspaceMember `json:"members"`
	Invites []domain.WorkspaceInvite `json:"invites"`
}

// JoinResult carries WalletError instead of failing workspace creation when its first wallet can't be created.
type JoinResult struct {
	Workspace   domain.Workspace `json:"workspace"`
	Wallet      *domain.Wallet   `json:"wallet,omitempty"`
	WalletError string           `json:"walletError,omitempty"`
}

func describeWorkspace(workspace domain.Workspace) domain.Workspace {
	workspace.Personal = strings.HasPrefix(workspace.ID, domain.PersonalWorkspaceID(""))
	return workspace
}

// activeWorkspace falls back to the personal workspace when the session's is unset or no longer joined.
func activeWorkspace(workspaces *sqlite.WorkspaceRepository, userID, sessionWorkspaceID string) (domain.Workspace, error) {
	if sessionWorkspaceID != "" {
		workspace, err := workspaces.MemberWorkspace(sessionWorkspaceID, userID)
		if err == nil {
			return describeWorkspace(workspace), nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return workspace, err
		}
	}
	workspace, err := workspaces.MemberWorkspace(domain.PersonalWorkspaceID(userID), userID)
	if errors.Is(err, repository.ErrNotFound) {
		if err := workspaces.EnsurePersonalWorkspace(userID, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return domain.Workspace{}, err
		}
		workspace, err = workspaces.MemberWorkspace(domain.PersonalWorkspaceID(userID), userID)
	}
	if err != nil {
		return workspace, fmt.Errorf("read personal workspace: %w", err)
	}
	return describeWorkspace(workspace), nil
}

// requestedWorkspace uses the workspace the client asked for, so tabs sharing a session stay independent.
func requestedWorkspace(workspaces *sqlite.WorkspaceRepository, userID, sessionWorkspaceID, requestedWorkspaceID string) (domain.Workspace, error) {
	if requestedWorkspaceID != "" {
		return memberWorkspace(workspaces, requestedWorkspaceID, userID)
	}
	return activeWorkspace(workspaces, userID, sessionWorkspaceID)
}

func memberWorkspace(workspaces *sqlite.WorkspaceRepository, workspaceID, userID string) (domain.Workspace, error) {
	workspace, err := workspaces.MemberWorkspace(workspaceID, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return workspace, apperror.WorkspaceNotFound.New()
	}
	if err != nil {
		return workspace, err
	}
	return describeWorkspace(workspace), nil
}

func requireOwner(workspaces *sqlite.WorkspaceRepository, workspaceID, userID string) (domain.Workspace, error) {
	workspace, err := memberWorkspace(workspaces, workspaceID, userID)
	if err != nil {
		return workspace, err
	}
	if workspace.Role != domain.RoleOwner {
		return workspace, apperror.WorkspaceOwnerRequired.New()
	}
	return workspace, nil
}

func (s *WorkspaceService) List(userID, sessionWorkspaceID string) (WorkspaceList, error) {
	active, err := activeWorkspace(s.workspaces, userID, sessionWorkspaceID)
	if err != nil {
		return WorkspaceList{}, err
	}
	workspaces, err := s.workspaces.WorkspacesForUser(userID)
	if err != nil {
		return WorkspaceList{}, err
	}
	for index := range workspaces {
		workspaces[index] = describeWorkspace(workspaces[index])
	}
	return WorkspaceList{Workspaces: workspaces, ActiveID: active.ID, Selected: sessionWorkspaceID == active.ID}, nil
}

func (s *WorkspaceService) Create(userID, sessionToken, name string) (JoinResult, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 60 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return JoinResult{}, apperror.InvalidWorkspaceName.New()
	}
	workspace := domain.Workspace{ID: uuid.NewString(), Name: name, Role: domain.RoleOwner, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := s.workspaces.CreateWorkspace(workspace, userID); err != nil {
		return JoinResult{}, err
	}
	if err := s.accounts.SetSessionWorkspace(auth.HashToken(sessionToken), workspace.ID); err != nil {
		return JoinResult{}, err
	}
	return s.provisionFirstWallet(userID, workspace), nil
}

func (s *WorkspaceService) Switch(userID, sessionToken, workspaceID string) (domain.Workspace, error) {
	workspace, err := memberWorkspace(s.workspaces, workspaceID, userID)
	if err != nil {
		return workspace, err
	}
	if err := s.accounts.SetSessionWorkspace(auth.HashToken(sessionToken), workspace.ID); err != nil {
		return workspace, err
	}
	return workspace, nil
}

func (s *WorkspaceService) Members(userID, workspaceID string) (WorkspaceMembers, error) {
	if _, err := memberWorkspace(s.workspaces, workspaceID, userID); err != nil {
		return WorkspaceMembers{}, err
	}
	members, err := s.workspaces.Members(workspaceID)
	if err != nil {
		return WorkspaceMembers{}, err
	}
	invites, err := s.workspaces.PendingInvites(workspaceID, time.Now().Unix())
	if err != nil {
		return WorkspaceMembers{}, err
	}
	return WorkspaceMembers{Members: members, Invites: invites}, nil
}

func (s *WorkspaceService) RemoveMember(userID, workspaceID, memberID string) error {
	if _, err := requireOwner(s.workspaces, workspaceID, userID); err != nil {
		return err
	}
	err := s.workspaces.RemoveMember(workspaceID, memberID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.MemberNotFound.New()
	}
	if errors.Is(err, repository.ErrConflict) {
		return apperror.LastWorkspaceOwner.New()
	}
	if errors.Is(err, repository.ErrTooFewSenders) {
		return apperror.ApprovalPolicyNeedsMember.New()
	}
	return err
}

func (s *WorkspaceService) ChangeRole(userID, workspaceID, memberID string, role domain.WorkspaceRole) error {
	if _, err := requireOwner(s.workspaces, workspaceID, userID); err != nil {
		return err
	}
	if !role.Valid() {
		return apperror.InvalidMemberRole.New()
	}
	err := s.workspaces.UpdateMemberRole(workspaceID, memberID, role)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.MemberNotFound.New()
	}
	if errors.Is(err, repository.ErrConflict) {
		return apperror.LastWorkspaceOwner.New()
	}
	if errors.Is(err, repository.ErrTooFewSenders) {
		return apperror.ApprovalPolicyNeedsMember.New()
	}
	return err
}

func (s *WorkspaceService) provisionFirstWallet(userID string, workspace domain.Workspace) JoinResult {
	result := JoinResult{Workspace: workspace}
	user, err := s.accounts.UserByID(userID)
	if err != nil {
		logError("read member for wallet workspace="+workspace.ID, err)
		result.WalletError = "The workspace wallet could not be created. Create one from Wallets."
		return result
	}
	wallet, err := s.wallets.Provision(userID, workspace.ID, memberWalletName(user.Email), "stellar")
	if err != nil {
		var failure *apperror.Error
		if errors.As(err, &failure) {
			result.WalletError = failure.Message
		} else {
			logError("provision member wallet workspace="+workspace.ID, err)
			result.WalletError = "Your wallet could not be created. Please try again later."
		}
		return result
	}
	result.Wallet = &wallet
	return result
}

func memberWalletName(email string) string {
	local, _, _ := strings.Cut(email, "@")
	return local + "'s wallet"
}
