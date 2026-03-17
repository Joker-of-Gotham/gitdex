package tui

import (
	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/executor"
	"github.com/Joker-of-Gotham/gitdex/internal/flow"
	"github.com/Joker-of-Gotham/gitdex/internal/helper"
)

// TUI message bus definitions (section-independent transport contracts).

type initMsg struct{}

type flowRoundMsg struct {
	flow  string
	round *flow.FlowRound
	err   error
}

type executionResultMsg struct {
	index  int
	result *executor.ExecutionResult
	err    error
}

type goalProgressMsg struct {
	goals []dotgitdex.Goal
}

type cruiseTickMsg struct{}
type flowRetryMsg struct{}
type cruiseCycleCompleteMsg struct{}

type analysisDoneMsg struct{}

type goalDecomposedMsg struct {
	goalTitle string
	todos     []dotgitdex.Todo
	err       error
}

type goalTriageMsg struct {
	goalTitle string
	result    *helper.GoalTriageResult
	err       error
}

type goalProgressUpdatedMsg struct {
	err    error
	replan bool // if true, replan after updating (failure path)
}

type creativeResultMsg struct {
	result *flow.CreativeResult
	err    error
}

type llmConnectivityMsg struct {
	role      string // "helper" or "planner"
	provider  string
	model     string
	ok        bool
	err       string
	latencyMs int64
}

type ollamaModelsMsg struct {
	models []OllamaModelInfo
	err    error
}

type gitRefreshMsg struct {
	info GitSnapshot
}

