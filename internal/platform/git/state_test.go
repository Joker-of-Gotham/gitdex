package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s\n%v", args, out, err)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "initial commit")
}

func TestReadLocalState_CleanRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)

	state, err := ReadLocalState(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Branch != "main" {
		t.Errorf("branch = %q, want %q", state.Branch, "main")
	}
	if !state.IsClean {
		t.Error("expected clean working tree")
	}
	if state.IsDetached {
		t.Error("expected attached HEAD")
	}
	if state.StagedCount != 0 {
		t.Errorf("staged count = %d, want 0", state.StagedCount)
	}
	if state.DirtyCount != 0 {
		t.Errorf("dirty count = %d, want 0", state.DirtyCount)
	}
	if len(state.HeadSHA) != 40 {
		t.Errorf("head SHA length = %d, want 40", len(state.HeadSHA))
	}
}

func TestReadLocalState_DirtyRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}

	state, err := ReadLocalState(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.IsClean {
		t.Error("expected dirty working tree")
	}
	if state.DirtyCount < 1 {
		t.Errorf("dirty count = %d, want >= 1", state.DirtyCount)
	}
}

func TestReadLocalState_StagedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)

	path := filepath.Join(dir, "staged.txt")
	if err := os.WriteFile(path, []byte("staged"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "staged.txt")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %s\n%v", out, err)
	}

	state, err := ReadLocalState(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.StagedCount < 1 {
		t.Errorf("staged count = %d, want >= 1", state.StagedCount)
	}
}

func TestReadLocalState_DetachedHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)

	cmd := exec.Command("git", "checkout", "--detach")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout --detach: %s\n%v", out, err)
	}

	state, err := ReadLocalState(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !state.IsDetached {
		t.Error("expected detached HEAD")
	}
}

func TestReadLocalState_InvalidPath(t *testing.T) {
	_, err := ReadLocalState(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestReadLocalState_Remotes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)

	cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %s\n%v", out, err)
	}

	state, err := ReadLocalState(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Remotes) != 1 {
		t.Fatalf("remotes count = %d, want 1", len(state.Remotes))
	}
	if state.Remotes[0] != "origin" {
		t.Errorf("remote = %q, want %q", state.Remotes[0], "origin")
	}
	if state.DefaultRemote != "https://github.com/test/repo.git" {
		t.Errorf("default remote = %q, want https://github.com/test/repo.git", state.DefaultRemote)
	}
}
