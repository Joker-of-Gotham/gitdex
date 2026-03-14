package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	gitplatform "github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type workflowFlowStepStatus string

const (
	workflowFlowPending           workflowFlowStepStatus = "pending"
	workflowFlowReady             workflowFlowStepStatus = "ready"
	workflowFlowSuggested         workflowFlowStepStatus = "suggested"
	workflowFlowRunning           workflowFlowStepStatus = "running"
	workflowFlowWaitingValidation workflowFlowStepStatus = "waiting_validation"
	workflowFlowRetrying          workflowFlowStepStatus = "retrying"
	workflowFlowPaused            workflowFlowStepStatus = "paused"
	workflowFlowDone              workflowFlowStepStatus = "done"
	workflowFlowFailed            workflowFlowStepStatus = "failed"
	workflowFlowDeadLetter        workflowFlowStepStatus = "deadletter"
	workflowFlowCompensated       workflowFlowStepStatus = "compensated"
	workflowFlowSkipped           workflowFlowStepStatus = "skipped"
)

type WorkflowStepPolicy struct {
	RetryBudget      int    `json:"retry_budget,omitempty"`
	BackoffSecs      int    `json:"backoff_secs,omitempty"`
	TimeoutSecs      int    `json:"timeout_secs,omitempty"`
	ApprovalRequired bool   `json:"approval_required,omitempty"`
	ConcurrencyKey   string `json:"concurrency_key,omitempty"`
	SchedulerSafe    bool   `json:"scheduler_safe,omitempty"`
}

