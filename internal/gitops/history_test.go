package gitops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryInspector_Log(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	cm := NewCommitManager(NewGitExecutor())
	hi := NewHistoryInspector(NewGitExecutor())

	// Add another commit so we have 2
	f := filepath.Join(dir, "extra.txt")
	if err := os.WriteFile(f, []byte("x\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := cm.Add(ctx, dir, "extra.txt"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := cm.Commit(ctx, dir, "second commit", nil); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	entries, err := hi.Log(ctx, dir, &LogOptions{MaxCount: 10})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) < 2 {
		t.Errorf("Log: expected at least 2 entries, got %d", len(entries))
	}
	for i, e := range entries {
		if e.SHA == "" {
			t.Errorf("Log[%d]: empty SHA", i)
		}
		if e.ShortSHA == "" {
			t.Errorf("Log[%d]: empty ShortSHA", i)
		}
		if e.Author == "" {
			t.Errorf("Log[%d]: empty Author", i)
		}
		if e.Subject == "" {
			t.Errorf("Log[%d]: empty Subject", i)
		}
	}
}

func TestHistoryInspector_RevParse(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	hi := NewHistoryInspector(NewGitExecutor())

	sha, err := hi.RevParse(ctx, dir, "HEAD")
	if err != nil {
		t.Fatalf("RevParse HEAD: %v", err)
	}
	if sha == "" {
		t.Error("RevParse: expected non-empty SHA")
	}
	if len(sha) != 40 {
		t.Errorf("RevParse: expected 40-char SHA, got %q (len=%d)", sha, len(sha))
	}
}

func TestHistoryInspector_CountObjects(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	hi := NewHistoryInspector(NewGitExecutor())

	du, err := hi.CountObjects(ctx, dir)
	if err != nil {
		t.Fatalf("CountObjects: %v", err)
	}
	if du == nil {
		t.Fatal("CountObjects: expected non-nil DiskUsage")
	}
	// Repo has at least the initial commit - count or size should be set
	if du.Count == 0 && du.Size == "" && du.InPack == 0 && du.PackSize == "" {
		t.Error("CountObjects: expected disk usage info to be populated")
	}
}
