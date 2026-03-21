package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/gitops"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
	return strings.TrimSpace(string(out))
}

func initBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "--bare")
	return dir
}

func initLocalRepo(t *testing.T, remotePath string) string {
	t.Helper()
	parent := t.TempDir()
	cloneDir := filepath.Join(parent, "local")
	runGit(t, parent, "clone", remotePath, "local")
	runGit(t, cloneDir, "config", "user.email", "test@test.com")
	runGit(t, cloneDir, "config", "user.name", "Test")
	return cloneDir
}

func TestGitPipeline_EndToEnd(t *testing.T) {
	// Create bare repo as "remote"
	remoteDir := initBareRepo(t)

	// Seed the bare repo with an initial commit (bare repos need content pushed)
	seedDir := t.TempDir()
	runGit(t, seedDir, "init")
	runGit(t, seedDir, "config", "user.email", "test@test.com")
	runGit(t, seedDir, "config", "user.name", "Test")
	readme := filepath.Join(seedDir, "README.md")
	_ = os.WriteFile(readme, []byte("# Remote\n"), 0o644)
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "initial")
	runGit(t, seedDir, "remote", "add", "origin", remoteDir)
	runGit(t, seedDir, "push", "-u", "origin", "HEAD")

	// Clone as local
	localDir := initLocalRepo(t, remoteDir)

	// Determine default branch (main or master)
	defaultBranch := runGit(t, localDir, "branch", "--show-current")

	// Use GitExecutor, Inspector, BranchManager, CommitManager for full pipeline
	executor := gitops.NewGitExecutor()
	ctx := context.Background()

	// 1. Create branch
	branchMgr := gitops.NewBranchManager(executor)
	if err := branchMgr.CreateBranch(ctx, localDir, "feature/test", defaultBranch); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := branchMgr.SwitchBranch(ctx, localDir, "feature/test"); err != nil {
		t.Fatalf("SwitchBranch: %v", err)
	}

	// 2. Add files, commit
	commitMgr := gitops.NewCommitManager(executor)
	newFile := filepath.Join(localDir, "feature.txt")
	_ = os.WriteFile(newFile, []byte("feature content\n"), 0o644)
	if err := commitMgr.Add(ctx, localDir, "feature.txt"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	result, err := commitMgr.Commit(ctx, localDir, "add feature file", nil)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if result == nil || result.SHA == "" {
		t.Fatal("expected non-empty commit SHA")
	}

	// 3. Push
	remoteMgr := gitops.NewRemoteManager(executor)
	if err := remoteMgr.Push(ctx, localDir, "origin", "feature/test", gitops.PushOptions{SetUpstream: true}); err != nil {
		t.Fatalf("Push: %v", err)
	}

	// 4. Verify with log
	inspector := gitops.NewInspector(executor)
	inspection, err := inspector.Inspect(ctx, localDir)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if inspection.LocalBranch != "feature/test" {
		t.Errorf("LocalBranch = %q, want feature/test", inspection.LocalBranch)
	}

	logOut := runGit(t, localDir, "log", "-1", "--oneline")
	if !strings.Contains(logOut, "add feature file") {
		t.Errorf("log should contain commit message, got: %s", logOut)
	}

	// Verify remote has the branch
	refs := runGit(t, remoteDir, "branch", "-a")
	if !strings.Contains(refs, "feature/test") {
		t.Errorf("remote should have feature/test branch, got: %s", refs)
	}
}
