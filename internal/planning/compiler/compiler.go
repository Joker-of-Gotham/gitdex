package compiler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/planning/intent"
)

// LLMClient is an optional dependency for LLM-assisted plan compilation.
type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// CompilerOptions configures the default compiler (deterministic + optional LLM).
type CompilerOptions struct {
	LLMClient   LLMClient // optional
	UseLLM      bool
	MaxSteps    int
	TimeoutSecs int
}

type Compiler interface {
	Compile(ctx context.Context, i intent.Intent) (*planning.Plan, error)
}

type DefaultCompiler struct {
	owner string
	repo  string
	opts  CompilerOptions
}

func New(owner, repo string) *DefaultCompiler {
	return NewWithOptions(owner, repo, CompilerOptions{})
}

func NewWithOptions(owner, repo string, opts CompilerOptions) *DefaultCompiler {
	if opts.MaxSteps <= 0 {
		opts.MaxSteps = 12
	}
	if opts.TimeoutSecs <= 0 {
		opts.TimeoutSecs = 60
	}
	return &DefaultCompiler{owner: owner, repo: repo, opts: opts}
}

func (c *DefaultCompiler) Compile(ctx context.Context, i intent.Intent) (*planning.Plan, error) {
	if strings.TrimSpace(i.RawInput) == "" {
		return nil, fmt.Errorf("empty intent: provide a goal or action")
	}

	now := time.Now().UTC()
	plan := &planning.Plan{
		SchemaVersion: "v1",
		PlanID:        planning.GeneratePlanID(),
		TaskID:        planning.GenerateTaskID(),
		Status:        planning.PlanDraft,
		Intent: planning.PlanIntent{
			Source:     string(i.Source),
			RawInput:   i.RawInput,
			ActionType: i.ActionType,
		},
		Scope: planning.PlanScope{
			Owner:  c.owner,
			Repo:   c.repo,
			Branch: i.Parameters["branch"],
		},
		Steps:     c.deriveSteps(i),
		RiskLevel: c.assessRisk(i),
		CreatedAt: now,
		UpdatedAt: now,
	}

	return plan, nil
}

// CompileWithLLM attempts LLM-assisted decomposition; falls back to deterministic Compile.
func (c *DefaultCompiler) CompileWithLLM(ctx context.Context, intentText string) (*planning.Plan, error) {
	intentText = strings.TrimSpace(intentText)
	if intentText == "" {
		return nil, fmt.Errorf("empty intent: provide a goal or action")
	}

	base := intent.NewChatIntent(intentText)
	if c.opts.UseLLM && c.opts.LLMClient != nil {
		sub, cancel := context.WithTimeout(ctx, time.Duration(c.opts.TimeoutSecs)*time.Second)
		defer cancel()
		if plan, err := c.compileViaLLM(sub, intentText); err == nil && plan != nil && len(plan.Steps) > 0 {
			return plan, nil
		}
	}
	return c.Compile(ctx, base)
}

type llmPlanStep struct {
	Action      string `json:"action"`
	Target      string `json:"target"`
	Description string `json:"description"`
	RiskLevel   string `json:"risk_level"`
}

type llmPlanEnvelope struct {
	Steps []llmPlanStep `json:"steps"`
}

func (c *DefaultCompiler) compileViaLLM(ctx context.Context, intentText string) (*planning.Plan, error) {
	sys := `You are a planning assistant for Gitdex. Given a user intent, respond with JSON ONLY, no markdown, in this exact shape:
{"steps":[{"action":"string","target":"string","description":"string","risk_level":"low|medium|high|critical"}]}
Rules:
- Produce between 1 and ` + fmt.Sprintf("%d", c.opts.MaxSteps) + ` steps.
- Actions should be short verb phrases (e.g. review, sync, test, open_pr).
- Targets should reference repository scope when relevant.`

	prompt := sys + "\n\nUser intent:\n" + intentText + "\n"
	out, err := c.opts.LLMClient.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	// Strip markdown fences if present
	if strings.HasPrefix(out, "```") {
		if idx := strings.Index(out, "\n"); idx != -1 {
			out = out[idx+1:]
		}
		out = strings.TrimSuffix(out, "```")
		out = strings.TrimSpace(out)
	}

	var env llmPlanEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		return nil, fmt.Errorf("parse LLM plan JSON: %w", err)
	}
	if len(env.Steps) == 0 {
		return nil, fmt.Errorf("LLM returned no steps")
	}
	if len(env.Steps) > c.opts.MaxSteps {
		env.Steps = env.Steps[:c.opts.MaxSteps]
	}

	now := time.Now().UTC()
	steps := make([]planning.PlanStep, 0, len(env.Steps))
	for i, s := range env.Steps {
		steps = append(steps, planning.PlanStep{
			Sequence:    i + 1,
			Action:      strings.TrimSpace(s.Action),
			Target:      strings.TrimSpace(s.Target),
			Description: strings.TrimSpace(s.Description),
			RiskLevel:   planningRisk(s.RiskLevel),
			Reversible:  true,
		})
	}

	return &planning.Plan{
		SchemaVersion: "v1",
		PlanID:        planning.GeneratePlanID(),
		TaskID:        planning.GenerateTaskID(),
		Status:        planning.PlanDraft,
		Intent: planning.PlanIntent{
			Source:     string(intent.SourceChat),
			RawInput:   intentText,
			ActionType: "llm_derived",
		},
		Scope: planning.PlanScope{
			Owner: c.owner,
			Repo:  c.repo,
		},
		Steps:     steps,
		RiskLevel: worstRisk(steps),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func planningRisk(s string) planning.RiskLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low":
		return planning.RiskLow
	case "medium":
		return planning.RiskMedium
	case "high":
		return planning.RiskHigh
	case "critical":
		return planning.RiskCritical
	default:
		return planning.RiskMedium
	}
}

func worstRisk(steps []planning.PlanStep) planning.RiskLevel {
	w := planning.RiskLow
	order := map[planning.RiskLevel]int{
		planning.RiskLow: 1, planning.RiskMedium: 2, planning.RiskHigh: 3, planning.RiskCritical: 4,
	}
	for _, s := range steps {
		if order[s.RiskLevel] > order[w] {
			w = s.RiskLevel
		}
	}
	return w
}

func (c *DefaultCompiler) deriveSteps(i intent.Intent) []planning.PlanStep {
	action := i.ActionType
	if action == "" || action == "chat_derived" {
		action = "review"
	}

	return []planning.PlanStep{
		{
			Sequence:    1,
			Action:      action,
			Target:      fmt.Sprintf("%s/%s", c.owner, c.repo),
			Description: fmt.Sprintf("Execute: %s", truncate(i.RawInput, 80)),
			RiskLevel:   c.assessRisk(i),
			Reversible:  true,
		},
	}
}

func (c *DefaultCompiler) assessRisk(i intent.Intent) planning.RiskLevel {
	branch := i.Parameters["branch"]
	env := i.Parameters["environment"]

	switch {
	case env == "production":
		return planning.RiskCritical
	case branch == "main" || branch == "master":
		return planning.RiskHigh
	case i.ActionType == "delete" || i.ActionType == "force_push":
		return planning.RiskHigh
	default:
		return planning.RiskLow
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
