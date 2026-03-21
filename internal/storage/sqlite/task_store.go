package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/orchestrator"
)

type TaskStore struct {
	db *sql.DB
}

func NewTaskStore(db *sql.DB) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) SaveTask(task *orchestrator.Task) error {
	if task == nil {
		return fmt.Errorf("cannot save nil task")
	}
	if task.TaskID == "" {
		task.TaskID = orchestrator.GenerateTaskID()
	}
	task.UpdatedAt = time.Now().UTC()

	stepsJSON, _ := json.Marshal(orNilSlice(task.Steps))

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO tasks (task_id, correlation_id, plan_id, status, current_step, steps, result, updated_at)
		VALUES (?, NULLIF(?,''), NULLIF(?,''), ?, ?, ?, '{}', ?)
		ON CONFLICT (task_id) DO UPDATE SET
			correlation_id = excluded.correlation_id,
			plan_id = excluded.plan_id,
			status = excluded.status,
			current_step = excluded.current_step,
			steps = excluded.steps,
			updated_at = excluded.updated_at
	`, task.TaskID, task.CorrelationID, task.PlanID, task.Status, task.CurrentStep, stepsJSON, task.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *TaskStore) GetTask(taskID string) (*orchestrator.Task, error) {
	var t orchestrator.Task
	var corrID, planID sql.NullString
	var stepsJSON []byte
	var updatedAtStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT task_id, correlation_id, plan_id, status, current_step, steps, updated_at
		FROM tasks WHERE task_id = ?
	`, taskID).Scan(&t.TaskID, &corrID, &planID, &t.Status, &t.CurrentStep, &stepsJSON, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task %q not found", taskID)
		}
		return nil, err
	}

	if corrID.Valid {
		t.CorrelationID = corrID.String
	}
	if planID.Valid {
		t.PlanID = planID.String
	}
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	_ = json.Unmarshal(stepsJSON, &t.Steps)
	return &t, nil
}

func (s *TaskStore) GetByCorrelationID(corrID string) (*orchestrator.Task, error) {
	var taskID string
	err := s.db.QueryRowContext(ctx(), `SELECT task_id FROM tasks WHERE correlation_id = ?`, corrID).Scan(&taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no task found for correlation %q", corrID)
		}
		return nil, err
	}
	return s.GetTask(taskID)
}

func (s *TaskStore) ListTasks() ([]*orchestrator.Task, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT task_id FROM tasks ORDER BY updated_at`)
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

	result := make([]*orchestrator.Task, 0, len(ids))
	for _, id := range ids {
		t, err := s.GetTask(id)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

func (s *TaskStore) UpdateTask(task *orchestrator.Task) error {
	if task == nil {
		return fmt.Errorf("cannot update nil task")
	}
	task.UpdatedAt = time.Now().UTC()

	stepsJSON, _ := json.Marshal(orNilSlice(task.Steps))

	res, err := s.db.ExecContext(ctx(), `
		UPDATE tasks SET correlation_id = NULLIF(?,''), plan_id = NULLIF(?,''), status = ?, current_step = ?, steps = ?, updated_at = ?
		WHERE task_id = ?
	`, task.CorrelationID, task.PlanID, task.Status, task.CurrentStep, stepsJSON, task.UpdatedAt.UTC().Format(time.RFC3339), task.TaskID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task %q not found", task.TaskID)
	}
	return nil
}

func (s *TaskStore) AppendEvent(event *orchestrator.TaskEvent) error {
	if event == nil {
		return fmt.Errorf("cannot append nil event")
	}
	if event.EventID == "" {
		event.EventID = orchestrator.GenerateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO task_events (event_id, task_id, from_status, to_status, step_index, detail, timestamp)
		VALUES (?, ?, ?, ?, ?, NULLIF(?,''), ?)
	`, event.EventID, event.TaskID, event.FromStatus, event.ToStatus, event.StepSequence, event.Message, event.Timestamp.UTC().Format(time.RFC3339))
	return err
}

func (s *TaskStore) GetEvents(taskID string) ([]*orchestrator.TaskEvent, error) {
	rows, err := s.db.QueryContext(ctx(), `
		SELECT event_id, task_id, from_status, to_status, step_index, detail, timestamp
		FROM task_events WHERE task_id = ? ORDER BY timestamp
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*orchestrator.TaskEvent
	for rows.Next() {
		var e orchestrator.TaskEvent
		var fromStatus, toStatus, detail sql.NullString
		var timestampStr string
		if err := rows.Scan(&e.EventID, &e.TaskID, &fromStatus, &toStatus, &e.StepSequence, &detail, &timestampStr); err != nil {
			return nil, err
		}
		if fromStatus.Valid {
			e.FromStatus = orchestrator.TaskStatus(fromStatus.String)
		}
		if toStatus.Valid {
			e.ToStatus = orchestrator.TaskStatus(toStatus.String)
		}
		if detail.Valid {
			e.Message = detail.String
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		result = append(result, &e)
	}
	return result, rows.Err()
}
