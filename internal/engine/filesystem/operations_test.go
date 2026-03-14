package filesystem

import (
	"context"
	"path/filepath"
	"testing"
)

func TestExecutorCreateUpdateDelete(t *testing.T) {
	exec := NewExecutor()
	path := filepath.Join(t.TempDir(), "nested", "file.txt")

	createRes, err := exec.Create(context.Background(), path, "hello")
	if err != nil || !createRes.Success {
		t.Fatalf("create failed: %v", err)
	}

	updateRes, err := exec.Update(context.Background(), path, "world", true)
	if err != nil || !updateRes.Success {
		t.Fatalf("update failed: %v", err)
	}
	if updateRes.BackupPath == "" {
		t.Fatal("expected backup path")
	}

	deleteRes, err := exec.Delete(context.Background(), path, false)
	if err != nil || !deleteRes.Success {
		t.Fatalf("delete failed: %v", err)
	}
}
