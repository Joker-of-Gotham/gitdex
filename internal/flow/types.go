package flow

import (
	"github.com/Joker-of-Gotham/gitdex/internal/contract"
	"github.com/Joker-of-Gotham/gitdex/internal/executor"
)

// FlowRound holds the output of a single flow round (analysis + suggestions).
type FlowRound struct {
	Flow          string // maintain, goal, creative
	RoundID       string
	AttemptID     string
	SliceID       string
	Suggestions   []contract.SuggestionItem
	Analysis      string
	GitContent    string         // snapshot used for this round
	TokensUsed    int            // estimated input tokens for this round
	TokensBudget  int            // max input tokens available
	TokenSections map[string]int // per-section token usage
}

// RoundResult captures what happened after executing a FlowRound.
type RoundResult struct {
	Executed   []ExecutedItem
	Skipped    []string
	HasError   bool
	NeedReplan bool
}

// ExecutedItem pairs a suggestion with its result.
type ExecutedItem struct {
	Item   contract.SuggestionItem
	Result *executor.ExecutionResult
}
