package flow

import (
	"context"

	"github.com/Joker-of-Gotham/gitdex/internal/collector"
	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/helper"
	"github.com/Joker-of-Gotham/gitdex/internal/knowledge"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

// GoalFlow orchestrates the goal-completion flow:
// collect → helper selects knowledge → planner generates suggestions.
type GoalFlow struct {
	Collector    *collector.GitCollector
	Helper       *helper.KnowledgeSelector
	GoalHelper   *helper.GoalMaintainer
	Planner      *planner.GoalPlanner
	Store        *dotgitdex.Manager
	KnReader     *knowledge.Reader
	ContextLimit int // max tokens for model context window; 0 = 32768
}

// Run executes one round of the goal-completion flow.
// Only the FIRST pending goal and its sub-tasks are sent to the planner.
func (f *GoalFlow) Run(ctx context.Context) (*FlowRound, error) {
	_, err := f.Collector.Refresh(ctx)
	if err != nil {
		return nil, err
	}

	assembler := NewContextAssembler("goal", f.ContextLimit)

	gitContent, err := f.Store.ReadGitContent()
	if err != nil {
		return nil, err
	}
	gitContent = assembler.AddGit(gitContent)

	outputLog := dotgitdex.NewOutputLog(f.Store)
	rawOutput, _ := outputLog.ReadRecent(assembler.OutputRounds())
	output := assembler.AddOutput(rawOutput)

	index, _ := f.Store.ReadIndexText()
	index = assembler.AddIndex(index)

	goals, _ := f.Store.ReadGoalList()
	pendingGoals := dotgitdex.PendingGoals(goals)
	if len(pendingGoals) == 0 {
		return &FlowRound{
			Flow:       "goal",
			Analysis:   "All goals completed.",
			GitContent: gitContent,
		}, nil
	}

	// Focus on the FIRST pending goal only — not all goals at once
	activeGoal := pendingGoals[0]
	goal := activeGoal.Title
	todoList := dotgitdex.FormatSingleGoalTodos(activeGoal)
	goal = assembler.AddGoal(goal)
	todoList = assembler.AddTodo(todoList)

	paths, err := f.Helper.SelectForGoal(ctx, gitContent, output, index, goal, todoList)
	if err != nil {
		paths = nil
	}

	knowledgeCtx, _ := f.KnReader.ReadByPaths(paths)
	knowledgeCtx = assembler.AddKnowledge(knowledgeCtx)

	suggestions, analysis, err := f.Planner.Plan(ctx, gitContent, output, knowledgeCtx, goal, todoList)
	if err != nil {
		return nil, err
	}

	return &FlowRound{
		Flow:          "goal",
		Suggestions:   suggestions,
		Analysis:      analysis,
		GitContent:    gitContent,
		TokensUsed:    assembler.TokensUsed(),
		TokensBudget:  assembler.TokensBudget(),
		TokenSections: assembler.SectionUsage(),
	}, nil
}

// UpdateGoalProgress uses the helper LLM to update goal completion status.
func (f *GoalFlow) UpdateGoalProgress(ctx context.Context) error {
	gitContent, _ := f.Store.ReadGitContent()
	outputLog := dotgitdex.NewOutputLog(f.Store)
	output, _ := outputLog.ReadRecent(3)
	return f.GoalHelper.UpdateGoalCompletion(ctx, gitContent, output)
}

// IsComplete checks if all goals are done and the repo is clean.
func (f *GoalFlow) IsComplete(ctx context.Context) bool {
	goals, err := f.Store.ReadGoalList()
	if err != nil || len(goals) == 0 {
		return true
	}
	pending := dotgitdex.PendingGoals(goals)
	if len(pending) > 0 {
		return false
	}
	state, err := f.Collector.Collect(ctx)
	if err != nil {
		return false
	}
	return len(state.WorkingTree) == 0 && len(state.StagingArea) == 0
}
