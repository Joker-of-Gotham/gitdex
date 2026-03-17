package app

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
)

func TestNew_ReturnsNonNil(t *testing.T) {
	a := New(Config{Version: "test"})
	if a == nil {
		t.Fatal("New returned nil")
	}
}

func TestResolveMode_Manual(t *testing.T) {
	cfg := &config.Config{}
	cfg.Automation.Mode = "manual"
	got := resolveMode(cfg)
	if got != "manual" {
		t.Errorf("expected manual, got %s", got)
	}
}

func TestResolveMode_Auto(t *testing.T) {
	cfg := &config.Config{}
	cfg.Automation.Mode = "auto"
	got := resolveMode(cfg)
	if got != "auto" {
		t.Errorf("expected auto, got %s", got)
	}
}

func TestResolveMode_Cruise(t *testing.T) {
	cfg := &config.Config{}
	cfg.Automation.Mode = "cruise"
	got := resolveMode(cfg)
	if got != "cruise" {
		t.Errorf("expected cruise, got %s", got)
	}
}

func TestResolveMode_Default(t *testing.T) {
	cfg := &config.Config{}
	cfg.Automation.Mode = "invalid"
	got := resolveMode(cfg)
	if got != "manual" {
		t.Errorf("expected manual for invalid mode, got %s", got)
	}
}

func TestResolveMode_Empty(t *testing.T) {
	cfg := &config.Config{}
	got := resolveMode(cfg)
	if got != "manual" {
		t.Errorf("expected manual for empty mode, got %s", got)
	}
}

func TestResolveGitBinary_FromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Adapters.Git.Binary = "custom-git"
	got := resolveGitBinary(cfg)
	if got != "custom-git" {
		t.Fatalf("expected custom-git, got %q", got)
	}
}

func TestResolveGitBinary_DefaultFallback(t *testing.T) {
	var cfg *config.Config
	got := resolveGitBinary(cfg)
	if got != "git" {
		t.Fatalf("expected git fallback, got %q", got)
	}
}
