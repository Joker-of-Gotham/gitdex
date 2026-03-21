package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
	"github.com/your-org/gitdex/internal/collaboration"
)

func TestCollabCreate_RequiresFlags(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"collab", "create"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when required flags omitted")
	}
}

func TestCollabCreate_ThenShow(t *testing.T) {
	objStore := collaboration.NewMemoryObjectStore()
	ctxStore := collaboration.NewMemoryContextStore()
	restoreObj := command.SetCollabObjectStoreForTest(objStore)
	restoreCtx := command.SetCollabContextStoreForTest(ctxStore)
	defer restoreObj()
	defer restoreCtx()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"collab", "create", "--type", "issue", "--repo", "owner/repo", "--title", "Test issue title"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when GitHub is not configured")
	}
	if !strings.Contains(err.Error(), "GitHub authentication") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollabComment_RequiresBody(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"collab", "comment", "owner/repo#1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --body omitted")
	}
}

func TestCollabClose_ThenReopen(t *testing.T) {
	objStore := collaboration.NewMemoryObjectStore()
	ctxStore := collaboration.NewMemoryContextStore()
	restoreObj := command.SetCollabObjectStoreForTest(objStore)
	restoreCtx := command.SetCollabContextStoreForTest(ctxStore)
	defer restoreObj()
	defer restoreCtx()

	root := command.NewRootCommand()

	root.SetArgs([]string{"collab", "create", "--type", "issue", "--repo", "x/y", "--title", "To close"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when GitHub is not configured")
	}
	if !strings.Contains(err.Error(), "GitHub authentication") {
		t.Fatalf("unexpected error: %v", err)
	}
}
