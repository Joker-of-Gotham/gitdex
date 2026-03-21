package doctor_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/app/doctor"
	"github.com/your-org/gitdex/internal/platform/config"
)

func TestRunReportsHealthyConfiguredEnvironment(t *testing.T) {
	userConfigDir := t.TempDir()
	_, nestedDir := createRepository(t)
	keyPath := filepath.Join(t.TempDir(), "app.pem")
	if err := os.WriteFile(keyPath, []byte("test-key"), 0o600); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}

	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if err := config.WriteFile(globalConfigPath, config.FileConfig{
		Output:   "json",
		LogLevel: "debug",
		Profile:  "local",
		Identity: config.IdentityConfig{
			Mode: "github-app",
			GitHubApp: config.GitHubAppConfig{
				Host:           "github.com",
				AppID:          "123",
				InstallationID: "456",
				PrivateKeyPath: keyPath,
			},
		},
	}); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GITDEX_GITHUB_API_URL", server.URL)

	report, err := doctor.Run(doctor.Options{
		ConfigOptions: config.Options{
			WorkingDir:    nestedDir,
			UserConfigDir: userConfigDir,
		},
		LookPath: func(file string) (string, error) {
			return "git", nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.Status != "pass" {
		t.Fatalf("Status = %q, want %q", report.Status, "pass")
	}
	if len(report.Checks) != 5 {
		t.Fatalf("len(Checks) = %d, want %d", len(report.Checks), 5)
	}
}

func TestRunReportsMissingConfigAndRepositoryContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GITDEX_GITHUB_API_URL", server.URL)

	report, err := doctor.Run(doctor.Options{
		ConfigOptions: config.Options{
			WorkingDir:    t.TempDir(),
			UserConfigDir: t.TempDir(),
		},
		LookPath: func(file string) (string, error) {
			return "git", nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.Status != "needs_attention" {
		t.Fatalf("Status = %q, want %q", report.Status, "needs_attention")
	}
	if report.Checks[0].Status != doctor.StatusNotConfigured {
		t.Fatalf("config check status = %q, want %q", report.Checks[0].Status, doctor.StatusNotConfigured)
	}
	if report.Checks[1].Status != doctor.StatusNotConfigured {
		t.Fatalf("repository check status = %q, want %q", report.Checks[1].Status, doctor.StatusNotConfigured)
	}
}

func TestRunReportsIncompleteIdentity(t *testing.T) {
	userConfigDir := t.TempDir()
	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if err := config.WriteFile(globalConfigPath, config.FileConfig{
		Identity: config.IdentityConfig{
			Mode: "github-app",
			GitHubApp: config.GitHubAppConfig{
				Host:  "github.com",
				AppID: "123",
			},
		},
	}); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GITDEX_GITHUB_API_URL", server.URL)

	report, err := doctor.Run(doctor.Options{
		ConfigOptions: config.Options{
			WorkingDir:    t.TempDir(),
			UserConfigDir: userConfigDir,
		},
		LookPath: func(file string) (string, error) {
			return "git", nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.Checks[2].Status != doctor.StatusIncomplete {
		t.Fatalf("identity check status = %q, want %q", report.Checks[2].Status, doctor.StatusIncomplete)
	}
}

func TestRunReportsIncompleteTokenIdentity(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GH_ENTERPRISE_TOKEN", "")
	t.Setenv("GITHUB_ENTERPRISE_TOKEN", "")
	userConfigDir := t.TempDir()
	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if err := config.WriteFile(globalConfigPath, config.FileConfig{
		Identity: config.IdentityConfig{
			Mode: "token",
			GitHubApp: config.GitHubAppConfig{
				Host: "github.com",
			},
		},
	}); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GITDEX_GITHUB_API_URL", server.URL)

	report, err := doctor.Run(doctor.Options{
		ConfigOptions: config.Options{
			WorkingDir:    t.TempDir(),
			UserConfigDir: userConfigDir,
		},
		LookPath: func(file string) (string, error) {
			return "git", nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.Checks[2].Status != doctor.StatusIncomplete {
		t.Fatalf("identity check status = %q, want %q", report.Checks[2].Status, doctor.StatusIncomplete)
	}
	if report.Checks[2].Source != "identity.github_pat" {
		t.Fatalf("identity check source = %q, want %q", report.Checks[2].Source, "identity.github_pat")
	}
}

func TestRunReportsConfiguredTokenIdentity(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GH_ENTERPRISE_TOKEN", "")
	t.Setenv("GITHUB_ENTERPRISE_TOKEN", "")
	userConfigDir := t.TempDir()
	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if err := config.WriteFile(globalConfigPath, config.FileConfig{
		Identity: config.IdentityConfig{
			Mode:      "token",
			GitHubPAT: "ghp_test",
			GitHubApp: config.GitHubAppConfig{
				Host: "github.com",
			},
		},
	}); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GITDEX_GITHUB_API_URL", server.URL)

	report, err := doctor.Run(doctor.Options{
		ConfigOptions: config.Options{
			WorkingDir:    t.TempDir(),
			UserConfigDir: userConfigDir,
		},
		LookPath: func(file string) (string, error) {
			return "git", nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.Checks[2].Status != doctor.StatusPass {
		t.Fatalf("identity check status = %q, want %q", report.Checks[2].Status, doctor.StatusPass)
	}
}

func TestRunAcceptsRuntimeFallbackTokenIdentity(t *testing.T) {
	userConfigDir := t.TempDir()
	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if err := config.WriteFile(globalConfigPath, config.FileConfig{
		Identity: config.IdentityConfig{
			Mode: "token",
			GitHubApp: config.GitHubAppConfig{
				Host: "github.com22321321123",
			},
		},
	}); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("GITHUB_TOKEN", "ghp_runtime_fallback")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GITDEX_GITHUB_API_URL", server.URL)

	report, err := doctor.Run(doctor.Options{
		ConfigOptions: config.Options{
			WorkingDir:    t.TempDir(),
			UserConfigDir: userConfigDir,
		},
		LookPath: func(file string) (string, error) {
			return "git", nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.Checks[2].Status != doctor.StatusPass {
		t.Fatalf("identity check status = %q, want %q", report.Checks[2].Status, doctor.StatusPass)
	}
}

func createRepository(t *testing.T) (string, string) {
	t.Helper()

	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("os.Mkdir(.git) failed: %v", err)
	}

	nestedDir := filepath.Join(repoRoot, "nested", "deeper")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll failed: %v", err)
	}

	return repoRoot, nestedDir
}
