package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	appchat "github.com/your-org/gitdex/internal/app/chat"
	"github.com/your-org/gitdex/internal/app/session"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/llm/adapter"
	"github.com/your-org/gitdex/internal/platform/config"
)

func TestResolveProvider_UsesConfiguredProvider(t *testing.T) {
	t.Setenv("GITDEX_LLM_PROVIDER", "")
	t.Setenv("GITDEX_LLM_API_KEY", "")
	t.Setenv("GITDEX_LLM_MODEL", "")
	t.Setenv("GITDEX_LLM_ENDPOINT", "")

	provider, err := resolveProvider(nil, config.LLMConfig{
		Provider: "deepseek",
		Model:    "deepseek-chat",
		APIKey:   "sk-test",
		Endpoint: "https://api.deepseek.com/v1",
	})
	if err != nil {
		t.Fatalf("resolveProvider returned error: %v", err)
	}
	if _, ok := provider.(*adapter.MockProvider); ok {
		t.Fatal("expected configured deepseek provider, got mock provider")
	}
}

func TestResolveProvider_UsesOllamaWithoutAPIKey(t *testing.T) {
	t.Setenv("GITDEX_LLM_PROVIDER", "")
	t.Setenv("GITDEX_LLM_API_KEY", "")
	t.Setenv("GITDEX_LLM_MODEL", "")
	t.Setenv("GITDEX_LLM_ENDPOINT", "")

	provider, err := resolveProvider(nil, config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3.2",
		Endpoint: "http://localhost:11434/v1",
	})
	if err != nil {
		t.Fatalf("resolveProvider returned error: %v", err)
	}
	if _, ok := provider.(*adapter.MockProvider); ok {
		t.Fatal("expected ollama provider, got mock provider")
	}
}

func TestResolveProvider_FallsBackToMockWhenHostedProviderHasNoAPIKey(t *testing.T) {
	t.Setenv("GITDEX_LLM_PROVIDER", "")
	t.Setenv("GITDEX_LLM_API_KEY", "")
	t.Setenv("GITDEX_LLM_MODEL", "")
	t.Setenv("GITDEX_LLM_ENDPOINT", "")

	provider, err := resolveProvider(nil, config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		Endpoint: "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatalf("resolveProvider returned error: %v", err)
	}
	if _, ok := provider.(*adapter.MockProvider); !ok {
		t.Fatalf("expected mock provider, got %T", provider)
	}
}

func TestChatExecuteSingleMessageWritesFileAndCommits(t *testing.T) {
	repoRoot := initAutonomyRepo(t)
	app := autonomyTestApp(t, repoRoot)
	defer func() { _ = app.StorageProvider.Close() }()

	restore := setAutonomyProviderForTest(&adapter.MockProvider{
		ChatCompletionFn: func(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return &adapter.ChatResponse{
				Content: `{
  "description": "add chat execute note",
  "steps": [
    {"order": 1, "action": "file.write", "args": {"path": "notes/chat-exec.txt", "content": "chat execute\n"}, "reversible": true, "description": "write the file"},
    {"order": 2, "action": "git.add", "args": {"path": "notes/chat-exec.txt"}, "reversible": true, "description": "stage the file"},
    {"order": 3, "action": "git.commit", "args": {"message": "add chat execute file"}, "reversible": false, "description": "commit the change"}
  ],
  "risk_level": "high",
  "rationale": "test chat execute"
}`,
			}, nil
		},
	})
	defer restore()

	var out bytes.Buffer
	var sessionCtx *session.TaskContext
	cmd := newChatCommand(&runtimeOptions{}, func() bootstrap.App { return app }, &sessionCtx, nil)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--execute", "add a note and commit it"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("chat --execute failed: %v\n%s", err, out.String())
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, "notes", "chat-exec.txt"))
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	if string(content) != "chat execute\n" {
		t.Fatalf("unexpected file content: %q", string(content))
	}

	logOut := runGit(t, repoRoot, "log", "-1", "--pretty=%s")
	if got := strings.TrimSpace(logOut); got != "add chat execute file" {
		t.Fatalf("latest commit = %q, want %q", got, "add chat execute file")
	}
}

func TestChatInteractiveBangExecutesIntent(t *testing.T) {
	repoRoot := initAutonomyRepo(t)
	app := autonomyTestApp(t, repoRoot)
	defer func() { _ = app.StorageProvider.Close() }()

	restore := setAutonomyProviderForTest(&adapter.MockProvider{
		ChatCompletionFn: func(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return &adapter.ChatResponse{
				Content: `{
  "description": "append interactive note",
  "steps": [
    {"order": 1, "action": "file.write", "args": {"path": "notes/interactive-chat.txt", "content": "interactive execute\n"}, "reversible": true, "description": "write the file"},
    {"order": 2, "action": "git.add", "args": {"path": "notes/interactive-chat.txt"}, "reversible": true, "description": "stage the file"},
    {"order": 3, "action": "git.commit", "args": {"message": "add interactive chat file"}, "reversible": false, "description": "commit the change"}
  ],
  "risk_level": "high",
  "rationale": "test interactive execute"
}`,
			}, nil
		},
	})
	defer restore()

	var out bytes.Buffer
	in := strings.NewReader("!add a note from interactive chat\nquit\n")
	var sessionCtx *session.TaskContext
	tc := getOrCreateSession(&sessionCtx, repoRoot, &runtimeOptions{})
	svc := appchat.NewService(&adapter.MockProvider{
		ChatCompletionFn: func(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return &adapter.ChatResponse{Content: "noop"}, nil
		},
	})

	cmd := newChatCommand(&runtimeOptions{}, func() bootstrap.App { return app }, &sessionCtx, nil)
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runInteractiveChat(cmd, svc, tc, "text", in, &out, app, false, "", "", autonomy.RiskHigh.String(), autonomy.RiskCritical.String()); err != nil {
		t.Fatalf("interactive chat failed: %v\n%s", err, out.String())
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, "notes", "interactive-chat.txt"))
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	if string(content) != "interactive execute\n" {
		t.Fatalf("unexpected file content: %q", string(content))
	}
}
