package service

import (
	"testing"

	"stellar-wallet-backend/internal/domain"
)

func TestWorkspaceAssets(t *testing.T) {
	euro := domain.CustomAsset{Code: "EURC", Issuer: "GEURISSUER"}
	cases := []struct {
		name     string
		userID   string
		change   func(f *workspaceFixture, userID, workspaceID string) ([]domain.CustomAsset, error)
		wantCode string
		wantLen  int
	}{
		{"member reads the defaults", "teammate", func(f *workspaceFixture, userID, workspaceID string) ([]domain.CustomAsset, error) {
			return f.service.Assets(userID, workspaceID)
		}, "", len(defaultTestAssets)},
		{"owner adds an asset", "owner", func(f *workspaceFixture, userID, workspaceID string) ([]domain.CustomAsset, error) {
			return f.service.AddAsset(userID, workspaceID, euro)
		}, "", len(defaultTestAssets) + 1},
		{"owner removes an asset", "owner", func(f *workspaceFixture, userID, workspaceID string) ([]domain.CustomAsset, error) {
			return f.service.RemoveAsset(userID, workspaceID, "USDC", "GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5")
		}, "", len(defaultTestAssets) - 1},
		{"member cannot add", "teammate", func(f *workspaceFixture, userID, workspaceID string) ([]domain.CustomAsset, error) {
			return f.service.AddAsset(userID, workspaceID, euro)
		}, "workspace_owner_required", 0},
		{"member cannot remove", "teammate", func(f *workspaceFixture, userID, workspaceID string) ([]domain.CustomAsset, error) {
			return f.service.RemoveAsset(userID, workspaceID, "USDC", "GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5")
		}, "workspace_owner_required", 0},
		{"asset needs code and issuer", "owner", func(f *workspaceFixture, userID, workspaceID string) ([]domain.CustomAsset, error) {
			return f.service.AddAsset(userID, workspaceID, domain.CustomAsset{Code: " ", Issuer: "GISSUER"})
		}, "invalid_asset", 0},
		{"outsider cannot read", "stranger", func(f *workspaceFixture, userID, workspaceID string) ([]domain.CustomAsset, error) {
			return f.service.Assets(userID, workspaceID)
		}, "workspace_not_found", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWorkspaceFixture(t)
			team := f.teamWorkspace(t)
			if _, err := f.service.Accept("teammate", f.sessions["teammate"], f.invite(t, team.ID, "teammate@example.com")); err != nil {
				t.Fatal(err)
			}
			assets, err := tc.change(f, tc.userID, team.ID)
			if tc.wantCode != "" {
				assertCode(t, err, tc.wantCode)
				return
			}
			if err != nil || len(assets) != tc.wantLen {
				t.Fatalf("assets = %#v, %v", assets, err)
			}
			personal, err := f.service.Assets("owner", domain.PersonalWorkspaceID("owner"))
			if err != nil || len(personal) != len(defaultTestAssets) {
				t.Fatalf("personal workspace assets changed = %#v, %v", personal, err)
			}
		})
	}
}

var defaultTestAssets = []string{"BTC", "ETH", "USDC", "USDT"}
