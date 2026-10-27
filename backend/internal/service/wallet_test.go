package service

import (
	"testing"

	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/realtime"
)

func TestWalletAccessFollowsWorkspaceMembership(t *testing.T) {
	f := newWorkspaceFixture(t)
	team := f.teamWorkspace(t)
	if _, err := f.service.Accept("teammate", f.sessions["teammate"], f.invite(t, team.ID, "teammate@example.com")); err != nil {
		t.Fatal(err)
	}
	store := f.store
	if err := store.CreateWallet(domain.Wallet{ID: "team-wallet", UserID: "owner", WorkspaceID: team.ID, Chain: "stellar", Symbol: "XLM", Status: domain.WalletReady}); err != nil {
		t.Fatal(err)
	}
	wallets := &WalletService{store: store, workspaces: f.workspaces, hub: realtime.NewHub()}

	cases := []struct {
		name string
		run  func() error
		code string
	}{
		{"member sees team wallet", func() error { _, err := wallets.Get("teammate", "team-wallet"); return err }, ""},
		{"stranger cannot see team wallet", func() error { _, err := wallets.Get("stranger", "team-wallet"); return err }, "wallet_not_found"},
		{"member cannot create wallets", func() error { _, err := wallets.Create("teammate", "", team.ID, "Extra", "stellar"); return err }, "workspace_owner_required"},
		{"member cannot sign", func() error { _, err := signableWallet(store, "teammate", "team-wallet"); return err }, "workspace_owner_required"},
		{"owner can sign", func() error { _, err := signableWallet(store, "owner", "team-wallet"); return err }, ""},
		{"stranger cannot list a requested workspace", func() error { _, err := wallets.List("stranger", "", team.ID); return err }, "workspace_not_found"},
		{"member cannot delete wallets", func() error { return wallets.Delete("teammate", "team-wallet") }, "workspace_owner_required"},
		{"removed member loses access", func() error {
			if err := f.service.RemoveMember("owner", team.ID, "teammate"); err != nil {
				t.Fatal(err)
			}
			_, err := wallets.Get("teammate", "team-wallet")
			return err
		}, "wallet_not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			assertCode(t, err, tc.code)
		})
	}

	list, err := wallets.List("owner", team.ID, "")
	if err != nil || len(list) != 1 || list[0].ID != "team-wallet" {
		t.Fatalf("team wallets = %#v, %v", list, err)
	}
	// Another tab switched the session to Personal; the requested workspace still wins.
	list, err = wallets.List("owner", domain.PersonalWorkspaceID("owner"), team.ID)
	if err != nil || len(list) != 1 || list[0].ID != "team-wallet" {
		t.Fatalf("requested team wallets = %#v, %v", list, err)
	}
}
