package conformance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	configpkg "github.com/your-org/gitdex/internal/platform/config"
)

func TestStarterSkeletonContainsRequiredPaths(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs failed: %v", err)
	}

	requiredPaths := []string{
		"README.md",
		".gitignore",
		".env.example",
		"Taskfile.yml",
		"Makefile",
		".golangci.yml",
		".goreleaser.yml",
		"configs/gitdex.example.yaml",
		"configs/policies/default/global.yaml",
		"configs/policies/default/repo_class_public.yaml",
		"configs/policies/default/repo_class_sensitive.yaml",
		"configs/policies/default/repo_class_release_critical.yaml",
		"cmd/gitdex/main.go",
		"cmd/gitdexd/main.go",
		"internal/app/bootstrap",
		"internal/app/version",
		"internal/cli/command/root.go",
		"internal/cli/completion/completion.go",
		"internal/cli/output",
		"internal/platform/config/config.go",
		"internal/platform/ids",
		"internal/platform/logging",
		"pkg/contracts/plan",
		"pkg/contracts/task",
		"pkg/contracts/audit",
		"pkg/contracts/campaign",
		"pkg/contracts/handoff",
		"pkg/contracts/api",
		"schema/json/plan.schema.json",
		"schema/json/task.schema.json",
		"schema/json/campaign.schema.json",
		"schema/json/audit_event.schema.json",
		"schema/json/handoff_pack.schema.json",
		"schema/json/api_error.schema.json",
		"schema/openapi/control_plane.yaml",
		"migrations/000001_init.sql",
		"migrations/000002_task_events.sql",
		"migrations/000003_repo_projections.sql",
		"migrations/000004_audit_records.sql",
		"scripts/dev",
		"scripts/ci",
		"scripts/fixtures",
		"test/integration",
		"test/e2e",
		"test/contracts",
		"test/conformance",
		"test/fixtures/repos",
		"test/fixtures/policies",
		"test/fixtures/webhooks",
		"test/fixtures/campaigns",
	}

	for _, rel := range requiredPaths {
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err != nil {
			t.Fatalf("required starter path %q missing: %v", rel, err)
		}
	}
}

func TestGoreleaserInjectsBuildVersion(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, ".goreleaser.yml"))
	if err != nil {
		t.Fatalf("os.ReadFile failed: %v", err)
	}

	const want = "github.com/your-org/gitdex/internal/app/version.Version={{.Version}}"
	if !strings.Contains(string(content), want) {
		t.Fatalf(".goreleaser.yml does not inject version with %q", want)
	}
}

func TestExampleConfigCoversStory12OnboardingFields(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs failed: %v", err)
	}

	cfg, err := configpkg.Load(configpkg.Options{
		RepoRoot:      repoRoot,
		WorkingDir:    repoRoot,
		UserConfigDir: t.TempDir(),
		ConfigFile:    filepath.Join(repoRoot, "configs", "gitdex.example.yaml"),
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Identity.Mode != "github-app" {
		t.Fatalf("Identity.Mode = %q, want %q", cfg.Identity.Mode, "github-app")
	}
	if cfg.Identity.GitHubApp.Host != "github.com" {
		t.Fatalf("Identity.GitHubApp.Host = %q, want %q", cfg.Identity.GitHubApp.Host, "github.com")
	}
	if cfg.Identity.GitHubApp.AppID == "" {
		t.Fatal("expected example config to include identity.github_app.app_id")
	}
	if cfg.Identity.GitHubApp.InstallationID == "" {
		t.Fatal("expected example config to include identity.github_app.installation_id")
	}
	if cfg.Identity.GitHubApp.PrivateKeyPath == "" {
		t.Fatal("expected example config to include identity.github_app.private_key_path")
	}
	if cfg.Daemon.HealthAddress == "" {
		t.Fatal("expected example config to include daemon.health_address")
	}
}
