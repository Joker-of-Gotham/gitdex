package gitops

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", args, out)
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	// Create initial commit
	f := filepath.Join(dir, "README.md")
	_ = os.WriteFile(f, []byte("# Test\n"), 0o644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial")
	return dir
}

func TestGitExecutor_Run_Success(t *testing.T) {
	dir := initTestRepo(t)
	executor := NewGitExecutor()
	ctx := context.Background()

	result, err := executor.Run(ctx, dir, "status")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Stdout == "" && result.Stderr == "" {
		t.Error("expected some output from git status")
	}
}

func TestGitExecutor_Run_NotARepo(t *testing.T) {
	dir := t.TempDir()
	executor := NewGitExecutor()
	ctx := context.Background()

	_, err := executor.Run(ctx, dir, "status")
	if err == nil {
		t.Fatal("expected error when running in non-repo")
	}
	gerr, ok := err.(*GitError)
	if !ok {
		t.Fatalf("expected *GitError, got %T: %v", err, err)
	}
	if gerr.Kind != ErrKindNotARepo {
		t.Errorf("Kind = %q, want %q", gerr.Kind, ErrKindNotARepo)
	}
}

func TestGitExecutor_Run_Timeout(t *testing.T) {
	dir := initTestRepo(t)
	executor := NewGitExecutorWithConfig("git", 1*time.Nanosecond)
	ctx := context.Background()

	_, err := executor.Run(ctx, dir, "status")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if ctx.Err() == nil && err.Error() == "" {
		t.Error("expected context canceled or timeout error")
	}
}

func TestGitExecutor_RunLines(t *testing.T) {
	dir := initTestRepo(t)
	executor := NewGitExecutor()
	ctx := context.Background()

	lines, err := executor.RunLines(ctx, dir, "branch")
	if err != nil {
		t.Fatalf("RunLines: %v", err)
	}
	if len(lines) == 0 {
		t.Error("expected at least one branch line")
	}
	// Main/master branch should be present (format: "* branch" or "  branch")
	found := false
	for _, line := range lines {
		name := strings.TrimPrefix(line, "* ")
		name = strings.TrimSpace(name)
		if name == "main" || name == "master" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected main or master branch in %v", lines)
	}
}
