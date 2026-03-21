package gitops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func initTempGitRepo(t *testing.T, dir string) {
	t.Helper()
	git := NewGitExecutor()
	ctx := context.Background()
	if _, err := git.Run(ctx, dir, "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := git.Run(ctx, dir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email: %v", err)
	}
	if _, err := git.Run(ctx, dir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name: %v", err)
	}
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("x\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := git.Run(ctx, dir, "add", "f"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := git.Run(ctx, dir, "commit", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if _, err := git.Run(ctx, dir, "branch", "-M", "main"); err != nil {
		t.Fatalf("git branch -M main: %v", err)
	}
}

func TestHygieneExecutor_Execute_Success(t *testing.T) {
	exec := NewHygieneExecutor(NewGitExecutor())
	ctx := context.Background()
	repoPath := t.TempDir()
	initTempGitRepo(t, repoPath)
	action := HygieneGCAggressive // gc works in any repo without remotes

	result, err := exec.Execute(ctx, repoPath, action)
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute: expected result, got nil")
	}
	if !result.Success {
		t.Errorf("Execute: expected success=true, got %v (error: %s)", result.Success, result.ErrorMessage)
	}
	if result.Action != action {
		t.Errorf("Action: got %q, want %q", result.Action, action)
	}
	if result.Summary == "" {
		t.Error("Summary: expected non-empty")
	}
}

func TestHygieneExecutor_Execute_Failure_EmptyRepoPath(t *testing.T) {
	exec := NewHygieneExecutor(NewGitExecutor())
	ctx := context.Background()
	action := HygieneCleanUntracked

	result, err := exec.Execute(ctx, "", action)
	if err != nil {
		t.Fatalf("Execute: expected result on failure, got error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute: expected HygieneResult for failure, got nil")
	}
	if result.Success {
		t.Error("Execute: expected success=false for empty repo path")
	}
	if result.ErrorMessage == "" {
		t.Error("ErrorMessage: expected non-empty for failure")
	}
	if result.Summary == "" {
		t.Error("Summary: expected non-empty for retry/handoff context")
	}
	if result.Action != action {
		t.Errorf("Action: got %q, want %q", result.Action, action)
	}
}

func TestHygieneExecutor_Execute_InvalidAction(t *testing.T) {
	exec := NewHygieneExecutor(NewGitExecutor())
	ctx := context.Background()
	repoPath := "/tmp/repo"
	action := HygieneAction("invalid_action")

	result, err := exec.Execute(ctx, repoPath, action)
	if err == nil {
		t.Fatal("Execute: expected error for invalid action")
	}
	if result != nil {
		t.Errorf("Execute: expected nil result for invalid action, got %+v", result)
	}
}

func TestHygieneExecutor_Execute_CancelledContext(t *testing.T) {
	exec := NewHygieneExecutor(NewGitExecutor())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repoPath := "/tmp/repo"
	action := HygieneGCAggressive

	result, err := exec.Execute(ctx, repoPath, action)
	if err != nil {
		t.Fatalf("Execute: expected result on cancel, got error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute: expected HygieneResult on cancel, got nil")
	}
	if result.Success {
		t.Error("Execute: expected success=false when context cancelled")
	}
	if result.ErrorMessage == "" {
		t.Error("ErrorMessage: expected non-empty on cancel")
	}
}
