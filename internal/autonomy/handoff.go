package autonomy

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
)

type HandoffPackage struct {
	PackageID       string            `json:"package_id" yaml:"package_id"`
	TaskID          string            `json:"task_id" yaml:"task_id"`
	TaskSummary     string            `json:"task_summary" yaml:"task_summary"`
	CurrentState    string            `json:"current_state" yaml:"current_state"`
	CompletedSteps  []string          `json:"completed_steps" yaml:"completed_steps"`
	PendingSteps    []string          `json:"pending_steps" yaml:"pending_steps"`
	ContextData     map[string]string `json:"context_data" yaml:"context_data"`
	Artifacts       []string          `json:"artifacts" yaml:"artifacts"`
	Recommendations []string          `json:"recommendations" yaml:"recommendations"`
	CreatedAt       time.Time         `json:"created_at" yaml:"created_at"`
}

type HandoffStore interface {
	SavePackage(pkg *HandoffPackage) error
	GetPackage(packageID string) (*HandoffPackage, error)
	ListPackages() ([]*HandoffPackage, error)
	GetByTaskID(taskID string) (*HandoffPackage, error)
}

type MemoryHandoffStore struct {
	mu       sync.RWMutex
	byID     map[string]*HandoffPackage
	byTaskID map[string]string
}

func NewMemoryHandoffStore() *MemoryHandoffStore {
	return &MemoryHandoffStore{
		byID:     make(map[string]*HandoffPackage),
		byTaskID: make(map[string]string),
	}
}

