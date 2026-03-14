package engine

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPipeline(t *testing.T) {
	p := NewPipeline("zen")
	require.NotNil(t, p)
}

func TestPipeline_AnalyzeNoLLM(t *testing.T) {
	p := NewPipeline("zen")
	state := &status.GitState{
		StagingArea: []git.FileStatus{{Path: "a.go", StagingCode: git.StatusAdded}},
	}
	_, err := p.Analyze(context.Background(), state, nil, AnalyzeOptions{})
	require.Error(t, err, "should fail without LLM provider")
}

func TestParseLLMResponse_FullObject(t *testing.T) {
	input := `{"analysis":"Repo has untracked files.","suggestions":[{"action":"Stage files","argv":["git","add","."],"reason":"Untracked files","risk":"safe","interaction":"auto"}]}`
	result, err := parseLLMResponse(nil, input)
	require.NoError(t, err)
	assert.Equal(t, "Repo has untracked files.", result.analysis)
	require.Len(t, result.suggestions, 1)
	assert.Equal(t, "Stage files", result.suggestions[0].Action)
	assert.Equal(t, []string{"git", "add", "."}, result.suggestions[0].Command)
}

func TestParseLLMResponse_ArrayFallback(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Upstream: "origin/main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURLValid: true,
			PushURLValid:  true,
			Reachable:     true,
		}},
	}
	input := `[{"action":"Push changes","argv":["git","push","origin","main"],"reason":"Ahead of remote","risk":"safe","interaction":"auto"}]`
	result, err := parseLLMResponse(state, input)
	require.NoError(t, err)
	require.Len(t, result.suggestions, 1)
	assert.Equal(t, []string{"git", "push", "origin", "main"}, result.suggestions[0].Command)
}

func TestParseLLMResponse_WithMarkdownFences(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Upstream: "origin/main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURLValid: true,
			PushURLValid:  true,
			Reachable:     true,
		}},
	}
	input := "```json\n" + `{"analysis":"test","suggestions":[{"action":"Pull","argv":["git","pull"],"reason":"Behind","risk":"safe","interaction":"auto"}]}` + "\n```"
	result, err := parseLLMResponse(state, input)
	require.NoError(t, err)
	require.Len(t, result.suggestions, 1)
	assert.Equal(t, "Pull", result.suggestions[0].Action)
}

func TestParseLLMResponse_EmptyArray(t *testing.T) {
	input := `{"analysis":"All clean","suggestions":[]}`
	result, err := parseLLMResponse(nil, input)
	require.NoError(t, err)
	assert.Len(t, result.suggestions, 0)
	assert.Equal(t, "All clean", result.analysis)
}

func TestParseLLMResponse_RejectsShortPlainText(t *testing.T) {
	input := "git stash pop"
	_, err := parseLLMResponse(nil, input)
	require.Error(t, err, "text under 20 chars should still be rejected")
}

func TestParseLLMResponse_AcceptsLongPlainAnalysis(t *testing.T) {
	input := "The repository is clean and up to date. No pending changes found. All branches are synchronized with the remote."
	result, err := parseLLMResponse(nil, input)
	require.NoError(t, err, "long plain text should be accepted as analysis fallback")
	assert.Contains(t, result.analysis, "repository is clean")
	assert.Empty(t, result.suggestions)
}

func TestParseLLMResponse_RejectsEmptyFence(t *testing.T) {
	input := "```"
	_, err := parseLLMResponse(nil, input)
	require.Error(t, err)
}

func TestParseLLMResponse_RiskLevels(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Upstream: "origin/main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURLValid: true,
			PushURLValid:  true,
			Reachable:     true,
		}},
	}
	input := `{"analysis":"","suggestions":[{"action":"Force push","argv":["git","push","--force"],"reason":"Rewrite","risk":"dangerous","interaction":"auto"}]}`
	result, err := parseLLMResponse(state, input)
	require.NoError(t, err)
	require.Len(t, result.suggestions, 1)
	assert.Equal(t, git.RiskDangerous, result.suggestions[0].RiskLevel)
}

func TestParseLLMResponse_InvalidCommand(t *testing.T) {
	input := `{"analysis":"","suggestions":[{"action":"List files","argv":["ls","-la"],"reason":"See files","risk":"safe","interaction":"auto"}]}`
	result, err := parseLLMResponse(nil, input)
	require.NoError(t, err)
	assert.Len(t, result.suggestions, 0)
	assert.Len(t, result.rejected, 1)
}

