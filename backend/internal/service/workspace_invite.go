package service

import (
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"

	"github.com/google/uuid"
)

type InviteResult struct {
	Invite    domain.WorkspaceInvite `json:"invite"`
	Link      string                 `json:"link"`
	EmailSent bool                   `json:"emailSent"`
}

type InvitePreview struct {
	InviteID      string              `json:"inviteId"`
	WorkspaceName string              `json:"workspaceName"`
	InvitedBy     string              `json:"invitedBy"`
	Email         string              `json:"email"`
	Status        domain.InviteStatus `json:"status"`
	// Only the token holder sees these; they pick the invite page's sign-in method.
	HasAccount  bool     `json:"hasAccount"`
	HasPassword bool     `json:"hasPassword"`
	Providers   []string `json:"providers"`
}

// Invite always returns the link so owners can share it when email is not configured.
func (s *WorkspaceService) Invite(userID, workspaceID, email string, role domain.WorkspaceRole) (InviteResult, error) {
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
	if role == "" {
		role = domain.RoleMember
	}
	if !role.Valid() {
		return InviteResult{}, apperror.InvalidMemberRole.New()
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
		Role:           role,
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
	if _, err := requireOwner(s.workspaces, workspaceID, userID); err != nil {
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
	preview := InvitePreview{InviteID: invite.ID, WorkspaceName: workspaceName, InvitedBy: invite.InvitedByEmail, Email: invite.Email, Status: inviteStatus(invite, time.Now().Unix()), Providers: []string{}}
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
	err := s.workspaces.AcceptInvite(invite.TokenHash, userID, now.UTC().Format(time.RFC3339), now.Unix())
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
	return JoinResult{Workspace: workspace}, nil
}
