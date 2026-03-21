package gitops

import (
	"strings"
	"testing"
)

func TestWriterLock_Acquire_Release(t *testing.T) {
	wl := NewWriterLock()
	owner, repo, ref := "owner1", "repo1", "main"
	taskID := "task-1"

	if err := wl.Acquire(owner, repo, ref, taskID); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	locked, holder := wl.IsLocked(owner, repo, ref)
	if !locked {
		t.Error("IsLocked: expected true after Acquire")
	}
	if holder != taskID {
		t.Errorf("IsLocked: holder=%q, want %q", holder, taskID)
	}
	if err := wl.Release(owner, repo, ref, taskID); err != nil {
		t.Fatalf("Release: %v", err)
	}
	locked, _ = wl.IsLocked(owner, repo, ref)
	if locked {
		t.Error("IsLocked: expected false after Release")
	}
}

func TestWriterLock_DoubleAcquire(t *testing.T) {
	wl := NewWriterLock()
	owner, repo, ref := "o", "r", "main"
	if err := wl.Acquire(owner, repo, ref, "task-a"); err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	err := wl.Acquire(owner, repo, ref, "task-b")
	if err == nil {
		t.Fatal("Double Acquire: expected error")
	}
	if !strings.Contains(err.Error(), "task-a") {
		t.Errorf("Double Acquire: error should mention task-a, got %q", err.Error())
	}
	// Cleanup
	_ = wl.Release(owner, repo, ref, "task-a")
}

func TestWriterLock_ReleaseWrongTask(t *testing.T) {
	wl := NewWriterLock()
	owner, repo, ref := "o", "r", "ref"
	if err := wl.Acquire(owner, repo, ref, "task-hold"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	err := wl.Release(owner, repo, ref, "task-wrong")
	if err == nil {
		t.Fatal("Release wrong task: expected error")
	}
	if !strings.Contains(err.Error(), "task-hold") {
		t.Errorf("Release wrong task: error should mention holder, got %q", err.Error())
	}
	// Cleanup
	_ = wl.Release(owner, repo, ref, "task-hold")
}

func TestWriterLock_IsLocked(t *testing.T) {
	wl := NewWriterLock()
	owner, repo, ref := "org", "proj", "master"

	locked, _ := wl.IsLocked(owner, repo, ref)
	if locked {
		t.Error("IsLocked: expected false when not locked")
	}
	if err := wl.Acquire(owner, repo, ref, "t1"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	locked, holder := wl.IsLocked(owner, repo, ref)
	if !locked {
		t.Error("IsLocked: expected true when locked")
	}
	if holder != "t1" {
		t.Errorf("IsLocked: holder=%q, want t1", holder)
	}
	_ = wl.Release(owner, repo, ref, "t1")
	locked, _ = wl.IsLocked(owner, repo, ref)
	if locked {
		t.Error("IsLocked: expected false after Release")
	}
}
