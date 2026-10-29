package sqlite

import (
	"database/sql"
	"fmt"

	"stellar-wallet-backend/internal/domain"
)

const testnetMarketMaker = "GAHPYWLK6YRN7CVYZOO4H3VDRZ7PVF5UJGLZCSPAEIKJE2XSWF5LAGER"

var defaultAssets = []domain.CustomAsset{
	{Code: "USDC", Issuer: "GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5"},
	{Code: "USDT", Issuer: testnetMarketMaker},
	{Code: "BTC", Issuer: testnetMarketMaker},
	{Code: "ETH", Issuer: testnetMarketMaker},
}

func seedWorkspaceAssets(tx *sql.Tx, workspaceID string) error {
	for _, asset := range defaultAssets {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO workspace_assets (workspace_id, code, issuer) VALUES (?, ?, ?)`,
			workspaceID, asset.Code, asset.Issuer); err != nil {
			return fmt.Errorf("seed workspace assets: %w", err)
		}
	}
	return nil
}

func (s *WorkspaceRepository) WorkspaceAssets(workspaceID string) ([]domain.CustomAsset, error) {
	rows, err := s.db.Query(`SELECT code, issuer FROM workspace_assets WHERE workspace_id = ? ORDER BY code`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace assets: %w", err)
	}
	defer rows.Close()
	assets := []domain.CustomAsset{}
	for rows.Next() {
		var asset domain.CustomAsset
		if err := rows.Scan(&asset.Code, &asset.Issuer); err != nil {
			return nil, fmt.Errorf("scan workspace asset: %w", err)
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace assets: %w", err)
	}
	return assets, nil
}

func (s *WorkspaceRepository) AddWorkspaceAsset(workspaceID string, asset domain.CustomAsset) error {
	return exec(s.db, "add workspace asset",
		`INSERT OR IGNORE INTO workspace_assets (workspace_id, code, issuer) VALUES (?, ?, ?)`, workspaceID, asset.Code, asset.Issuer)
}

func (s *WorkspaceRepository) RemoveWorkspaceAsset(workspaceID, code, issuer string) error {
	return exec(s.db, "remove workspace asset",
		`DELETE FROM workspace_assets WHERE workspace_id = ? AND code = ? AND issuer = ?`, workspaceID, code, issuer)
}
