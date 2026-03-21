package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitManager_Add_Commit(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	cm := NewCommitManager(NewGitExecutor())

	f := filepath.Join(dir, "newfile.txt")
	if err := os.WriteFile(f, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := cm.Add(ctx, dir, "newfile.txt"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	result, err := cm.Commit(ctx, dir, "add newfile", nil)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if result == nil || result.SHA == "" {
		t.Fatal("Commit: expected non-empty SHA")
	}
	if len(result.SHA) != 40 {
		t.Errorf("Commit: expected 40-char SHA, got %q", result.SHA)
	}
	if result.Summary != "add newfile" {
		t.Errorf("Commit: Summary=%q, want add newfile", result.Summary)
	}
}

func TestCommitManager_AddAll(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	cm := NewCommitManager(NewGitExecutor())
	hi := NewHistoryInspector(NewGitExecutor())

	// Create multiple files
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	if err := cm.AddAll(ctx, dir); err != nil {
		t.Fatalf("AddAll: %v", err)
	}

	// Verify staged via ls-files --cached
	files, err := hi.LsFiles(ctx, dir, &LsFilesOptions{Cached: true})
	if err != nil {
		t.Fatalf("LsFiles: %v", err)
	}
	hasA := false
	hasB := false
	hasC := false
	for _, f := range files {
		if strings.HasSuffix(f, "a.txt") || f == "a.txt" {
			hasA = true
		}
		if strings.HasSuffix(f, "b.txt") || f == "b.txt" {
			hasB = true
		}
		if strings.HasSuffix(f, "c.txt") || f == "c.txt" {
			hasC = true
		}
	}
	if !hasA || !hasB || !hasC {
		t.Errorf("AddAll: expected a.txt, b.txt, c.txt staged, got %v", files)
	}
}

func TestCommitManager_StashPush_StashList_StashPop(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	cm := NewCommitManager(NewGitExecutor())

	// Must stage changes (add but don't commit) - stash does not include untracked by default
	f := filepath.Join(dir, "stashed.txt")
	if err := os.WriteFile(f, []byte("stash me\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := cm.Add(ctx, dir, "stashed.txt"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cm.StashPush(ctx, dir, "wip: stashed changes", false); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	entries, err := cm.StashList(ctx, dir)
	if err != nil {
		t.Fatalf("StashList: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("StashList: expected 1 entry, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Message, "stashed") {
		t.Errorf("StashList: expected message containing 'stashed', got %q", entries[0].Message)
	}

	if err := cm.StashPop(ctx, dir, 0); err != nil {
		t.Fatalf("StashPop: %v", err)
	}

	// Verify file is back
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("ReadFile after pop: %v", err)
	}
	content := strings.TrimSpace(string(data))
	if content != "stash me" {
		t.Errorf("StashPop: file content = %q, want stash me", string(data))
	}
}
