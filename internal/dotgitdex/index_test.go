package dotgitdex

import (
	"testing"
)

func TestWriteReadIndex(t *testing.T) {
	tmp := t.TempDir()
	mgr := New(tmp)
	_ = mgr.Init()

	entries := []IndexEntry{
		{KnowledgeID: "sync#fetch-upstream", Path: "/k/sync.yaml", Title: "fetch-upstream", Description: "Sync upstream", Tags: []string{"sync"}, Domain: "git", TrustLevel: "builtin"},
		{KnowledgeID: "branch#create", Path: "/k/branch.yaml", Title: "create", Description: "Branch ops", Domain: "git"},
	}

	if err := mgr.WriteIndex(entries); err != nil {
		t.Fatal(err)
	}
	got, err := mgr.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0].KnowledgeID != "sync#fetch-upstream" {
		t.Errorf("got[0].KnowledgeID = %q", got[0].KnowledgeID)
	}
}