func TestParseLLMResponse_NeedsInput(t *testing.T) {
	input := `{"analysis":"No remote","suggestions":[{"action":"Add remote","argv":["git","remote","add","origin","<remote-url>"],"reason":"Need remote","risk":"safe","interaction":"needs_input","inputs":[{"key":"remote_url","label":"Remote URL","placeholder":"https://github.com/user/repo.git","arg_index":4}]}]}`
	result, err := parseLLMResponse(nil, input)
	require.NoError(t, err)
	require.Len(t, result.suggestions, 1)
	assert.Equal(t, git.NeedsInput, result.suggestions[0].Interaction)
	require.Len(t, result.suggestions[0].Inputs, 1)
	assert.Equal(t, "remote_url", result.suggestions[0].Inputs[0].Key)
	assert.Equal(t, "Remote URL", result.suggestions[0].Inputs[0].Label)
	assert.Equal(t, 4, result.suggestions[0].Inputs[0].ArgIndex)
}

func TestParseLLMResponse_LLMSource(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Upstream: "origin/main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURLValid: true,
			PushURLValid:  true,
			Reachable:     true,
		}},
	}
	input := `{"analysis":"test","suggestions":[{"action":"Push","argv":["git","push"],"reason":"Ahead","risk":"safe","interaction":"auto"}]}`
	result, err := parseLLMResponse(state, input)
	require.NoError(t, err)
	require.Len(t, result.suggestions, 1)
	assert.Equal(t, git.SourceLLM, result.suggestions[0].Source)
}

func TestParseLLMResponse_DoesNotHardRejectPushWithoutUsableRemote(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Upstream: "origin/main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURLValid: true,
			PushURLValid:  false,
			Reachable:     false,
		}},
	}
	input := `{"analysis":"test","suggestions":[{"action":"Push","argv":["git","push","-u","origin","master"],"reason":"Ahead","risk":"safe","interaction":"auto"}]}`
	result, err := parseLLMResponse(state, input)
	require.NoError(t, err)
	assert.Len(t, result.suggestions, 1)
	assert.Empty(t, result.rejected)
}

func TestParseLLMResponse_AllowsPushWithValidUncheckedSSHRemote(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Upstream: "origin/main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:                "origin",
			FetchURL:            "git@github.com:user/repo.git",
			PushURL:             "git@github.com:user/repo.git",
			FetchURLValid:       true,
			PushURLValid:        true,
			ReachabilityChecked: false,
			Reachable:           false,
		}},
	}
	input := `{"analysis":"test","suggestions":[{"action":"Push","argv":["git","push","origin","main"],"reason":"Ahead","risk":"safe","interaction":"auto"}]}`
	result, err := parseLLMResponse(state, input)
	require.NoError(t, err)
	require.Len(t, result.suggestions, 1)
	assert.Empty(t, result.rejected)
}

func TestSuppressRepeatedSuccessfulSuggestions(t *testing.T) {
	suggestions := []git.Suggestion{
		{Action: "Pull", Command: []string{"git", "pull", "origin", "master"}},
		{Action: "Commit", Command: []string{"git", "commit", "-m", "test"}},
	}
	recent := []prompt.OperationRecord{
		{Command: "git pull origin master", Result: "success"},
	}
	filtered := suppressRepeatedSuccessfulSuggestions(suggestions, recent)
	require.Len(t, filtered, 1)
	assert.Equal(t, "Commit", filtered[0].Action)
}

func TestSuppressRepeatedSuccessfulSuggestions_FiltersViewedAdvisory(t *testing.T) {
	suggestions := []git.Suggestion{
		{Action: "Review .gitignore", Interaction: git.InfoOnly},
		{Action: "Commit", Command: []string{"git", "commit", "-m", "test"}},
	}
	recent := []prompt.OperationRecord{
		{Type: "viewed", Action: "Review .gitignore", Result: "viewed"},
	}

	filtered := suppressRepeatedSuccessfulSuggestions(suggestions, recent)
	require.Len(t, filtered, 1)
	assert.Equal(t, "Commit", filtered[0].Action)
}

func TestParseLLMResponse_StripsQuotedCommitMessageArg(t *testing.T) {
	input := `{"analysis":"test","suggestions":[{"action":"Commit","argv":["git","commit","-m","\"Fix commit failure\""],"reason":"Commit changes","risk":"safe","interaction":"auto"}]}`
	result, err := parseLLMResponse(nil, input)
	require.NoError(t, err)
	require.Len(t, result.suggestions, 1)
	assert.Equal(t, []string{"git", "commit", "-m", "Fix commit failure"}, result.suggestions[0].Command)
}

