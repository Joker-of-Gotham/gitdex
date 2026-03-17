package contract

import "context"

// PlannerAPI defines planner boundary against the flow layer.
type PlannerAPI interface {
	Plan(ctx context.Context, payload PlannerPayload) (PlannerResult, error)
}

// RuntimeAPI defines executor boundary against the flow layer.
type RuntimeAPI interface {
	Execute(ctx context.Context, sequenceID int, suggestion SuggestionItem) ActionResult
}

// PlannerPayload is a normalized planner input contract.
type PlannerPayload struct {
	Flow       string `json:"flow"`
	GitContent string `json:"git_content"`
	Output     string `json:"output"`
	Knowledge  string `json:"knowledge"`
	Goal       string `json:"goal,omitempty"`
	TodoList   string `json:"todo_list,omitempty"`
}

// PlannerResult is a normalized planner output contract.
type PlannerResult struct {
	Analysis    string           `json:"analysis"`
	Suggestions []SuggestionItem `json:"suggestions"`
}

