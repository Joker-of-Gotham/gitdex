package flow

import (
	"context"

	"github.com/Joker-of-Gotham/gitdex/internal/collector"
	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/helper"
	"github.com/Joker-of-Gotham/gitdex/internal/knowledge"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

// MaintainFlow orchestrates the repository maintenance flow:
// collect → helper selects knowledge → planner generates suggestions.
type MaintainFlow struct {
	Collector    *collector.GitCollector
	Helper       *helper.KnowledgeSelector
	Planner      *planner.MaintenancePlanner
	Store        *dotgitdex.Manager
	KnReader     *knowledge.Reader
	ContextLimit int // max tokens for model context window; 0 = 32768
}

// Run executes one round of the maintenance flow.
func (f *MaintainFlow) Run(ctx context.Context) (*FlowRound, error) {
	_, err := f.Collector.Refresh(ctx)
	if err != nil {
		return nil, err
	}

	assembler := NewContextAssembler("maintain", f.ContextLimit)

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

	paths, err := f.Helper.SelectForMaintain(ctx, gitContent, output, index)
	if err != nil {
		paths = nil
	}

	knowledgeCtx, _ := f.KnReader.ReadByPaths(paths)
	knowledgeCtx = assembler.AddKnowledge(knowledgeCtx)

	suggestions, analysis, err := f.Planner.Plan(ctx, gitContent, output, knowledgeCtx)
	if err != nil {
		return nil, err
	}

	return &FlowRound{
		Flow:          "maintain",
		Suggestions:   suggestions,
		Analysis:      analysis,
		GitContent:    gitContent,
		TokensUsed:    assembler.TokensUsed(),
		TokensBudget:  assembler.TokensBudget(),
		TokenSections: assembler.SectionUsage(),
	}, nil
}

// IsClean checks if the repository needs no further maintenance.
func (f *MaintainFlow) IsClean(ctx context.Context) bool {
	state, err := f.Collector.Collect(ctx)
	if err != nil {
		return false
	}
	return len(state.WorkingTree) == 0 && len(state.StagingArea) == 0
}
