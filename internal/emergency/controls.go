package emergency

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/orchestrator"
)

type ControlAction string

const (
	ControlPauseTask         ControlAction = "pause_task"
	ControlPauseScope        ControlAction = "pause_scope"
	ControlSuspendCapability ControlAction = "suspend_capability"
	ControlKillSwitch        ControlAction = "kill_switch"
	ControlHaltAll           ControlAction = "halt_all"
	ControlPauseAll          ControlAction = "pause_all"
)

type ControlRequest struct {
	Action    ControlAction `json:"action" yaml:"action"`
	Scope     string        `json:"scope" yaml:"scope"`
	Reason    string        `json:"reason" yaml:"reason"`
	Actor     string        `json:"actor" yaml:"actor"`
	Timestamp time.Time     `json:"timestamp" yaml:"timestamp"`
}

type ControlResult struct {
	Request        ControlRequest `json:"request" yaml:"request"`
	Success        bool           `json:"success" yaml:"success"`
	AffectedTasks  []string       `json:"affected_tasks" yaml:"affected_tasks"`
	AffectedScopes []string       `json:"affected_scopes" yaml:"affected_scopes"`
	Message        string         `json:"message" yaml:"message"`
}

type ControlEngine interface {
	Execute(request ControlRequest) (*ControlResult, error)
}

type DefaultControlEngine struct {
	mu             sync.RWMutex
	activeControls []ControlRequest
	taskStore      orchestrator.TaskStore
	auditLedger    audit.AuditLedger
	taskController autonomy.TaskController
}

// NewControlEngine creates a ControlEngine with real store dependencies.
func NewControlEngine(taskStore orchestrator.TaskStore, auditLedger audit.AuditLedger, taskController autonomy.TaskController) *DefaultControlEngine {
	return &DefaultControlEngine{
		activeControls: nil,
		taskStore:      taskStore,
		auditLedger:    auditLedger,
		taskController: taskController,
	}
}

// NewDefaultControlEngine creates a ControlEngine with in-memory stores (backward compatible).
func NewDefaultControlEngine() *DefaultControlEngine {
	taskStore := orchestrator.NewMemoryTaskStore()
	auditLedger := audit.NewMemoryAuditLedger()
	taskController := autonomy.NewTaskController(taskStore, auditLedger)
	return NewControlEngine(taskStore, auditLedger, taskController)
}

func (e *DefaultControlEngine) Execute(request ControlRequest) (*ControlResult, error) {
	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now().UTC()
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	result := &ControlResult{
		Request:        request,
		Success:        true,
		AffectedTasks:  nil,
		AffectedScopes: nil,
		Message:        "",
	}

	switch request.Action {
	case ControlHaltAll:
		if e.taskStore == nil || e.taskController == nil {
			return nil, fmt.Errorf("emergency halt requires configured task store and controller")
		}
		affected, haltErr := e.haltAllTasks(request)
		result.AffectedTasks = affected
		if haltErr != nil {
			result.Success = false
			result.Message = fmt.Sprintf("halt_all: %d task(s) cancelled (with errors: %v)", len(affected), haltErr)
		} else {
			result.Message = fmt.Sprintf("halt_all: %d task(s) cancelled", len(affected))
		}
		if e.auditLedger != nil {
			_ = e.auditLedger.Append(&audit.AuditEntry{
				EventType: audit.EventEmergencyControl,
				Actor:     request.Actor,
				Action:    string(ControlHaltAll),
				Target:    "all",
				Timestamp: request.Timestamp,
			})
		}
	case ControlPauseAll:
		if e.taskStore == nil || e.taskController == nil {
			return nil, fmt.Errorf("emergency pause requires configured task store and controller")
		}
		affected, pauseErr := e.pauseAllTasks(request)
		result.AffectedTasks = affected
		if pauseErr != nil {
			result.Success = false
			result.Message = fmt.Sprintf("pause_all: %d task(s) paused (with errors: %v)", len(affected), pauseErr)
		} else {
			result.Message = fmt.Sprintf("pause_all: %d task(s) paused", len(affected))
		}
		if e.auditLedger != nil {
			_ = e.auditLedger.Append(&audit.AuditEntry{
				EventType: audit.EventEmergencyControl,
				Actor:     request.Actor,
				Action:    string(ControlPauseAll),
				Target:    "all",
				Timestamp: request.Timestamp,
			})
		}
	case ControlPauseTask:
		if e.taskController == nil {
			result.Success = false
			result.Message = "pause_task requires a configured task controller"
			return result, nil
		}
		res, err := e.taskController.Execute(context.Background(), autonomy.TaskControlRequest{
			Action:    autonomy.TaskControlPause,
			TaskID:    request.Scope,
			Reason:    request.Reason,
			Actor:     request.Actor,
			Timestamp: request.Timestamp,
		})
		if err != nil {
			return nil, err
		}
		result.Success = res.Success
		result.AffectedTasks = []string{request.Scope}
		result.Message = res.Message
		if e.auditLedger != nil {
			_ = e.auditLedger.Append(&audit.AuditEntry{
				EventType: audit.EventEmergencyControl,
				Actor:     request.Actor,
				Action:    string(ControlPauseTask),
				Target:    request.Scope,
				Timestamp: request.Timestamp,
			})
		}
	case ControlPauseScope:
		result.Success = false
		result.AffectedScopes = []string{request.Scope}
		result.Message = fmt.Sprintf("pause_scope for %s is not implemented against a real scope registry yet", request.Scope)
	case ControlSuspendCapability:
		result.Success = false
		result.AffectedScopes = []string{request.Scope}
		result.Message = fmt.Sprintf("suspend_capability for %s requires a real capability registry", request.Scope)
	case ControlKillSwitch:
		if e.taskStore != nil && e.taskController != nil {
			affected, killErr := e.haltAllTasks(request)
			result.AffectedTasks = affected
			if killErr != nil {
				result.Success = false
				result.Message = fmt.Sprintf("kill switch: %d task(s) cancelled (with errors: %v)", len(affected), killErr)
			} else if len(affected) > 0 {
				result.Message = fmt.Sprintf("kill switch: %d task(s) cancelled", len(affected))
			} else {
				result.Message = "kill switch activated; no active tasks"
			}
			result.AffectedScopes = []string{"*"}
			if e.auditLedger != nil {
				_ = e.auditLedger.Append(&audit.AuditEntry{
					EventType: audit.EventEmergencyControl,
					Actor:     request.Actor,
					Action:    string(ControlKillSwitch),
					Target:    "all",
					Timestamp: request.Timestamp,
				})
			}
		} else {
			result.Message = "kill switch activated (simulated); all tasks affected"
			result.AffectedTasks = []string{"*"}
			result.AffectedScopes = []string{"*"}
		}
	default:
		result.Success = false
		result.Message = fmt.Sprintf("unknown control action: %s", request.Action)
		return result, nil
	}

	e.activeControls = append(e.activeControls, request)
	return result, nil
}

