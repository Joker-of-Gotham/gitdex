package autonomy

import (
	"context"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/orchestrator"
)

type TaskControlAction string

const (
	TaskControlPause    TaskControlAction = "pause"
	TaskControlResume   TaskControlAction = "resume"
	TaskControlCancel   TaskControlAction = "cancel"
	TaskControlTakeover TaskControlAction = "takeover"
)

type TaskControlRequest struct {
	Action    TaskControlAction `json:"action" yaml:"action"`
	TaskID    string            `json:"task_id" yaml:"task_id"`
	Reason    string            `json:"reason" yaml:"reason"`
	Actor     string            `json:"actor" yaml:"actor"`
	Timestamp time.Time         `json:"timestamp" yaml:"timestamp"`
}

type TaskControlResult struct {
	Request        TaskControlRequest `json:"request" yaml:"request"`
	Success        bool               `json:"success" yaml:"success"`
	PreviousStatus string             `json:"previous_status" yaml:"previous_status"`
	NewStatus      string             `json:"new_status" yaml:"new_status"`
	Message        string             `json:"message" yaml:"message"`
}

type TaskController interface {
	Execute(ctx context.Context, request TaskControlRequest) (*TaskControlResult, error)
}

type DefaultTaskController struct {
	taskStore   orchestrator.TaskStore
	auditLedger audit.AuditLedger
}

// NewTaskController creates a TaskController with real store dependencies.
func NewTaskController(taskStore orchestrator.TaskStore, auditLedger audit.AuditLedger) *DefaultTaskController {
	return &DefaultTaskController{
		taskStore:   taskStore,
		auditLedger: auditLedger,
	}
}

// NewDefaultTaskController creates a TaskController with in-memory stores (backward compatible).
func NewDefaultTaskController() *DefaultTaskController {
	return NewTaskController(
		orchestrator.NewMemoryTaskStore(),
		audit.NewMemoryAuditLedger(),
	)
}

func (c *DefaultTaskController) Execute(ctx context.Context, request TaskControlRequest) (*TaskControlResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now().UTC()
	}

	result := &TaskControlResult{
		Request: request,
		Success: true,
	}

	if c.taskStore == nil {
		// Fallback for tests without real stores
		result.PreviousStatus = "running"
		result.NewStatus = string(taskControlActionToStatus(request.Action))
		result.Message = fmt.Sprintf("task %s %s (no store)", request.TaskID, request.Action)
		return result, nil
	}

	task, err := c.taskStore.GetTask(request.TaskID)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("task %q not found: %v", request.TaskID, err)
		return result, nil
	}

	prev := string(task.Status)
	result.PreviousStatus = prev

	var newStatus orchestrator.TaskStatus
	switch request.Action {
	case TaskControlPause:
		if prev != "executing" && prev != "running" && prev != "queued" {
			result.Success = false
			result.NewStatus = prev
			result.Message = fmt.Sprintf("cannot pause task in status %s", prev)
			c.recordAudit(request, prev, prev, result.Success)
			return result, nil
		}
		newStatus = orchestrator.TaskPaused
		result.Message = fmt.Sprintf("task %s paused", request.TaskID)
	case TaskControlResume:
		if prev != "paused" {
			result.Success = false
			result.NewStatus = prev
			result.Message = fmt.Sprintf("cannot resume task in status %s", prev)
			c.recordAudit(request, prev, prev, result.Success)
			return result, nil
		}
		newStatus = orchestrator.TaskQueued
		result.Message = fmt.Sprintf("task %s resumed to queued", request.TaskID)
	case TaskControlCancel:
		newStatus = orchestrator.TaskCancelled
		result.Message = fmt.Sprintf("task %s cancelled", request.TaskID)
	case TaskControlTakeover:
		newStatus = orchestrator.TaskManual
		result.Message = fmt.Sprintf("task %s taken over to manual", request.TaskID)
	default:
		result.Success = false
		result.NewStatus = prev
		result.Message = fmt.Sprintf("unknown control action: %s", request.Action)
		c.recordAudit(request, prev, prev, false)
		return result, nil
	}

	task.Status = newStatus
	task.UpdatedAt = time.Now().UTC()
	if err := c.taskStore.UpdateTask(task); err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("failed to update task: %v", err)
		c.recordAudit(request, prev, prev, false)
		return result, nil
	}

	result.NewStatus = string(newStatus)
	c.recordAudit(request, prev, string(newStatus), true)
	return result, nil
}

func taskControlActionToStatus(a TaskControlAction) orchestrator.TaskStatus {
	switch a {
	case TaskControlPause:
		return orchestrator.TaskPaused
	case TaskControlResume:
		return orchestrator.TaskQueued
	case TaskControlCancel:
		return orchestrator.TaskCancelled
	case TaskControlTakeover:
		return orchestrator.TaskManual
	default:
		return ""
	}
}

func (c *DefaultTaskController) recordAudit(req TaskControlRequest, prev, next string, success bool) {
	if c.auditLedger == nil {
		return
	}
	_ = c.auditLedger.Append(&audit.AuditEntry{
		TaskID:    req.TaskID,
		EventType: audit.EventTaskControl,
		Actor:     req.Actor,
		Action:    string(req.Action),
		Target:    req.TaskID,
		Timestamp: req.Timestamp,
	})
}
