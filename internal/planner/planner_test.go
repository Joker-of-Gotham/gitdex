package planner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

type mockLLM struct {
	generateFunc func(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error)
}

func (m *mockLLM) Name() string { return "mock" }

func (m *mockLLM) Generate(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &llm.GenerateResponse{Text: `{"analysis":"","suggestions":[]}`}, nil
}

func (m *mockLLM) GenerateStream(ctx context.Context, req llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{Done: true}
	close(ch)
	return ch, nil
}

func (m *mockLLM) IsAvailable(ctx context.Context) bool { return true }

func (m *mockLLM) ModelInfo(ctx context.Context) (*llm.ModelInfo, error) {
	return &llm.ModelInfo{Name: "mock"}, nil
}

func (m *mockLLM) ListModels(ctx context.Context) ([]llm.ModelInfo, error) {
	return []llm.ModelInfo{{Name: "mock"}}, nil
}

func (m *mockLLM) SetModel(name string) {}

func (m *mockLLM) SetModelForRole(role llm.ModelRole, name string) {}

func TestCleanJSON(t *testing.T) {
	fenced := "```json\n{\"analysis\":\"ok\",\"suggestions\":[{\"name\":\"test\",\"action\":{\"type\":\"git_command\"},\"reason\":\"r\"}]}\n```"
	mock := &mockLLM{
		generateFunc: func(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
			return &llm.GenerateResponse{Text: fenced}, nil
		},
	}
	p := &MaintenancePlanner{LLM: mock}
	sug, analysis, err := p.Plan(context.Background(), "git", "out", "knowledge")
	if err != nil {
		t.Fatal(err)
	}
	if analysis != "ok" {
		t.Errorf("analysis = %q, want ok", analysis)
	}
	if len(sug) != 1 || sug[0].Name != "test" {
		t.Errorf("suggestions = %v", sug)
	}
}

func TestMaintenancePlanner_Plan(t *testing.T) {
	resp := plannerResponse{
		Analysis: "repo is clean",
		Suggestions: []SuggestionItem{{
			Name:   "fetch all",
			Action: ActionSpec{Type: "git_command", Command: "git fetch --all"},
			Reason: "sync remotes",
		}},
	}
	raw, _ := json.Marshal(resp)
	mock := &mockLLM{
		generateFunc: func(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
			return &llm.GenerateResponse{Text: string(raw)}, nil
		},
	}
	p := &MaintenancePlanner{LLM: mock}
	sug, analysis, err := p.Plan(context.Background(), "git", "out", "knowledge")
	if err != nil {
		t.Fatal(err)
	}
	if analysis != "repo is clean" {
		t.Errorf("analysis = %q", analysis)
	}
	if len(sug) != 1 || sug[0].Name != "fetch all" || sug[0].Action.Command != "git fetch --all" {
		t.Errorf("suggestions = %v", sug)
	}
}

func TestGoalPlanner_Plan(t *testing.T) {
	resp := plannerResponse{
		Analysis: "goal progress",
		Suggestions: []SuggestionItem{{
			Name:   "commit",
			Action: ActionSpec{Type: "git_command", Command: "git commit -m \"done\""},
			Reason: "finish goal",
		}},
	}
	raw, _ := json.Marshal(resp)
	mock := &mockLLM{
		generateFunc: func(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
			return &llm.GenerateResponse{Text: string(raw)}, nil
		},
	}
	p := &GoalPlanner{LLM: mock}
	sug, analysis, err := p.Plan(context.Background(), "git", "out", "knowledge", "my goal", "[]")
	if err != nil {
		t.Fatal(err)
	}
	if analysis != "goal progress" {
		t.Errorf("analysis = %q", analysis)
	}
	if len(sug) != 1 || sug[0].Name != "commit" {
		t.Errorf("suggestions = %v", sug)
	}
}

func TestCreativePlanner_Generate(t *testing.T) {
	out := CreativeOutput{
		Analysis:      "opportunities",
		GitdexGoals:   []string{"PR workflow", "CI setup"},
		CreativeGoals: []string{"Consider docs"},
	}
	raw, _ := json.Marshal(out)
	mock := &mockLLM{
		generateFunc: func(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
			return &llm.GenerateResponse{Text: string(raw)}, nil
		},
	}
	p := &CreativePlanner{LLM: mock}
	result, err := p.Generate(context.Background(), "git", "out", "index", "goals", "todo", "github")
	if err != nil {
		t.Fatal(err)
	}
	if result.Analysis != "opportunities" {
		t.Errorf("Analysis = %q", result.Analysis)
	}
	if len(result.GitdexGoals) != 2 || result.GitdexGoals[0] != "PR workflow" {
		t.Errorf("GitdexGoals = %v", result.GitdexGoals)
	}
	if len(result.CreativeGoals) != 1 || result.CreativeGoals[0] != "Consider docs" {
		t.Errorf("CreativeGoals = %v", result.CreativeGoals)
	}
}