func (e *DefaultControlEngine) haltAllTasks(req ControlRequest) ([]string, error) {
	tasks, err := e.taskStore.ListTasks()
	if err != nil {
		return nil, fmt.Errorf("halt_all: list tasks: %w", err)
	}
	ctx := context.Background()
	affected := make([]string, 0, len(tasks))
	var firstErr error
	for _, t := range tasks {
		if t.Status.IsTerminal() {
			continue
		}
		res, execErr := e.taskController.Execute(ctx, autonomy.TaskControlRequest{
			Action:    autonomy.TaskControlCancel,
			TaskID:    t.TaskID,
			Actor:     req.Actor,
			Timestamp: req.Timestamp,
		})
		if execErr != nil && firstErr == nil {
			firstErr = fmt.Errorf("cancel task %s: %w", t.TaskID, execErr)
		}
		if res != nil && res.Success {
			affected = append(affected, t.TaskID)
		}
	}
	return affected, firstErr
}

func (e *DefaultControlEngine) pauseAllTasks(req ControlRequest) ([]string, error) {
	tasks, err := e.taskStore.ListTasks()
	if err != nil {
		return nil, fmt.Errorf("pause_all: list tasks: %w", err)
	}
	ctx := context.Background()
	affected := make([]string, 0, len(tasks))
	var firstErr error
	for _, t := range tasks {
		if t.Status.IsTerminal() {
			continue
		}
		res, execErr := e.taskController.Execute(ctx, autonomy.TaskControlRequest{
			Action:    autonomy.TaskControlPause,
			TaskID:    t.TaskID,
			Actor:     req.Actor,
			Timestamp: req.Timestamp,
		})
		if execErr != nil && firstErr == nil {
			firstErr = fmt.Errorf("pause task %s: %w", t.TaskID, execErr)
		}
		if res != nil && res.Success {
			affected = append(affected, t.TaskID)
		}
	}
	return affected, firstErr
}

// taskMatchesScope returns true if the scope string matches this task's identifiers
// (exact match on task, correlation, or plan ID, or substring match for partial scopes).
func taskMatchesScope(t *orchestrator.Task, scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	if t.TaskID == scope || t.CorrelationID == scope || t.PlanID == scope {
		return true
	}
	return strings.Contains(t.TaskID, scope) ||
		strings.Contains(t.CorrelationID, scope) ||
		strings.Contains(t.PlanID, scope)
}

func (e *DefaultControlEngine) pauseScopeTasks(req ControlRequest) ([]string, error) {
	tasks, err := e.taskStore.ListTasks()
	if err != nil {
		return nil, fmt.Errorf("pause_scope: list tasks: %w", err)
	}
	ctx := context.Background()
	affected := make([]string, 0)
	var firstErr error
	for _, t := range tasks {
		if t.Status.IsTerminal() {
			continue
		}
		if !taskMatchesScope(t, req.Scope) {
			continue
		}
		res, execErr := e.taskController.Execute(ctx, autonomy.TaskControlRequest{
			Action:    autonomy.TaskControlPause,
			TaskID:    t.TaskID,
			Actor:     req.Actor,
			Timestamp: req.Timestamp,
		})
		if execErr != nil && firstErr == nil {
			firstErr = fmt.Errorf("pause task %s: %w", t.TaskID, execErr)
		}
		if res != nil && res.Success {
			affected = append(affected, t.TaskID)
		}
	}
	return affected, firstErr
}

func (e *DefaultControlEngine) Status() []ControlRequest {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]ControlRequest, len(e.activeControls))
	copy(out, e.activeControls)
	return out
}
