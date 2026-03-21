package integration_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCollabTriageCommand_Registered(t *testing.T) {
	root := command.NewRootCommand()
	cmd, _, err := root.Find([]string{"collab", "triage"})
	if err != nil {
		t.Fatalf("collab triage not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("collab triage command is nil")
	}
}

func TestCollabTriageCommand_RequiresRepo(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "triage"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when repo cannot be determined")
	}
}

func TestCollabTriageCommand_WithRepo(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "triage", "--repo", "owner/repo"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("collab triage failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Triage Results") && !strings.Contains(stdout.String(), "No objects") {
		t.Errorf("unexpected output: %s", stdout.String())
	}
}

func TestCollabTriageCommand_JSONOutput(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "triage", "--repo", "owner/repo", "--output", "json"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("collab triage failed: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, stdout.String())
	}
	if _, ok := result["triage_results"]; !ok {
		t.Errorf("JSON missing triage_results: %v", result)
	}
}

func TestCollabSummaryCommand_WithRepo(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "summary", "--repo", "owner/repo", "--period", "7d"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("collab summary failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Activity Summary") && !strings.Contains(stdout.String(), "owner/repo") {
		t.Errorf("unexpected output: %s", stdout.String())
	}
}
