package main

import (
	"bytes"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestGitdexEntrypointCommandExecutesVersion(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer

	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if out.Len() == 0 {
		t.Fatal("expected version output")
	}
}
