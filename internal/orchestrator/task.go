package orchestrator

import (
	"crypto/rand"
	"fmt"
	"time"
)

type TaskStatus string

const (
	TaskQueued               TaskStatus = "queued"
	TaskExecuting            TaskStatus = "executing"
	TaskReconciling          TaskStatus = "reconciling"
	TaskSucceeded            TaskStatus = "succeeded"
	TaskFailedHandoffPending TaskStatus = "failed_handoff_pending"
	TaskQuarantined          TaskStatus = "quarantined"
	TaskCancelled            TaskStatus = "cancelled"
	TaskFailedWithHandoff    TaskStatus = "failed_with_handoff_complete"
	TaskPaused               TaskStatus = "paused"
	TaskManual               TaskStatus = "manual"
)

func (s TaskStatus) IsTerminal() bool {
	switch s {
	case TaskSucceeded, TaskCancelled, TaskFailedWithHandoff:
		return true
	}
	return false
}

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepSucceeded StepStatus = "succeeded"
	StepFailed    StepStatus = "failed"
	StepSkipped   StepStatus = "skipped"
)

type StepResult struct {
	Sequence     int        `json:"sequence" yaml:"sequence"`
	Action       string     `json:"action" yaml:"action"`
	Target       string     `json:"target" yaml:"target"`
	Description  string     `json:"description,omitempty" yaml:"description,omitempty"`
	Status       StepStatus `json:"status" yaml:"status"`
	StartedAt    *time.Time `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty" yaml:"finished_at,omitempty"`
	Output       string     `json:"output,omitempty" yaml:"output,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty" yaml:"error_message,omitempty"`
}

type TaskEvent struct {
	EventID      string     `json:"event_id" yaml:"event_id"`
	TaskID       string     `json:"task_id" yaml:"task_id"`
	FromStatus   TaskStatus `json:"from_status" yaml:"from_status"`
	ToStatus     TaskStatus `json:"to_status" yaml:"to_status"`
	StepSequence int        `json:"step_sequence,omitempty" yaml:"step_sequence,omitempty"`
	Message      string     `json:"message" yaml:"message"`
	Timestamp    time.Time  `json:"timestamp" yaml:"timestamp"`
}

type Task struct {
	TaskID        string       `json:"task_id" yaml:"task_id"`
	CorrelationID string       `json:"correlation_id" yaml:"correlation_id"`
	PlanID        string       `json:"plan_id" yaml:"plan_id"`
	Status        TaskStatus   `json:"status" yaml:"status"`
	CurrentStep   int          `json:"current_step" yaml:"current_step"`
	Steps         []StepResult `json:"steps" yaml:"steps"`
	StartedAt     *time.Time   `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	UpdatedAt     time.Time    `json:"updated_at" yaml:"updated_at"`
	FinishedAt    *time.Time   `json:"finished_at,omitempty" yaml:"finished_at,omitempty"`
}

func (t *Task) CompletedSteps() []StepResult {
	var completed []StepResult
	for _, s := range t.Steps {
		if s.Status == StepSucceeded || s.Status == StepFailed || s.Status == StepSkipped {
			completed = append(completed, s)
		}
	}
	return completed
}

func (t *Task) RunningStep() *StepResult {
	for i := range t.Steps {
		if t.Steps[i].Status == StepRunning {
			return &t.Steps[i]
		}
	}
	return nil
}

func GenerateTaskID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("task_%x", b)
}

func GenerateCorrelationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("corr_%x", b)
}

func GenerateEventID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("evt_%x", b)
}
