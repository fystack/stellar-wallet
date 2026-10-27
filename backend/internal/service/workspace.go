package service

import (
	"errors"
	"fmt"
	"net/mail"
	"slices"
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

// WalletProvisioner creates the wallet each member gets when they join a workspace.
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

type InviteResult struct {
	Invite    domain.WorkspaceInvite `json:"invite"`
	Link      string                 `json:"link"`
	EmailSent bool                   `json:"emailSent"`
}

type InvitePreview struct {
	WorkspaceName string              `json:"workspaceName"`
	InvitedBy     string              `json:"invitedBy"`
	Email         string              `json:"email"`
	Status        domain.InviteStatus `json:"status"`
	// Only the token holder sees these; they pick the invite page's sign-in method.
	HasAccount  bool     `json:"hasAccount"`
	HasPassword bool     `json:"hasPassword"`
	Providers   []string `json:"providers"`
}

// JoinResult carries WalletError instead of failing the join when wallet provisioning fails.
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

func requireOwner(workspaces *sqlite.WorkspaceRepository, workspaceID, userID string) error {
	workspace, err := memberWorkspace(workspaces, workspaceID, userID)
	if err != nil {
		return err
	}
	if workspace.Role != domain.RoleOwner {
		return apperror.WorkspaceOwnerRequired.New()
	}
	return nil
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
	return s.provisionMemberWallet(userID, workspace), nil
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

// Invite always returns the link so owners can share it when email is not configured.
func (s *WorkspaceService) Invite(userID, workspaceID, email string) (InviteResult, error) {
	workspace, err := memberWorkspace(s.workspaces, workspaceID, userID)
	if err != nil {
		return InviteResult{}, err
	}
	if workspace.Role != domain.RoleOwner {
		return InviteResult{}, apperror.WorkspaceOwnerRequired.New()
	}
	if workspace.Personal {
		return InviteResult{}, apperror.PersonalWorkspace.New()
	}
	email = strings.ToLower(strings.TrimSpace(email))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return InviteResult{}, apperror.InvalidInviteEmail.New()
	}
	member, err := s.workspaces.IsMemberEmail(workspaceID, email)
	if err != nil {
		return InviteResult{}, err
	}
	if member {
		return InviteResult{}, apperror.AlreadyWorkspaceMember.New()
	}
	inviter, err := s.accounts.UserByID(userID)
	if err != nil {
		return InviteResult{}, fmt.Errorf("read inviter: %w", err)
	}
	token, err := auth.RandomToken()
	if err != nil {
		return InviteResult{}, fmt.Errorf("generate invite token: %w", err)
	}
	now := time.Now()
	invite := domain.WorkspaceInvite{
		ID:             uuid.NewString(),
		WorkspaceID:    workspaceID,
		Email:          email,
		TokenHash:      auth.HashToken(token),
		InvitedBy:      userID,
		InvitedByEmail: inviter.Email,
		CreatedAt:      now.UTC().Format(time.RFC3339),
		ExpiresAt:      now.Add(inviteTTL).Unix(),
	}
	if err := s.workspaces.CreateInvite(invite, now.Unix()); err != nil {
		return InviteResult{}, err
	}
	result := InviteResult{Invite: invite, Link: s.frontendURL + "/#/invite/" + token}
	if s.mailer != nil {
		subject := fmt.Sprintf("%s invited you to %s on Stellar MPC Wallet", inviter.Email, workspace.Name)
		body := fmt.Sprintf("%s invited you to join the %q workspace on Stellar MPC Wallet.\n\nAccept the invite:\n%s\n\nThe link expires in 7 days and works once. Sign in with %s to accept it.\n",
			inviter.Email, workspace.Name, result.Link, email)
		if err := s.mailer.Send(email, subject, body); err != nil {
			logError("send invite email workspace="+workspaceID, err)
		} else {
			result.EmailSent = true
		}
	}
	return result, nil
}

func (s *WorkspaceService) RevokeInvite(userID, workspaceID, inviteID string) error {
	if err := requireOwner(s.workspaces, workspaceID, userID); err != nil {
		return err
	}
	revoked, err := s.workspaces.RevokeInvite(workspaceID, inviteID, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	if !revoked {
		return apperror.InviteNotFound.New()
	}
	return nil
}

func (s *WorkspaceService) RemoveMember(userID, workspaceID, memberID string) error {
	if err := requireOwner(s.workspaces, workspaceID, userID); err != nil {
		return err
	}
	err := s.workspaces.RemoveMember(workspaceID, memberID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.MemberNotFound.New()
	}
	if errors.Is(err, repository.ErrConflict) {
		return apperror.LastWorkspaceOwner.New()
	}
	return err
}

func inviteStatus(invite domain.WorkspaceInvite, now int64) domain.InviteStatus {
	switch {
	case invite.AcceptedAt != "":
		return domain.InviteAccepted
	case invite.RevokedAt != "":
		return domain.InviteRevoked
	case invite.ExpiresAt <= now:
		return domain.InviteExpired
	}
	return domain.InvitePending
}

func (s *WorkspaceService) inviteByToken(token string) (domain.WorkspaceInvite, error) {
	if token == "" {
		return domain.WorkspaceInvite{}, apperror.InviteNotFound.New()
	}
	invite, err := s.workspaces.InviteByToken(auth.HashToken(token))
	if errors.Is(err, repository.ErrNotFound) {
		return invite, apperror.InviteNotFound.New()
	}
	return invite, err
}

// PreviewInvite is public so the invite page can show who invited whom before sign-in.
func (s *WorkspaceService) PreviewInvite(token string) (InvitePreview, error) {
	invite, err := s.inviteByToken(token)
	if err != nil {
		return InvitePreview{}, err
	}
	workspaceName, err := s.workspaces.WorkspaceName(invite.WorkspaceID)
	if err != nil {
		return InvitePreview{}, err
	}
	preview := InvitePreview{WorkspaceName: workspaceName, InvitedBy: invite.InvitedByEmail, Email: invite.Email, Status: inviteStatus(invite, time.Now().Unix()), Providers: []string{}}
	if preview.Status != domain.InvitePending {
		return preview, nil
	}
	userID, _, err := s.accounts.UserCredentials(invite.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return preview, nil
	}
	if err != nil {
		return InvitePreview{}, err
	}
	account, err := s.accounts.Account(userID)
	if err != nil {
		return InvitePreview{}, fmt.Errorf("read invited account: %w", err)
	}
	preview.HasAccount, preview.HasPassword = true, account.HasPassword
	for _, identity := range account.Identities {
		preview.Providers = append(preview.Providers, identity.Provider)
	}
	return preview, nil
}

func (s *WorkspaceService) Accept(userID, sessionToken, token string) (JoinResult, error) {
	invite, err := s.inviteByToken(token)
	if err != nil {
		return JoinResult{}, err
	}
	if inviteStatus(invite, time.Now().Unix()) != domain.InvitePending {
		return JoinResult{}, apperror.InviteUnavailable.New()
	}
	user, err := s.accounts.UserByID(userID)
	if err != nil {
		return JoinResult{}, fmt.Errorf("read invited user: %w", err)
	}
	if !strings.EqualFold(user.Email, invite.Email) {
		return JoinResult{}, apperror.InviteEmailMismatch.New()
	}
	return s.join(userID, sessionToken, invite)
}

type Invitations struct {
	Invites []domain.WorkspaceInvite `json:"invites"`
	// Verified is false when no email is confirmed by Google or GitHub.
	Verified bool `json:"verified"`
}

// verifiedEmails come from linked providers only: password sign-up never verifies the address.
func (s *WorkspaceService) verifiedEmails(userID string) ([]string, error) {
	account, err := s.accounts.Account(userID)
	if err != nil {
		return nil, fmt.Errorf("read account: %w", err)
	}
	emails := []string{}
	for _, identity := range account.Identities {
		email := strings.ToLower(identity.Email)
		if !slices.Contains(emails, email) {
			emails = append(emails, email)
		}
	}
	return emails, nil
}

func (s *WorkspaceService) Invitations(userID string) (Invitations, error) {
	emails, err := s.verifiedEmails(userID)
	if err != nil {
		return Invitations{}, err
	}
	invites, err := s.workspaces.PendingInvitesForUser(userID, emails, time.Now().Unix())
	if err != nil {
		return Invitations{}, err
	}
	return Invitations{Invites: invites, Verified: len(emails) > 0}, nil
}

// invitationFor reports other people's invites as not found, so IDs reveal nothing.
func (s *WorkspaceService) invitationFor(userID, inviteID string) (domain.WorkspaceInvite, error) {
	invite, err := s.workspaces.InviteByID(inviteID)
	if errors.Is(err, repository.ErrNotFound) {
		return invite, apperror.InviteNotFound.New()
	}
	if err != nil {
		return invite, err
	}
	emails, err := s.verifiedEmails(userID)
	if err != nil {
		return invite, err
	}
	if !slices.Contains(emails, strings.ToLower(invite.Email)) {
		return invite, apperror.InviteNotFound.New()
	}
	if inviteStatus(invite, time.Now().Unix()) != domain.InvitePending {
		return invite, apperror.InviteUnavailable.New()
	}
	return invite, nil
}

func (s *WorkspaceService) AcceptInvitation(userID, sessionToken, inviteID string) (JoinResult, error) {
	invite, err := s.invitationFor(userID, inviteID)
	if err != nil {
		return JoinResult{}, err
	}
	return s.join(userID, sessionToken, invite)
}

func (s *WorkspaceService) DeclineInvitation(userID, inviteID string) error {
	invite, err := s.invitationFor(userID, inviteID)
	if err != nil {
		return err
	}
	declined, err := s.workspaces.RevokeInvite(invite.WorkspaceID, invite.ID, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	if !declined {
		return apperror.InviteUnavailable.New()
	}
	return nil
}

func (s *WorkspaceService) join(userID, sessionToken string, invite domain.WorkspaceInvite) (JoinResult, error) {
	now := time.Now()
	joined, err := s.workspaces.AcceptInvite(invite.TokenHash, userID, now.UTC().Format(time.RFC3339), now.Unix())
	if errors.Is(err, repository.ErrConflict) {
		return JoinResult{}, apperror.InviteUnavailable.New()
	}
	if err != nil {
		return JoinResult{}, err
	}
	workspace, err := s.Switch(userID, sessionToken, invite.WorkspaceID)
	if err != nil {
		return JoinResult{}, err
	}
	if !joined {
		return JoinResult{Workspace: workspace}, nil
	}
	return s.provisionMemberWallet(userID, workspace), nil
}

func (s *WorkspaceService) provisionMemberWallet(userID string, workspace domain.Workspace) JoinResult {
	result := JoinResult{Workspace: workspace}
	user, err := s.accounts.UserByID(userID)
	if err != nil {
		logError("read member for wallet workspace="+workspace.ID, err)
		result.WalletError = "Your wallet could not be created. Ask a workspace owner to create one."
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
