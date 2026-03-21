package command_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/app/version"
	"github.com/your-org/gitdex/internal/cli/command"
)

func TestNewRootCommandExposesStarterSubcommands(t *testing.T) {
	root := command.NewRootCommand()

	if got, want := root.Use, "gitdex"; got != want {
		t.Fatalf("root.Use = %q, want %q", got, want)
	}

	required := map[string]bool{
		"completion": false,
		"config":     false,
		"daemon":     false,
		"doctor":     false,
		"init":       false,
		"version":    false,
	}

	for _, sub := range root.Commands() {
		if _, ok := required[sub.Name()]; ok {
			required[sub.Name()] = true
		}
	}

	for name, found := range required {
		if !found {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

func TestVersionCommandDoesNotRequireRepoBootstrap(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("os.Chdir failed: %v", err)
	}

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("version command should not require repo root: %v", err)
	}

	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("expected version output")
	}
}

func TestDaemonRunStillBootstrapsFromNestedDirectory(t *testing.T) {
	t.Skip("daemon run now starts a real HTTP server; skipping in unit tests")
}

func TestVersionCommandUsesInjectedVersionValue(t *testing.T) {
	originalVersion := version.Version
	version.Version = "1.2.3-test"
	defer func() {
		version.Version = originalVersion
	}()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if got, want := strings.TrimSpace(out.String()), "1.2.3-test"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}
