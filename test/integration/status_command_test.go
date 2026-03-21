package integration_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func initTestGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s\n%v", args, out, err)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	return dir
}

func TestStatusCommand_TextOutput_NoGitHub(t *testing.T) {
	dir := initTestGitRepo(t)

	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)
	t.Setenv("GITDEX_OUTPUT", "text")

	root := command.NewRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"status", "--owner", "test-owner", "--repo", "test-repo"})

	oldWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(oldWd) }()

	err := root.Execute()
	if err != nil {
		t.Fatalf("command error: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "test-owner/test-repo") {
		t.Errorf("output missing owner/repo, got:\n%s", output)
	}
	if !strings.Contains(output, "healthy") && !strings.Contains(output, "unknown") {
		t.Errorf("output missing state label, got:\n%s", output)
	}
	if !strings.Contains(output, "Local") {
		t.Errorf("output missing Local section, got:\n%s", output)
	}
	if !strings.Contains(output, "Remote") {
		t.Errorf("output missing Remote section, got:\n%s", output)
	}
}

func TestStatusCommand_JSONOutput_NoGitHub(t *testing.T) {
	dir := initTestGitRepo(t)

	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"status", "--owner", "test-owner", "--repo", "test-repo", "--output", "json"})

	oldWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(oldWd) }()

	err := root.Execute()
	if err != nil {
		t.Fatalf("command error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, stdout.String())
	}

	requiredFields := []string{"owner", "repo", "overall_label", "local", "remote", "collaboration", "workflows", "deployments"}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("JSON output missing field %q", field)
		}
	}

	if result["owner"] != "test-owner" {
		t.Errorf("owner = %q, want %q", result["owner"], "test-owner")
	}
}

func TestStatusCommand_YAMLOutput_NoGitHub(t *testing.T) {
	dir := initTestGitRepo(t)

	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"status", "--owner", "test-owner", "--repo", "test-repo", "--output", "yaml"})

	oldWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(oldWd) }()

	err := root.Execute()
	if err != nil {
		t.Fatalf("command error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "owner: test-owner") {
		t.Errorf("YAML output missing owner field, got:\n%s", output)
	}
	if !strings.Contains(output, "overall_label:") {
		t.Errorf("YAML output missing overall_label, got:\n%s", output)
	}
}

func TestStatusCommand_MissingOwnerRepo(t *testing.T) {
	dir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"status"})

	oldWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(oldWd) }()

	err := root.Execute()
	if err == nil {
		t.Error("expected error when owner/repo cannot be determined")
	}
}
