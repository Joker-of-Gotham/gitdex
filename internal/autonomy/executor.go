package autonomy

import (
	"context"
	"fmt"
	"time"
)

type ActionHandler func(ctx context.Context, args map[string]string) (string, error)

type ExecutionResult struct {
	PlanID       string        `json:"plan_id"`
	Success      bool          `json:"success"`
	StepsRun     int           `json:"steps_run"`
	StepsTotal   int           `json:"steps_total"`
	Error        string        `json:"error,omitempty"`
	StepResults  []StepResult  `json:"step_results"`
	Duration     time.Duration `json:"duration"`
}

type StepResult struct {
	Order   int    `json:"order"`
	Action  string `json:"action"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

type PlanExecutor struct {
	handlers    map[string]ActionHandler
	guard       *Guardrails
	policyGate  *PolicyGate
	onProgress  func(planID string, step int, total int, action string)
}

func NewPlanExecutor(guard *Guardrails) *PlanExecutor {
	return &PlanExecutor{
		handlers: make(map[string]ActionHandler),
		guard:    guard,
	}
}

func (e *PlanExecutor) RegisterHandler(action string, handler ActionHandler) {
	e.handlers[action] = handler
}

func (e *PlanExecutor) SetProgressHandler(fn func(planID string, step int, total int, action string)) {
	e.onProgress = fn
}

func (e *PlanExecutor) SetPolicyGate(pg *PolicyGate) {
	e.policyGate = pg
}

func (e *PlanExecutor) Execute(ctx context.Context, plan ActionPlan) ExecutionResult {
	start := time.Now()
	result := ExecutionResult{
		PlanID:     plan.ID,
		StepsTotal: len(plan.Steps),
	}

	for i, step := range plan.Steps {
		if ctx.Err() != nil {
			result.Error = "cancelled"
			break
		}

		allowed, reason := e.guard.CheckPolicy(ActionPlan{Steps: []PlanStep{step}})
		if !allowed {
			result.StepResults = append(result.StepResults, StepResult{
				Order:  step.Order,
				Action: step.Action,
				Error:  fmt.Sprintf("blocked by guardrail: %s", reason),
			})
			result.Error = reason
			break
		}

		if e.policyGate != nil {
			stepRisk := e.guard.EvaluateRisk(ActionPlan{Steps: []PlanStep{step}})
			gate, policyReason := e.policyGate.Evaluate(step.Action, stepRisk)
			switch gate {
			case GateBlocked:
				msg := policyReason
				if msg == "" {
					msg = "policy gate blocked"
				}
				result.StepResults = append(result.StepResults, StepResult{
					Order:  step.Order,
					Action: step.Action,
					Error:  fmt.Sprintf("policy blocked: %s", msg),
				})
				result.Error = msg
			case GateManual:
				msg := policyReason
				if msg == "" {
					msg = "manual approval required"
				}
				result.StepResults = append(result.StepResults, StepResult{
					Order:  step.Order,
					Action: step.Action,
					Error:  fmt.Sprintf("policy manual: %s", msg),
				})
				result.Error = msg
			default:
				if need, n := e.policyGate.RequiresApproval(step.Action); need && n > 0 {
					msg := fmt.Sprintf("policy requires %d approval(s)", n)
					result.StepResults = append(result.StepResults, StepResult{
						Order:  step.Order,
						Action: step.Action,
						Error:  msg,
					})
					result.Error = msg
				}
			}
			if result.Error != "" {
				break
			}
		}

		if e.onProgress != nil {
			e.onProgress(plan.ID, i+1, len(plan.Steps), step.Action)
		}

		handler, ok := e.handlers[step.Action]
		if !ok {
			result.StepResults = append(result.StepResults, StepResult{
				Order:  step.Order,
				Action: step.Action,
				Error:  fmt.Sprintf("no handler for action: %s", step.Action),
			})
			result.Error = fmt.Sprintf("no handler: %s", step.Action)
			break
		}

		output, err := handler(ctx, step.Args)
		sr := StepResult{
			Order:   step.Order,
			Action:  step.Action,
			Success: err == nil,
			Output:  output,
		}
		if err != nil {
			sr.Error = err.Error()
			result.StepResults = append(result.StepResults, sr)
			result.Error = err.Error()
			break
		}

		result.StepResults = append(result.StepResults, sr)
		result.StepsRun = i + 1
	}

	result.Duration = time.Since(start)
	result.Success = result.Error == "" && result.StepsRun == result.StepsTotal
	return result
}
