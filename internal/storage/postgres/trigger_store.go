package postgres

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/autonomy"
)

type TriggerStore struct {
	pool *pgxpool.Pool
}

func NewTriggerStore(pool *pgxpool.Pool) *TriggerStore {
	return &TriggerStore{pool: pool}
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

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO trigger_configs (trigger_id, trigger_type, name, source, pattern, action_template, enabled, created_at)
		VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), $7, $8)
		ON CONFLICT (trigger_id) DO UPDATE SET
			trigger_type = EXCLUDED.trigger_type,
			name = EXCLUDED.name,
			source = EXCLUDED.source,
			pattern = EXCLUDED.pattern,
			action_template = EXCLUDED.action_template,
			enabled = EXCLUDED.enabled
	`, cfg.TriggerID, cfg.TriggerType, cfg.Name, cfg.Source, cfg.Pattern, cfg.ActionTemplate, cfg.Enabled, cfg.CreatedAt)
	return err
}

func (s *TriggerStore) GetTrigger(triggerID string) (*autonomy.TriggerConfig, error) {
	var cfg autonomy.TriggerConfig
	var source, pattern, actionTemplate *string

	err := s.pool.QueryRow(ctx(), `
		SELECT trigger_id, trigger_type, name, source, pattern, action_template, enabled, created_at
		FROM trigger_configs WHERE trigger_id = $1
	`, triggerID).Scan(&cfg.TriggerID, &cfg.TriggerType, &cfg.Name, &source, &pattern, &actionTemplate, &cfg.Enabled, &cfg.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("trigger %q not found", triggerID)
		}
		return nil, err
	}
	if source != nil {
		cfg.Source = *source
	}
	if pattern != nil {
		cfg.Pattern = *pattern
	}
	if actionTemplate != nil {
		cfg.ActionTemplate = *actionTemplate
	}
	return &cfg, nil
}

func (s *TriggerStore) ListTriggers() ([]*autonomy.TriggerConfig, error) {
	rows, err := s.pool.Query(ctx(), `SELECT trigger_id FROM trigger_configs ORDER BY created_at`)
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
	cmd, err := s.pool.Exec(ctx(), `UPDATE trigger_configs SET enabled = TRUE WHERE trigger_id = $1`, triggerID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("trigger %q not found", triggerID)
	}
	return nil
}

func (s *TriggerStore) DisableTrigger(triggerID string) error {
	cmd, err := s.pool.Exec(ctx(), `UPDATE trigger_configs SET enabled = FALSE WHERE trigger_id = $1`, triggerID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
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

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO trigger_events (event_id, trigger_id, trigger_type, source_event, resulting_task_id, timestamp)
		VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), NULLIF($5,''), $6)
	`, ev.EventID, ev.TriggerID, ev.TriggerType, ev.SourceEvent, ev.ResultingTaskID, ev.Timestamp)
	return err
}

func (s *TriggerStore) ListTriggerEvents(triggerID string, limit int) ([]*autonomy.TriggerEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx(), `
		SELECT event_id, trigger_id, trigger_type, source_event, resulting_task_id, timestamp
		FROM trigger_events WHERE trigger_id = $1 ORDER BY timestamp DESC LIMIT $2
	`, triggerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*autonomy.TriggerEvent
	for rows.Next() {
		var ev autonomy.TriggerEvent
		var triggerType, sourceEvent, resultingTaskID *string
		if err := rows.Scan(&ev.EventID, &ev.TriggerID, &triggerType, &sourceEvent, &resultingTaskID, &ev.Timestamp); err != nil {
			return nil, err
		}
		if triggerType != nil {
			ev.TriggerType = autonomy.TriggerType(*triggerType)
		}
		if sourceEvent != nil {
			ev.SourceEvent = *sourceEvent
		}
		if resultingTaskID != nil {
			ev.ResultingTaskID = *resultingTaskID
		}
		result = append(result, &ev)
	}
	return result, rows.Err()
}
