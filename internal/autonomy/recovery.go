package autonomy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
)

type RecoveryStrategy string

const (
	RecoveryRetry              RecoveryStrategy = "retry"
	RecoveryRollback           RecoveryStrategy = "rollback"
	RecoveryEscalate           RecoveryStrategy = "escalate"
	RecoverySkip               RecoveryStrategy = "skip"
	RecoveryManualIntervention RecoveryStrategy = "manual_intervention"
)

type RecoveryRequest struct {
	TaskID     string           `json:"task_id" yaml:"task_id"`
	Strategy   RecoveryStrategy `json:"strategy" yaml:"strategy"`
	MaxRetries int              `json:"max_retries" yaml:"max_retries"`
	Reason     string           `json:"reason" yaml:"reason"`
	Actor      string           `json:"actor" yaml:"actor"`
}

type RecoveryResult struct {
	Request     RecoveryRequest `json:"request" yaml:"request"`
	Success     bool            `json:"success" yaml:"success"`
	Attempts    int             `json:"attempts" yaml:"attempts"`
	FinalStatus string          `json:"final_status" yaml:"final_status"`
	Message     string          `json:"message" yaml:"message"`
	RecoveredAt time.Time       `json:"recovered_at" yaml:"recovered_at"`
}

type RecoveryAssessment struct {
	TaskID              string           `json:"task_id" yaml:"task_id"`
	FailureType         string           `json:"failure_type" yaml:"failure_type"`
	RecommendedStrategy RecoveryStrategy `json:"recommended_strategy" yaml:"recommended_strategy"`
	RiskLevel           string           `json:"risk_level" yaml:"risk_level"`
	Details             string           `json:"details" yaml:"details"`
}

type RecoveryEngine interface {
	Assess(ctx context.Context, taskID string) (*RecoveryAssessment, error)
	Execute(ctx context.Context, request RecoveryRequest) (*RecoveryResult, error)
}

type DefaultRecoveryEngine struct {
	mu           sync.RWMutex
	history      []RecoveryResult
	taskStore    orchestrator.TaskStore
	planStore    planning.PlanStore
	auditLedger  audit.AuditLedger
	handoffStore HandoffStore
}

// NewRecoveryEngine creates a RecoveryEngine with real store dependencies.
func NewRecoveryEngine(taskStore orchestrator.TaskStore, planStore planning.PlanStore, auditLedger audit.AuditLedger, handoffStore HandoffStore) *DefaultRecoveryEngine {
	return &DefaultRecoveryEngine{
		history:      make([]RecoveryResult, 0),
		taskStore:    taskStore,
		planStore:    planStore,
		auditLedger:  auditLedger,
		handoffStore: handoffStore,
	}
}

func (e *DefaultRecoveryEngine) Assess(ctx context.Context, taskID string) (*RecoveryAssessment, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if e.taskStore == nil {
		return nil, fmt.Errorf("task store is required for recovery assessment; configure TaskStore")
	}

	task, err := e.taskStore.GetTask(taskID)
	if err != nil {
		return &RecoveryAssessment{
			TaskID:              taskID,
			FailureType:         "not_found",
			RecommendedStrategy: RecoveryManualIntervention,
			RiskLevel:           "high",
			Details:             fmt.Sprintf("task %s not found: %v", taskID, err),
		}, nil
	}
	state := string(task.Status)
	failureType := "blocked"
	if state == string(orchestrator.TaskFailedHandoffPending) || state == string(orchestrator.TaskQuarantined) {
		failureType = "failed"
	} else if state == "drifted" {
		failureType = "drifted"
	}
	recommended := RecoveryRetry
	riskLevel := "low"
	if failureType == "failed" {
		recommended = RecoveryRollback
		riskLevel = "medium"
	} else if failureType == "drifted" {
		recommended = RecoveryManualIntervention
		riskLevel = "high"
	}
	return &RecoveryAssessment{
		TaskID:              taskID,
		FailureType:         failureType,
		RecommendedStrategy: recommended,
		RiskLevel:           riskLevel,
		Details:             fmt.Sprintf("task %s (status=%s)", taskID, state),
	}, nil
}

var validRecoveryStrategies = map[RecoveryStrategy]struct{}{
	RecoveryRetry:              {},
	RecoveryRollback:           {},
	RecoveryEscalate:           {},
	RecoverySkip:               {},
	RecoveryManualIntervention: {},
}

