package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/autonomy"
)

type TriggerStore struct {
	db *sql.DB
}

func NewTriggerStore(db *sql.DB) *TriggerStore {
	return &TriggerStore{db: db}
}

func (s *TriggerStore) SaveTrigger(cfg *autonomy.TriggerConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil trigger config")
	}
	if cfg.TriggerID == "" {
		cfg.TriggerID = "tr_" + uuid.New().String()[:8]
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = time.Now().UTC()
	}

	enabled := boolToInt(cfg.Enabled)

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO trigger_configs (trigger_id, trigger_type, name, source, pattern, action_template, enabled, created_at)
		VALUES (?, ?, ?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), ?, ?)
		ON CONFLICT (trigger_id) DO UPDATE SET
			trigger_type = excluded.trigger_type,
			name = excluded.name,
			source = excluded.source,
			pattern = excluded.pattern,
			action_template = excluded.action_template,
			enabled = excluded.enabled
	`, cfg.TriggerID, cfg.TriggerType, cfg.Name, cfg.Source, cfg.Pattern, cfg.ActionTemplate, enabled, formatTime(cfg.CreatedAt))
	return err
}

func (s *TriggerStore) GetTrigger(triggerID string) (*autonomy.TriggerConfig, error) {
	var cfg autonomy.TriggerConfig
	var source, pattern, actionTemplate sql.NullString
	var createdAtStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT trigger_id, trigger_type, name, source, pattern, action_template, enabled, created_at
		FROM trigger_configs WHERE trigger_id = ?
	`, triggerID).Scan(&cfg.TriggerID, &cfg.TriggerType, &cfg.Name, &source, &pattern, &actionTemplate, &cfg.Enabled, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("trigger %q not found", triggerID)
		}
		return nil, err
	}
	if source.Valid {
		cfg.Source = source.String
	}
	if pattern.Valid {
		cfg.Pattern = pattern.String
	}
	if actionTemplate.Valid {
		cfg.ActionTemplate = actionTemplate.String
	}
	cfg.CreatedAt, _ = parseTime(createdAtStr)
	return &cfg, nil
}

func (s *TriggerStore) ListTriggers() ([]*autonomy.TriggerConfig, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT trigger_id FROM trigger_configs ORDER BY created_at`)
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

	result := make([]*autonomy.TriggerConfig, 0, len(ids))
	for _, id := range ids {
		cfg, err := s.GetTrigger(id)
		if err != nil {
			return nil, err
		}
		result = append(result, cfg)
	}
	return result, nil
}

func (s *TriggerStore) EnableTrigger(triggerID string) error {
	res, err := s.db.ExecContext(ctx(), `UPDATE trigger_configs SET enabled = 1 WHERE trigger_id = ?`, triggerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("trigger %q not found", triggerID)
	}
	return nil
}

func (s *TriggerStore) DisableTrigger(triggerID string) error {
	res, err := s.db.ExecContext(ctx(), `UPDATE trigger_configs SET enabled = 0 WHERE trigger_id = ?`, triggerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("trigger %q not found", triggerID)
	}
	return nil
}

func (s *TriggerStore) AppendTriggerEvent(ev *autonomy.TriggerEvent) error {
	if ev == nil {
		return fmt.Errorf("cannot append nil trigger event")
	}
	if ev.EventID == "" {
		ev.EventID = "tev_" + uuid.New().String()[:8]
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO trigger_events (event_id, trigger_id, trigger_type, source_event, resulting_task_id, timestamp)
		VALUES (?, ?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), ?)
	`, ev.EventID, ev.TriggerID, string(ev.TriggerType), ev.SourceEvent, ev.ResultingTaskID, formatTime(ev.Timestamp))
	return err
}

func (s *TriggerStore) ListTriggerEvents(triggerID string, limit int) ([]*autonomy.TriggerEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx(), `
		SELECT event_id, trigger_id, trigger_type, source_event, resulting_task_id, timestamp
		FROM trigger_events WHERE trigger_id = ? ORDER BY timestamp DESC LIMIT ?
	`, triggerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*autonomy.TriggerEvent
	for rows.Next() {
		var ev autonomy.TriggerEvent
		var triggerType, sourceEvent, resultingTaskID sql.NullString
		var timestampStr string
		if err := rows.Scan(&ev.EventID, &ev.TriggerID, &triggerType, &sourceEvent, &resultingTaskID, &timestampStr); err != nil {
			return nil, err
		}
		if triggerType.Valid {
			ev.TriggerType = autonomy.TriggerType(triggerType.String)
		}
		if sourceEvent.Valid {
			ev.SourceEvent = sourceEvent.String
		}
		if resultingTaskID.Valid {
			ev.ResultingTaskID = resultingTaskID.String
		}
		ev.Timestamp, _ = parseTime(timestampStr)
		result = append(result, &ev)
	}
	return result, rows.Err()
}
