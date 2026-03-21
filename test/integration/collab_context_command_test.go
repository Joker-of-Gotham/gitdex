package integration_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCollabLinkCommand_Registered(t *testing.T) {
	root := command.NewRootCommand()
	cmd, _, err := root.Find([]string{"collab", "link"})
	if err != nil {
		t.Fatalf("collab link not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("collab link command is nil")
	}
}

func TestCollabLinkCommand_RequiresArgs(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "link"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when args missing")
	}
}

func TestCollabLinkCommand_Runs(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "link", "owner/repo#1", "owner/repo#2", "--type", "relates_to"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("collab link failed: %v", err)
	}
}

func TestCollabContextCommand_Registered(t *testing.T) {
	root := command.NewRootCommand()
	cmd, _, err := root.Find([]string{"collab", "context"})
	if err != nil {
		t.Fatalf("collab context not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("collab context command is nil")
	}
}

func TestCollabContextCommand_RequiresArg(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "context"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when object_ref missing")
	}
}

func TestCollabContextCommand_ShowsContext(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "context", "owner/repo#1"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("collab context failed: %v", err)
	}
}

func TestCollabContextCommand_JSONOutput(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"collab", "context", "owner/repo#1", "--output", "json"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("collab context failed: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, stdout.String())
	}
	if _, ok := result["primary_object_ref"]; !ok {
		t.Errorf("JSON missing primary_object_ref: %v", result)
	}
}
