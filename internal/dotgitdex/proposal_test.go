package dotgitdex

import "testing"

func TestAppendAndReadProposals(t *testing.T) {
	tmp := t.TempDir()
	mgr := New(tmp)
	_ = mgr.Init()

	_ = mgr.AppendCreativeProposal([]string{"Add CI pipeline", "Write docs"})
	_ = mgr.AppendCreativeProposal([]string{"Add tests"})

	items, err := mgr.ReadCreativeProposals()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	if items[2] != "Add tests" {
		t.Errorf("items[2] = %q", items[2])
	}
}
