package postgres_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/storage/postgres"
)

func TestNewProvider_InvalidDSN(t *testing.T) {
	_, err := postgres.NewProvider("invalid://bad-dsn", 10)
	if err == nil {
		t.Fatal("NewProvider(invalid DSN): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "postgres") {
		t.Errorf("error should mention postgres: %v", err)
	}
}

func TestNewProvider_EmptyDSN(t *testing.T) {
	// pgxpool.ParseConfig may still fail on malformed DSN
	_, err := postgres.NewProvider("", 10)
	if err == nil {
		t.Fatal("NewProvider(empty DSN): expected error, got nil")
	}
}
