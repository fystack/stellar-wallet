package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository/sqlite"
)

type provisionerStub struct {
	err   error
	calls []string
}

func (p *provisionerStub) Provision(userID, workspaceID, name, _ string) (domain.Wallet, error) {
	p.calls = append(p.calls, userID+"@"+workspaceID)
	return domain.Wallet{ID: "wallet-" + userID, UserID: userID, WorkspaceID: workspaceID, Name: name}, p.err
}

type mailerStub struct {
	err  error
	sent []string
}

func (m *mailerStub) Send(to, _, body string) error {
	m.sent = append(m.sent, to+"\n"+body)
	return m.err
}

type workspaceFixture struct {
	service     *WorkspaceService
	workspaces  *sqlite.WorkspaceRepository
	accounts    *sqlite.AuthRepository
	store       *sqlite.Store
	provisioner *provisionerStub
	mailer      *mailerStub
	sessions    map[string]string
}

func newWorkspaceFixture(t *testing.T) *workspaceFixture {
	t.Helper()
	db, err := sqlite.Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	fixture := &workspaceFixture{
		workspaces:  sqlite.NewWorkspaceRepository(db),
		accounts:    sqlite.NewAuthRepository(db),
		store:       sqlite.NewStore(db),
		provisioner: &provisionerStub{},
		mailer:      &mailerStub{},
		sessions:    map[string]string{},
	}
	fixture.service = NewWorkspaceService(fixture.workspaces, fixture.accounts, fixture.provisioner, fixture.mailer, "http://localhost:5173/")
	for _, id := range []string{"owner", "teammate", "stranger"} {
		if err := fixture.accounts.CreateUser(domain.User{ID: id, Email: id + "@example.com", CreatedAt: "2026-10-01T00:00:00Z"}); err != nil {
			t.Fatal(err)
		}
		token := "session-" + id
		now := time.Now()
		if err := fixture.accounts.CreateSession(domain.Session{Hash: auth.HashToken(token), UserID: id, CreatedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix()}); err != nil {
			t.Fatal(err)
		}
		fixture.sessions[id] = token
	}
	return fixture
}

func (f *workspaceFixture) teamWorkspace(t *testing.T) domain.Workspace {
	t.Helper()
	result, err := f.service.Create("owner", f.sessions["owner"], "Treasury")
	if err != nil {
		t.Fatal(err)
	}
	return result.Workspace
}

func (f *workspaceFixture) invite(t *testing.T, workspaceID, email string) string {
	t.Helper()
	result, err := f.service.Invite("owner", workspaceID, email)
	if err != nil {
		t.Fatal(err)
	}
	_, token, found := strings.Cut(result.Link, "/#/invite/")
	if !found || token == "" {
		t.Fatalf("invite link = %q", result.Link)
	}
	return token
}

func (f *workspaceFixture) sessionWorkspace(t *testing.T, userID string) string {
	t.Helper()
	session, err := f.accounts.Session(auth.HashToken(f.sessions[userID]))
	if err != nil {
		t.Fatal(err)
	}
	return session.WorkspaceID
}

func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatalf("error = %v, want %s", err, code)
	}
}

