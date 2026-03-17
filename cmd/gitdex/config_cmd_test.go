package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunConfigCommand_UnknownSubcommand(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	t.Setenv("GITDEX_LLM_PRIMARY_MODEL", "qwen2.5:3b")

	if err := runConfigCommand([]string{"unknown"}); err == nil {
		t.Fatalf("expected error for unknown config subcommand")
	}
}

func TestMaybeHandleConfigCommand_NonConfig(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"gitdex"}
	if err := maybeHandleConfigCommand(); err != nil {
		t.Fatalf("expected nil for non-config command, got: %v", err)
	}
}

