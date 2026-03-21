package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/storage/sqlite"
)

func TestNewProvider(t *testing.T) {
	provider, err := sqlite.NewProvider(":memory:")
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	defer provider.Close()

	if provider == nil {
		t.Fatal("NewProvider: expected non-nil provider")
	}
}

func TestProvider_Migrate(t *testing.T) {
	provider, err := sqlite.NewProvider(":memory:")
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()
	if err := provider.Migrate(ctx); err != nil {
		t.Errorf("Migrate: %v", err)
	}
}

func TestProvider_Close(t *testing.T) {
	provider, err := sqlite.NewProvider(":memory:")
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	if err := provider.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestProvider_StoreAccessorsAfterMigrate(t *testing.T) {
	provider, err := sqlite.NewProvider(":memory:")
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()
	if err := provider.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if provider.PlanStore() == nil {
		t.Error("PlanStore: expected non-nil after Migrate")
	}
	if provider.TaskStore() == nil {
		t.Error("TaskStore: expected non-nil after Migrate")
	}
}

func TestNewProvider_CreatesParentDirectory(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "nested", "state", "gitdex.sqlite")

	provider, err := sqlite.NewProvider(dsn)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	defer provider.Close()

	if _, err := os.Stat(dsn); err != nil {
		t.Fatalf("expected sqlite file to exist: %v", err)
	}
}
