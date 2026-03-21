package bbolt_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/storage/bbolt"
)

func TestNewProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	provider, err := bbolt.NewProvider(path)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	defer provider.Close()

	if provider == nil {
		t.Fatal("NewProvider: expected non-nil provider")
	}
}

func TestProvider_Close(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	provider, err := bbolt.NewProvider(path)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	if err := provider.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestProvider_Migrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	provider, err := bbolt.NewProvider(path)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()
	if err := provider.Migrate(ctx); err != nil {
		t.Errorf("Migrate: expected nil, got %v", err)
	}
}

func TestProvider_PlanStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	provider, err := bbolt.NewProvider(path)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	defer provider.Close()

	store := provider.PlanStore()
	if store == nil {
		t.Fatal("PlanStore: expected non-nil")
	}
}

func TestNewProvider_CreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state", "gitdex.db")

	provider, err := bbolt.NewProvider(path)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	defer provider.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected bbolt file to exist: %v", err)
	}
}
