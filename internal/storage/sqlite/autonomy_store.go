package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/autonomy"
)

type AutonomyStore struct {
	db *sql.DB
}

func NewAutonomyStore(db *sql.DB) *AutonomyStore {
	return &AutonomyStore{db: db}
}

func (s *AutonomyStore) SaveConfig(cfg *autonomy.AutonomyConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil config")
	}
	if cfg.ConfigID == "" {
		cfg.ConfigID = "cfg_" + generateShortID()
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = time.Now().UTC()
	}

	caJSON, _ := json.Marshal(orNilSlice(cfg.CapabilityAutonomies))

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO autonomy_configs (config_id, name, capability_autonomies, default_level, is_active, created_at)
		VALUES (?, ?, ?, NULLIF(?,''), 0, ?)
		ON CONFLICT (config_id) DO UPDATE SET
			name = excluded.name,
			capability_autonomies = excluded.capability_autonomies,
			default_level = excluded.default_level
	`, cfg.ConfigID, cfg.Name, caJSON, cfg.DefaultLevel, formatTime(cfg.CreatedAt))
	if err != nil {
		return err
	}

	var count int
	_ = s.db.QueryRowContext(ctx(), `SELECT COUNT(*) FROM autonomy_configs WHERE is_active = 1`).Scan(&count)
	if count == 0 {
		_, _ = s.db.ExecContext(ctx(), `UPDATE autonomy_configs SET is_active = 1 WHERE config_id = ?`, cfg.ConfigID)
	}
	return nil
}

func (s *AutonomyStore) GetConfig(configID string) (*autonomy.AutonomyConfig, error) {
	var c autonomy.AutonomyConfig
	var caJSON []byte
	var defaultLevel sql.NullString
	var createdAtStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT config_id, name, capability_autonomies, default_level, created_at
		FROM autonomy_configs WHERE config_id = ?
	`, configID).Scan(&c.ConfigID, &c.Name, &caJSON, &defaultLevel, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("config %q not found", configID)
		}
		return nil, err
	}

	_ = json.Unmarshal(caJSON, &c.CapabilityAutonomies)
	if defaultLevel.Valid {
		c.DefaultLevel = autonomy.AutonomyLevel(defaultLevel.String)
	}
	c.CreatedAt, _ = parseTime(createdAtStr)
	return &c, nil
}

func (s *AutonomyStore) GetActiveConfig() (*autonomy.AutonomyConfig, error) {
	var configID string
	err := s.db.QueryRowContext(ctx(), `SELECT config_id FROM autonomy_configs WHERE is_active = 1 LIMIT 1`).Scan(&configID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.GetConfig(configID)
}

func (s *AutonomyStore) SetActiveConfig(configID string) error {
	_, err := s.db.ExecContext(ctx(), `UPDATE autonomy_configs SET is_active = 0`)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx(), `UPDATE autonomy_configs SET is_active = 1 WHERE config_id = ?`, configID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("config %q not found", configID)
	}
	return nil
}

func (s *AutonomyStore) ListConfigs() ([]*autonomy.AutonomyConfig, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT config_id FROM autonomy_configs ORDER BY created_at`)
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

	result := make([]*autonomy.AutonomyConfig, 0, len(ids))
	for _, id := range ids {
		c, err := s.GetConfig(id)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}
