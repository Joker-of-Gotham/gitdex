package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/daemon/service"
	"github.com/your-org/gitdex/internal/storage"
)

func TestRunRequiresProvider(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := service.Run(ctx, service.Config{}, nil)
	if err == nil {
		t.Fatal("expected error when provider is nil")
	}
}

func TestRunWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	cfg := service.Config{
		Address:         "127.0.0.1:0",
		ShutdownTimeout: 1 * time.Second,
	}
	_ = service.Run(ctx, cfg, provider)
}

func TestDefaultConfig(t *testing.T) {
	cfg := service.DefaultConfig()
	if cfg.Address != "127.0.0.1:7777" {
		t.Fatalf("expected default address 127.0.0.1:7777, got %q", cfg.Address)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("expected 10s shutdown timeout, got %v", cfg.ShutdownTimeout)
	}
}
