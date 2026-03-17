package flow

import (
	"context"

	"github.com/Joker-of-Gotham/gitdex/internal/collector"
	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/helper"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

// CreativeResult holds the output of a creative flow round.
type CreativeResult struct {
	NewGitdexGoals []string
	NewCreative    []string
	Discarded      []string
}

// CreativeFlow generates new goals from repository + GitHub context.
type CreativeFlow struct {
	GitCollector *collector.GitCollector
	GHCollector  *collector.GitHubCollector
	Planner      *planner.CreativePlanner
	Reviewer     *helper.ProposalReviewer
	Store        *dotgitdex.Manager
	ContextLimit int // max tokens for model context window; 0 = 32768
}

// Run executes the creative flow: collect all context → Planner generates goals → review and triage.
func (f *CreativeFlow) Run(ctx context.Context) (*CreativeResult, error) {
	state, err := f.GitCollector.Refresh(ctx)
	if err != nil {
		return nil, err
	}

	assembler := NewContextAssembler("creative", f.ContextLimit)

	gitContent, _ := f.Store.ReadGitContent()
	gitContent = assembler.AddGit(gitContent)

	outputLog := dotgitdex.NewOutputLog(f.Store)
	rawOutput, _ := outputLog.ReadRecent(assembler.OutputRounds())
	output := assembler.AddOutput(rawOutput)

	index, _ := f.Store.ReadIndexText()
	index = assembler.AddIndex(index)

	goals, _ := f.Store.ReadGoalList()
	goalText := dotgitdex.FormatPendingGoals(goals)
	goalText = assembler.AddGoal(goalText)
	todoList := assembler.AddTodo(goalText)

	ghCtx, _ := f.GHCollector.Collect(ctx, state)
	githubText := ghCtx.FormatForPrompt()
	githubText = assembler.AddGitHub(githubText)

	creative, err := f.Planner.Generate(ctx, gitContent, output, index, goalText, todoList, githubText)
	if err != nil {
		return nil, err
	}

	review, err := f.Reviewer.ReviewProposals(ctx, creative.GitdexGoals, creative.CreativeGoals, goals)
	if err != nil {
		return &CreativeResult{
			NewGitdexGoals: creative.GitdexGoals,
			NewCreative:    creative.CreativeGoals,
		}, nil
	}

	if len(review.ApprovedGitdexGoals) > 0 {
		newGoals := make([]dotgitdex.Goal, 0, len(goals)+len(review.ApprovedGitdexGoals))
		newGoals = append(newGoals, goals...)
		for _, title := range review.ApprovedGitdexGoals {
			newGoals = append(newGoals, dotgitdex.Goal{Title: title})
		}
		_ = f.Store.WriteGoalList(newGoals)
	}

	if len(review.ApprovedCreative) > 0 {
		_ = f.Store.AppendCreativeProposal(review.ApprovedCreative)
	}
	if len(review.Discarded) > 0 {
		_ = f.Store.AppendDiscardedProposal(review.Discarded)
	}

	return &CreativeResult{
		NewGitdexGoals: review.ApprovedGitdexGoals,
		NewCreative:    review.ApprovedCreative,
		Discarded:      review.Discarded,
	}, nil
}
