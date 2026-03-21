package planning

import "fmt"

var validTransitions = map[PlanStatus]map[ApprovalAction]PlanStatus{
	PlanReviewRequired: {
		ActionApprove: PlanApproved,
		ActionReject:  PlanBlocked,
		ActionDefer:   PlanDraft,
		ActionEdit:    PlanReviewRequired,
	},
	PlanBlocked: {
		ActionEdit: PlanReviewRequired,
	},
	PlanDraft: {
		ActionEdit: PlanDraft,
	},
}

func ValidateTransition(current PlanStatus, action ApprovalAction) (PlanStatus, error) {
	actions, ok := validTransitions[current]
	if !ok {
		return "", fmt.Errorf("plan in status %q does not accept review actions", current)
	}
	next, ok := actions[action]
	if !ok {
		return "", transitionError(current, action)
	}
	return next, nil
}

func transitionError(current PlanStatus, action ApprovalAction) error {
	switch {
	case current == PlanBlocked && action == ActionApprove:
		return fmt.Errorf("blocked plans cannot be directly approved; edit the plan to reduce risk or request an override first")
	case current == PlanApproved:
		return fmt.Errorf("plan is already approved and cannot be modified through review actions")
	case current == PlanExecuting:
		return fmt.Errorf("plan is currently executing and cannot be modified")
	case current == PlanCompleted:
		return fmt.Errorf("plan has already completed and cannot be modified")
	default:
		return fmt.Errorf("action %q is not valid for plan in status %q", action, current)
	}
}
