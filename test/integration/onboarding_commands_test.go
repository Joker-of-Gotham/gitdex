package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestInitWritesGlobalConfigOutsideRepository(t *testing.T) {
	userConfigDir := t.TempDir()
	workingDir := t.TempDir()
	keyPath := filepath.Join(t.TempDir(), "app.pem")
	if err := os.WriteFile(keyPath, []byte("test-key"), 0o600); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}

	t.Setenv("GITDEX_USER_CONFIG_DIR", userConfigDir)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working directory failed: %v", err)
		}
	}()
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("os.Chdir failed: %v", err)
	}

	root := command.NewRootCommand()
	var out bytes.Buffer
	var errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{
		"--output", "json",
		"init",
		"--non-interactive",
		"--default-output", "json",
		"--github-app-id", "123",
		"--github-installation-id", "456",
		"--github-private-key-path", keyPath,
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if _, err := os.Stat(globalConfigPath); err != nil {
		t.Fatalf("expected global config file: %v", err)
	}

	var payload struct {
		WrittenFiles []string `json:"written_files"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal failed: %v; output=%q", err, out.String())
	}
	if len(payload.WrittenFiles) != 1 || payload.WrittenFiles[0] != globalConfigPath {
		t.Fatalf("unexpected written files: %#v", payload.WrittenFiles)
	}

	configOutput := runRootCommand(t, workingDir, func() *cobra.Command { return command.NewRootCommand() }, []string{
		"--output", "json",
		"config", "show",
	})
	var configPayload struct {
		Config struct {
			Output   string `json:"output"`
			LogLevel string `json:"log_level"`
		} `json:"config"`
		Paths struct {
			RepositoryDetected bool     `json:"repository_detected"`
			ActiveFiles        []string `json:"active_files"`
		} `json:"paths"`
		Sources map[string]string `json:"sources"`
	}
	if err := json.Unmarshal([]byte(configOutput), &configPayload); err != nil {
		t.Fatalf("json.Unmarshal(config show) failed: %v; output=%q", err, configOutput)
	}
	if configPayload.Paths.RepositoryDetected {
		t.Fatal("expected repository to remain undetected outside a repo context")
	}
	if len(configPayload.Paths.ActiveFiles) != 1 || configPayload.Paths.ActiveFiles[0] != globalConfigPath {
		t.Fatalf("unexpected active files: %#v", configPayload.Paths.ActiveFiles)
	}
	if configPayload.Config.Output != "json" {
		t.Fatalf("Output = %q, want %q", configPayload.Config.Output, "json")
	}
	if configPayload.Config.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want %q", configPayload.Config.LogLevel, "info")
	}
	if got := configPayload.Sources["log_level"]; got != "global" {
		t.Fatalf("log_level source = %q, want %q", got, "global")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GITDEX_GITHUB_API_URL", server.URL)

	doctorOutput := runRootCommand(t, workingDir, func() *cobra.Command { return command.NewRootCommand() }, []string{
		"--output", "json",
		"doctor",
	})
	var doctorPayload struct {
		Status string `json:"status"`
		Checks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(doctorOutput), &doctorPayload); err != nil {
		t.Fatalf("json.Unmarshal(doctor) failed: %v; output=%q", err, doctorOutput)
	}
	if doctorPayload.Status != "needs_attention" {
		t.Fatalf("doctor status = %q, want %q", doctorPayload.Status, "needs_attention")
	}
	if len(doctorPayload.Checks) != 5 {
		t.Fatalf("len(checks) = %d, want %d", len(doctorPayload.Checks), 5)
	}
	if doctorPayload.Checks[0].ID != "config.load" || doctorPayload.Checks[0].Status != "pass" {
		t.Fatalf("unexpected config.load check: %#v", doctorPayload.Checks[0])
	}
	if doctorPayload.Checks[1].ID != "repository.context" || doctorPayload.Checks[1].Status != "not_configured" {
		t.Fatalf("unexpected repository.context check: %#v", doctorPayload.Checks[1])
	}
}

func TestConfigShowAndDoctorReportStableStructuredOutputFromNestedRepository(t *testing.T) {
	userConfigDir := t.TempDir()
	repoRoot, nestedDir := createRepository(t)
	keyPath := filepath.Join(t.TempDir(), "app.pem")
	if err := os.WriteFile(keyPath, []byte("test-key"), 0o600); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}
	t.Setenv("GITDEX_USER_CONFIG_DIR", userConfigDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GITDEX_GITHUB_API_URL", server.URL)

	runRootCommand(t, nestedDir, func() *cobra.Command { return command.NewRootCommand() }, []string{
		"init",
		"--non-interactive",
		"--write-repo",
		"--default-profile", "repo-profile",
		"--github-app-id", "123",
		"--github-installation-id", "456",
		"--github-private-key-path", keyPath,
	})

	configOutput := runRootCommand(t, nestedDir, func() *cobra.Command { return command.NewRootCommand() }, []string{
		"--output", "json",
		"config", "show",
	})
	var configPayload struct {
		Config struct {
			Profile string `json:"profile"`
		} `json:"config"`
		Paths struct {
			RepositoryDetected bool   `json:"repository_detected"`
			RepositoryRoot     string `json:"repository_root"`
		} `json:"paths"`
		Sources map[string]string `json:"sources"`
	}
	if err := json.Unmarshal([]byte(configOutput), &configPayload); err != nil {
		t.Fatalf("json.Unmarshal(config show) failed: %v; output=%q", err, configOutput)
	}
	if !configPayload.Paths.RepositoryDetected {
		t.Fatal("expected repository to be detected")
	}
	if configPayload.Paths.RepositoryRoot != repoRoot {
		t.Fatalf("RepositoryRoot = %q, want %q", configPayload.Paths.RepositoryRoot, repoRoot)
	}
	if configPayload.Config.Profile != "repo-profile" {
		t.Fatalf("Profile = %q, want %q", configPayload.Config.Profile, "repo-profile")
	}
	if got := configPayload.Sources["profile"]; got != "repo" {
		t.Fatalf("profile source = %q, want %q", got, "repo")
	}

	doctorOutput := runRootCommand(t, nestedDir, func() *cobra.Command { return command.NewRootCommand() }, []string{
		"--output", "json",
		"doctor",
	})
	var doctorPayload struct {
		Status string `json:"status"`
		Checks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(doctorOutput), &doctorPayload); err != nil {
		t.Fatalf("json.Unmarshal(doctor) failed: %v; output=%q", err, doctorOutput)
	}
	if doctorPayload.Status != "pass" {
		t.Fatalf("doctor status = %q, want %q", doctorPayload.Status, "pass")
	}
	if len(doctorPayload.Checks) != 5 {
		t.Fatalf("len(checks) = %d, want %d", len(doctorPayload.Checks), 5)
	}
}

func runRootCommand(t *testing.T, workingDir string, factory func() *cobra.Command, args []string) string {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working directory failed: %v", err)
		}
	}()
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("os.Chdir failed: %v", err)
	}

	root := factory()
	var out bytes.Buffer
	var errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v; stderr=%q", err, errOut.String())
	}

	return out.String()
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
