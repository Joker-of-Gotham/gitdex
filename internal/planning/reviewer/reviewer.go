package reviewer

import (
	"context"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/policy"
)

type PlanEdits struct {
	Branch        *string                 `json:"branch,omitempty" yaml:"branch,omitempty"`
	ExecutionMode *planning.ExecutionMode `json:"execution_mode,omitempty" yaml:"execution_mode,omitempty"`
}

type Reviewer struct {
	store  planning.PlanStore
	policy policy.Engine
}

func New(store planning.PlanStore, policyEngine policy.Engine) *Reviewer {
	return &Reviewer{store: store, policy: policyEngine}
}

func (r *Reviewer) Approve(ctx context.Context, planID, actor, reason string, mode *planning.ExecutionMode) error {
	plan, err := r.store.Get(planID)
	if err != nil {
		return fmt.Errorf("cannot find plan: %w", err)
	}

	newStatus, err := planning.ValidateTransition(plan.Status, planning.ActionApprove)
	if err != nil {
		return err
	}

	prevStatus := plan.Status
	if mode != nil {
		plan.ExecutionMode = *mode
	}
	if plan.ExecutionMode == "" {
		plan.ExecutionMode = planning.ModeExecute
	}

	record := &planning.ApprovalRecord{
		PlanID:         planID,
		Action:         planning.ActionApprove,
		Actor:          actor,
		Reason:         reason,
		PreviousStatus: prevStatus,
		NewStatus:      newStatus,
		CreatedAt:      time.Now().UTC(),
	}
	if err := r.store.SaveApproval(record); err != nil {
		return fmt.Errorf("failed to record approval: %w", err)
	}

	plan.Status = newStatus
	plan.UpdatedAt = time.Now().UTC()
	if err := r.store.Save(plan); err != nil {
		return fmt.Errorf("failed to save approved plan: %w", err)
	}
	return nil
}

func (r *Reviewer) Reject(_ context.Context, planID, actor, reason string) error {
	if reason == "" {
		return fmt.Errorf("rejection reason is required")
	}

	plan, err := r.store.Get(planID)
	if err != nil {
		return fmt.Errorf("cannot find plan: %w", err)
	}

	newStatus, err := planning.ValidateTransition(plan.Status, planning.ActionReject)
	if err != nil {
		return err
	}

	prevStatus := plan.Status

	record := &planning.ApprovalRecord{
		PlanID:         planID,
		Action:         planning.ActionReject,
		Actor:          actor,
		Reason:         reason,
		PreviousStatus: prevStatus,
		NewStatus:      newStatus,
		CreatedAt:      time.Now().UTC(),
	}
	if err := r.store.SaveApproval(record); err != nil {
		return fmt.Errorf("failed to record rejection: %w", err)
	}

	if err := r.store.UpdateStatus(planID, newStatus); err != nil {
		return fmt.Errorf("failed to update plan status: %w", err)
	}
	return nil
}

func (r *Reviewer) Defer(_ context.Context, planID, actor, reason string) error {
	plan, err := r.store.Get(planID)
	if err != nil {
		return fmt.Errorf("cannot find plan: %w", err)
	}

	newStatus, err := planning.ValidateTransition(plan.Status, planning.ActionDefer)
	if err != nil {
		return err
	}

	prevStatus := plan.Status

	record := &planning.ApprovalRecord{
		PlanID:         planID,
		Action:         planning.ActionDefer,
		Actor:          actor,
		Reason:         reason,
		PreviousStatus: prevStatus,
		NewStatus:      newStatus,
		CreatedAt:      time.Now().UTC(),
	}
	if err := r.store.SaveApproval(record); err != nil {
		return fmt.Errorf("failed to record deferral: %w", err)
	}

	if err := r.store.UpdateStatus(planID, newStatus); err != nil {
		return fmt.Errorf("failed to update plan status: %w", err)
	}
	return nil
}

func (r *Reviewer) Edit(ctx context.Context, planID, actor string, edits PlanEdits) error {
	plan, err := r.store.Get(planID)
	if err != nil {
		return fmt.Errorf("cannot find plan: %w", err)
	}

	_, err = planning.ValidateTransition(plan.Status, planning.ActionEdit)
	if err != nil {
		return err
	}

	prevStatus := plan.Status
	changed := false

	if edits.Branch != nil && *edits.Branch != plan.Scope.Branch {
		plan.Scope.Branch = *edits.Branch
		changed = true
	}
	if edits.ExecutionMode != nil && *edits.ExecutionMode != plan.ExecutionMode {
		plan.ExecutionMode = *edits.ExecutionMode
		changed = true
	}

	if !changed {
		return fmt.Errorf("no changes specified in edit request")
	}

	result, err := r.policy.Evaluate(ctx, plan)
	if err != nil {
		return fmt.Errorf("policy re-evaluation failed after edit: %w", err)
	}
	plan.PolicyResult = result

	var newStatus planning.PlanStatus
	switch result.Verdict {
	case planning.VerdictBlocked:
		newStatus = planning.PlanBlocked
	default:
		newStatus = planning.PlanReviewRequired
	}

	record := &planning.ApprovalRecord{
		PlanID:         planID,
		Action:         planning.ActionEdit,
		Actor:          actor,
		Reason:         fmt.Sprintf("plan edited: branch=%v, mode=%v", edits.Branch, edits.ExecutionMode),
		PreviousStatus: prevStatus,
		NewStatus:      newStatus,
		CreatedAt:      time.Now().UTC(),
	}
	if err := r.store.SaveApproval(record); err != nil {
		return fmt.Errorf("failed to record edit: %w", err)
	}

	plan.Status = newStatus
	plan.UpdatedAt = time.Now().UTC()
	if err := r.store.Save(plan); err != nil {
		return fmt.Errorf("failed to save edited plan: %w", err)
	}
	return nil
}
