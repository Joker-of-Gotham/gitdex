package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/orchestrator"
)

type TaskStore struct {
	pool *pgxpool.Pool
}

func NewTaskStore(pool *pgxpool.Pool) *TaskStore {
	return &TaskStore{pool: pool}
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

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO tasks (task_id, correlation_id, plan_id, status, current_step, steps, result, updated_at)
		VALUES ($1, NULLIF($2,''), NULLIF($3,''), $4, $5, $6, '{}', $7)
		ON CONFLICT (task_id) DO UPDATE SET
			correlation_id = EXCLUDED.correlation_id,
			plan_id = EXCLUDED.plan_id,
			status = EXCLUDED.status,
			current_step = EXCLUDED.current_step,
			steps = EXCLUDED.steps,
			updated_at = EXCLUDED.updated_at
	`, task.TaskID, task.CorrelationID, task.PlanID, task.Status, task.CurrentStep, stepsJSON, task.UpdatedAt)
	return err
}

func (s *TaskStore) GetTask(taskID string) (*orchestrator.Task, error) {
	var t orchestrator.Task
	var corrID, planID *string
	var stepsJSON []byte

	err := s.pool.QueryRow(ctx(), `
		SELECT task_id, correlation_id, plan_id, status, current_step, steps, updated_at
		FROM tasks WHERE task_id = $1
	`, taskID).Scan(&t.TaskID, &corrID, &planID, &t.Status, &t.CurrentStep, &stepsJSON, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task %q not found", taskID)
		}
		return nil, err
	}

	if corrID != nil {
		t.CorrelationID = *corrID
	}
	if planID != nil {
		t.PlanID = *planID
	}
	_ = json.Unmarshal(stepsJSON, &t.Steps)
	return &t, nil
}

func (s *TaskStore) GetByCorrelationID(corrID string) (*orchestrator.Task, error) {
	var taskID string
	err := s.pool.QueryRow(ctx(), `SELECT task_id FROM tasks WHERE correlation_id = $1`, corrID).Scan(&taskID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no task found for correlation %q", corrID)
		}
		return nil, err
	}
	return s.GetTask(taskID)
}

func (s *TaskStore) ListTasks() ([]*orchestrator.Task, error) {
	rows, err := s.pool.Query(ctx(), `SELECT task_id FROM tasks ORDER BY updated_at`)
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

	cmd, err := s.pool.Exec(ctx(), `
		UPDATE tasks SET correlation_id = NULLIF($1,''), plan_id = NULLIF($2,''), status = $3, current_step = $4, steps = $5, updated_at = $6
		WHERE task_id = $7
	`, task.CorrelationID, task.PlanID, task.Status, task.CurrentStep, stepsJSON, task.UpdatedAt, task.TaskID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
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

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO task_events (event_id, task_id, from_status, to_status, step_index, detail, timestamp)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), $7)
	`, event.EventID, event.TaskID, event.FromStatus, event.ToStatus, event.StepSequence, event.Message, event.Timestamp)
	return err
}

func (s *TaskStore) GetEvents(taskID string) ([]*orchestrator.TaskEvent, error) {
	rows, err := s.pool.Query(ctx(), `
		SELECT event_id, task_id, from_status, to_status, step_index, detail, timestamp
		FROM task_events WHERE task_id = $1 ORDER BY timestamp
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*orchestrator.TaskEvent
	for rows.Next() {
		var e orchestrator.TaskEvent
		var fromStatus, toStatus, detail *string
		if err := rows.Scan(&e.EventID, &e.TaskID, &fromStatus, &toStatus, &e.StepSequence, &detail, &e.Timestamp); err != nil {
			return nil, err
		}
		if fromStatus != nil {
			e.FromStatus = orchestrator.TaskStatus(*fromStatus)
		}
		if toStatus != nil {
			e.ToStatus = orchestrator.TaskStatus(*toStatus)
		}
		if detail != nil {
			e.Message = *detail
		}
		result = append(result, &e)
	}
	return result, rows.Err()
}