func TestPipelineAnalyze_UsesStructuredThinking(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURLValid: true,
			PushURLValid:  true,
			Reachable:     true,
		}},
	}
	provider := fakeLLMProvider{
		response: &llm.GenerateResponse{
			Text:     `{"analysis":"ok","suggestions":[]}`,
			Thinking: "checked branch, working tree, and remotes",
			Raw:      `{"output_text":"{\"analysis\":\"ok\",\"suggestions\":[]}"}`,
		},
	}
	pipeline := NewPipelineWithLLM("zen", provider, config.LLMConfig{})

	result, err := pipeline.Analyze(context.Background(), state, nil, AnalyzeOptions{})
	require.NoError(t, err)
	assert.Equal(t, "checked branch, working tree, and remotes", result.Thinking)
	assert.Contains(t, result.Trace.RawResponse, "\"output_text\"")
}

func TestPipelineAnalyze_RepairsRejectedSuggestions(t *testing.T) {
	state := &status.GitState{
		LocalBranch:   git.BranchInfo{Name: "main"},
		LocalBranches: []string{"main", "dev"},
		RemoteInfos: []git.RemoteInfo{{
			Name:          "origin",
			FetchURLValid: true,
			PushURLValid:  true,
			Reachable:     true,
		}},
		Remotes: []string{"origin"},
	}
	provider := &sequenceLLMProvider{
		responses: []*llm.GenerateResponse{
			{
				Text: `{"analysis":"test","suggestions":[{"action":"Switch to main","argv":["git","checkout","main"],"reason":"go to current branch","risk":"safe","interaction":"auto"},{"action":"Inspect status","argv":["git","status"],"reason":"safe inspection","risk":"safe","interaction":"auto"}]}`,
			},
			{
				Text: `[{"action":"Inspect branch details","argv":["git","branch","-vv"],"reason":"current branch is already checked out; inspect tracking state instead","risk":"safe","interaction":"auto"}]`,
			},
		},
	}
	pipeline := NewPipelineWithLLM("zen", provider, config.LLMConfig{})

	result, err := pipeline.Analyze(context.Background(), state, nil, AnalyzeOptions{})
	require.NoError(t, err)
	require.Len(t, result.Suggestions, 2)
	assert.Equal(t, []string{"git", "status"}, result.Suggestions[0].Command)
	assert.Equal(t, []string{"git", "branch", "-vv"}, result.Suggestions[1].Command)
	assert.Contains(t, result.Analysis, "Repaired 1 invalid suggestion")
	assert.Contains(t, strings.Join(result.Trace.Rejected, "\n"), "invalid for current repository state")
}

type fakeLLMProvider struct {
	response *llm.GenerateResponse
}

func (f fakeLLMProvider) Name() string { return "fake" }

func (f fakeLLMProvider) Generate(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return f.response, nil
}

func (f fakeLLMProvider) GenerateStream(context.Context, llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	return nil, assert.AnError
}

func (f fakeLLMProvider) IsAvailable(context.Context) bool { return true }

func (f fakeLLMProvider) ModelInfo(context.Context) (*llm.ModelInfo, error) {
	return &llm.ModelInfo{Name: "fake"}, nil
}

func (f fakeLLMProvider) ListModels(context.Context) ([]llm.ModelInfo, error) { return nil, nil }

func (f fakeLLMProvider) SetModel(string) {}

func (f fakeLLMProvider) SetModelForRole(llm.ModelRole, string) {}

type sequenceLLMProvider struct {
	responses []*llm.GenerateResponse
	mu        sync.Mutex
}

func (s *sequenceLLMProvider) Name() string { return "sequence" }

func (s *sequenceLLMProvider) Generate(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.responses) == 0 {
		return nil, assert.AnError
	}
	resp := s.responses[0]
	s.responses = s.responses[1:]
	return resp, nil
}

func (s *sequenceLLMProvider) GenerateStream(context.Context, llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	return nil, assert.AnError
}

func (s *sequenceLLMProvider) IsAvailable(context.Context) bool { return true }

func (s *sequenceLLMProvider) ModelInfo(context.Context) (*llm.ModelInfo, error) {
	return &llm.ModelInfo{Name: "sequence"}, nil
}

func (s *sequenceLLMProvider) ListModels(context.Context) ([]llm.ModelInfo, error) { return nil, nil }

func (s *sequenceLLMProvider) SetModel(string) {}

func (s *sequenceLLMProvider) SetModelForRole(llm.ModelRole, string) {}