func TestWorkspaceInviteAndAcceptProvisionsMemberWallet(t *testing.T) {
	f := newWorkspaceFixture(t)
	workspace := f.teamWorkspace(t)
	if workspace.Role != domain.RoleOwner || f.sessionWorkspace(t, "owner") != workspace.ID {
		t.Fatalf("created workspace = %#v, session workspace = %q", workspace, f.sessionWorkspace(t, "owner"))
	}

	result, err := f.service.Invite("owner", workspace.ID, " Teammate@Example.com ")
	if err != nil {
		t.Fatal(err)
	}
	if !result.EmailSent || result.Invite.Email != "teammate@example.com" || !strings.HasPrefix(result.Link, "http://localhost:5173/#/invite/") {
		t.Fatalf("invite result = %#v", result)
	}
	if len(f.mailer.sent) != 1 || !strings.Contains(f.mailer.sent[0], result.Link) {
		t.Fatalf("mail = %#v", f.mailer.sent)
	}
	_, token, _ := strings.Cut(result.Link, "/#/invite/")

	preview, err := f.service.PreviewInvite(token)
	if err != nil || preview.Status != domain.InvitePending || preview.WorkspaceName != "Treasury" || preview.InvitedBy != "owner@example.com" || !preview.HasAccount {
		t.Fatalf("preview = %#v, %v", preview, err)
	}
	newcomer, err := f.service.PreviewInvite(f.invite(t, workspace.ID, "newcomer@example.com"))
	if err != nil || newcomer.HasAccount || newcomer.HasPassword || len(newcomer.Providers) != 0 {
		t.Fatalf("preview for an email without an account = %#v, %v", newcomer, err)
	}
	social, err := f.accounts.SocialUser(domain.Identity{Provider: "google", Subject: "google-social", Email: "social@example.com"}, "")
	if err != nil {
		t.Fatal(err)
	}
	socialPreview, err := f.service.PreviewInvite(f.invite(t, workspace.ID, social.Email))
	if err != nil || !socialPreview.HasAccount || socialPreview.HasPassword || len(socialPreview.Providers) != 1 || socialPreview.Providers[0] != "google" {
		t.Fatalf("preview for a Google-only account = %#v, %v", socialPreview, err)
	}

	joined, err := f.service.Accept("teammate", f.sessions["teammate"], token)
	if err != nil {
		t.Fatal(err)
	}
	if joined.Workspace.ID != workspace.ID || joined.Workspace.Role != domain.RoleMember || joined.Wallet == nil || joined.Wallet.Name != "teammate's wallet" {
		t.Fatalf("join result = %#v", joined)
	}
	if f.sessionWorkspace(t, "teammate") != workspace.ID {
		t.Fatal("session was not switched to the joined workspace")
	}
	wantCalls := []string{"owner@" + workspace.ID, "teammate@" + workspace.ID}
	if strings.Join(f.provisioner.calls, ",") != strings.Join(wantCalls, ",") {
		t.Fatalf("provisioned = %v, want %v", f.provisioner.calls, wantCalls)
	}

	members, err := f.service.Members("teammate", workspace.ID)
	if err != nil || len(members.Members) != 2 || len(members.Invites) != 2 {
		t.Fatalf("members = %#v, %v", members, err)
	}
	if preview, _ := f.service.PreviewInvite(token); preview.Status != domain.InviteAccepted {
		t.Fatalf("preview after accept = %#v", preview)
	}
	_, err = f.service.Accept("teammate", f.sessions["teammate"], token)
	assertCode(t, err, "invite_unavailable")
}

func TestWorkspaceInviteRejections(t *testing.T) {
	cases := []struct {
		name   string
		userID string
		target func(f *workspaceFixture, team domain.Workspace) string
		email  string
		code   string
	}{
		{"member cannot invite", "teammate", func(_ *workspaceFixture, team domain.Workspace) string { return team.ID }, "new@example.com", "workspace_owner_required"},
		{"personal workspace", "owner", func(_ *workspaceFixture, _ domain.Workspace) string { return domain.PersonalWorkspaceID("owner") }, "new@example.com", "personal_workspace"},
		{"invalid email", "owner", func(_ *workspaceFixture, team domain.Workspace) string { return team.ID }, "not-an-email", "invalid_request"},
		{"already a member", "owner", func(_ *workspaceFixture, team domain.Workspace) string { return team.ID }, "teammate@example.com", "already_member"},
		{"not a member", "stranger", func(_ *workspaceFixture, team domain.Workspace) string { return team.ID }, "new@example.com", "workspace_not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			team := f.teamWorkspace(t)
			if _, err := f.service.Accept("teammate", f.sessions["teammate"], f.invite(t, team.ID, "teammate@example.com")); err != nil {
				t.Fatal(err)
			}
			if _, err := f.service.List("owner", ""); err != nil {
				t.Fatal(err)
			}
			sentBefore := len(f.mailer.sent)
			_, err := f.service.Invite(tc.userID, tc.target(f, team), tc.email)
			assertCode(t, err, tc.code)
			if len(f.mailer.sent) != sentBefore {
				t.Fatal("rejected invite sent an email")
			}
		})
	}
}

func TestWorkspaceAcceptRejections(t *testing.T) {
	cases := []struct {
		name  string
		token func(f *workspaceFixture, team domain.Workspace) string
		code  string
	}{
		{"unknown token", func(_ *workspaceFixture, _ domain.Workspace) string { return "missing" }, "invite_not_found"},
		{"different email", func(f *workspaceFixture, team domain.Workspace) string {
			return f.invite(t, team.ID, "someone-else@example.com")
		}, "invite_email_mismatch"},
		{"revoked", func(f *workspaceFixture, team domain.Workspace) string {
			token := f.invite(t, team.ID, "teammate@example.com")
			invites, _ := f.service.Members("owner", team.ID)
			if err := f.service.RevokeInvite("owner", team.ID, invites.Invites[0].ID); err != nil {
				t.Fatal(err)
			}
			return token
		}, "invite_unavailable"},
		{"replaced by a newer invite", func(f *workspaceFixture, team domain.Workspace) string {
			token := f.invite(t, team.ID, "teammate@example.com")
			f.invite(t, team.ID, "teammate@example.com")
			return token
		}, "invite_unavailable"},
		{"expired", func(f *workspaceFixture, team domain.Workspace) string {
			expired := domain.WorkspaceInvite{ID: "expired", WorkspaceID: team.ID, Email: "teammate@example.com", TokenHash: auth.HashToken("expired-token"), InvitedBy: "owner", CreatedAt: "2026-01-01T00:00:00Z", ExpiresAt: time.Now().Add(-time.Minute).Unix()}
			if err := f.workspaces.CreateInvite(expired, time.Now().Unix()); err != nil {
				t.Fatal(err)
			}
			return "expired-token"
		}, "invite_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			team := f.teamWorkspace(t)
			token := tc.token(f, team)
			provisionedBefore := len(f.provisioner.calls)
			_, err := f.service.Accept("teammate", f.sessions["teammate"], token)
			assertCode(t, err, tc.code)
			if len(f.provisioner.calls) != provisionedBefore {
				t.Fatal("rejected accept provisioned a wallet")
			}
			if _, err := f.service.Members("teammate", team.ID); err == nil {
				t.Fatal("rejected accept added a member")
			}
		})
	}
}

