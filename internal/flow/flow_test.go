package flow

import (
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/executor"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

func TestFlowRound_Fields(t *testing.T) {
	round := &FlowRound{
		Flow:       "maintain",
		Analysis:   "repo is clean",
		GitContent: "branch: main",
		Suggestions: []planner.SuggestionItem{
			{Name: "fetch", Action: planner.ActionSpec{Type: "git_command", Command: "git fetch"}, Reason: "sync"},
		},
	}
	if round.Flow != "maintain" {
		t.Errorf("expected Flow=maintain, got %s", round.Flow)
	}
	if len(round.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(round.Suggestions))
	}
	if round.Suggestions[0].Name != "fetch" {
		t.Errorf("expected suggestion name=fetch, got %s", round.Suggestions[0].Name)
	}
}

func TestRoundResult_Fields(t *testing.T) {
	result := &RoundResult{
		Executed: []ExecutedItem{
			{
				Item:   planner.SuggestionItem{Name: "a"},
				Result: &executor.ExecutionResult{Success: true, Stdout: "ok"},
			},
			{
				Item:   planner.SuggestionItem{Name: "b"},
				Result: &executor.ExecutionResult{Success: false, Stderr: "fail"},
			},
		},
		HasError:   true,
		NeedReplan: true,
	}
	if len(result.Executed) != 2 {
		t.Fatalf("expected 2 executed items, got %d", len(result.Executed))
	}
	if !result.HasError {
		t.Error("expected HasError=true")
	}
	if !result.NeedReplan {
		t.Error("expected NeedReplan=true")
	}
	if result.Executed[0].Result.Success != true {
		t.Error("expected first item success")
	}
	if result.Executed[1].Result.Success != false {
		t.Error("expected second item failure")
	}
}

func TestExecutedItem_Duration(t *testing.T) {
	item := ExecutedItem{
		Item: planner.SuggestionItem{Name: "test"},
		Result: &executor.ExecutionResult{
			Success:  true,
			Duration: 150 * time.Millisecond,
		},
	}
	if item.Result.Duration != 150*time.Millisecond {
		t.Errorf("expected 150ms duration, got %v", item.Result.Duration)
	}
}

func TestCreativeResult_Fields(t *testing.T) {
	cr := &CreativeResult{
		NewGitdexGoals: []string{"goal-a", "goal-b"},
		NewCreative:    []string{"idea-1"},
		Discarded:      []string{"bad-idea"},
	}
	if len(cr.NewGitdexGoals) != 2 {
		t.Errorf("expected 2 gitdex goals, got %d", len(cr.NewGitdexGoals))
	}
	if len(cr.NewCreative) != 1 {
		t.Errorf("expected 1 creative goal, got %d", len(cr.NewCreative))
	}
	if len(cr.Discarded) != 1 {
		t.Errorf("expected 1 discarded, got %d", len(cr.Discarded))
	}
}

func TestFlowRound_EmptySuggestions(t *testing.T) {
	round := &FlowRound{
		Flow:     "goal",
		Analysis: "all complete",
	}
	if len(round.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(round.Suggestions))
	}
}

func TestRoundResult_Empty(t *testing.T) {
	result := &RoundResult{}
	if result.HasError {
		t.Error("empty result should not have error")
	}
	if result.NeedReplan {
		t.Error("empty result should not need replan")
	}
	if len(result.Executed) != 0 {
		t.Error("empty result should have no executed items")
	}
}
