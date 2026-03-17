package helper

import (
	"context"
	"errors"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

type mockLLM struct {
	response *llm.GenerateResponse
	err      error
}

func (m *mockLLM) Name() string { return "mock" }

func (m *mockLLM) Generate(_ context.Context, _ llm.GenerateRequest) (*llm.GenerateResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockLLM) GenerateStream(_ context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockLLM) IsAvailable(_ context.Context) bool { return true }

func (m *mockLLM) ModelInfo(_ context.Context) (*llm.ModelInfo, error) {
	return &llm.ModelInfo{Name: "mock"}, nil
}

func (m *mockLLM) ListModels(_ context.Context) ([]llm.ModelInfo, error) {
	return []llm.ModelInfo{{Name: "mock"}}, nil
}

func (m *mockLLM) SetModel(_ string) {}

func (m *mockLLM) SetModelForRole(_ llm.ModelRole, _ string) {}

func setupStore(t testing.TB) *dotgitdex.Manager {
	t.Helper()
	tmp := t.TempDir()
	store := dotgitdex.New(tmp)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestSelectForMaintain_cleanJSONStripsMarkdownFences(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: "```json\n{\"selected_knowledge\": [\"a.yaml\", \"b.yaml\"]}\n```",
		},
	}
	ks := &KnowledgeSelector{LLM: mock, Store: store, Language: "en"}
	got, err := ks.SelectForMaintain(context.Background(), "git", "out", "index")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.yaml", "b.yaml"}
	if len(got) != len(want) || (len(got) > 0 && (got[0] != want[0] || got[1] != want[1])) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSelectForMaintain_cleanJSONStripsPlainFences(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: "```\n{\"selected_knowledge\": [\"x.yaml\"]}\n```",
		},
	}
	ks := &KnowledgeSelector{LLM: mock, Store: store, Language: "en"}
	got, err := ks.SelectForMaintain(context.Background(), "git", "out", "index")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "x.yaml" {
		t.Errorf("got %v, want [x.yaml]", got)
	}
}

func TestSelectForMaintain_ValidJSON(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"selected_knowledge": ["init.yaml", "sync.yaml"]}`,
		},
	}
	ks := &KnowledgeSelector{LLM: mock, Store: store, Language: "en"}
	got, err := ks.SelectForMaintain(context.Background(), "git", "out", "index")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"init.yaml", "sync.yaml"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSelectForGoal_ValidJSON(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"selected_knowledge": ["staging.yaml", "platform_github.yaml"]}`,
		},
	}
	ks := &KnowledgeSelector{LLM: mock, Store: store, Language: "en"}
	got, err := ks.SelectForGoal(context.Background(), "git", "out", "index", "Deploy to prod", "- [ ] Step 1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"staging.yaml", "platform_github.yaml"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUpdateGoalCompletion_mergeGoalUpdates(t *testing.T) {
	store := setupStore(t)
	initial := []dotgitdex.Goal{
		{Title: "Goal A", Completed: false, Todos: []dotgitdex.Todo{{Title: "Todo A1", Completed: false}, {Title: "Todo A2", Completed: false}}},
		{Title: "Goal B", Completed: false, Todos: []dotgitdex.Todo{{Title: "Todo B1", Completed: false}}},
	}
	if err := store.WriteGoalList(initial); err != nil {
		t.Fatal(err)
	}
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"goals": [{"title": "Goal A", "completed": false, "todos": [{"title": "Todo A1", "completed": true}, {"title": "Todo A2", "completed": false}]}, {"title": "Goal B", "completed": true, "todos": [{"title": "Todo B1", "completed": true}]}]}`,
		},
	}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	if err := gm.UpdateGoalCompletion(context.Background(), "git", "output"); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadGoalList()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d goals, want 2", len(got))
	}
	if got[0].Todos[0].Completed != true || got[0].Todos[1].Completed != false {
		t.Errorf("Goal A todos: got A1=%v A2=%v, want A1=true A2=false", got[0].Todos[0].Completed, got[0].Todos[1].Completed)
	}
	if got[1].Completed != true || got[1].Todos[0].Completed != true {
		t.Errorf("Goal B: got completed=%v Todo B1=%v, want both true", got[1].Completed, got[1].Todos[0].Completed)
	}
}

func TestReviewProposals_ValidJSON(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"approved_gitdex": ["g1", "g2"], "approved_creative": ["c1"], "discarded": ["d1"]}`,
		},
	}
	pr := &ProposalReviewer{LLM: mock, Store: store, Language: "en"}
	got, err := pr.ReviewProposals(context.Background(), []string{"g1", "g2"}, []string{"c1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ApprovedGitdexGoals) != 2 || got.ApprovedGitdexGoals[0] != "g1" || got.ApprovedGitdexGoals[1] != "g2" {
		t.Errorf("ApprovedGitdexGoals: got %v, want [g1 g2]", got.ApprovedGitdexGoals)
	}
	if len(got.ApprovedCreative) != 1 || got.ApprovedCreative[0] != "c1" {
		t.Errorf("ApprovedCreative: got %v, want [c1]", got.ApprovedCreative)
	}
	if len(got.Discarded) != 1 || got.Discarded[0] != "d1" {
		t.Errorf("Discarded: got %v, want [d1]", got.Discarded)
	}
}

