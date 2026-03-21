package planning

import (
	"crypto/rand"
	"fmt"
	"time"
)

type PlanStatus string

const (
	PlanDraft          PlanStatus = "draft"
	PlanReviewRequired PlanStatus = "review_required"
	PlanApproved       PlanStatus = "approved"
	PlanBlocked        PlanStatus = "blocked"
	PlanExecuting      PlanStatus = "executing"
	PlanCompleted      PlanStatus = "completed"
)

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type PolicyVerdict string

const (
	VerdictAllowed   PolicyVerdict = "allowed"
	VerdictEscalated PolicyVerdict = "escalated"
	VerdictBlocked   PolicyVerdict = "blocked"
	VerdictDegraded  PolicyVerdict = "degraded"
)

type PlanScope struct {
	Owner       string   `json:"owner" yaml:"owner"`
	Repo        string   `json:"repo" yaml:"repo"`
	Branch      string   `json:"branch,omitempty" yaml:"branch,omitempty"`
	Environment string   `json:"environment,omitempty" yaml:"environment,omitempty"`
	Objects     []string `json:"objects,omitempty" yaml:"objects,omitempty"`
}

type PlanStep struct {
	Sequence    int       `json:"sequence" yaml:"sequence"`
	Action      string    `json:"action" yaml:"action"`
	Target      string    `json:"target" yaml:"target"`
	Description string    `json:"description" yaml:"description"`
	RiskLevel   RiskLevel `json:"risk_level" yaml:"risk_level"`
	Reversible  bool      `json:"reversible" yaml:"reversible"`
}

type PolicyResult struct {
	Verdict           PolicyVerdict `json:"verdict" yaml:"verdict"`
	Reason            string        `json:"reason" yaml:"reason"`
	RequiredApprovals []string      `json:"required_approvals,omitempty" yaml:"required_approvals,omitempty"`
	RiskFactors       []string      `json:"risk_factors,omitempty" yaml:"risk_factors,omitempty"`
	Explanation       string        `json:"explanation" yaml:"explanation"`
}

type Plan struct {
	SchemaVersion string        `json:"schema_version" yaml:"schema_version"`
	PlanID        string        `json:"plan_id" yaml:"plan_id"`
	TaskID        string        `json:"task_id,omitempty" yaml:"task_id,omitempty"`
	Status        PlanStatus    `json:"status" yaml:"status"`
	Intent        PlanIntent    `json:"intent" yaml:"intent"`
	Scope         PlanScope     `json:"scope" yaml:"scope"`
	Steps         []PlanStep    `json:"steps" yaml:"steps"`
	RiskLevel     RiskLevel     `json:"risk_level" yaml:"risk_level"`
	PolicyResult  *PolicyResult `json:"policy_result,omitempty" yaml:"policy_result,omitempty"`
	ExecutionMode ExecutionMode `json:"execution_mode,omitempty" yaml:"execution_mode,omitempty"`
	DeferredUntil *time.Time    `json:"deferred_until,omitempty" yaml:"deferred_until,omitempty"`
	EvidenceRefs  []string      `json:"evidence_refs,omitempty" yaml:"evidence_refs,omitempty"`
	CreatedAt     time.Time     `json:"created_at" yaml:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at" yaml:"updated_at"`
}

type ExecutionMode string

const (
	ModeObserve   ExecutionMode = "observe"
	ModeRecommend ExecutionMode = "recommend"
	ModeDryRun    ExecutionMode = "dry_run"
	ModeExecute   ExecutionMode = "execute"
)

func ValidExecutionMode(m ExecutionMode) bool {
	switch m {
	case ModeObserve, ModeRecommend, ModeDryRun, ModeExecute:
		return true
	}
	return false
}

type ApprovalAction string

const (
	ActionApprove ApprovalAction = "approve"
	ActionReject  ApprovalAction = "reject"
	ActionEdit    ApprovalAction = "edit"
	ActionDefer   ApprovalAction = "defer"
)

type ApprovalRecord struct {
	RecordID       string         `json:"record_id" yaml:"record_id"`
	PlanID         string         `json:"plan_id" yaml:"plan_id"`
	Action         ApprovalAction `json:"action" yaml:"action"`
	Actor          string         `json:"actor" yaml:"actor"`
	Reason         string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	PreviousStatus PlanStatus     `json:"previous_status" yaml:"previous_status"`
	NewStatus      PlanStatus     `json:"new_status" yaml:"new_status"`
	CreatedAt      time.Time      `json:"created_at" yaml:"created_at"`
}

type PlanIntent struct {
	Source     string `json:"source" yaml:"source"`
	RawInput   string `json:"raw_input" yaml:"raw_input"`
	ActionType string `json:"action_type" yaml:"action_type"`
}

func GenerateApprovalID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("approval_%x", b)
}

func GeneratePlanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("plan_%x", b)
}

func GenerateTaskID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("task_%x", b)
}
