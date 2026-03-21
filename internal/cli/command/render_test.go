package command

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/app/doctor"
	"github.com/your-org/gitdex/internal/app/setup"
	"github.com/your-org/gitdex/internal/platform/config"
)

func TestRenderSetupTextIncludesWrittenFilesAndNextSteps(t *testing.T) {
	var out bytes.Buffer

	err := renderSetupText(&out, setup.Result{
		WrittenFiles: []string{"/tmp/global.yaml", "/tmp/repo.yaml"},
		Config: config.Snapshot{
			Config: config.FileConfig{
				Output:   "json",
				LogLevel: "debug",
				Profile:  "repo",
			},
			Sources: map[string]config.ValueSource{
				"output":    config.SourceGlobal,
				"log_level": config.SourceEnv,
				"profile":   config.SourceRepo,
			},
		},
		NextSteps: []string{"gitdex doctor", "gitdex config show"},
	})
	if err != nil {
		t.Fatalf("renderSetupText returned error: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		"Setup complete",
		"/tmp/global.yaml",
		"/tmp/repo.yaml",
		"Output: json (source: global)",
		"Log level: debug (source: env)",
		"Profile: repo (source: repo)",
		"gitdex doctor",
		"gitdex config show",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered setup output missing %q: %q", want, rendered)
		}
	}
}

func TestRenderDoctorTextIncludesFixAndSource(t *testing.T) {
	var out bytes.Buffer

	err := renderDoctorText(&out, doctor.Report{
		Status: "needs_attention",
		Checks: []doctor.Check{
			{
				ID:      "identity.github_app",
				Status:  doctor.StatusIncomplete,
				Summary: "GitHub App identity is incomplete",
				Detail:  "Missing installation ID",
				Fix:     "Fill the missing field and rerun doctor.",
				Source:  "identity.github_app",
			},
		},
	})
	if err != nil {
		t.Fatalf("renderDoctorText returned error: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		"Doctor status: needs_attention",
		"[incomplete] identity.github_app",
		"Summary: GitHub App identity is incomplete",
		"Detail: Missing installation ID",
		"Fix: Fill the missing field and rerun doctor.",
		"Source: identity.github_app",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered doctor output missing %q: %q", want, rendered)
		}
	}
}

func TestRenderConfigTextIncludesSourcesAndPaths(t *testing.T) {
	var out bytes.Buffer

	err := renderConfigText(&out, config.Snapshot{
		Config: config.FileConfig{
			Output:   "yaml",
			LogLevel: "warn",
			Profile:  "repo",
			Identity: config.IdentityConfig{
				Mode: "github-app",
				GitHubApp: config.GitHubAppConfig{
					Host: "ghe.example.test",
				},
			},
			Git: config.GitConfig{
				WorkspaceRoots: []string{"D:/Code", "E:/Repos"},
			},
			Storage: config.StorageConfig{
				Type: "sqlite",
			},
		},
		Paths: config.ConfigPaths{
			GlobalConfig:   "/tmp/global.yaml",
			RepoConfig:     "/repo/.gitdex/config.yaml",
			RepositoryRoot: "/repo",
			ActiveFiles:    []string{"/tmp/global.yaml", "/repo/.gitdex/config.yaml"},
		},
		Sources: map[string]config.ValueSource{
			"output":                   config.SourceGlobal,
			"log_level":                config.SourceEnv,
			"profile":                  config.SourceRepo,
			"identity.mode":            config.SourceDefault,
			"identity.github_app.host": config.SourceGlobal,
			"storage.type":             config.SourceGlobal,
		},
	})
	if err != nil {
		t.Fatalf("renderConfigText returned error: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		"Effective Gitdex configuration",
		"Output: yaml (source: global)",
		"Log level: warn (source: env)",
		"Profile: repo (source: repo)",
		"Identity mode: github-app (source: default)",
		"GitHub host: ghe.example.test (source: global)",
		"Workspace roots: D:/Code, E:/Repos",
		"Storage backend: sqlite (source: global)",
		"Storage dsn: (auto/default)",
		"Global config: /tmp/global.yaml",
		"Active config files: /tmp/global.yaml, /repo/.gitdex/config.yaml",
		"Repo config: /repo/.gitdex/config.yaml",
		"Repository root: /repo",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered config output missing %q: %q", want, rendered)
		}
	}
}

func TestNormalizeConfigSnapshotFillsDefaultSQLiteDSN(t *testing.T) {
	snapshot := normalizeConfigSnapshot(config.Snapshot{
		Config: config.FileConfig{
			Storage: config.StorageConfig{
				Type: "sqlite",
			},
		},
		Paths: config.ConfigPaths{
			GlobalConfig: "/tmp/gitdex/config.yaml",
		},
	})

	if got, want := snapshot.Config.Storage.DSN, filepath.Join("/tmp/gitdex", "gitdex.sqlite"); got != want {
		t.Fatalf("Storage.DSN = %q, want %q", got, want)
	}
}
