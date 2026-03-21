package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/daemon/server"
	"github.com/your-org/gitdex/internal/storage"
)

// Config holds daemon runtime configuration.
type Config struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	OnGitHubWebhook func(context.Context, *autonomy.TriggerConfig, string, *autonomy.TriggerEvent) error
	Cruise          *autonomy.CruiseEngine
}

// DefaultConfig returns default daemon configuration.
func DefaultConfig() Config {
	return Config{
		Address:         "127.0.0.1:7777",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// Run starts the HTTP control plane daemon, blocking until shutdown.
// It creates and starts the HTTP server, handles SIGINT/SIGTERM for graceful shutdown,
// and runs until terminated.
func Run(ctx context.Context, cfg Config, provider storage.StorageProvider) error {
	if provider == nil {
		return fmt.Errorf("storage provider is required")
	}
	if cfg.Address == "" {
		cfg.Address = DefaultConfig().Address
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = DefaultConfig().ShutdownTimeout
	}

	srvCfg := server.Config{
		Address:         cfg.Address,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		ShutdownTimeout: cfg.ShutdownTimeout,
		OnGitHubWebhook: cfg.OnGitHubWebhook,
		Cruise:          cfg.Cruise,
	}
	srv := server.New(srvCfg, provider)

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	// Wait for ctx cancellation or OS signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
		log.Printf("daemon: context cancelled: %v", ctx.Err())
	case sig := <-sigCh:
		log.Printf("daemon: received signal %v", sig)
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
