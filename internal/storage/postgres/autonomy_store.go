package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/autonomy"
)

type AutonomyStore struct {
	pool *pgxpool.Pool
}

func NewAutonomyStore(pool *pgxpool.Pool) *AutonomyStore {
	return &AutonomyStore{pool: pool}
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

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO autonomy_configs (config_id, name, capability_autonomies, default_level, is_active, created_at)
		VALUES ($1, $2, $3, NULLIF($4,''), FALSE, $5)
		ON CONFLICT (config_id) DO UPDATE SET
			name = EXCLUDED.name,
			capability_autonomies = EXCLUDED.capability_autonomies,
			default_level = EXCLUDED.default_level
	`, cfg.ConfigID, cfg.Name, caJSON, cfg.DefaultLevel, cfg.CreatedAt)
	if err != nil {
		return err
	}

	var count int
	_ = s.pool.QueryRow(ctx(), `SELECT COUNT(*) FROM autonomy_configs WHERE is_active`).Scan(&count)
	if count == 0 {
		_, _ = s.pool.Exec(ctx(), `UPDATE autonomy_configs SET is_active = TRUE WHERE config_id = $1`, cfg.ConfigID)
	}
	return nil
}

func (s *AutonomyStore) GetConfig(configID string) (*autonomy.AutonomyConfig, error) {
	var c autonomy.AutonomyConfig
	var caJSON []byte

	err := s.pool.QueryRow(ctx(), `
		SELECT config_id, name, capability_autonomies, default_level, created_at
		FROM autonomy_configs WHERE config_id = $1
	`, configID).Scan(&c.ConfigID, &c.Name, &caJSON, &c.DefaultLevel, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("config %q not found", configID)
		}
		return nil, err
	}

	_ = json.Unmarshal(caJSON, &c.CapabilityAutonomies)
	return &c, nil
}

func (s *AutonomyStore) GetActiveConfig() (*autonomy.AutonomyConfig, error) {
	var configID string
	err := s.pool.QueryRow(ctx(), `SELECT config_id FROM autonomy_configs WHERE is_active LIMIT 1`).Scan(&configID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.GetConfig(configID)
}

func (s *AutonomyStore) SetActiveConfig(configID string) error {
	_, err := s.pool.Exec(ctx(), `UPDATE autonomy_configs SET is_active = FALSE`)
	if err != nil {
		return err
	}
	cmd, err := s.pool.Exec(ctx(), `UPDATE autonomy_configs SET is_active = TRUE WHERE config_id = $1`, configID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("config %q not found", configID)
	}
	return nil
}

func (s *AutonomyStore) ListConfigs() ([]*autonomy.AutonomyConfig, error) {
	rows, err := s.pool.Query(ctx(), `SELECT config_id FROM autonomy_configs ORDER BY created_at`)
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