func TestWorkspaceJoinSucceedsWhenWalletProvisioningFails(t *testing.T) {
	f := newWorkspaceFixture(t)
	team := f.teamWorkspace(t)
	f.provisioner.err = apperror.MPCUnavailable.New()
	joined, err := f.service.Accept("teammate", f.sessions["teammate"], f.invite(t, team.ID, "teammate@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if joined.Wallet != nil || joined.WalletError != apperror.MPCUnavailable.New().Message {
		t.Fatalf("join result = %#v", joined)
	}
	if _, err := f.service.Members("teammate", team.ID); err != nil {
		t.Fatalf("member was not added: %v", err)
	}
}

func TestWorkspaceInviteWithoutMailerReturnsLink(t *testing.T) {
	f := newWorkspaceFixture(t)
	f.service.mailer = nil
	team := f.teamWorkspace(t)
	result, err := f.service.Invite("owner", team.ID, "teammate@example.com")
	if err != nil || result.EmailSent || result.Link == "" {
		t.Fatalf("result = %#v, %v", result, err)
	}
}

func TestWorkspaceMembershipManagement(t *testing.T) {
	cases := []struct {
		name string
		run  func(f *workspaceFixture, team domain.Workspace) error
		code string
	}{
		{"owner removes member", func(f *workspaceFixture, team domain.Workspace) error {
			return f.service.RemoveMember("owner", team.ID, "teammate")
		}, ""},
		{"last owner cannot be removed", func(f *workspaceFixture, team domain.Workspace) error {
			return f.service.RemoveMember("owner", team.ID, "owner")
		}, "last_owner"},
		{"member cannot remove", func(f *workspaceFixture, team domain.Workspace) error {
			return f.service.RemoveMember("teammate", team.ID, "owner")
		}, "workspace_owner_required"},
		{"unknown member", func(f *workspaceFixture, team domain.Workspace) error {
			return f.service.RemoveMember("owner", team.ID, "stranger")
		}, "member_not_found"},
		{"unknown invite", func(f *workspaceFixture, team domain.Workspace) error {
			return f.service.RevokeInvite("owner", team.ID, "missing")
		}, "invite_not_found"},
		{"switch to foreign workspace", func(f *workspaceFixture, team domain.Workspace) error {
			_, err := f.service.Switch("stranger", f.sessions["stranger"], team.ID)
			return err
		}, "workspace_not_found"},
		{"blank workspace name", func(f *workspaceFixture, _ domain.Workspace) error {
			_, err := f.service.Create("owner", f.sessions["owner"], "  ")
			return err
		}, "invalid_request"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			team := f.teamWorkspace(t)
			if _, err := f.service.Accept("teammate", f.sessions["teammate"], f.invite(t, team.ID, "teammate@example.com")); err != nil {
				t.Fatal(err)
			}
			err := tc.run(f, team)
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			assertCode(t, err, tc.code)
		})
	}
}

func TestActiveWorkspaceFallsBackToPersonal(t *testing.T) {
	f := newWorkspaceFixture(t)
	team := f.teamWorkspace(t)
	list, err := f.service.List("owner", team.ID)
	if err != nil || list.ActiveID != team.ID || !list.Selected || len(list.Workspaces) != 2 {
		t.Fatalf("list = %#v, %v", list, err)
	}
	list, err = f.service.List("stranger", team.ID)
	if err != nil || list.ActiveID != domain.PersonalWorkspaceID("stranger") || list.Selected || len(list.Workspaces) != 1 || !list.Workspaces[0].Personal {
		t.Fatalf("stranger list = %#v, %v", list, err)
	}
}

