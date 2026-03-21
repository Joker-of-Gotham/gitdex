package storage_test

import (
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/storage"
)

func TestDefaultDSN(t *testing.T) {
	baseDir := filepath.Join("config", "gitdex")

	if got := storage.DefaultDSN(baseDir, storage.BackendSQLite); got != filepath.Join(baseDir, "gitdex.sqlite") {
		t.Fatalf("DefaultDSN(sqlite) = %q", got)
	}
	if got := storage.DefaultDSN(baseDir, storage.BackendBBolt); got != filepath.Join(baseDir, "gitdex.db") {
		t.Fatalf("DefaultDSN(bbolt) = %q", got)
	}
	if got := storage.DefaultDSN(baseDir, storage.BackendPostgres); got != "" {
		t.Fatalf("DefaultDSN(postgres) = %q, want empty", got)
	}
}

func TestConfigNormalizedFillsFileBackedDSN(t *testing.T) {
	baseDir := filepath.Join("config", "gitdex")

	sqliteCfg := (storage.Config{Type: storage.BackendSQLite}).Normalized(baseDir)
	if sqliteCfg.DSN != filepath.Join(baseDir, "gitdex.sqlite") {
		t.Fatalf("sqlite normalized dsn = %q", sqliteCfg.DSN)
	}

	bboltCfg := (storage.Config{Type: storage.BackendBBolt}).Normalized(baseDir)
	if bboltCfg.DSN != filepath.Join(baseDir, "gitdex.db") {
		t.Fatalf("bbolt normalized dsn = %q", bboltCfg.DSN)
	}

	postgresCfg := (storage.Config{Type: storage.BackendPostgres}).Normalized(baseDir)
	if postgresCfg.DSN != "" {
		t.Fatalf("postgres normalized dsn = %q, want empty", postgresCfg.DSN)
	}
}
