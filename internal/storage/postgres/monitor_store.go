package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/autonomy"
)

type MonitorStore struct {
	pool *pgxpool.Pool
}

func NewMonitorStore(pool *pgxpool.Pool) *MonitorStore {
	return &MonitorStore{pool: pool}
}

func (s *MonitorStore) SaveMonitorConfig(cfg *autonomy.MonitorConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil monitor config")
	}

	checksJSON, _ := json.Marshal(orNilSlice(cfg.Checks))

	if cfg.MonitorID == "" {
		cfg.MonitorID = "mon_" + uuid.New().String()[:8]
	}

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO monitor_configs (monitor_id, repo_owner, repo_name, interval, checks, enabled)
		VALUES ($1, $2, $3, NULLIF($4,''), $5, $6)
		ON CONFLICT (monitor_id) DO UPDATE SET
			repo_owner = EXCLUDED.repo_owner,
			repo_name = EXCLUDED.repo_name,
			interval = EXCLUDED.interval,
			checks = EXCLUDED.checks,
			enabled = EXCLUDED.enabled
	`, cfg.MonitorID, cfg.RepoOwner, cfg.RepoName, cfg.Interval, checksJSON, cfg.Enabled)
	return err
}

func (s *MonitorStore) GetMonitorConfig(monitorID string) (*autonomy.MonitorConfig, error) {
	var cfg autonomy.MonitorConfig
	var checksJSON []byte
	var interval *string

	err := s.pool.QueryRow(ctx(), `
		SELECT monitor_id, repo_owner, repo_name, interval, checks, enabled
		FROM monitor_configs WHERE monitor_id = $1
	`, monitorID).Scan(&cfg.MonitorID, &cfg.RepoOwner, &cfg.RepoName, &interval, &checksJSON, &cfg.Enabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("monitor %q not found", monitorID)
		}
		return nil, err
	}
	if interval != nil {
		cfg.Interval = *interval
	}
	_ = json.Unmarshal(checksJSON, &cfg.Checks)
	return &cfg, nil
}

func (s *MonitorStore) ListMonitorConfigs() ([]*autonomy.MonitorConfig, error) {
	rows, err := s.pool.Query(ctx(), `SELECT monitor_id FROM monitor_configs ORDER BY monitor_id`)
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

	result := make([]*autonomy.MonitorConfig, 0, len(ids))
	for _, id := range ids {
		cfg, err := s.GetMonitorConfig(id)
		if err != nil {
			return nil, err
		}
		result = append(result, cfg)
	}
	return result, nil
}

func (s *MonitorStore) AppendEvent(ev *autonomy.MonitorEvent) error {
	if ev == nil {
		return fmt.Errorf("cannot append nil event")
	}
	if ev.EventID == "" {
		ev.EventID = "ev_" + uuid.New().String()[:8]
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO monitor_events (event_id, monitor_id, repo_owner, repo_name, check_name, status, message, timestamp)
		VALUES ($1, $2, $3, $4, NULLIF($5,''), NULLIF($6,''), NULLIF($7,''), $8)
	`, ev.EventID, ev.MonitorID, ev.RepoOwner, ev.RepoName, ev.CheckName, ev.Status, ev.Message, ev.Timestamp)
	return err
}

func (s *MonitorStore) ListEvents(filter autonomy.MonitorEventFilter) ([]*autonomy.MonitorEvent, error) {
	query := `SELECT event_id, monitor_id, repo_owner, repo_name, check_name, status, message, timestamp FROM monitor_events WHERE 1=1`
	args := []interface{}{}
	n := 1

	if filter.MonitorID != "" {
		query += fmt.Sprintf(" AND monitor_id = $%d", n)
		args = append(args, filter.MonitorID)
		n++
	}
	if filter.RepoOwner != "" {
		query += fmt.Sprintf(" AND repo_owner = $%d", n)
		args = append(args, filter.RepoOwner)
		n++
	}
	if filter.RepoName != "" {
		query += fmt.Sprintf(" AND repo_name = $%d", n)
		args = append(args, filter.RepoName)
		n++
	}
	query += " ORDER BY timestamp DESC"
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := s.pool.Query(ctx(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*autonomy.MonitorEvent
	for rows.Next() {
		var ev autonomy.MonitorEvent
		var checkName, status, message *string
		if err := rows.Scan(&ev.EventID, &ev.MonitorID, &ev.RepoOwner, &ev.RepoName, &checkName, &status, &message, &ev.Timestamp); err != nil {
			return nil, err
		}
		if checkName != nil {
			ev.CheckName = *checkName
		}
		if status != nil {
			ev.Status = *status
		}
		if message != nil {
			ev.Message = *message
		}
		result = append(result, &ev)
	}
	return result, rows.Err()
}

func (s *MonitorStore) RemoveMonitorConfig(monitorID string) error {
	cmd, err := s.pool.Exec(ctx(), `DELETE FROM monitor_configs WHERE monitor_id = $1`, monitorID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("monitor %q not found", monitorID)
	}
	return nil
}