func TestSelectForMaintain_LLMError(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{err: errors.New("llm failed")}
	ks := &KnowledgeSelector{LLM: mock, Store: store, Language: "en"}
	_, err := ks.SelectForMaintain(context.Background(), "git", "out", "index")
	if err == nil {
		t.Error("expected error from LLM")
	}
}

func TestDecomposeGoal_ValidJSON(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"achievable":true,"category":"gitdex","reason":"can do","todos": [{"title": "Create new branch feature/test"}, {"title": "Add test file"}, {"title": "Commit and push"}]}`,
		},
	}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	todos, err := gm.DecomposeGoal(context.Background(), "Create feature branch with test file", "branch: master\nwork: clean")
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 3 {
		t.Fatalf("got %d todos, want 3", len(todos))
	}
	if todos[0].Title != "Create new branch feature/test" {
		t.Errorf("todo[0].Title = %q, want 'Create new branch feature/test'", todos[0].Title)
	}
	if todos[0].Completed {
		t.Error("todo[0] should not be completed")
	}
}

func TestDecomposeGoal_JSONWithMarkdownFences(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: "```json\n{\"achievable\":true,\"category\":\"gitdex\",\"reason\":\"ok\",\"todos\": [{\"title\": \"Step 1\"}, {\"title\": \"Step 2\"}]}\n```",
		},
	}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	todos, err := gm.DecomposeGoal(context.Background(), "Some goal", "branch: main")
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 {
		t.Fatalf("got %d todos, want 2", len(todos))
	}
}

func TestDecomposeGoal_EmptyTodos(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"achievable":true,"category":"gitdex","reason":"trivial","todos": []}`,
		},
	}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	todos, err := gm.DecomposeGoal(context.Background(), "Trivial goal", "branch: main")
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 0 {
		t.Errorf("got %d todos, want 0", len(todos))
	}
}

func TestDecomposeGoal_LLMError(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{err: errors.New("llm unavailable")}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	_, err := gm.DecomposeGoal(context.Background(), "Some goal", "branch: main")
	if err == nil {
		t.Error("expected error from LLM")
	}
}

func TestDecomposeGoal_SkipsEmptyTitles(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"achievable":true,"category":"gitdex","reason":"ok","todos": [{"title": "Valid task"}, {"title": ""}, {"title": "Another task"}]}`,
		},
	}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	todos, err := gm.DecomposeGoal(context.Background(), "Goal", "branch: main")
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 {
		t.Fatalf("got %d todos, want 2 (empty title should be skipped)", len(todos))
	}
}

func TestTriageGoal_CreativeCategory(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"achievable":false,"category":"creative","reason":"strategic suggestion","todos":[]}`,
		},
	}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	result, err := gm.TriageAndDecomposeGoal(context.Background(), "Improve code quality", "branch: main")
	if err != nil {
		t.Fatal(err)
	}
	if result.Category != "creative" {
		t.Errorf("expected category='creative', got %q", result.Category)
	}
	if result.Achievable {
		t.Error("expected achievable=false for creative goal")
	}
	if len(result.Todos) != 0 {
		t.Errorf("expected 0 todos for creative goal, got %d", len(result.Todos))
	}
}

func TestTriageGoal_DiscardCategory(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"achievable":false,"category":"discard","reason":"impossible goal","todos":[]}`,
		},
	}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	result, err := gm.TriageAndDecomposeGoal(context.Background(), "Launch rocket", "branch: main")
	if err != nil {
		t.Fatal(err)
	}
	if result.Category != "discard" {
		t.Errorf("expected category='discard', got %q", result.Category)
	}
}

func TestTriageGoal_DefaultCategoryFromAchievable(t *testing.T) {
	store := setupStore(t)
	mock := &mockLLM{
		response: &llm.GenerateResponse{
			Text: `{"achievable":true,"reason":"can do","todos":[{"title":"step 1"}]}`,
		},
	}
	gm := &GoalMaintainer{LLM: mock, Store: store, Language: "en"}
	result, err := gm.TriageAndDecomposeGoal(context.Background(), "Some goal", "branch: main")
	if err != nil {
		t.Fatal(err)
	}
	if result.Category != "gitdex" {
		t.Errorf("expected category='gitdex' when achievable=true and no category set, got %q", result.Category)
	}
}
