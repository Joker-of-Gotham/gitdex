package dotgitdex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	mgr := New("/tmp/repo")
	if mgr.Root != filepath.Join("/tmp/repo", dirName) {
		t.Errorf("Root = %q, want %q", mgr.Root, filepath.Join("/tmp/repo", dirName))
	}
}

func TestInit(t *testing.T) {
	tmp := t.TempDir()
	mgr := New(tmp)
	if err := mgr.Init(); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{mgr.MaintainDir(), mgr.KnowledgeDir(), mgr.GoalListDir(), mgr.ProposalDir()} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory %q to exist", dir)
		}
	}
}
