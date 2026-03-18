package flow

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/contract"
	"github.com/Joker-of-Gotham/gitdex/internal/executor"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

// Orchestrator coordinates the three flows based on the current mode.
type Orchestrator struct {
	Maintain *MaintainFlow
	Goal     *GoalFlow
	Creative *CreativeFlow
	Runner   *executor.Runner
	Logger   *executor.ExecutionLogger
	Mode     string        // manual, auto, cruise
	Interval time.Duration // cruise scan interval

	Machine *StateMachine

	mu       sync.Mutex
	roundSeq int
	attempt  int
	sliceSeq int
}

// RunMaintainRound executes a single maintenance round.
func (o *Orchestrator) RunMaintainRound(ctx context.Context) (*FlowRound, error) {
	o.ensureMachine()
	o.Runner.ResetRound()
	_ = o.Machine.Transition(EventAnalyzeStart)
	o.Logger.SetFlow("maintain")
	round, err := o.Maintain.Run(ctx)
	if err != nil {
		_ = o.Machine.Transition(EventAnalyzeFail)
		return nil, err
	}
	o.applyRoundMeta(round, "maintain")
	_ = o.Machine.Transition(EventAnalyzeSuccess)
	return round, nil
}

// RunGoalRound executes a single goal-completion round.
func (o *Orchestrator) RunGoalRound(ctx context.Context) (*FlowRound, error) {
	o.ensureMachine()
	o.Runner.ResetRound()
	_ = o.Machine.Transition(EventAnalyzeStart)
	o.Logger.SetFlow("goal")
	round, err := o.Goal.Run(ctx)
	if err != nil {
		_ = o.Machine.Transition(EventAnalyzeFail)
		return nil, err
	}
	o.applyRoundMeta(round, "goal")
	_ = o.Machine.Transition(EventAnalyzeSuccess)
	return round, nil
}

// RunCreativeRound executes the creative flow.
func (o *Orchestrator) RunCreativeRound(ctx context.Context) (*CreativeResult, error) {
	o.ensureMachine()
	_ = o.Machine.Transition(EventAnalyzeStart)
	o.Logger.SetFlow("creative")
	result, err := o.Creative.Run(ctx)
	if err != nil {
		_ = o.Machine.Transition(EventAnalyzeFail)
		return nil, err
	}
	_ = o.Machine.Transition(EventAnalyzeSuccess)
	return result, nil
}

// ExecuteRound runs all suggestions in a FlowRound through the Runner.
func (o *Orchestrator) ExecuteRound(ctx context.Context, round *FlowRound) (*RoundResult, error) {
	o.ensureMachine()
	_ = o.Machine.Transition(EventExecuteStart)
	if round == nil || len(round.Suggestions) == 0 {
		_ = o.Machine.Transition(EventExecuteSuccess)
		_ = o.Machine.Transition(EventRefreshDone)
		return &RoundResult{}, nil
	}

	result := &RoundResult{}
	for i, item := range round.Suggestions {
		execResult := o.Runner.ExecuteSuggestion(ctx, i+1, item)
		result.Executed = append(result.Executed, ExecutedItem{
			Item:   item,
			Result: execResult,
		})
		if !execResult.Success {
			result.HasError = true
			result.NeedReplan = true
			_ = o.Machine.Transition(EventExecuteFail)
			break
		}
	}
	if !result.HasError {
		_ = o.Machine.Transition(EventExecuteSuccess)
		_ = o.Machine.Transition(EventRefreshDone)
	}

	if err := o.Logger.Flush(); err != nil {
		return result, fmt.Errorf("flush logger: %w", err)
	}

	return result, nil
}

// ExecuteSingleSuggestion executes one suggestion and returns the result.
func (o *Orchestrator) ExecuteSingleSuggestion(ctx context.Context, seqID int, item contract.SuggestionItem) *executor.ExecutionResult {
	return o.Runner.ExecuteSuggestion(ctx, seqID, item)
}

// FlushLog persists the current execution log.
func (o *Orchestrator) FlushLog() error {
	return o.Logger.Flush()
}

func (o *Orchestrator) ensureMachine() {
	if o.Machine == nil {
		o.Machine = NewStateMachine()
	}
}

func (o *Orchestrator) applyRoundMeta(round *FlowRound, flow string) {
	if round == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.roundSeq++
	if flow == "creative" || o.roundSeq == 1 {
		o.sliceSeq++
	}
	if flow == "goal" || flow == "maintain" {
		o.attempt++
	}
	round.RoundID = "round-" + strconv.Itoa(o.roundSeq)
	round.AttemptID = "attempt-" + strconv.Itoa(o.attempt)
	round.SliceID = "slice-" + strconv.Itoa(o.sliceSeq)
}

// IsMaintainClean checks if the repo needs maintenance.
func (o *Orchestrator) IsMaintainClean(ctx context.Context) bool {
	if o.Maintain == nil {
		return true
	}
	return o.Maintain.IsClean(ctx)
}

// IsGoalComplete checks if all goals are done.
func (o *Orchestrator) IsGoalComplete(ctx context.Context) bool {
	return o.Goal.IsComplete(ctx)
}

// UpdateGoalProgress updates goal completion via helper LLM.
func (o *Orchestrator) UpdateGoalProgress(ctx context.Context) error {
	return o.Goal.UpdateGoalProgress(ctx)
}

// SetProviders updates all helper and planner LLM references at runtime.
func (o *Orchestrator) SetProviders(helperLLM, plannerLLM llm.LLMProvider) {
	if o.Maintain != nil {
		if o.Maintain.Helper != nil {
			o.Maintain.Helper.LLM = helperLLM
		}
		if o.Maintain.Planner != nil {
			o.Maintain.Planner.LLM = plannerLLM
		}
	}
	if o.Goal != nil {
		if o.Goal.Helper != nil {
			o.Goal.Helper.LLM = helperLLM
		}
		if o.Goal.GoalHelper != nil {
			o.Goal.GoalHelper.LLM = helperLLM
		}
		if o.Goal.Planner != nil {
			o.Goal.Planner.LLM = plannerLLM
		}
	}
	if o.Creative != nil {
		if o.Creative.Planner != nil {
			o.Creative.Planner.LLM = plannerLLM
		}
		if o.Creative.Reviewer != nil {
			o.Creative.Reviewer.LLM = helperLLM
		}
	}
}
