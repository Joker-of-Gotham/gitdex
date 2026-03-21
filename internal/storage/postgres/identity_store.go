package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/identity"
)

type IdentityStore struct {
	pool *pgxpool.Pool
}

func NewIdentityStore(pool *pgxpool.Pool) *IdentityStore {
	return &IdentityStore{pool: pool}
}

func (s *IdentityStore) SaveIdentity(id *identity.AppIdentity) error {
	if id == nil {
		return fmt.Errorf("cannot save nil identity")
	}
	if id.IdentityID == "" {
		id.IdentityID = "id_" + fmt.Sprintf("%x", time.Now().UnixNano())[:8]
	}
	if id.CreatedAt.IsZero() {
		id.CreatedAt = time.Now().UTC()
	}

	capabilitiesJSON, _ := json.Marshal(orNilSlice(id.Capabilities))
	scopeGrantsJSON, _ := json.Marshal(orNilSlice(id.ScopeGrants))

	if id.Capabilities == nil || len(id.Capabilities) == 0 {
		capabilitiesJSON, _ = json.Marshal([]identity.Capability{})
	}
	if id.ScopeGrants == nil || len(id.ScopeGrants) == 0 {
		scopeGrantsJSON, _ = json.Marshal([]identity.ScopeGrant{})
	}

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO identities (identity_id, identity_type, app_id, installation_id, org_scope, repo_scope, capabilities, scope_grants, is_current, created_at)
		VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), $7, $8, COALESCE($9, FALSE), $10)
		ON CONFLICT (identity_id) DO UPDATE SET
			identity_type = EXCLUDED.identity_type,
			app_id = EXCLUDED.app_id,
			installation_id = EXCLUDED.installation_id,
			org_scope = EXCLUDED.org_scope,
			repo_scope = EXCLUDED.repo_scope,
			capabilities = EXCLUDED.capabilities,
			scope_grants = EXCLUDED.scope_grants,
			is_current = EXCLUDED.is_current
	`, id.IdentityID, id.IdentityType, id.AppID, id.InstallationID, id.OrgScope, id.RepoScope, capabilitiesJSON, scopeGrantsJSON, false, id.CreatedAt)
	if err != nil {
		return err
	}

	// If this is the first identity, set as current
	if id.Capabilities != nil && len(id.Capabilities) > 0 {
		var count int
		_ = s.pool.QueryRow(ctx(), `SELECT COUNT(*) FROM identities WHERE is_current = TRUE`).Scan(&count)
		if count == 0 {
			_ = s.SetCurrentIdentity(id.IdentityID)
		}
	}
	return nil
}

func (s *IdentityStore) GetIdentity(identityID string) (*identity.AppIdentity, error) {
	var id identity.AppIdentity
	var appID, installationID, orgScope, repoScope *string
	var capabilitiesJSON, scopeGrantsJSON []byte

	err := s.pool.QueryRow(ctx(), `
		SELECT identity_id, identity_type, app_id, installation_id, org_scope, repo_scope, capabilities, scope_grants, created_at
		FROM identities WHERE identity_id = $1
	`, identityID).Scan(&id.IdentityID, &id.IdentityType, &appID, &installationID, &orgScope, &repoScope, &capabilitiesJSON, &scopeGrantsJSON, &id.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("identity %q not found", identityID)
		}
		return nil, err
	}

	if appID != nil {
		id.AppID = *appID
	}
	if installationID != nil {
		id.InstallationID = *installationID
	}
	if orgScope != nil {
		id.OrgScope = *orgScope
	}
	if repoScope != nil {
		id.RepoScope = *repoScope
	}
	_ = json.Unmarshal(capabilitiesJSON, &id.Capabilities)
	_ = json.Unmarshal(scopeGrantsJSON, &id.ScopeGrants)
	return &id, nil
}

func (s *IdentityStore) ListIdentities() ([]*identity.AppIdentity, error) {
	rows, err := s.pool.Query(ctx(), `SELECT identity_id FROM identities ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*identity.AppIdentity, 0, len(ids))
	for _, identityID := range ids {
		id, err := s.GetIdentity(identityID)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

func (s *IdentityStore) GetCurrentIdentity() (*identity.AppIdentity, error) {
	var identityID string
	err := s.pool.QueryRow(ctx(), `SELECT identity_id FROM identities WHERE is_current = TRUE LIMIT 1`).Scan(&identityID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.GetIdentity(identityID)
}

func (s *IdentityStore) SetCurrentIdentity(identityID string) error {
	_, err := s.pool.Exec(ctx(), `UPDATE identities SET is_current = FALSE`)
	if err != nil {
		return err
	}
	cmd, err := s.pool.Exec(ctx(), `UPDATE identities SET is_current = TRUE WHERE identity_id = $1`, identityID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("identity %q not found", identityID)
	}
	return nil
}
