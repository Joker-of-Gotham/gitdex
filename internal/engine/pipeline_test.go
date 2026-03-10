package engine

import (
	"context"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
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

func TestParseLLMResponse_RejectsPlainText(t *testing.T) {
	input := "git stash pop"
	_, err := parseLLMResponse(nil, input)
	require.Error(t, err)
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