func (e *DefaultRecoveryEngine) Execute(ctx context.Context, request RecoveryRequest) (*RecoveryResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if _, ok := validRecoveryStrategies[request.Strategy]; !ok {
		return nil, fmt.Errorf("invalid recovery strategy %q; use retry, rollback, escalate, skip, or manual_intervention", request.Strategy)
	}

	maxRetries := request.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	if e.taskStore == nil {
		return nil, fmt.Errorf("task store is required for recovery execution; configure TaskStore")
	}
	if e.auditLedger == nil {
		return nil, fmt.Errorf("audit ledger is required for recovery execution; configure AuditLedger")
	}

	// Count prior recovery attempts from AuditLedger
	entries, _ := e.auditLedger.Query(audit.AuditFilter{
		TaskID:    request.TaskID,
		EventType: audit.EventRecovery,
	})
	priorRetries := len(entries)

	task, err := e.taskStore.GetTask(request.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task %q not found: %w", request.TaskID, err)
	}

	var result RecoveryResult
	retriable := priorRetries < maxRetries && (request.Strategy == RecoveryRetry || request.Strategy == RecoveryRollback)

	if retriable && (request.Strategy == RecoveryRetry || request.Strategy == RecoveryRollback) {
		// Reset task to queued for retry
		task.Status = orchestrator.TaskQueued
		task.UpdatedAt = time.Now().UTC()
		if err := e.taskStore.UpdateTask(task); err != nil {
			return nil, fmt.Errorf("failed to update task for retry: %w", err)
		}
		result = RecoveryResult{
			Request:     request,
			Success:     true,
			Attempts:    priorRetries + 1,
			FinalStatus: "queued",
			Message:     fmt.Sprintf("task %s reset to queued for retry (attempt %d/%d)", request.TaskID, priorRetries+1, maxRetries),
			RecoveredAt: time.Now().UTC(),
		}
	} else if request.Strategy == RecoveryManualIntervention || !retriable {
		// Create HandoffPackage for manual handoff
		if e.handoffStore != nil {
			if _, err := GenerateHandoffPackageFromStores(e.handoffStore, e.taskStore, e.planStore, e.auditLedger, request.TaskID); err != nil {
				return nil, fmt.Errorf("failed to create handoff package: %w", err)
			}
		}
		task.Status = orchestrator.TaskFailedHandoffPending
		task.UpdatedAt = time.Now().UTC()
		_ = e.taskStore.UpdateTask(task)
		result = RecoveryResult{
			Request:     request,
			Success:     true,
			Attempts:    priorRetries + 1,
			FinalStatus: "manual",
			Message:     fmt.Sprintf("task %s marked for manual handoff (exceeded retries or manual_intervention)", request.TaskID),
			RecoveredAt: time.Now().UTC(),
		}
	} else {
		// Escalate, Skip - update task status
		var newStatus orchestrator.TaskStatus
		switch request.Strategy {
		case RecoveryEscalate:
			newStatus = orchestrator.TaskFailedHandoffPending
		case RecoverySkip:
			newStatus = orchestrator.TaskCancelled
		default:
			newStatus = orchestrator.TaskFailedHandoffPending
		}
		task.Status = newStatus
		task.UpdatedAt = time.Now().UTC()
		_ = e.taskStore.UpdateTask(task)
		result = RecoveryResult{
			Request:     request,
			Success:     true,
			Attempts:    priorRetries + 1,
			FinalStatus: string(newStatus),
			Message:     fmt.Sprintf("task %s recovered using %s", request.TaskID, request.Strategy),
			RecoveredAt: time.Now().UTC(),
		}
	}

	// Record recovery event in AuditLedger
	_ = e.auditLedger.Append(&audit.AuditEntry{
		TaskID:       request.TaskID,
		EventType:    audit.EventRecovery,
		Actor:        request.Actor,
		Action:       string(request.Strategy),
		Target:       request.TaskID,
		PolicyResult: result.FinalStatus,
		Timestamp:    time.Now().UTC(),
	})

	e.mu.Lock()
	e.history = append(e.history, result)
	e.mu.Unlock()
	return &result, nil
}

func (e *DefaultRecoveryEngine) History(taskID string) []RecoveryResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if taskID == "" {
		out := make([]RecoveryResult, len(e.history))
		copy(out, e.history)
		return out
	}

	var filtered []RecoveryResult
	for _, r := range e.history {
		if r.Request.TaskID == taskID {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
