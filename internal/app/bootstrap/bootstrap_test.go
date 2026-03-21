package bootstrap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/platform/config"
)

func TestLoadReturnsRepoRootAndVersion(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("filepath.Abs failed: %v", err)
	}

	app, err := bootstrap.Load(bootstrap.Options{
		RepoRoot: repoRoot,
		Version:  "test-version",
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if app.RepoRoot != repoRoot {
		t.Fatalf("RepoRoot = %q, want %q", app.RepoRoot, repoRoot)
	}

	if app.Version != "test-version" {
		t.Fatalf("Version = %q, want %q", app.Version, "test-version")
	}
}

func TestLoadDefaultsSQLiteDSNWhenMissing(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("filepath.Abs failed: %v", err)
	}

	userConfigDir := t.TempDir()
	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if err := config.WriteFile(globalConfigPath, config.FileConfig{
		Storage: config.StorageConfig{
			Type: "sqlite",
		},
	}); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	app, err := bootstrap.Load(bootstrap.Options{
		RepoRoot:      repoRoot,
		WorkingDir:    repoRoot,
		UserConfigDir: userConfigDir,
		Version:       "test-version",
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	defer app.StorageProvider.Close()

	expectedDSN := filepath.Join(userConfigDir, "gitdex", "gitdex.sqlite")
	if app.Config.Storage.DSN != expectedDSN {
		t.Fatalf("Storage.DSN = %q, want %q", app.Config.Storage.DSN, expectedDSN)
	}

	if _, err := os.Stat(expectedDSN); err != nil {
		t.Fatalf("sqlite database file should exist: %v", err)
	}
}
