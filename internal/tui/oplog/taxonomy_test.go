package oplog

import "testing"

func TestBuildFailureDashboard(t *testing.T) {
	entries := []Entry{
		{Type: EntryCmdFail, Summary: "HTTP 404 not found"},
		{Type: EntryCmdFail, Summary: "HTTP 403 forbidden"},
		{Type: EntryLLMError, Summary: "timeout while calling provider"},
		{Type: EntryCmdSuccess, Summary: "ok"},
	}
	d := BuildFailureDashboard(entries)
	if d.Total != 3 {
		t.Fatalf("expected total=3, got %d", d.Total)
	}
	if d.Buckets["not_found"] != 1 {
		t.Fatalf("expected not_found bucket=1, got %d", d.Buckets["not_found"])
	}
	if d.Buckets["auth_permission"] != 1 {
		t.Fatalf("expected auth_permission bucket=1, got %d", d.Buckets["auth_permission"])
	}
	if d.Buckets["network_transient"] != 1 {
		t.Fatalf("expected network_transient bucket=1, got %d", d.Buckets["network_transient"])
	}
}
