package gitops

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func mustInitGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	env := append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	// Create a tracked file for diff tests
	if err := os.WriteFile(filepath.Join(dir, "bar.txt"), []byte("original\n"), 0644); err != nil {
		t.Fatalf("write bar.txt failed: %v", err)
	}
	cmd = exec.Command("git", "add", "bar.txt")
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
	// Ensure main branch exists (init may create master on older git)
	cmd = exec.Command("git", "branch", "-m", "main")
	cmd.Dir = dir
	_ = cmd.Run() // ignore error if already main
}

func TestWorktreeManager_Create(t *testing.T) {
	dir := t.TempDir()
	mustInitGitRepo(t, dir)

	mgr := NewWorktreeManager(NewGitExecutor())
	wtDir := filepath.Join(dir, "..", "gitdex-worktree-feature")
	cfg := WorktreeConfig{
		RepoPath:    dir,
		Branch:      "feature",
		WorktreeDir: wtDir,
		StartPoint:  "main", // create new branch from main
	}

	wt, err := mgr.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if wt == nil {
		t.Fatal("expected non-nil Worktree")
	}
	if wt.Config.RepoPath != cfg.RepoPath {
		t.Errorf("RepoPath: got %q, want %q", wt.Config.RepoPath, cfg.RepoPath)
	}
	if wt.Config.Branch != cfg.Branch {
		t.Errorf("Branch: got %q, want %q", wt.Config.Branch, cfg.Branch)
	}
	if wt.Status != WorktreeStatusActive && wt.Status != WorktreeStatusClean {
		t.Errorf("Status: got %q, want active or clean", wt.Status)
	}
	if wt.Config.WorktreeDir == "" {
		t.Error("WorktreeDir should be set")
	}

	// Cleanup
	_ = mgr.Discard(context.Background(), wtDir)
}

func TestWorktreeManager_Create_MissingRepoPath(t *testing.T) {
	mgr := NewWorktreeManager(NewGitExecutor())
	_, err := mgr.Create(context.Background(), WorktreeConfig{Branch: "main"})
	if err == nil {
		t.Fatal("expected error for missing repo path")
	}
}

func TestWorktreeManager_Create_MissingBranch(t *testing.T) {
	mgr := NewWorktreeManager(NewGitExecutor())
	_, err := mgr.Create(context.Background(), WorktreeConfig{RepoPath: "/tmp/repo"})
	if err == nil {
		t.Fatal("expected error for missing branch")
	}
}

func TestWorktreeManager_Create_ExplicitWorktreeDir(t *testing.T) {
	dir := t.TempDir()
	mustInitGitRepo(t, dir)

	mgr := NewWorktreeManager(NewGitExecutor())
	wtDir := filepath.Join(dir, "..", "wt-explicit")
	cfg := WorktreeConfig{
		RepoPath:    dir,
		Branch:      "feature",
		WorktreeDir: wtDir,
		StartPoint:  "main",
	}
	wt, err := mgr.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if wt.Config.WorktreeDir != cfg.WorktreeDir {
		t.Errorf("WorktreeDir: got %q, want %q", wt.Config.WorktreeDir, cfg.WorktreeDir)
	}

	_ = mgr.Discard(context.Background(), wtDir)
}

func TestWorktreeManager_Inspect(t *testing.T) {
	dir := t.TempDir()
	mustInitGitRepo(t, dir)

	mgr := NewWorktreeManager(NewGitExecutor())
	wtDir := filepath.Join(dir, "..", "wt-inspect")
	cfg := WorktreeConfig{RepoPath: dir, Branch: "inspect-branch", WorktreeDir: wtDir, StartPoint: "main"}
	_, err := mgr.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = mgr.Discard(context.Background(), wtDir) }()

	wt, err := mgr.Inspect(context.Background(), wtDir)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}
	if wt == nil {
		t.Fatal("expected non-nil Worktree")
	}
	if wt.Config.WorktreeDir != wtDir {
		t.Errorf("WorktreeDir: got %q, want %q", wt.Config.WorktreeDir, wtDir)
	}
	if wt.Status != WorktreeStatusClean && wt.Status != WorktreeStatusActive {
		t.Errorf("Status: got %q", wt.Status)
	}
}

func TestWorktreeManager_Inspect_EmptyDir(t *testing.T) {
	mgr := NewWorktreeManager(NewGitExecutor())
	_, err := mgr.Inspect(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty worktree dir")
	}
}

func TestWorktreeManager_Diff(t *testing.T) {
	dir := t.TempDir()
	mustInitGitRepo(t, dir)

	mgr := NewWorktreeManager(NewGitExecutor())
	wtDir := filepath.Join(dir, "..", "wt-diff")
	cfg := WorktreeConfig{RepoPath: dir, Branch: "diff-branch", WorktreeDir: wtDir, StartPoint: "main"}
	_, err := mgr.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = mgr.Discard(context.Background(), wtDir) }()

	// Modify the tracked bar.txt (from mustInitGitRepo) to get a diff
	barPath := filepath.Join(wtDir, "bar.txt")
	if err := os.WriteFile(barPath, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("write bar.txt failed: %v", err)
	}

	diff, err := mgr.Diff(context.Background(), wtDir)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestWorktreeManager_Diff_EmptyDir(t *testing.T) {
	mgr := NewWorktreeManager(NewGitExecutor())
	_, err := mgr.Diff(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty worktree dir")
	}
}

func TestWorktreeManager_Discard(t *testing.T) {
	dir := t.TempDir()
	mustInitGitRepo(t, dir)

	mgr := NewWorktreeManager(NewGitExecutor())
	wtDir := filepath.Join(dir, "..", "wt-discard")
	cfg := WorktreeConfig{RepoPath: dir, Branch: "discard-branch", WorktreeDir: wtDir, StartPoint: "main"}
	_, err := mgr.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = mgr.Discard(context.Background(), wtDir)
	if err != nil {
		t.Fatalf("Discard failed: %v", err)
	}
	if _, err := os.Stat(wtDir); err == nil {
		t.Error("worktree dir should be removed after discard")
	}
}

func TestWorktreeManager_Discard_EmptyDir(t *testing.T) {
	mgr := NewWorktreeManager(NewGitExecutor())
	err := mgr.Discard(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty worktree dir")
	}
}

func TestWorktreeStatus_Constants(t *testing.T) {
	statuses := []WorktreeStatus{
		WorktreeStatusActive,
		WorktreeStatusDirty,
		WorktreeStatusClean,
		WorktreeStatusRemoved,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("WorktreeStatus constant should not be empty")
		}
	}
}

func TestWorktree_RoundTripJSON(t *testing.T) {
	original := &Worktree{
		Config: WorktreeConfig{
			RepoPath:    "/repo",
			Branch:      "main",
			WorktreeDir: "/worktree",
		},
		Status:      WorktreeStatusActive,
		DiffSummary: "test diff",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Worktree
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Config.RepoPath != original.Config.RepoPath {
		t.Errorf("RepoPath: got %q, want %q", decoded.Config.RepoPath, original.Config.RepoPath)
	}
	if decoded.Config.Branch != original.Config.Branch {
		t.Errorf("Branch: got %q, want %q", decoded.Config.Branch, original.Config.Branch)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, original.Status)
	}
}
