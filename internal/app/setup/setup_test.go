package setup_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/app/setup"
)

func TestRunWritesGlobalAndRepoConfigsNonInteractively(t *testing.T) {
	userConfigDir := t.TempDir()
	repoRoot, nestedDir := createRepository(t)
	keyPath := filepath.Join(t.TempDir(), "app.pem")
	if err := os.WriteFile(keyPath, []byte("test-key"), 0o600); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}

	result, err := setup.Run(setup.Options{
		WorkingDir:           nestedDir,
		UserConfigDir:        userConfigDir,
		NonInteractive:       true,
		DefaultOutput:        "json",
		DefaultLogLevel:      "debug",
		DefaultProfile:       "repo-profile",
		GitHubAppID:          "123",
		GitHubInstallationID: "456",
		GitHubPrivateKeyPath: keyPath,
		WriteGlobal:          true,
		WriteGlobalSet:       true,
		WriteRepo:            true,
		WriteRepoSet:         true,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(result.WrittenFiles) != 2 {
		t.Fatalf("len(WrittenFiles) = %d, want %d", len(result.WrittenFiles), 2)
	}

	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if _, err := os.Stat(globalConfigPath); err != nil {
		t.Fatalf("expected global config file: %v", err)
	}

	repoConfigPath := filepath.Join(repoRoot, ".gitdex", "config.yaml")
	if _, err := os.Stat(repoConfigPath); err != nil {
		t.Fatalf("expected repo config file: %v", err)
	}

	if result.Config.Config.Output != "json" {
		t.Fatalf("Config.Output = %q, want %q", result.Config.Config.Output, "json")
	}
	if result.Config.Config.Profile != "repo-profile" {
		t.Fatalf("Config.Profile = %q, want %q", result.Config.Config.Profile, "repo-profile")
	}
	if result.Config.Sources["profile"] != "repo" {
		t.Fatalf("profile source = %q, want %q", result.Config.Sources["profile"], "repo")
	}
}

func TestRunSupportsInteractiveInput(t *testing.T) {
	userConfigDir := t.TempDir()
	workingDir := t.TempDir()
	keyPath := filepath.Join(t.TempDir(), "interactive.pem")
	if err := os.WriteFile(keyPath, []byte("interactive-key"), 0o600); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}

	input := bytes.NewBufferString("\n\n123\n456\n" + keyPath + "\njson\nwarn\nlocal\ny\n")
	var prompts bytes.Buffer

	result, err := setup.Run(setup.Options{
		In:            input,
		Out:           &prompts,
		WorkingDir:    workingDir,
		UserConfigDir: userConfigDir,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(result.WrittenFiles) != 1 {
		t.Fatalf("len(WrittenFiles) = %d, want %d", len(result.WrittenFiles), 1)
	}
	if result.Config.Config.Output != "json" {
		t.Fatalf("Config.Output = %q, want %q", result.Config.Config.Output, "json")
	}
	if prompts.Len() == 0 {
		t.Fatal("expected interactive prompts to be written")
	}
}

func TestRunRejectsRepoWriteWithoutRepositoryContext(t *testing.T) {
	_, err := setup.Run(setup.Options{
		WorkingDir:     t.TempDir(),
		UserConfigDir:  t.TempDir(),
		NonInteractive: true,
		WriteGlobal:    false,
		WriteGlobalSet: true,
		WriteRepo:      true,
		WriteRepoSet:   true,
	})
	if err == nil {
		t.Fatal("expected error when repo write is requested without repository context")
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
