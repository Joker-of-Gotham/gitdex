package regression

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/engine"
	gitcli "github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveRepoRegression_DoesNotRepeatRecentSuccessfulSyncCommands(t *testing.T) {
	repoDir := envAny("GITDEX_REAL_REGRESSION_REPO", "GITMANUAL_REAL_REGRESSION_REPO")
	if repoDir == "" {
		t.Skip("set GITDEX_REAL_REGRESSION_REPO (or legacy GITMANUAL_REAL_REGRESSION_REPO) to run live regression")
	}
	require.DirExists(t, repoDir)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(repoDir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	gitExec, err := gitcli.NewCLIExecutor()
	require.NoError(t, err)

	watcher := status.NewStatusWatcher(gitExec)
	stateCtx, cancelState := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelState()

	state, err := watcher.GetStatus(stateCtx)
	require.NoError(t, err)
	require.NotEmpty(t, state.LocalBranch.Name, "live regression expects a named branch")

	remote := primaryRemoteName(state)
	require.NotEmpty(t, remote, "live regression expects at least one remote")

	t.Logf(
		"live repo state: branch=%s upstream=%s working=%d staged=%d remotes=%d",
		state.LocalBranch.Name,
		state.LocalBranch.Upstream,
		len(state.WorkingTree),
		len(state.StagingArea),
		len(state.RemoteInfos),
	)

	client := newLiveRegressionClient(t)
	pipeline := engine.NewPipelineWithLLM("zen", client, nil)

	recentOps := []prompt.OperationRecord{
		{Command: strings.Join([]string{"git", "pull", remote, state.LocalBranch.Name}, " "), Result: "success"},
		{Command: strings.Join([]string{"git", "push", remote, state.LocalBranch.Name}, " "), Result: "success"},
	}

	analyzeCtx, cancelAnalyze := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancelAnalyze()

	result, err := pipeline.Analyze(analyzeCtx, state, recentOps, engine.AnalyzeOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimSpace(result.Analysis))

	pullCmd := normalizeCommandIdentity(strings.Join([]string{"git", "pull", remote, state.LocalBranch.Name}, " "))
	pushCmd := normalizeCommandIdentity(strings.Join([]string{"git", "push", remote, state.LocalBranch.Name}, " "))
	for _, suggestion := range result.Suggestions {
		cmd := normalizeCommandIdentity(strings.Join(suggestion.Command, " "))
		assert.NotEqual(t, pullCmd, cmd, "LLM should not repeat a successful pull command")
		assert.NotEqual(t, pushCmd, cmd, "LLM should not repeat a successful push command")
	}
}

func TestLiveRepoRegression_WorktreeOnlyDoesNotJumpToCommit(t *testing.T) {
	if envAny("GITDEX_ENABLE_LIVE_REGRESSION", "GITMANUAL_ENABLE_LIVE_REGRESSION") != "1" {
		t.Skip("set GITDEX_ENABLE_LIVE_REGRESSION=1 (or legacy GITMANUAL_ENABLE_LIVE_REGRESSION=1) to run live regression")
	}

	repoDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(repoDir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	gitExec, err := gitcli.NewCLIExecutor()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, _, err = gitExec.Exec(ctx, "init")
	require.NoError(t, err)
	_, _, err = gitExec.Exec(ctx, "config", "user.name", "gitdex regression")
	require.NoError(t, err)
	_, _, err = gitExec.Exec(ctx, "config", "user.email", "gitdex@example.com")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("tracked.txt", []byte("v1\n"), 0o600))
	_, _, err = gitExec.Exec(ctx, "add", "tracked.txt")
	require.NoError(t, err)
	_, _, err = gitExec.Exec(ctx, "commit", "-m", "initial commit")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("tracked.txt", []byte("v2\n"), 0o600))

	watcher := status.NewStatusWatcher(gitExec)
	state, err := watcher.GetStatus(ctx)
	require.NoError(t, err)
	require.Len(t, state.WorkingTree, 1)
	require.Empty(t, state.StagingArea)

	client := newLiveRegressionClient(t)
	pipeline := engine.NewPipelineWithLLM("zen", client, nil)
	recentOps := []prompt.OperationRecord{
		{Command: `git commit -m "Fix commit failure"`, Result: "failed: nothing added to commit"},
	}

	analyzeCtx, cancelAnalyze := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancelAnalyze()

	result, err := pipeline.Analyze(analyzeCtx, state, recentOps, engine.AnalyzeOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimSpace(result.Analysis))
	require.NotEmpty(t, result.Suggestions)

	for _, suggestion := range result.Suggestions {
		cmd := normalizeCommandIdentity(strings.Join(suggestion.Command, " "))
		assert.NotContains(t, cmd, "git commit -a", "LLM should not skip staging with git commit -a")
		assert.NotContains(t, cmd, "git commit -am", "LLM should not skip staging with git commit -am")
		assert.NotContains(t, strings.ToLower(suggestion.Action), "commit", "worktree-only state should not jump straight to commit")
	}
}

func newLiveRegressionClient(t *testing.T) *ollama.OllamaClient {
	t.Helper()

	endpoint := envAny("GITDEX_REAL_REGRESSION_ENDPOINT", "GITMANUAL_REAL_REGRESSION_ENDPOINT")
	client := ollama.NewClient(endpoint, "")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	models, err := client.ListModels(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, models, "live regression expects at least one local Ollama model")

	model := envAny("GITDEX_REAL_REGRESSION_MODEL", "GITMANUAL_REAL_REGRESSION_MODEL")
	if model == "" {
		for _, preferred := range []string{"qwen3:8b", "qwen2.5:3b", "gemma3:latest"} {
			if modelExists(models, preferred) {
				model = preferred
				break
			}
		}
	}
	if model == "" {
		model = models[0].Name
	}

	client.SetModel(model)
	t.Logf("live regression using Ollama model: %s", model)
	return client
}

func modelExists(models []llm.ModelInfo, name string) bool {
	for _, m := range models {
		if strings.EqualFold(strings.TrimSpace(m.Name), strings.TrimSpace(name)) {
			return true
		}
	}
	return false
}

func primaryRemoteName(state *status.GitState) string {
	if state == nil {
		return ""
	}
	for _, info := range state.RemoteInfos {
		if strings.TrimSpace(info.Name) != "" {
			return info.Name
		}
	}
	if len(state.Remotes) > 0 {
		return strings.TrimSpace(state.Remotes[0])
	}
	return ""
}

func normalizeCommandIdentity(cmd string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(cmd))), " ")
}

func envAny(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}