func (f *workspaceFixture) verify(t *testing.T, userID, email string) {
	t.Helper()
	if _, err := f.accounts.SocialUser(domain.Identity{Provider: "google", Subject: "google-" + userID, Email: email}, userID); err != nil {
		t.Fatal(err)
	}
}

func TestInvitationsAreListedForVerifiedEmailsOnly(t *testing.T) {
	f := newWorkspaceFixture(t)
	team := f.teamWorkspace(t)
	f.invite(t, team.ID, "teammate@example.com")
	f.invite(t, team.ID, "teammate.personal@gmail.com")

	unverified, err := f.service.Invitations("teammate")
	if err != nil || unverified.Verified || len(unverified.Invites) != 0 {
		t.Fatalf("unverified invitations = %#v, %v", unverified, err)
	}

	f.verify(t, "teammate", "teammate.personal@gmail.com")
	verified, err := f.service.Invitations("teammate")
	if err != nil || !verified.Verified || len(verified.Invites) != 1 {
		t.Fatalf("verified invitations = %#v, %v", verified, err)
	}
	invite := verified.Invites[0]
	if invite.Email != "teammate.personal@gmail.com" || invite.WorkspaceName != "Treasury" || invite.InvitedByEmail != "owner@example.com" {
		t.Fatalf("listed invite = %#v", invite)
	}

	joined, err := f.service.AcceptInvitation("teammate", f.sessions["teammate"], invite.ID)
	if err != nil || joined.Workspace.ID != team.ID || joined.Wallet == nil {
		t.Fatalf("accept = %#v, %v", joined, err)
	}
	if f.sessionWorkspace(t, "teammate") != team.ID {
		t.Fatal("session was not switched to the joined workspace")
	}
	if after, _ := f.service.Invitations("teammate"); len(after.Invites) != 0 {
		t.Fatalf("invitations after joining = %#v", after)
	}
}

func TestInvitationActionRejections(t *testing.T) {
	cases := []struct {
		name string
		run  func(f *workspaceFixture, inviteID string) error
		code string
	}{
		{"decline", func(f *workspaceFixture, inviteID string) error {
			return f.service.DeclineInvitation("teammate", inviteID)
		}, ""},
		{"accept after decline", func(f *workspaceFixture, inviteID string) error {
			if err := f.service.DeclineInvitation("teammate", inviteID); err != nil {
				t.Fatal(err)
			}
			_, err := f.service.AcceptInvitation("teammate", f.sessions["teammate"], inviteID)
			return err
		}, "invite_unavailable"},
		{"someone else's invite", func(f *workspaceFixture, inviteID string) error {
			f.verify(t, "stranger", "stranger@example.com")
			_, err := f.service.AcceptInvitation("stranger", f.sessions["stranger"], inviteID)
			return err
		}, "invite_not_found"},
		{"someone else declines", func(f *workspaceFixture, inviteID string) error {
			f.verify(t, "stranger", "stranger@example.com")
			return f.service.DeclineInvitation("stranger", inviteID)
		}, "invite_not_found"},
		{"revoked by the owner", func(f *workspaceFixture, inviteID string) error {
			if err := f.service.RevokeInvite("owner", f.mustTeam(t), inviteID); err != nil {
				t.Fatal(err)
			}
			_, err := f.service.AcceptInvitation("teammate", f.sessions["teammate"], inviteID)
			return err
		}, "invite_unavailable"},
		{"unknown invite", func(f *workspaceFixture, _ string) error {
			_, err := f.service.AcceptInvitation("teammate", f.sessions["teammate"], "missing")
			return err
		}, "invite_not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			team := f.teamWorkspace(t)
			f.invite(t, team.ID, "teammate@example.com")
			f.verify(t, "teammate", "teammate@example.com")
			listed, err := f.service.Invitations("teammate")
			if err != nil || len(listed.Invites) != 1 {
				t.Fatalf("invitations = %#v, %v", listed, err)
			}
			provisionedBefore := len(f.provisioner.calls)
			err = tc.run(f, listed.Invites[0].ID)
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
				if after, _ := f.service.Invitations("teammate"); len(after.Invites) != 0 {
					t.Fatalf("declined invite still listed: %#v", after)
				}
				if pending, _ := f.service.Members("owner", team.ID); len(pending.Invites) != 0 {
					t.Fatalf("declined invite still pending for the owner: %#v", pending.Invites)
				}
				return
			}
			assertCode(t, err, tc.code)
			if len(f.provisioner.calls) != provisionedBefore {
				t.Fatal("rejected invitation provisioned a wallet")
			}
		})
	}
}

func (f *workspaceFixture) mustTeam(t *testing.T) string {
	t.Helper()
	list, err := f.service.List("owner", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, workspace := range list.Workspaces {
		if !workspace.Personal {
			return workspace.ID
		}
	}
	t.Fatal("owner has no team workspace")
	return ""
}
