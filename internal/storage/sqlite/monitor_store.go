package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/autonomy"
)

type MonitorStore struct {
	db *sql.DB
}

func NewMonitorStore(db *sql.DB) *MonitorStore {
	return &MonitorStore{db: db}
}

func (s *MonitorStore) SaveMonitorConfig(cfg *autonomy.MonitorConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil monitor config")
	}

	checksJSON, _ := json.Marshal(orNilSlice(cfg.Checks))

	if cfg.MonitorID == "" {
		cfg.MonitorID = "mon_" + uuid.New().String()[:8]
	}

	enabled := 1
	if !cfg.Enabled {
		enabled = 0
	}

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO monitor_configs (monitor_id, repo_owner, repo_name, interval, checks, enabled)
		VALUES (?, ?, ?, NULLIF(?,''), ?, ?)
		ON CONFLICT (monitor_id) DO UPDATE SET
			repo_owner = excluded.repo_owner,
			repo_name = excluded.repo_name,
			interval = excluded.interval,
			checks = excluded.checks,
			enabled = excluded.enabled
	`, cfg.MonitorID, cfg.RepoOwner, cfg.RepoName, cfg.Interval, checksJSON, enabled)
	return err
}

func (s *MonitorStore) GetMonitorConfig(monitorID string) (*autonomy.MonitorConfig, error) {
	var cfg autonomy.MonitorConfig
	var checksJSON []byte
	var interval sql.NullString
	var enabled int

	err := s.db.QueryRowContext(ctx(), `
		SELECT monitor_id, repo_owner, repo_name, interval, checks, enabled
		FROM monitor_configs WHERE monitor_id = ?
	`, monitorID).Scan(&cfg.MonitorID, &cfg.RepoOwner, &cfg.RepoName, &interval, &checksJSON, &enabled)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("monitor %q not found", monitorID)
		}
		return nil, err
	}
	if interval.Valid {
		cfg.Interval = interval.String
	}
	cfg.Enabled = intToBool(enabled)
	_ = json.Unmarshal(checksJSON, &cfg.Checks)
	return &cfg, nil
}

func (s *MonitorStore) ListMonitorConfigs() ([]*autonomy.MonitorConfig, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT monitor_id FROM monitor_configs ORDER BY monitor_id`)
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

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO monitor_events (event_id, monitor_id, repo_owner, repo_name, check_name, status, message, timestamp)
		VALUES (?, ?, ?, ?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), ?)
	`, ev.EventID, ev.MonitorID, ev.RepoOwner, ev.RepoName, ev.CheckName, ev.Status, ev.Message, formatTime(ev.Timestamp))
	return err
}

func (s *MonitorStore) ListEvents(filter autonomy.MonitorEventFilter) ([]*autonomy.MonitorEvent, error) {
	query := `SELECT event_id, monitor_id, repo_owner, repo_name, check_name, status, message, timestamp FROM monitor_events WHERE 1=1`
	args := []any{}

	if filter.MonitorID != "" {
		query += ` AND monitor_id = ?`
		args = append(args, filter.MonitorID)
	}
	if filter.RepoOwner != "" {
		query += ` AND repo_owner = ?`
		args = append(args, filter.RepoOwner)
	}
	if filter.RepoName != "" {
		query += ` AND repo_name = ?`
		args = append(args, filter.RepoName)
	}
	query += ` ORDER BY timestamp DESC`
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := s.db.QueryContext(ctx(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*autonomy.MonitorEvent
	for rows.Next() {
		var ev autonomy.MonitorEvent
		var checkName, status, message sql.NullString
		var timestampStr string
		if err := rows.Scan(&ev.EventID, &ev.MonitorID, &ev.RepoOwner, &ev.RepoName, &checkName, &status, &message, &timestampStr); err != nil {
			return nil, err
		}
		if checkName.Valid {
			ev.CheckName = checkName.String
		}
		if status.Valid {
			ev.Status = status.String
		}
		if message.Valid {
			ev.Message = message.String
		}
		ev.Timestamp, _ = parseTime(timestampStr)
		result = append(result, &ev)
	}
	return result, rows.Err()
}

func (s *MonitorStore) RemoveMonitorConfig(monitorID string) error {
	res, err := s.db.ExecContext(ctx(), `DELETE FROM monitor_configs WHERE monitor_id = ?`, monitorID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("monitor %q not found", monitorID)
	}
	return nil
}
