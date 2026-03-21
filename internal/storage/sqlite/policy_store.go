package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/policy"
)

type PolicyStore struct {
	db *sql.DB
}

func NewPolicyStore(db *sql.DB) *PolicyStore {
	return &PolicyStore{db: db}
}

func (s *PolicyStore) SaveBundle(bundle *policy.PolicyBundle) error {
	if bundle == nil {
		return fmt.Errorf("cannot save nil bundle")
	}
	if bundle.BundleID == "" {
		bundle.BundleID = "bundle_" + generateShortID()
	}
	if bundle.Version == "" {
		bundle.Version = "1.0.0"
	}
	if bundle.CreatedAt.IsZero() {
		bundle.CreatedAt = time.Now().UTC()
	}

	cg, _ := json.Marshal(orNilSlice(bundle.CapabilityGrants))
	pt, _ := json.Marshal(orNilSlice(bundle.ProtectedTargets))
	ar, _ := json.Marshal(orNilSlice(bundle.ApprovalRules))
	rt, _ := json.Marshal(orNilMap(bundle.RiskThresholds))
	dhr, _ := json.Marshal(orNilSlice(bundle.DataHandlingRules))

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO policy_bundles (bundle_id, name, version, capability_grants, protected_targets, approval_rules, risk_thresholds, data_handling_rules, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?)
		ON CONFLICT (bundle_id) DO UPDATE SET
			name = excluded.name,
			version = excluded.version,
			capability_grants = excluded.capability_grants,
			protected_targets = excluded.protected_targets,
			approval_rules = excluded.approval_rules,
			risk_thresholds = excluded.risk_thresholds,
			data_handling_rules = excluded.data_handling_rules
	`, bundle.BundleID, bundle.Name, bundle.Version, cg, pt, ar, rt, dhr, formatTime(bundle.CreatedAt))
	if err != nil {
		return err
	}

	var count int
	_ = s.db.QueryRowContext(ctx(), `SELECT COUNT(*) FROM policy_bundles WHERE is_active = 1`).Scan(&count)
	if count == 0 {
		_, _ = s.db.ExecContext(ctx(), `UPDATE policy_bundles SET is_active = 1 WHERE bundle_id = ?`, bundle.BundleID)
	}
	return nil
}

func (s *PolicyStore) GetBundle(bundleID string) (*policy.PolicyBundle, error) {
	var b policy.PolicyBundle
	var cg, pt, ar, rt, dhr []byte
	var createdAtStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT bundle_id, name, version, capability_grants, protected_targets, approval_rules, risk_thresholds, data_handling_rules, created_at
		FROM policy_bundles WHERE bundle_id = ?
	`, bundleID).Scan(&b.BundleID, &b.Name, &b.Version, &cg, &pt, &ar, &rt, &dhr, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bundle %q not found", bundleID)
		}
		return nil, err
	}

	_ = json.Unmarshal(cg, &b.CapabilityGrants)
	_ = json.Unmarshal(pt, &b.ProtectedTargets)
	_ = json.Unmarshal(ar, &b.ApprovalRules)
	_ = json.Unmarshal(rt, &b.RiskThresholds)
	_ = json.Unmarshal(dhr, &b.DataHandlingRules)
	b.CreatedAt, _ = parseTime(createdAtStr)
	return &b, nil
}

func (s *PolicyStore) ListBundles() ([]*policy.PolicyBundle, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT bundle_id FROM policy_bundles ORDER BY created_at`)
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

	result := make([]*policy.PolicyBundle, 0, len(ids))
	for _, id := range ids {
		b, err := s.GetBundle(id)
		if err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, nil
}

func (s *PolicyStore) GetActiveBundle() (*policy.PolicyBundle, error) {
	var bundleID string
	err := s.db.QueryRowContext(ctx(), `SELECT bundle_id FROM policy_bundles WHERE is_active = 1 LIMIT 1`).Scan(&bundleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.GetBundle(bundleID)
}

func (s *PolicyStore) SetActiveBundle(bundleID string) error {
	_, err := s.db.ExecContext(ctx(), `UPDATE policy_bundles SET is_active = 0`)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx(), `UPDATE policy_bundles SET is_active = 1 WHERE bundle_id = ?`, bundleID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("bundle %q not found", bundleID)
	}
	return nil
}