func (s *MemoryHandoffStore) SavePackage(pkg *HandoffPackage) error {
	if pkg == nil {
		return fmt.Errorf("cannot save nil package")
	}
	if pkg.TaskID == "" {
		return fmt.Errorf("TaskID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := copyHandoffPackage(pkg)
	if cp.PackageID == "" {
		cp.PackageID = "pkg_" + uuid.New().String()[:8]
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now().UTC()
	}
	s.byID[cp.PackageID] = cp
	s.byTaskID[cp.TaskID] = cp.PackageID
	// Write back generated fields to caller's struct (inside lock, after store)
	pkg.PackageID = cp.PackageID
	pkg.CreatedAt = cp.CreatedAt
	return nil
}

func (s *MemoryHandoffStore) GetPackage(packageID string) (*HandoffPackage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pkg, ok := s.byID[packageID]
	if !ok {
		return nil, fmt.Errorf("package %q not found", packageID)
	}
	return copyHandoffPackage(pkg), nil
}

func (s *MemoryHandoffStore) ListPackages() ([]*HandoffPackage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*HandoffPackage, 0, len(s.byID))
	for _, pkg := range s.byID {
		result = append(result, copyHandoffPackage(pkg))
	}
	return result, nil
}

func (s *MemoryHandoffStore) GetByTaskID(taskID string) (*HandoffPackage, error) {
	s.mu.RLock()
	packageID, ok := s.byTaskID[taskID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no package for task %q", taskID)
	}
	return s.GetPackage(packageID)
}

// GenerateHandoffPackage is deprecated: use GenerateHandoffPackageFromStores with HandoffStore, TaskStore,
// PlanStore, and AuditLedger so the package is built from real task/audit/plan data.
func GenerateHandoffPackage(_ HandoffStore, _ string) (*HandoffPackage, error) {
	return nil, fmt.Errorf("handoff package generation requires task and plan context: use GenerateHandoffPackageFromStores(HandoffStore, TaskStore, PlanStore, AuditLedger, taskID) instead of GenerateHandoffPackage")
}

func GenerateHandoffPackageFromStores(store HandoffStore, taskStore orchestrator.TaskStore, planStore planning.PlanStore, auditLedger audit.AuditLedger, taskID string) (*HandoffPackage, error) {
	if store == nil {
		return nil, fmt.Errorf("handoff store is required")
	}
	if taskStore == nil {
		return nil, fmt.Errorf("task store is required")
	}
	task, err := taskStore.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	var plan *planning.Plan
	if planStore != nil {
		plan, _ = planStore.GetByTaskID(taskID)
	}
	var entries []*audit.AuditEntry
	if auditLedger != nil {
		entries, _ = auditLedger.GetByTask(taskID)
	}

	completed := make([]string, 0, len(task.Steps))
	pending := make([]string, 0, len(task.Steps))
	recommendations := []string{}
	for _, step := range task.Steps {
		label := fmt.Sprintf("%d:%s", step.Sequence, step.Action)
		if step.Description != "" {
			label = fmt.Sprintf("%s (%s)", label, step.Description)
		}
		switch step.Status {
		case orchestrator.StepSucceeded, orchestrator.StepSkipped:
			completed = append(completed, label)
		default:
			pending = append(pending, label)
			if step.ErrorMessage != "" {
				recommendations = append(recommendations, fmt.Sprintf("Investigate step %d failure: %s", step.Sequence, step.ErrorMessage))
			}
		}
	}

	contextData := map[string]string{
		"task_id":        task.TaskID,
		"task_status":    string(task.Status),
		"correlation_id": task.CorrelationID,
		"current_step":   fmt.Sprintf("%d", task.CurrentStep),
	}
	if task.PlanID != "" {
		contextData["plan_id"] = task.PlanID
	}
	if plan != nil {
		contextData["intent_source"] = plan.Intent.Source
		contextData["intent_action"] = plan.Intent.ActionType
		contextData["repo_owner"] = plan.Scope.Owner
		contextData["repo_name"] = plan.Scope.Repo
		contextData["branch"] = plan.Scope.Branch
		contextData["environment"] = plan.Scope.Environment
	}

	artifacts := make([]string, 0, len(entries)+len(task.Steps))
	for _, entry := range entries {
		artifacts = append(artifacts, entry.EntryID)
		artifacts = append(artifacts, entry.EvidenceRefs...)
	}
	for _, step := range task.Steps {
		if step.Output != "" {
			artifacts = append(artifacts, fmt.Sprintf("step:%d:output", step.Sequence))
		}
	}
	artifacts = dedupeStrings(artifacts)
	recommendations = dedupeStrings(recommendations)
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Review pending steps and latest audit events before resuming.")
	}

	summary := fmt.Sprintf("Task %s in status %s", task.TaskID, task.Status)
	if plan != nil && plan.Intent.RawInput != "" {
		summary = plan.Intent.RawInput
	}

	pkg := &HandoffPackage{
		TaskID:          taskID,
		TaskSummary:     summary,
		CurrentState:    string(task.Status),
		CompletedSteps:  completed,
		PendingSteps:    pending,
		ContextData:     contextData,
		Artifacts:       artifacts,
		Recommendations: recommendations,
		CreatedAt:       time.Now().UTC(),
	}
	if err := store.SavePackage(pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func copyHandoffPackage(pkg *HandoffPackage) *HandoffPackage {
	cp := *pkg
	if len(pkg.CompletedSteps) > 0 {
		cp.CompletedSteps = make([]string, len(pkg.CompletedSteps))
		copy(cp.CompletedSteps, pkg.CompletedSteps)
	}
	if len(pkg.PendingSteps) > 0 {
		cp.PendingSteps = make([]string, len(pkg.PendingSteps))
		copy(cp.PendingSteps, pkg.PendingSteps)
	}
	if len(pkg.ContextData) > 0 {
		cp.ContextData = make(map[string]string, len(pkg.ContextData))
		for k, v := range pkg.ContextData {
			cp.ContextData[k] = v
		}
	}
	if len(pkg.Artifacts) > 0 {
		cp.Artifacts = make([]string, len(pkg.Artifacts))
		copy(cp.Artifacts, pkg.Artifacts)
	}
	if len(pkg.Recommendations) > 0 {
		cp.Recommendations = make([]string, len(pkg.Recommendations))
		copy(cp.Recommendations, pkg.Recommendations)
	}
	return &cp
}
