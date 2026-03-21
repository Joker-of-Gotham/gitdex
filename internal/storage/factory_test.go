package storage_test

import (
	"testing"

	"github.com/your-org/gitdex/internal/storage"
)

func TestNewProvider_Memory(t *testing.T) {
	cfg := storage.Config{Type: storage.BackendMemory}
	provider, err := storage.NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider(memory): %v", err)
	}
	if provider == nil {
		t.Fatal("NewProvider(memory): expected non-nil provider")
	}
}

func TestNewProvider_UnknownType(t *testing.T) {
	cfg := storage.Config{Type: storage.BackendType("unknown")}
	provider, err := storage.NewProvider(cfg)
	if err == nil {
		t.Fatal("NewProvider(unknown): expected error, got nil")
	}
	if provider != nil {
		t.Fatal("NewProvider(unknown): expected nil provider")
	}
}

func TestBackendTypeConstants(t *testing.T) {
	// Ensure all constants exist and are distinct
	types := map[storage.BackendType]bool{
		storage.BackendMemory:   true,
		storage.BackendPostgres: true,
		storage.BackendSQLite:   true,
		storage.BackendBBolt:    true,
	}
	if len(types) != 4 {
		t.Errorf("expected 4 distinct BackendType constants, got %d", len(types))
	}
}

func TestConfig_Constructor(t *testing.T) {
	cfg := storage.Config{
		Type:         storage.BackendSQLite,
		DSN:          ":memory:",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		AutoMigrate:  true,
	}
	if cfg.Type != storage.BackendSQLite || cfg.DSN != ":memory:" {
		t.Errorf("Config struct not constructed correctly: %+v", cfg)
	}
}
