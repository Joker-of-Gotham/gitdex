package autonomyexec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/llm/adapter"
	"github.com/your-org/gitdex/internal/platform/config"
	"github.com/your-org/gitdex/internal/storage"
)

func TestRunExecuteWritesFileAndCommits(t *testing.T) {
	repoRoot := initRepo(t)
	app := testApp(t, repoRoot)
	defer func() { _ = app.StorageProvider.Close() }()

	restore := SetProviderOverrideForTest(&adapter.MockProvider{
		ChatCompletionFn: func(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return &adapter.ChatResponse{
				Content: `{
  "description": "write smoke file",
  "steps": [
    {"order": 1, "action": "file.write", "args": {"path": "notes/exec.txt", "content": "shared runtime\n"}, "reversible": true, "description": "write"},
    {"order": 2, "action": "git.add", "args": {"path": "notes/exec.txt"}, "reversible": true, "description": "stage"},
    {"order": 3, "action": "git.commit", "args": {"message": "shared runtime commit"}, "reversible": false, "description": "commit"}
  ],
  "risk_level": "high",
  "rationale": "test"
}`,
			}, nil
		},
	})
	defer restore()

	result, err := Run(context.Background(), app, Request{
		RepoRoot:          repoRoot,
		Intent:            "write file",
		Execute:           true,
		AutoThreshold:     3,
		ApprovalThreshold: 4,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Mode != "execute" {
		t.Fatalf("Mode = %q", result.Mode)
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, "notes", "exec.txt"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(content) != "shared runtime\n" {
		t.Fatalf("unexpected content: %q", string(content))
	}

	if got := strings.TrimSpace(runGit(t, repoRoot, "log", "-1", "--pretty=%s")); got != "shared runtime commit" {
		t.Fatalf("latest commit = %q", got)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	repoRoot := t.TempDir()
	runGit(t, repoRoot, "init", "-b", "main")
	runGit(t, repoRoot, "config", "user.email", "autonomy@test.local")
	runGit(t, repoRoot, "config", "user.name", "Autonomy Test")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "commit", "-m", "seed")
	return repoRoot
}

func testApp(t *testing.T, repoRoot string) bootstrap.App {
	t.Helper()

	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatalf("new memory provider: %v", err)
	}

	return bootstrap.App{
		RepoRoot: repoRoot,
		Config: config.Config{
			FileConfig: config.FileConfig{
				Output: "text",
				Storage: config.StorageConfig{
					Type: string(storage.BackendMemory),
				},
				LLM: config.LLMConfig{
					Provider: "openai",
					Model:    "gpt-4o-mini",
					APIKey:   "test-key",
				},
			},
			Paths: config.ConfigPaths{
				WorkingDir:         repoRoot,
				RepositoryRoot:     repoRoot,
				RepositoryDetected: true,
			},
		},
		StorageProvider: provider,
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=echo",
		"LANG=C",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}
