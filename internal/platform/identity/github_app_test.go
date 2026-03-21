package identity

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/platform/config"
)

func generateTestKey(t *testing.T, dir string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
	path := filepath.Join(dir, "test-key.pem")
	if err := os.WriteFile(path, pemBlock, 0600); err != nil {
		t.Fatalf("write key file: %v", err)
	}
	return path
}

func validConfig(keyPath string) config.IdentityConfig {
	return config.IdentityConfig{
		Mode: "github-app",
		GitHubApp: config.GitHubAppConfig{
			AppID:          "12345",
			InstallationID: "67890",
			PrivateKeyPath: keyPath,
			Host:           "github.com",
		},
	}
}

func TestNewGitHubAppTransport_Success(t *testing.T) {
	dir := t.TempDir()
	keyPath := generateTestKey(t, dir)

	result, err := NewGitHubAppTransport(validConfig(keyPath), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Transport == nil {
		t.Fatal("expected non-nil transport")
	}
	if result.Host != "github.com" {
		t.Errorf("host = %q, want %q", result.Host, "github.com")
	}
}

func TestNewGitHubAppTransport_GHESHost(t *testing.T) {
	dir := t.TempDir()
	keyPath := generateTestKey(t, dir)

	cfg := validConfig(keyPath)
	cfg.GitHubApp.Host = "github.example.com"

	result, err := NewGitHubAppTransport(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Host != "github.example.com" {
		t.Errorf("host = %q, want %q", result.Host, "github.example.com")
	}
}

func TestNewGitHubAppTransport_ErrNoIdentity_WrongMode(t *testing.T) {
	_, err := NewGitHubAppTransport(config.IdentityConfig{Mode: "pat"}, nil)
	if !errors.Is(err, ErrNoIdentity) {
		t.Errorf("expected ErrNoIdentity, got %v", err)
	}
}

func TestNewGitHubAppTransport_ErrMissingField(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.IdentityConfig
	}{
		{
			name: "missing_app_id",
			cfg: config.IdentityConfig{
				Mode:      "github-app",
				GitHubApp: config.GitHubAppConfig{InstallationID: "1", PrivateKeyPath: "/tmp/k.pem"},
			},
		},
		{
			name: "missing_installation_id",
			cfg: config.IdentityConfig{
				Mode:      "github-app",
				GitHubApp: config.GitHubAppConfig{AppID: "1", PrivateKeyPath: "/tmp/k.pem"},
			},
		},
		{
			name: "missing_private_key_path",
			cfg: config.IdentityConfig{
				Mode:      "github-app",
				GitHubApp: config.GitHubAppConfig{AppID: "1", InstallationID: "1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGitHubAppTransport(tt.cfg, nil)
			if !errors.Is(err, ErrMissingField) {
				t.Errorf("expected ErrMissingField, got %v", err)
			}
		})
	}
}

func TestNewGitHubAppTransport_ErrInvalidAppID(t *testing.T) {
	_, err := NewGitHubAppTransport(config.IdentityConfig{
		Mode: "github-app",
		GitHubApp: config.GitHubAppConfig{
			AppID:          "notanumber",
			InstallationID: "1",
			PrivateKeyPath: "/tmp/k.pem",
		},
	}, nil)
	if !errors.Is(err, ErrMissingField) {
		t.Errorf("expected ErrMissingField, got %v", err)
	}
}

func TestNewGitHubAppTransport_ErrKeyFileNotExist(t *testing.T) {
	_, err := NewGitHubAppTransport(config.IdentityConfig{
		Mode: "github-app",
		GitHubApp: config.GitHubAppConfig{
			AppID:          "1",
			InstallationID: "1",
			PrivateKeyPath: "/nonexistent/key.pem",
		},
	}, nil)
	if !errors.Is(err, ErrInvalidKeyFile) {
		t.Errorf("expected ErrInvalidKeyFile, got %v", err)
	}
}

func TestNewGitHubAppTransport_ErrInvalidKeyContent(t *testing.T) {
	dir := t.TempDir()
	badKeyPath := filepath.Join(dir, "bad-key.pem")
	if err := os.WriteFile(badKeyPath, []byte("not a valid PEM key"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := NewGitHubAppTransport(config.IdentityConfig{
		Mode: "github-app",
		GitHubApp: config.GitHubAppConfig{
			AppID:          "1",
			InstallationID: "1",
			PrivateKeyPath: badKeyPath,
		},
	}, nil)
	if !errors.Is(err, ErrInvalidKeyFile) {
		t.Errorf("expected ErrInvalidKeyFile, got %v", err)
	}
}

func TestIsIdentityConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.IdentityConfig
		want bool
	}{
		{"fully_configured", config.IdentityConfig{Mode: "github-app", GitHubApp: config.GitHubAppConfig{AppID: "1", InstallationID: "2", PrivateKeyPath: "/k.pem"}}, true},
		{"wrong_mode", config.IdentityConfig{Mode: "pat", GitHubApp: config.GitHubAppConfig{AppID: "1", InstallationID: "2", PrivateKeyPath: "/k.pem"}}, false},
		{"missing_app_id", config.IdentityConfig{Mode: "github-app", GitHubApp: config.GitHubAppConfig{InstallationID: "2", PrivateKeyPath: "/k.pem"}}, false},
		{"empty_config", config.IdentityConfig{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIdentityConfigured(tt.cfg)
			if got != tt.want {
				t.Errorf("IsIdentityConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveTransport_UsesEnvironmentTokenFallback(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_env_fallback")

	result, err := ResolveTransport(config.IdentityConfig{
		Mode: "token",
		GitHubApp: config.GitHubAppConfig{
			Host: "github.com22321321123",
		},
	}, nil)
	if err != nil {
		t.Fatalf("ResolveTransport returned error: %v", err)
	}
	if result == nil || result.Transport == nil {
		t.Fatal("expected non-nil transport")
	}
	if result.Host != "github.com" {
		t.Fatalf("host = %q, want %q", result.Host, "github.com")
	}
	if result.Source != SourceEnvToken {
		t.Fatalf("source = %q, want %q", result.Source, SourceEnvToken)
	}
}

func TestResolveTransport_UsesGitHubCLIWhenConfigAndEnvAreMissing(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GH_ENTERPRISE_TOKEN", "")
	t.Setenv("GITHUB_ENTERPRISE_TOKEN", "")
	originalRunner := ghTokenRunner
	t.Cleanup(func() { ghTokenRunner = originalRunner })
	ghTokenRunner = func(host string) (string, error) {
		if host == "github.com" {
			return "ghp_cli_token", nil
		}
		return "", os.ErrNotExist
	}

	result, err := ResolveTransport(config.IdentityConfig{
		Mode: "token",
		GitHubApp: config.GitHubAppConfig{
			Host: "github.invalid.local",
		},
	}, nil)
	if err != nil {
		t.Fatalf("ResolveTransport returned error: %v", err)
	}
	if result.Host != "github.com" {
		t.Fatalf("host = %q, want github.com", result.Host)
	}
	if result.Source != SourceGitHubCLI {
		t.Fatalf("source = %q, want %q", result.Source, SourceGitHubCLI)
	}
}
