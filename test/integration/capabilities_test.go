package integration_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCapabilities_TextOutput(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"capabilities"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Available capabilities") {
		t.Error("expected header 'Available capabilities'")
	}
	required := []string{"chat", "doctor", "capabilities", "config", "version", "init"}
	for _, cmd := range required {
		if !strings.Contains(output, cmd) {
			t.Errorf("capabilities output missing command %q", cmd)
		}
	}
}

func TestCapabilities_JSONOutput(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--output", "json", "capabilities"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var result struct {
		Capabilities []struct {
			Command     string `json:"command"`
			Description string `json:"description"`
			Available   bool   `json:"available"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("json decode failed: %v\nraw: %s", err, out.String())
	}

	if len(result.Capabilities) == 0 {
		t.Fatal("expected at least one capability")
	}

	found := false
	for _, cap := range result.Capabilities {
		if cap.Command == "" {
			t.Error("capability has empty command")
		}
		if cap.Description == "" {
			t.Error("capability has empty description")
		}
		if strings.Contains(cap.Command, "chat") {
			found = true
		}
	}
	if !found {
		t.Error("capabilities JSON missing chat command")
	}
}

func TestCapabilities_YAMLOutput(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--output", "yaml", "capabilities"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "capabilities:") {
		t.Errorf("YAML output should contain 'capabilities:', got %q", output)
	}
}

func TestCapabilities_IncludesRepoGroup(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"capabilities"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(out.String(), "repo") {
		t.Error("capabilities should list repo command group")
	}
}

func TestHelpRepo(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"help", "repo"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "repository") {
		t.Errorf("help repo should describe repo operations, got %q", output)
	}
}

func TestHelpChat(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"help", "chat"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(out.String(), "natural language") {
		t.Error("help chat should describe natural language interaction")
	}
}
