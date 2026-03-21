package policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/your-org/gitdex/internal/planning"
)

type Engine interface {
	Evaluate(ctx context.Context, plan *planning.Plan) (*planning.PolicyResult, error)
}

type DefaultEngine struct{}

func NewDefaultEngine() *DefaultEngine {
	return &DefaultEngine{}
}

func (e *DefaultEngine) Evaluate(_ context.Context, plan *planning.Plan) (*planning.PolicyResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("cannot evaluate nil plan")
	}

	riskFactors := e.collectRiskFactors(plan)
	verdict, reason, explanation := e.decide(plan, riskFactors)

	result := &planning.PolicyResult{
		Verdict:     verdict,
		Reason:      reason,
		RiskFactors: riskFactors,
		Explanation: explanation,
	}

	if verdict == planning.VerdictEscalated {
		result.RequiredApprovals = []string{"repository_admin"}
	}

	return result, nil
}

func (e *DefaultEngine) collectRiskFactors(plan *planning.Plan) []string {
	var factors []string

	if plan.Scope.Branch == "main" || plan.Scope.Branch == "master" {
		factors = append(factors, "targets protected branch")
	}
	if strings.EqualFold(plan.Scope.Environment, "production") {
		factors = append(factors, "targets production environment")
	}
	if plan.RiskLevel == planning.RiskHigh || plan.RiskLevel == planning.RiskCritical {
		factors = append(factors, fmt.Sprintf("overall risk level is %s", plan.RiskLevel))
	}
	for _, step := range plan.Steps {
		if !step.Reversible {
			factors = append(factors, fmt.Sprintf("step %d (%s) is irreversible", step.Sequence, step.Action))
		}
	}

	return factors
}

func (e *DefaultEngine) decide(plan *planning.Plan, riskFactors []string) (planning.PolicyVerdict, string, string) {
	switch plan.RiskLevel {
	case planning.RiskCritical:
		return planning.VerdictBlocked,
			"critical risk level requires explicit override",
			fmt.Sprintf("This plan targets a critical scope and cannot proceed automatically. Risk factors: %s. Request manual override or narrow the scope.",
				strings.Join(riskFactors, "; "))

	case planning.RiskHigh:
		return planning.VerdictEscalated,
			"high risk level requires approval",
			fmt.Sprintf("This plan has elevated risk and requires approval before execution. Risk factors: %s. An administrator must review and approve this plan.",
				strings.Join(riskFactors, "; "))

	case planning.RiskMedium:
		if len(riskFactors) > 0 {
			return planning.VerdictEscalated,
				"medium risk with additional factors requires review",
				fmt.Sprintf("This plan has moderate risk with contributing factors: %s. Review recommended before proceeding.",
					strings.Join(riskFactors, "; "))
		}
		return planning.VerdictAllowed,
			"medium risk within acceptable bounds",
			"This plan has moderate risk but falls within acceptable operational bounds."

	default:
		return planning.VerdictAllowed,
			"low risk operation within policy",
			"This plan is a low-risk operation and can proceed without additional approval."
	}
}
