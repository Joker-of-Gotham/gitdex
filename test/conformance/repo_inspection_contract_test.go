package conformance

import (
	"encoding/json"
	"testing"

	"github.com/your-org/gitdex/internal/gitops"
)

func TestRepoInspection_RoundTrip(t *testing.T) {
	original := &gitops.RepoInspection{
		RepoPath:       "/test",
		LocalBranch:    "main",
		RemoteBranch:   "origin/main",
		Ahead:          1,
		Behind:         2,
		HasUncommitted: true,
		HasUntracked:   false,
		Divergence:     gitops.DivDiverged,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded gitops.RepoInspection
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Divergence != original.Divergence {
		t.Errorf("Divergence: got %q, want %q", decoded.Divergence, original.Divergence)
	}
	if decoded.Ahead != original.Ahead {
		t.Errorf("Ahead: got %d, want %d", decoded.Ahead, original.Ahead)
	}
}

func TestSyncPreview_RoundTrip(t *testing.T) {
	original := &gitops.SyncPreview{
		AffectedFiles: 5,
		MergeStrategy: "fast-forward",
		ConflictRisk:  "none",
		Description:   "test preview",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded gitops.SyncPreview
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.AffectedFiles != original.AffectedFiles {
		t.Errorf("AffectedFiles: got %d, want %d", decoded.AffectedFiles, original.AffectedFiles)
	}
}

func TestSyncResult_RoundTrip(t *testing.T) {
	original := &gitops.SyncResult{
		Success:      false,
		Conflicts:    1,
		ErrorMessage: "diverged",
		Description:  "blocked",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded gitops.SyncResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Success != original.Success {
		t.Errorf("Success: got %v, want %v", decoded.Success, original.Success)
	}
	if decoded.Conflicts != original.Conflicts {
		t.Errorf("Conflicts: got %d, want %d", decoded.Conflicts, original.Conflicts)
	}
}
