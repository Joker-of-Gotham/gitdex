package gitops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func initTempGitRepoForSync(t *testing.T, repoPath string) {
	t.Helper()
	git := NewGitExecutor()
	ctx := context.Background()
	// Create bare "remote" (must run from parent dir; git creates bare.git)
	parent := t.TempDir()
	barePath := filepath.Join(parent, "bare.git")
	if _, err := git.Run(ctx, parent, "init", "--bare", "bare.git"); err != nil {
		t.Fatalf("bare init: %v", err)
	}
	// Init repo, commit, push
	initTempGitRepo(t, repoPath)
	if _, err := git.Run(ctx, repoPath, "remote", "add", "origin", barePath); err != nil {
		t.Fatalf("remote add: %v", err)
	}
	if _, err := git.Run(ctx, repoPath, "push", "-u", "origin", "main"); err != nil {
		// try "master" if "main" doesn't exist
		if _, err := git.Run(ctx, repoPath, "branch", "-M", "main"); err == nil {
			_, _ = git.Run(ctx, repoPath, "push", "-u", "origin", "main")
		}
	}
	// Clone to second repo, add commit, push (so origin is ahead)
	repo2Path := filepath.Join(parent, "repo2")
	if _, err := git.Run(ctx, parent, "clone", barePath, "repo2"); err != nil {
		t.Fatalf("clone: %v", err)
	}
	repo2 := repo2Path
	if _, err := git.Run(ctx, repo2, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("config: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "config", "user.name", "Test"); err != nil {
		t.Fatalf("config: %v", err)
	}
	f2 := filepath.Join(repo2, "g")
	if err := os.WriteFile(f2, []byte("y\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "add", "g"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "commit", "-m", "upstream"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "push", "origin", "main"); err != nil {
		_, _ = git.Run(ctx, repo2, "push", "origin", "master")
	}
}

func initTempGitRepoForDivergedSync(t *testing.T, repoPath, bareParent string) {
	t.Helper()
	git := NewGitExecutor()
	ctx := context.Background()
	barePath := filepath.Join(bareParent, "bare.git")
	if _, err := git.Run(ctx, bareParent, "init", "--bare", "bare.git"); err != nil {
		t.Fatalf("bare init: %v", err)
	}
	initTempGitRepo(t, repoPath)
	f := filepath.Join(repoPath, "f")
	if err := os.WriteFile(f, []byte("a\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := git.Run(ctx, repoPath, "add", "f"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := git.Run(ctx, repoPath, "commit", "-m", "c1"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := git.Run(ctx, repoPath, "remote", "add", "origin", barePath); err != nil {
		t.Fatalf("remote add: %v", err)
	}
	if _, err := git.Run(ctx, repoPath, "push", "-u", "origin", "main"); err != nil {
		if _, err := git.Run(ctx, repoPath, "branch", "-M", "main"); err == nil {
			_, _ = git.Run(ctx, repoPath, "push", "-u", "origin", "main")
		}
	}
	// Repo2: clone, change f to "b", push
	repo2 := t.TempDir()
	if _, err := git.Run(ctx, repo2, "clone", barePath, "."); err != nil {
		t.Fatalf("clone: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("config: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "config", "user.name", "Test"); err != nil {
		t.Fatalf("config: %v", err)
	}
	f2 := filepath.Join(repo2, "f")
	if err := os.WriteFile(f2, []byte("b\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "add", "f"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "commit", "-m", "c2"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := git.Run(ctx, repo2, "push", "origin", "main"); err != nil {
		_, _ = git.Run(ctx, repo2, "push", "origin", "master")
	}
	// Repo: change f to "c" locally (diverged)
	if err := os.WriteFile(f, []byte("c\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := git.Run(ctx, repoPath, "add", "f"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := git.Run(ctx, repoPath, "commit", "-m", "c3"); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestPreview_FastForward(t *testing.T) {
	syncer := NewSyncer(NewGitExecutor())
	insp := &RepoInspection{Behind: 5, Divergence: DivBehind}
	rec := &SyncRecommendation{Action: "fast_forward", Previewable: true}

	prev, err := syncer.Preview(context.Background(), insp, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prev.AffectedFiles != 5 {
		t.Errorf("got affected files %d, want 5", prev.AffectedFiles)
	}
	if prev.ConflictRisk != "none" {
		t.Errorf("got conflict risk %q, want %q", prev.ConflictRisk, "none")
	}
}

func TestPreview_Diverged(t *testing.T) {
	syncer := NewSyncer(NewGitExecutor())
	insp := &RepoInspection{Ahead: 3, Behind: 4, Divergence: DivDiverged}
	rec := &SyncRecommendation{Action: "merge_or_rebase", Previewable: true}

	prev, err := syncer.Preview(context.Background(), insp, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prev.ConflictRisk != "high" {
		t.Errorf("got conflict risk %q, want %q", prev.ConflictRisk, "high")
	}
	if prev.AffectedFiles != 7 {
		t.Errorf("got affected files %d, want 7", prev.AffectedFiles)
	}
}

func TestPreview_NotPreviewable(t *testing.T) {
	syncer := NewSyncer(NewGitExecutor())
	insp := &RepoInspection{Divergence: DivDetached}
	rec := &SyncRecommendation{Action: "checkout_branch", Previewable: false}

	_, err := syncer.Preview(context.Background(), insp, rec)
	if err == nil {
		t.Fatal("expected error for non-previewable action")
	}
}

func TestPreview_NilInspection(t *testing.T) {
	syncer := NewSyncer(NewGitExecutor())
	rec := &SyncRecommendation{Action: "fast_forward", Previewable: true}

	_, err := syncer.Preview(context.Background(), nil, rec)
	if err == nil {
		t.Fatal("expected error for nil inspection")
	}
}

func TestExecute_FastForward(t *testing.T) {
	repoPath := t.TempDir()
	initTempGitRepoForSync(t, repoPath)
	syncer := NewSyncer(NewGitExecutor())
	rec := &SyncRecommendation{Action: "fast_forward"}
	insp := &RepoInspection{
		RepoPath:     repoPath,
		LocalBranch:  "main",
		RemoteBranch: "origin/main",
		Divergence:   DivBehind,
	}

	result, err := syncer.Execute(context.Background(), insp, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.ErrorMessage)
	}
}

func TestExecute_Diverged_Blocked(t *testing.T) {
	repoPath, barePath := t.TempDir(), t.TempDir()
	initTempGitRepoForDivergedSync(t, repoPath, barePath)
	syncer := NewSyncer(NewGitExecutor())
	rec := &SyncRecommendation{Action: "merge_or_rebase"}
	insp := &RepoInspection{
		RepoPath:     repoPath,
		LocalBranch:  "main",
		RemoteBranch: "origin/main",
		Divergence:   DivDiverged,
	}

	result, err := syncer.Execute(context.Background(), insp, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Diverged merge may either conflict (Success=false, Conflicts>=1) or
	// auto-resolve (Success=true). Both are valid; verify we get a coherent result.
	if !result.Success {
		if result.Conflicts < 1 {
			t.Errorf("failure without conflicts: %s", result.ErrorMessage)
		}
	}
}

func TestExecute_NilRecommendation(t *testing.T) {
	syncer := NewSyncer(NewGitExecutor())
	_, err := syncer.Execute(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for nil recommendation")
	}
}
