package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/storage"
)

type Server struct {
	router          chi.Router
	provider        storage.StorageProvider
	cruise          *autonomy.CruiseEngine
	httpSrv         *http.Server
	onGitHubWebhook func(context.Context, *autonomy.TriggerConfig, string, *autonomy.TriggerEvent) error
}

// Config holds the HTTP server configuration.
type Config struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	OnGitHubWebhook func(context.Context, *autonomy.TriggerConfig, string, *autonomy.TriggerEvent) error
	// Cruise optional: when set, cruise and metrics endpoints expose engine state.
	Cruise *autonomy.CruiseEngine
}

// DefaultConfig returns default server configuration.
func DefaultConfig() Config {
	return Config{
		Address:         "127.0.0.1:7777",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// New creates a new HTTP server with chi router and registered routes.
func New(cfg Config, provider storage.StorageProvider) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(cfg.WriteTimeout))
	r.Use(middleware.RequestID)

	s := &Server{
		router:          r,
		provider:        provider,
		cruise:          cfg.Cruise,
		onGitHubWebhook: cfg.OnGitHubWebhook,
		httpSrv: &http.Server{
			Addr:         cfg.Address,
			Handler:      r,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
	}
	s.registerRoutes()
	return s
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	log.Printf("daemon: HTTP server listening on %s", s.httpSrv.Addr)
	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// Handler returns the HTTP handler for use with httptest.Server.
func (s *Server) Handler() http.Handler {
	return s.router
}
