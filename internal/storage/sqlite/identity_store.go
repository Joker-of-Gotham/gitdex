package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/identity"
)

type IdentityStore struct {
	db *sql.DB
}

func NewIdentityStore(db *sql.DB) *IdentityStore {
	return &IdentityStore{db: db}
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

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO identities (identity_id, identity_type, app_id, installation_id, org_scope, repo_scope, capabilities, scope_grants, is_current, created_at)
		VALUES (?, ?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), ?, ?, 0, ?)
		ON CONFLICT (identity_id) DO UPDATE SET
			identity_type = excluded.identity_type,
			app_id = excluded.app_id,
			installation_id = excluded.installation_id,
			org_scope = excluded.org_scope,
			repo_scope = excluded.repo_scope,
			capabilities = excluded.capabilities,
			scope_grants = excluded.scope_grants,
			is_current = excluded.is_current
	`, id.IdentityID, id.IdentityType, id.AppID, id.InstallationID, id.OrgScope, id.RepoScope, capabilitiesJSON, scopeGrantsJSON, formatTime(id.CreatedAt))
	if err != nil {
		return err
	}

	if id.Capabilities != nil && len(id.Capabilities) > 0 {
		var count int
		_ = s.db.QueryRowContext(ctx(), `SELECT COUNT(*) FROM identities WHERE is_current = 1`).Scan(&count)
		if count == 0 {
			_ = s.SetCurrentIdentity(id.IdentityID)
		}
	}
	return nil
}

func (s *IdentityStore) GetIdentity(identityID string) (*identity.AppIdentity, error) {
	var id identity.AppIdentity
	var appID, installationID, orgScope, repoScope sql.NullString
	var capabilitiesJSON, scopeGrantsJSON []byte
	var createdAtStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT identity_id, identity_type, app_id, installation_id, org_scope, repo_scope, capabilities, scope_grants, created_at
		FROM identities WHERE identity_id = ?
	`, identityID).Scan(&id.IdentityID, &id.IdentityType, &appID, &installationID, &orgScope, &repoScope, &capabilitiesJSON, &scopeGrantsJSON, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("identity %q not found", identityID)
		}
		return nil, err
	}

	if appID.Valid {
		id.AppID = appID.String
	}
	if installationID.Valid {
		id.InstallationID = installationID.String
	}
	if orgScope.Valid {
		id.OrgScope = orgScope.String
	}
	if repoScope.Valid {
		id.RepoScope = repoScope.String
	}
	_ = json.Unmarshal(capabilitiesJSON, &id.Capabilities)
	_ = json.Unmarshal(scopeGrantsJSON, &id.ScopeGrants)
	id.CreatedAt, _ = parseTime(createdAtStr)
	return &id, nil
}

func (s *IdentityStore) ListIdentities() ([]*identity.AppIdentity, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT identity_id FROM identities ORDER BY created_at`)
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
	err := s.db.QueryRowContext(ctx(), `SELECT identity_id FROM identities WHERE is_current = 1 LIMIT 1`).Scan(&identityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.GetIdentity(identityID)
}

func (s *IdentityStore) SetCurrentIdentity(identityID string) error {
	_, err := s.db.ExecContext(ctx(), `UPDATE identities SET is_current = 0`)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx(), `UPDATE identities SET is_current = 1 WHERE identity_id = ?`, identityID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("identity %q not found", identityID)
	}
	return nil
}
