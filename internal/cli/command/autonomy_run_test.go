package command

import (
	"bytes"
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

func TestAutonomyRunOnceExecuteWritesFileAndCommits(t *testing.T) {
	repoRoot := initAutonomyRepo(t)
	app := autonomyTestApp(t, repoRoot)
	defer func() { _ = app.StorageProvider.Close() }()

	restore := setAutonomyProviderForTest(&adapter.MockProvider{
		ChatCompletionFn: func(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return &adapter.ChatResponse{
				Content: `{
  "description": "add notes file",
  "steps": [
    {"order": 1, "action": "file.write", "args": {"path": "notes/auto.txt", "content": "hello autonomy\n"}, "reversible": true, "description": "write the file"},
    {"order": 2, "action": "git.add", "args": {"path": "notes/auto.txt"}, "reversible": true, "description": "stage the file"},
    {"order": 3, "action": "git.commit", "args": {"message": "add auto file"}, "reversible": false, "description": "commit the change"}
  ],
  "risk_level": "high",
  "rationale": "test execution"
}`,
			}, nil
		},
	})
	defer restore()

	var out bytes.Buffer
	cmd := newAutonomyGroupCommand(&runtimeOptions{}, func() bootstrap.App { return app })
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"run-once", "--intent", "add a note", "--execute", "--auto-threshold", "high"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("autonomy run-once execute failed: %v\n%s", err, out.String())
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, "notes", "auto.txt"))
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	if string(content) != "hello autonomy\n" {
		t.Fatalf("unexpected file content: %q", string(content))
	}

	logOut := runGit(t, repoRoot, "log", "-1", "--pretty=%s")
	if got := strings.TrimSpace(logOut); got != "add auto file" {
		t.Fatalf("latest commit = %q, want %q", got, "add auto file")
	}
}

func TestAutonomyRunOncePlanOnlyDoesNotMutateRepo(t *testing.T) {
	repoRoot := initAutonomyRepo(t)
	app := autonomyTestApp(t, repoRoot)
	defer func() { _ = app.StorageProvider.Close() }()

	restore := setAutonomyProviderForTest(&adapter.MockProvider{
		ChatCompletionFn: func(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return &adapter.ChatResponse{
				Content: `{
  "description": "add notes file",
  "steps": [
    {"order": 1, "action": "file.write", "args": {"path": "notes/preview.txt", "content": "preview only\n"}, "reversible": true, "description": "write the file"}
  ],
  "risk_level": "high",
  "rationale": "test preview"
}`,
			}, nil
		},
	})
	defer restore()

	var out bytes.Buffer
	cmd := newAutonomyGroupCommand(&runtimeOptions{}, func() bootstrap.App { return app })
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"run-once", "--intent", "plan only"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("autonomy run-once plan-only failed: %v\n%s", err, out.String())
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "notes", "preview.txt")); !os.IsNotExist(err) {
		t.Fatalf("preview mode should not write files, stat err = %v", err)
	}
	if !strings.Contains(out.String(), "Pending:") {
		t.Fatalf("expected pending plans in output, got:\n%s", out.String())
	}
}

func initAutonomyRepo(t *testing.T) string {
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

func autonomyTestApp(t *testing.T, repoRoot string) bootstrap.App {
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