type DeadLetterEntry struct {
	StepIndex   int       `json:"step_index,omitempty"`
	Identity    string    `json:"identity,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	At          time.Time `json:"at,omitempty"`
	Acked       bool      `json:"acked,omitempty"`
	AckedAt     time.Time `json:"acked_at,omitempty"`
	Attempts    int       `json:"attempts,omitempty"`
	NextRetryAt time.Time `json:"next_retry_at,omitempty"`
	LedgerRefs  []string  `json:"ledger_refs,omitempty"`
}

type CompensationRef struct {
	StepIndex  int       `json:"step_index,omitempty"`
	Identity   string    `json:"identity,omitempty"`
	Summary    string    `json:"summary,omitempty"`
	LedgerID   string    `json:"ledger_id,omitempty"`
	RecordedAt time.Time `json:"recorded_at,omitempty"`
}

type WorkflowRunStep struct {
	Index         int                              `json:"index"`
	Identity      string                           `json:"identity"`
	Status        workflowFlowStepStatus           `json:"status"`
	UpdatedAt     time.Time                        `json:"updated_at,omitempty"`
	SuggestionRef string                           `json:"suggestion_ref,omitempty"`
	LastDetail    string                           `json:"last_detail,omitempty"`
	Attempt       int                              `json:"attempt,omitempty"`
	RetryBudget   int                              `json:"retry_budget,omitempty"`
	BackoffSecs   int                              `json:"backoff_secs,omitempty"`
	TimeoutSecs   int                              `json:"timeout_secs,omitempty"`
	NextRetryAt   time.Time                        `json:"next_retry_at,omitempty"`
	ApprovalReq   bool                             `json:"approval_required,omitempty"`
	ApprovalState string                           `json:"approval_state,omitempty"`
	ApprovedAt    time.Time                        `json:"approved_at,omitempty"`
	Concurrency   string                           `json:"concurrency_key,omitempty"`
	DeadLetter    string                           `json:"dead_letter,omitempty"`
	Compensation  string                           `json:"compensation,omitempty"`
	LedgerRefs    []string                         `json:"ledger_refs,omitempty"`
	Policy        WorkflowStepPolicy               `json:"policy,omitempty"`
	DeadLetterRef *DeadLetterEntry                 `json:"dead_letter_ref,omitempty"`
	Step          prompt.WorkflowOrchestrationStep `json:"step"`
}

type WorkflowRunState struct {
	RunID             string            `json:"run_id,omitempty"`
	CheckpointVersion int               `json:"checkpoint_version,omitempty"`
	WorkflowID        string            `json:"workflow_id"`
	WorkflowLabel     string            `json:"workflow_label"`
	Goal              string            `json:"goal,omitempty"`
	UpdatedAt         time.Time         `json:"updated_at,omitempty"`
	Health            string            `json:"health,omitempty"`
	PausedReason      string            `json:"paused_reason,omitempty"`
	ApprovalState     string            `json:"approval_state,omitempty"`
	ApprovalDetail    string            `json:"approval_detail,omitempty"`
	SelectedStepIndex int               `json:"selected_step_index,omitempty"`
	NextRetryAt       time.Time         `json:"next_retry_at,omitempty"`
	NextRetryStep     string            `json:"next_retry_step,omitempty"`
	ActiveLocks       map[string]string `json:"active_locks,omitempty"`
	DeadLetterEntries []DeadLetterEntry `json:"dead_letter_entries,omitempty"`
	CompensationRefs  []CompensationRef `json:"compensation_refs,omitempty"`
	Steps             []WorkflowRunStep `json:"steps,omitempty"`
}

type workflowFlowStep = WorkflowRunStep
type workflowFlowState = WorkflowRunState

func materializeWorkflowFlow(platformID gitplatform.Platform, plan *prompt.WorkflowOrchestration) *workflowFlowState {
	if plan == nil || len(plan.Steps) == 0 {
		return nil
	}
	steps := make([]workflowFlowStep, 0, len(plan.Steps))
	for idx, step := range plan.Steps {
		retryBudget := workflowStepRetryBudget(step)
		backoffSecs := workflowStepBackoff(step)
		policy := WorkflowStepPolicy{
			RetryBudget:      retryBudget,
			BackoffSecs:      backoffSecs,
			TimeoutSecs:      workflowStepTimeout(step),
			ApprovalRequired: workflowStepApproval(platformID, step),
			ConcurrencyKey:   workflowStepConcurrency(step),
			SchedulerSafe:    workflowStepSchedulerSafe(platformID, step),
		}
		steps = append(steps, workflowFlowStep{
			Index:       idx,
			Identity:    workflowStepIdentity(step),
			Status:      workflowFlowPending,
			UpdatedAt:   time.Now(),
			RetryBudget: retryBudget,
			BackoffSecs: backoffSecs,
			TimeoutSecs: policy.TimeoutSecs,
			ApprovalReq: policy.ApprovalRequired,
			ApprovalState: workflowStepApprovalState(
				policy.ApprovalRequired,
				false,
			),
			Concurrency: policy.ConcurrencyKey,
			Policy:      policy,
			Step:        cloneWorkflowOrchestrationStep(step),
		})
	}
	return &workflowFlowState{
		RunID:             newWorkflowRunID(plan),
		CheckpointVersion: 1,
		WorkflowID:        strings.TrimSpace(plan.WorkflowID),
		WorkflowLabel:     strings.TrimSpace(plan.WorkflowLabel),
		Goal:              strings.TrimSpace(plan.Goal),
		UpdatedAt:         time.Now(),
		Health:            "pending",
		ApprovalState:     "pending",
		SelectedStepIndex: 0,
		ActiveLocks:       map[string]string{},
		Steps:             steps,
	}
}

func newWorkflowRunID(plan *prompt.WorkflowOrchestration) string {
	base := strings.TrimSpace(firstNonEmpty(plan.WorkflowID, plan.WorkflowLabel, "workflow"))
	base = strings.ReplaceAll(strings.ToLower(base), " ", "-")
	return base + "-" + time.Now().Format("20060102T150405.000000000")
}

func cloneWorkflowOrchestrationStep(step prompt.WorkflowOrchestrationStep) prompt.WorkflowOrchestrationStep {
	return prompt.WorkflowOrchestrationStep{
		Title:      strings.TrimSpace(step.Title),
		Rationale:  strings.TrimSpace(step.Rationale),
		Capability: strings.TrimSpace(step.Capability),
		Flow:       strings.TrimSpace(step.Flow),
		Operation:  strings.TrimSpace(step.Operation),
		ResourceID: strings.TrimSpace(step.ResourceID),
		Scope:      cloneStringMap(step.Scope),
		Query:      cloneStringMap(step.Query),
		Payload:    cloneRaw(step.Payload),
		Validate:   cloneRaw(step.Validate),
		Rollback:   cloneRaw(step.Rollback),
	}
}

func workflowStepIdentity(step prompt.WorkflowOrchestrationStep) string {
	parts := []string{
		git.PlatformExecIdentity(&git.PlatformExecInfo{
			CapabilityID: step.Capability,
			Flow:         step.Flow,
			Operation:    step.Operation,
			ResourceID:   step.ResourceID,
			Scope:        cloneStringMap(step.Scope),
		}),
	}
	if encoded, ok := workflowIdentityJSON(step.Query); ok {
		parts = append(parts, encoded)
	}
	if encoded, ok := workflowIdentityJSON(step.Payload); ok {
		parts = append(parts, encoded)
	}
	if encoded, ok := workflowIdentityJSON(step.Validate); ok {
		parts = append(parts, encoded)
	}
	if encoded, ok := workflowIdentityJSON(step.Rollback); ok {
		parts = append(parts, encoded)
	}
	return strings.Join(parts, ":")
}

func workflowIdentityJSON(value any) (string, bool) {
	switch raw := value.(type) {
	case nil:
		return "", false
	case json.RawMessage:
		text := strings.TrimSpace(string(raw))
		return text, text != "" && text != "null"
	}
	encoded, err := marshalWorkflowJSON(value)
	if err != nil {
		return "", false
	}
	text := strings.TrimSpace(string(encoded))
	return text, text != "" && text != "null"
}

func (m *Model) syncWorkflowFlowFromPlan() {
	m.workflowFlow = materializeWorkflowFlow(m.detectedPlatform(), m.workflowPlan)
	m.refreshWorkflowRunState("")
	m.syncTaskMemory()
}

func (m *Model) reconcileWorkflowFlowSuggestions() {
	if m.workflowFlow == nil || len(m.workflowFlow.Steps) == 0 {
		return
	}
	now := time.Now()
	for idx := range m.workflowFlow.Steps {
		step := &m.workflowFlow.Steps[idx]
		if workflowFlowTerminal(step.Status) || step.Status == workflowFlowRunning || step.Status == workflowFlowPaused || step.Status == workflowFlowWaitingValidation {
			continue
		}
		if step.Status == workflowFlowRetrying && !step.NextRetryAt.IsZero() && step.NextRetryAt.After(now) {
			continue
		}
		step.SuggestionRef = ""
		step.LastDetail = ""
		step.Status = workflowFlowReady
		for _, suggestion := range m.suggestions {
			if suggestion.Interaction != git.PlatformExec || suggestion.PlatformOp == nil {
				continue
			}
			if !workflowStepMatchesSuggestion(step.Step, suggestion) {
				continue
			}
			step.Status = workflowFlowSuggested
			step.SuggestionRef = strings.TrimSpace(firstNonEmpty(suggestion.Action, git.PlatformExecIdentity(suggestion.PlatformOp)))
			step.LastDetail = strings.TrimSpace(suggestion.Reason)
			step.UpdatedAt = now
			break
		}
		if step.Status == workflowFlowReady && step.ApprovalReq {
			step.LastDetail = strings.TrimSpace(firstNonEmpty(step.LastDetail, "approval required"))
		}
	}
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = now
	m.syncTaskMemory()
}

func (m *Model) markWorkflowFlowRunning(op *git.PlatformExecInfo) {
	step := m.findWorkflowFlowStep(op)
	if step == nil {
		return
	}
	step.Status = workflowFlowRunning
	step.Attempt++
	step.UpdatedAt = time.Now()
	step.SuggestionRef = strings.TrimSpace(platformActionTitle(op))
	step.LastDetail = "running"
	step.NextRetryAt = time.Time{}
	if step.Policy.ApprovalRequired {
		step.ApprovalState = workflowStepApprovalState(true, true)
		step.ApprovedAt = step.UpdatedAt
	}
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = step.UpdatedAt
	m.syncTaskMemory()
}

func (m *Model) markWorkflowFlowResult(op *git.PlatformExecInfo, failed bool, detail string) {
	step := m.findWorkflowFlowStep(op)
	if step == nil {
		return
	}
	if failed {
		if step.Attempt <= maxInt(1, step.RetryBudget) {
			step.Status = workflowFlowRetrying
			step.NextRetryAt = time.Now().Add(time.Duration(maxInt(1, step.BackoffSecs*maxInt(1, step.Attempt))) * time.Second)
			step.LastDetail = strings.TrimSpace(firstNonEmpty(detail, "retry scheduled"))
		} else {
			step.Status = workflowFlowDeadLetter
			step.DeadLetter = strings.TrimSpace(firstNonEmpty(detail, "step moved to dead letter"))
			step.LastDetail = step.DeadLetter
			step.DeadLetterRef = m.recordDeadLetterEntry(step, step.DeadLetter)
		}
		step.ApprovalState = workflowStepApprovalState(step.Policy.ApprovalRequired, false)
		step.ApprovedAt = time.Time{}
	} else {
		switch strings.ToLower(strings.TrimSpace(op.Flow)) {
		case "rollback":
			step.Status = workflowFlowCompensated
			step.Compensation = strings.TrimSpace(firstNonEmpty(detail, "rollback applied"))
			step.DeadLetter = ""
			if ref := m.recordCompensationRef(step, step.Compensation); ref != nil {
				step.Compensation = strings.TrimSpace(firstNonEmpty(step.Compensation, ref.Summary))
			}
		case "mutate":
			if len(step.Step.Validate) > 0 {
				step.Status = workflowFlowWaitingValidation
			} else {
				step.Status = workflowFlowDone
			}
		default:
			step.Status = workflowFlowDone
		}
		step.LastDetail = strings.TrimSpace(firstNonEmpty(detail, step.LastDetail))
		step.ApprovalState = workflowStepApprovalState(step.Policy.ApprovalRequired, step.Policy.ApprovalRequired)
		if step.Policy.ApprovalRequired && step.ApprovedAt.IsZero() {
			step.ApprovedAt = time.Now()
		}
	}
	step.UpdatedAt = time.Now()
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = step.UpdatedAt
	m.syncTaskMemory()
}

func (m *Model) completeWorkflowFlowValidation(op *git.PlatformExecInfo, detail string, failed bool) {
	step := m.findWorkflowFlowStep(op)
	if step == nil {
		return
	}
	if failed {
		m.markWorkflowFlowResult(op, true, detail)
		return
	}
	step.Status = workflowFlowDone
	step.UpdatedAt = time.Now()
	step.LastDetail = strings.TrimSpace(firstNonEmpty(detail, "validation completed"))
	step.ApprovalState = workflowStepApprovalState(step.Policy.ApprovalRequired, step.Policy.ApprovalRequired)
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = step.UpdatedAt
	m.syncTaskMemory()
}

func (m *Model) recordWorkflowFlowLedger(op *git.PlatformExecInfo, ledgerID string) {
	step := m.findWorkflowFlowStep(op)
	if step == nil || strings.TrimSpace(ledgerID) == "" {
		return
	}
	for _, existing := range step.LedgerRefs {
		if strings.EqualFold(existing, ledgerID) {
			return
		}
	}
	step.LedgerRefs = append(step.LedgerRefs, strings.TrimSpace(ledgerID))
	if step.DeadLetterRef != nil {
		step.DeadLetterRef.LedgerRefs = appendIfMissing(step.DeadLetterRef.LedgerRefs, strings.TrimSpace(ledgerID))
	}
	step.UpdatedAt = time.Now()
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = step.UpdatedAt
	m.syncTaskMemory()
}

func (m *Model) promoteDueWorkflowRetries(now time.Time) bool {
	if m.workflowFlow == nil {
		return false
	}
	changed := false
	for idx := range m.workflowFlow.Steps {
		step := &m.workflowFlow.Steps[idx]
		if step.Status != workflowFlowRetrying {
			continue
		}
		if !step.NextRetryAt.IsZero() && step.NextRetryAt.After(now) {
			continue
		}
		step.Status = workflowFlowReady
		step.NextRetryAt = time.Time{}
		step.UpdatedAt = now
		step.LastDetail = strings.TrimSpace(firstNonEmpty(step.LastDetail, "retry ready"))
		step.DeadLetter = ""
		step.DeadLetterRef = nil
		step.ApprovalState = workflowStepApprovalState(step.Policy.ApprovalRequired, false)
		step.ApprovedAt = time.Time{}
		changed = true
	}
	if changed {
		m.refreshWorkflowRunState("")
		m.workflowFlow.UpdatedAt = now
		m.syncTaskMemory()
	}
	return changed
}

func (m *Model) pauseWorkflowFlow(reason string) {
	if step := m.selectedWorkflowStep(); m.pauseWorkflowStep(step, reason) {
		return
	}
	m.pauseAllWorkflowSteps(reason)
}

func (m *Model) pauseAllWorkflowSteps(reason string) {
	if m.workflowFlow == nil {
		return
	}
	now := time.Now()
	for idx := range m.workflowFlow.Steps {
		step := &m.workflowFlow.Steps[idx]
		if step.Status == workflowFlowRunning || step.Status == workflowFlowRetrying || step.Status == workflowFlowReady || step.Status == workflowFlowSuggested {
			step.Status = workflowFlowPaused
			step.UpdatedAt = now
			step.LastDetail = strings.TrimSpace(firstNonEmpty(reason, step.LastDetail, "paused"))
		}
	}
	m.refreshWorkflowRunState(reason)
	m.workflowFlow.UpdatedAt = now
	m.syncTaskMemory()
}

func (m *Model) resumeWorkflowFlow() {
	if step := m.selectedWorkflowStep(); m.resumeWorkflowStep(step) {
		return
	}
	m.resumeAllWorkflowSteps()
}

func (m *Model) resumeAllWorkflowSteps() {
	if m.workflowFlow == nil {
		return
	}
	now := time.Now()
	for idx := range m.workflowFlow.Steps {
		step := &m.workflowFlow.Steps[idx]
		if step.Status == workflowFlowPaused {
			step.Status = workflowFlowReady
			step.UpdatedAt = now
			step.LastDetail = strings.TrimSpace(firstNonEmpty(step.LastDetail, "resumed"))
		}
	}
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = now
	m.syncTaskMemory()
}

func (m Model) workflowHasPausedSteps() bool {
	if m.workflowFlow == nil {
		return false
	}
	for _, step := range m.workflowFlow.Steps {
		if step.Status == workflowFlowPaused {
			return true
		}
	}
	return false
}

func (m Model) workflowHasDeadLetters() bool {
	if m.workflowFlow == nil {
		return false
	}
	for _, step := range m.workflowFlow.Steps {
		if step.Status == workflowFlowDeadLetter {
			return true
		}
	}
	return false
}

func (m *Model) retryDeadLetterWorkflowStep() (string, bool) {
	if m.workflowFlow == nil {
		return "", false
	}
	return m.retryWorkflowStep(m.preferredDeadLetterStep())
}

func (m *Model) retryWorkflowStep(step *workflowFlowStep) (string, bool) {
	if m.workflowFlow == nil || step == nil || step.Status != workflowFlowDeadLetter {
		return "", false
	}
	now := time.Now()
	step.Status = workflowFlowReady
	step.DeadLetter = ""
	step.NextRetryAt = time.Time{}
	step.DeadLetterRef = nil
	step.UpdatedAt = now
	step.LastDetail = strings.TrimSpace(firstNonEmpty(step.LastDetail, "operator retry queued"))
	step.ApprovalState = workflowStepApprovalState(step.Policy.ApprovalRequired, false)
	step.ApprovedAt = time.Time{}
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = now
	m.syncTaskMemory()
	return step.Step.Title, true
}

func (m *Model) ackDeadLetterWorkflowStep() (string, bool) {
	if m.workflowFlow == nil {
		return "", false
	}
	return m.ackWorkflowStep(m.preferredDeadLetterStep())
}

func (m *Model) ackWorkflowStep(step *workflowFlowStep) (string, bool) {
	if m.workflowFlow == nil || step == nil || step.Status != workflowFlowDeadLetter || step.DeadLetterRef == nil {
		return "", false
	}
	now := time.Now()
	step.DeadLetterRef.Acked = true
	step.DeadLetterRef.AckedAt = now
	for deadIdx := range m.workflowFlow.DeadLetterEntries {
		if m.workflowFlow.DeadLetterEntries[deadIdx].Identity != step.Identity {
			continue
		}
		m.workflowFlow.DeadLetterEntries[deadIdx].Acked = true
		m.workflowFlow.DeadLetterEntries[deadIdx].AckedAt = now
	}
	step.LastDetail = strings.TrimSpace(firstNonEmpty(step.LastDetail, "dead-letter acknowledged"))
	step.UpdatedAt = now
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = now
	m.syncTaskMemory()
	return step.Step.Title, true
}

func (m *Model) skipDeadLetterWorkflowStep(reason string) (string, bool) {
	if m.workflowFlow == nil {
		return "", false
	}
	return m.skipWorkflowStep(m.preferredDeadLetterStep(), reason)
}

func (m *Model) skipWorkflowStep(step *workflowFlowStep, reason string) (string, bool) {
	if m.workflowFlow == nil || step == nil || step.Status != workflowFlowDeadLetter {
		return "", false
	}
	now := time.Now()
	step.Status = workflowFlowSkipped
	step.LastDetail = strings.TrimSpace(firstNonEmpty(reason, "operator skipped dead-letter step"))
	step.UpdatedAt = now
	if step.DeadLetterRef != nil {
		step.DeadLetterRef.Acked = true
		step.DeadLetterRef.AckedAt = now
	}
	for deadIdx := range m.workflowFlow.DeadLetterEntries {
		if m.workflowFlow.DeadLetterEntries[deadIdx].Identity != step.Identity {
			continue
		}
		m.workflowFlow.DeadLetterEntries[deadIdx].Acked = true
		m.workflowFlow.DeadLetterEntries[deadIdx].AckedAt = now
	}
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = now
	m.syncTaskMemory()
	return step.Step.Title, true
}

func (m Model) compensableWorkflowStep() *workflowFlowStep {
	if m.workflowFlow == nil {
		return nil
	}
	if step := m.selectedWorkflowStep(); step != nil {
		if (step.Status == workflowFlowDeadLetter || step.Status == workflowFlowFailed) &&
			len(step.Step.Rollback) > 0 &&
			strings.TrimSpace(step.Step.Capability) != "" {
			return step
		}
	}
	for idx := range m.workflowFlow.Steps {
		step := &m.workflowFlow.Steps[idx]
		if step.Status != workflowFlowDeadLetter && step.Status != workflowFlowFailed {
			continue
		}
		if len(step.Step.Rollback) == 0 {
			continue
		}
		if strings.TrimSpace(step.Step.Capability) == "" {
			continue
		}
		return step
	}
	return nil
}

func (m Model) preferredDeadLetterStep() *workflowFlowStep {
	if m.workflowFlow == nil {
		return nil
	}
	if step := m.selectedWorkflowStep(); step != nil && step.Status == workflowFlowDeadLetter {
		return step
	}
	for idx := range m.workflowFlow.Steps {
		if m.workflowFlow.Steps[idx].Status == workflowFlowDeadLetter {
			return &m.workflowFlow.Steps[idx]
		}
	}
	return nil
}

func (m Model) compensationRequestForStep(step *workflowFlowStep) (platformExecRequest, bool) {
	if step == nil {
		return platformExecRequest{}, false
	}
	op := &git.PlatformExecInfo{
		CapabilityID:    strings.TrimSpace(step.Step.Capability),
		Flow:            "rollback",
		Operation:       strings.TrimSpace(step.Step.Operation),
		ResourceID:      strings.TrimSpace(step.Step.ResourceID),
		Scope:           cloneStringMap(step.Step.Scope),
		RollbackPayload: cloneRaw(step.Step.Rollback),
	}
	return platformExecRequest{Op: op}, true
}

func (m *Model) findWorkflowFlowStep(op *git.PlatformExecInfo) *workflowFlowStep {
	if m.workflowFlow == nil || op == nil {
		return nil
	}
	for idx := range m.workflowFlow.Steps {
		if workflowStepMatchesPlatformOp(m.workflowFlow.Steps[idx].Step, op) {
			return &m.workflowFlow.Steps[idx]
		}
	}
	return nil
}

func (m Model) selectedWorkflowStep() *workflowFlowStep {
	if m.workflowFlow == nil || len(m.workflowFlow.Steps) == 0 {
		return nil
	}
	idx := m.workflowFlow.SelectedStepIndex
	if idx < 0 || idx >= len(m.workflowFlow.Steps) {
		return nil
	}
	return &m.workflowFlow.Steps[idx]
}

func (m *Model) moveWorkflowStepSelection(delta int) (string, bool) {
	if m.workflowFlow == nil || len(m.workflowFlow.Steps) == 0 || delta == 0 {
		return "", false
	}
	if m.workflowFlow.SelectedStepIndex < 0 || m.workflowFlow.SelectedStepIndex >= len(m.workflowFlow.Steps) {
		m.workflowFlow.SelectedStepIndex = 0
	}
	next := m.workflowFlow.SelectedStepIndex + delta
	if next < 0 {
		next = len(m.workflowFlow.Steps) - 1
	}
	if next >= len(m.workflowFlow.Steps) {
		next = 0
	}
	m.workflowFlow.SelectedStepIndex = next
	m.refreshWorkflowRunState("")
	m.syncTaskMemory()
	return strings.TrimSpace(firstNonEmpty(m.workflowFlow.Steps[next].Step.Title, m.workflowFlow.Steps[next].Identity)), true
}

func (m *Model) pauseWorkflowStep(step *workflowFlowStep, reason string) bool {
	if m.workflowFlow == nil || step == nil {
		return false
	}
	switch step.Status {
	case workflowFlowRunning, workflowFlowRetrying, workflowFlowReady, workflowFlowSuggested:
		step.Status = workflowFlowPaused
		step.UpdatedAt = time.Now()
		step.LastDetail = strings.TrimSpace(firstNonEmpty(reason, step.LastDetail, "paused"))
		m.refreshWorkflowRunState(reason)
		m.workflowFlow.UpdatedAt = step.UpdatedAt
		m.syncTaskMemory()
		return true
	default:
		return false
	}
}

func (m *Model) resumeWorkflowStep(step *workflowFlowStep) bool {
	if m.workflowFlow == nil || step == nil || step.Status != workflowFlowPaused {
		return false
	}
	step.Status = workflowFlowReady
	step.UpdatedAt = time.Now()
	step.LastDetail = strings.TrimSpace(firstNonEmpty(step.LastDetail, "resumed"))
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = step.UpdatedAt
	m.syncTaskMemory()
	return true
}

func (m *Model) approveWorkflowStep(step *workflowFlowStep, reason string) bool {
	if m.workflowFlow == nil || step == nil || !step.Policy.ApprovalRequired {
		return false
	}
	if strings.EqualFold(step.ApprovalState, "approved") {
		return false
	}
	now := time.Now()
	step.ApprovalState = workflowStepApprovalState(true, true)
	step.ApprovedAt = now
	step.UpdatedAt = now
	step.LastDetail = strings.TrimSpace(firstNonEmpty(reason, step.LastDetail, "operator approved"))
	m.refreshWorkflowRunState("")
	m.workflowFlow.UpdatedAt = now
	m.syncTaskMemory()
	return true
}

func workflowStepMatchesSuggestion(step prompt.WorkflowOrchestrationStep, suggestion git.Suggestion) bool {
	if suggestion.PlatformOp == nil {
		return false
	}
	return workflowStepMatchesPlatformOp(step, suggestion.PlatformOp)
}

func workflowStepMatchesPlatformOp(step prompt.WorkflowOrchestrationStep, op *git.PlatformExecInfo) bool {
	if op == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(step.Capability), strings.TrimSpace(op.CapabilityID)) {
		return false
	}
	if step.Flow != "" && !strings.EqualFold(strings.TrimSpace(step.Flow), strings.TrimSpace(op.Flow)) {
		if !(strings.EqualFold(strings.TrimSpace(step.Flow), "mutate") &&
			((strings.EqualFold(strings.TrimSpace(op.Flow), "validate") && len(step.Validate) > 0) ||
				(strings.EqualFold(strings.TrimSpace(op.Flow), "rollback") && len(step.Rollback) > 0))) {
			return false
		}
	}
	if step.Operation != "" && !strings.EqualFold(strings.TrimSpace(step.Operation), strings.TrimSpace(op.Operation)) {
		return false
	}
	if step.ResourceID != "" && !strings.EqualFold(strings.TrimSpace(step.ResourceID), strings.TrimSpace(op.ResourceID)) {
		return false
	}
	for key, value := range step.Scope {
		if !strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(op.Scope[key])) {
			return false
		}
	}
	if len(step.Query) == 0 {
		return true
	}
	for key, value := range step.Query {
		if !strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(op.Query[key])) {
			return false
		}
	}
	return true
}

func (m Model) workflowFlowCounts() (total, pending, running, done, failed int) {
	if m.workflowFlow == nil {
		return 0, 0, 0, 0, 0
	}
	total = len(m.workflowFlow.Steps)
	for _, step := range m.workflowFlow.Steps {
		switch step.Status {
		case workflowFlowDone, workflowFlowCompensated, workflowFlowSkipped:
			done++
		case workflowFlowRunning, workflowFlowWaitingValidation:
			running++
		case workflowFlowFailed, workflowFlowDeadLetter:
			failed++
		default:
			pending++
		}
	}
	return total, pending, running, done, failed
}

func workflowFlowTerminal(status workflowFlowStepStatus) bool {
	switch status {
	case workflowFlowDone, workflowFlowCompensated, workflowFlowSkipped, workflowFlowDeadLetter:
		return true
	default:
		return false
	}
}

func workflowStepApprovalState(required, approved bool) string {
	switch {
	case !required:
		return "not_required"
	case approved:
		return "approved"
	default:
		return "required"
	}
}

func workflowStepRetryBudget(step prompt.WorkflowOrchestrationStep) int {
	if strings.EqualFold(strings.TrimSpace(step.Flow), "inspect") || strings.EqualFold(strings.TrimSpace(step.Flow), "validate") {
		return 2
	}
	return 1
}

func workflowStepBackoff(step prompt.WorkflowOrchestrationStep) int {
	if strings.EqualFold(strings.TrimSpace(step.Flow), "inspect") || strings.EqualFold(strings.TrimSpace(step.Flow), "validate") {
		return 10
	}
	return 30
}

func workflowStepTimeout(step prompt.WorkflowOrchestrationStep) int {
	if strings.EqualFold(strings.TrimSpace(step.Flow), "mutate") || strings.EqualFold(strings.TrimSpace(step.Flow), "rollback") {
		return 45
	}
	return 25
}

func workflowStepApproval(platformID gitplatform.Platform, step prompt.WorkflowOrchestrationStep) bool {
	meta := platformRequestMeta(platformID, &git.PlatformExecInfo{
		CapabilityID: step.Capability,
		Flow:         step.Flow,
		Operation:    step.Operation,
	})
	return meta.ApprovalRequired
}

func workflowStepConcurrency(step prompt.WorkflowOrchestrationStep) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(step.Capability)),
		strings.ToLower(strings.TrimSpace(step.ResourceID)),
	}
	for key, value := range step.Scope {
		parts = append(parts, strings.ToLower(strings.TrimSpace(key))+"="+strings.ToLower(strings.TrimSpace(value)))
	}
	return strings.Trim(strings.Join(parts, ":"), ":")
}

func workflowStepSchedulerSafe(platformID gitplatform.Platform, step prompt.WorkflowOrchestrationStep) bool {
	meta := platformRequestMeta(platformID, &git.PlatformExecInfo{
		CapabilityID: step.Capability,
		Flow:         step.Flow,
		Operation:    step.Operation,
	})
	return meta.SchedulerSafe
}

func (m *Model) refreshWorkflowRunState(pausedReason string) {
	if m.workflowFlow == nil {
		return
	}
	m.workflowFlow.CheckpointVersion++
	if m.workflowFlow.ActiveLocks == nil {
		m.workflowFlow.ActiveLocks = map[string]string{}
	}
	clear(m.workflowFlow.ActiveLocks)
	for key, owner := range m.automationLocks {
		m.workflowFlow.ActiveLocks[key] = owner
	}
	if strings.TrimSpace(pausedReason) != "" {
		m.workflowFlow.PausedReason = strings.TrimSpace(pausedReason)
	}
	if !m.workflowHasPausedSteps() && strings.TrimSpace(pausedReason) == "" {
		m.workflowFlow.PausedReason = ""
	}
	if m.workflowFlow.SelectedStepIndex < 0 || m.workflowFlow.SelectedStepIndex >= len(m.workflowFlow.Steps) {
		m.workflowFlow.SelectedStepIndex = 0
	}
	var (
		nextRetryAt    time.Time
		nextRetryStep  string
		pendingApprove []string
		activeCount    int
	)
	for idx := range m.workflowFlow.Steps {
		step := &m.workflowFlow.Steps[idx]
		if step.Status == workflowFlowRetrying && !step.NextRetryAt.IsZero() {
			if nextRetryAt.IsZero() || step.NextRetryAt.Before(nextRetryAt) {
				nextRetryAt = step.NextRetryAt
				nextRetryStep = strings.TrimSpace(firstNonEmpty(step.Step.Title, step.Identity))
			}
		}
		if !workflowFlowTerminal(step.Status) {
			activeCount++
		}
		if step.Policy.ApprovalRequired &&
			!workflowFlowTerminal(step.Status) &&
			step.Status != workflowFlowRunning &&
			step.Status != workflowFlowWaitingValidation &&
			!strings.EqualFold(step.ApprovalState, "approved") {
			pendingApprove = append(pendingApprove, strings.TrimSpace(firstNonEmpty(step.Step.Title, step.Identity)))
		}
	}
	m.workflowFlow.NextRetryAt = nextRetryAt
	m.workflowFlow.NextRetryStep = nextRetryStep
	if len(pendingApprove) > 0 {
		m.workflowFlow.ApprovalDetail = fmt.Sprintf("%d step(s) pending approval: %s", len(pendingApprove), strings.Join(pendingApprove, ", "))
	} else if selected := m.selectedWorkflowStep(); selected != nil && strings.TrimSpace(selected.ApprovalState) != "" {
		m.workflowFlow.ApprovalDetail = strings.TrimSpace(firstNonEmpty(selected.Step.Title, selected.Identity)) + ": " + selected.ApprovalState
	} else {
		m.workflowFlow.ApprovalDetail = ""
	}
	switch {
	case m.workflowHasDeadLetters():
		m.workflowFlow.Health = "attention_required"
		m.workflowFlow.ApprovalState = "attention_required"
	case m.workflowHasPausedSteps():
		m.workflowFlow.Health = "paused"
		m.workflowFlow.ApprovalState = "paused"
	case m.workflowRequiresApproval():
		m.workflowFlow.Health = "approval_pending"
		m.workflowFlow.ApprovalState = "approval_required"
	case activeCount == 0 && len(m.workflowFlow.Steps) > 0:
		m.workflowFlow.Health = "complete"
		m.workflowFlow.ApprovalState = "clear"
	case m.workflowHasRunningSteps():
		m.workflowFlow.Health = "running"
		m.workflowFlow.ApprovalState = "clear"
	default:
		m.workflowFlow.Health = "ready"
		m.workflowFlow.ApprovalState = "clear"
	}
	m.workflowFlow.UpdatedAt = time.Now()
}

func (m Model) workflowRequiresApproval() bool {
	if m.workflowFlow == nil {
		return false
	}
	for _, step := range m.workflowFlow.Steps {
		if step.Policy.ApprovalRequired &&
			!workflowFlowTerminal(step.Status) &&
			step.Status != workflowFlowRunning &&
			step.Status != workflowFlowWaitingValidation &&
			!strings.EqualFold(step.ApprovalState, "approved") {
			return true
		}
	}
	return false
}

func (m Model) workflowHasRunningSteps() bool {
	if m.workflowFlow == nil {
		return false
	}
	for _, step := range m.workflowFlow.Steps {
		if step.Status == workflowFlowRunning || step.Status == workflowFlowWaitingValidation {
			return true
		}
	}
	return false
}

func (m *Model) recordDeadLetterEntry(step *workflowFlowStep, reason string) *DeadLetterEntry {
	if m.workflowFlow == nil || step == nil {
		return nil
	}
	entry := DeadLetterEntry{
		StepIndex:   step.Index,
		Identity:    step.Identity,
		Reason:      strings.TrimSpace(reason),
		At:          time.Now(),
		Attempts:    step.Attempt,
		NextRetryAt: step.NextRetryAt,
		LedgerRefs:  append([]string(nil), step.LedgerRefs...),
	}
	m.workflowFlow.DeadLetterEntries = append(m.workflowFlow.DeadLetterEntries, entry)
	ref := entry
	return &ref
}

func (m *Model) recordCompensationRef(step *workflowFlowStep, summary string) *CompensationRef {
	if m.workflowFlow == nil || step == nil {
		return nil
	}
	ref := CompensationRef{
		StepIndex:  step.Index,
		Identity:   step.Identity,
		Summary:    strings.TrimSpace(firstNonEmpty(summary, "compensation applied")),
		RecordedAt: time.Now(),
	}
	if len(step.LedgerRefs) > 0 {
		ref.LedgerID = step.LedgerRefs[len(step.LedgerRefs)-1]
	}
	m.workflowFlow.CompensationRefs = append(m.workflowFlow.CompensationRefs, ref)
	return &m.workflowFlow.CompensationRefs[len(m.workflowFlow.CompensationRefs)-1]
}

func appendIfMissing(values []string, candidate string) []string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return values
	}
	for _, existing := range values {
		if strings.EqualFold(strings.TrimSpace(existing), candidate) {
			return values
		}
	}
	return append(values, candidate)
}
