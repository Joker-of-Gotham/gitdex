package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/llm/adapter"
)

type ActionPlan struct {
	ID           string     `json:"id"`
	Description  string     `json:"description"`
	Steps        []PlanStep `json:"steps"`
	RiskLevel    RiskLevel  `json:"-"`
	RiskLevelStr string     `json:"risk_level"`
	Rationale    string     `json:"rationale"`
	CreatedAt    time.Time  `json:"created_at"`
}

type PlanStep struct {
	Order       int               `json:"order"`
	Action      string            `json:"action"`
	Args        map[string]string `json:"args"`
	Reversible  bool              `json:"reversible"`
	Description string            `json:"description"`
}

type Planner struct {
	provider     adapter.Provider
	systemPrompt string
	repoContext  func() string
}

func NewPlanner(provider adapter.Provider, repoContextFn func() string) *Planner {
	return &Planner{
		provider:    provider,
		repoContext: repoContextFn,
		systemPrompt: `You are the Gitdex autonomous planner.
Analyze repository state and produce structured action plans only.

Return JSON only.

For repository analysis mode, return a JSON array:
[
  {
    "description": "short plan summary",
    "steps": [
      {
        "order": 1,
        "action": "git.status",
        "args": {"path": "."},
        "reversible": true,
        "description": "optional step description"
      }
    ],
    "risk_level": "low|medium|high|critical",
    "rationale": "why this plan is useful"
  }
]

For single-intent mode, return one JSON object with the same shape.

Supported actions:
- file.mkdir
- file.write
- file.append
- file.delete
- file.move
- file.copy
- git.status
- git.fetch
- git.pull
- git.push
- git.add
- git.reset
- git.restore
- git.commit
- git.commit.amend
- git.branch.create
- git.branch.delete
- git.branch.rename
- git.checkout
- git.merge
- git.rebase
- git.cherry-pick
- git.stash
- git.tag
- git.gc
- git.clean
- git.log
- github.pr.create
- github.pr.merge
- github.pr.close
- github.pr.comment
- github.pr.review
- github.issue.create
- github.issue.close
- github.issue.reopen
- github.issue.comment
- github.issue.label
- github.issue.assign
- github.release.create
- github.workflow.trigger

Risk guidance:
- low: read-only inspection, labels, comments, safe cleanup
- medium: commit, pull, merge, workflow dispatch, release creation
- high: push, branch creation, file modifications, cherry-pick, rebase
- critical: destructive or irreversible operations

Rules:
- Prefer the smallest useful plan.
- Use file actions only when a local clone is available in context.
- Use GitHub actions for remote-only contexts.
- Do not invent action names outside the supported set.
- If no action is needed in analysis mode, return [].
- Do not return Markdown or prose outside the JSON payload.`,
	}
}

func (p *Planner) AnalyzeAndPlan(ctx context.Context) ([]ActionPlan, error) {
	if p.provider == nil {
		return nil, fmt.Errorf("LLM provider not configured")
	}

	repoState := ""
	if p.repoContext != nil {
		repoState = p.repoContext()
	}

	messages := []adapter.ChatMessage{
		{Role: "system", Content: p.systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Current repository context:\n%s\n\nAnalyze the state and return JSON action plans.", repoState)},
	}

	resp, err := p.provider.ChatCompletion(ctx, adapter.ChatRequest{
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	return ParsePlans(resp.Content)
}

func (p *Planner) PlanFromUserIntent(ctx context.Context, intent string) (*ActionPlan, error) {
	if p.provider == nil {
		return nil, fmt.Errorf("LLM provider not configured")
	}

	repoState := ""
	if p.repoContext != nil {
		repoState = p.repoContext()
	}

	userPrompt := fmt.Sprintf("User intent: %s\n\nReturn one JSON object for the best action plan.", intent)
	if repoState != "" {
		userPrompt = fmt.Sprintf("Current repository context:\n%s\n\n%s", repoState, userPrompt)
	}

	messages := []adapter.ChatMessage{
		{Role: "system", Content: p.systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := p.provider.ChatCompletion(ctx, adapter.ChatRequest{
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	var plan ActionPlan
	if err := json.Unmarshal([]byte(resp.Content), &plan); err != nil {
		return nil, fmt.Errorf("parse plan failed: %w", err)
	}
	plan.CreatedAt = time.Now()
	if plan.RiskLevelStr != "" {
		plan.RiskLevel = ParseRiskLevel(plan.RiskLevelStr)
	}
	return &plan, nil
}

func ParsePlans(raw string) ([]ActionPlan, error) {
	var plans []ActionPlan
	if err := json.Unmarshal([]byte(raw), &plans); err != nil {
		return nil, fmt.Errorf("parse plans: %w", err)
	}
	now := time.Now()
	for i := range plans {
		if plans[i].CreatedAt.IsZero() {
			plans[i].CreatedAt = now
		}
		if plans[i].ID == "" {
			plans[i].ID = fmt.Sprintf("plan-%d", i+1)
		}
		if plans[i].RiskLevelStr != "" {
			plans[i].RiskLevel = ParseRiskLevel(plans[i].RiskLevelStr)
		}
	}
	return plans, nil
}
