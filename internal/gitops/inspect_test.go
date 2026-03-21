package gitops

import (
	"encoding/json"
	"testing"
)

func TestDivergenceState_Values(t *testing.T) {
	states := []DivergenceState{DivSynced, DivAhead, DivBehind, DivDiverged, DivDetached, DivNoUpstream}
	seen := make(map[DivergenceState]bool)
	for _, s := range states {
		if s == "" {
			t.Error("divergence state should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate state: %s", s)
		}
		seen[s] = true
	}
}

func TestRepoInspection_JSONContract(t *testing.T) {
	insp := &RepoInspection{
		RepoPath:       "/test/repo",
		LocalBranch:    "main",
		RemoteBranch:   "origin/main",
		Ahead:          2,
		Behind:         3,
		HasUncommitted: true,
		HasUntracked:   false,
		Divergence:     DivDiverged,
	}

	data, err := json.Marshal(insp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"repo_path"`, `"local_branch"`, `"remote_branch"`,
		`"ahead"`, `"behind"`, `"has_uncommitted"`, `"has_untracked"`, `"divergence"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !containsStr(raw, f) {
			t.Errorf("JSON missing field %s", f)
		}
	}
}

func TestSyncRecommendation_JSONContract(t *testing.T) {
	rec := &SyncRecommendation{
		Action:      "fast_forward",
		RiskLevel:   "low",
		Description: "test",
		Previewable: true,
	}
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	for _, f := range []string{`"action"`, `"risk_level"`, `"description"`, `"previewable"`} {
		if !containsStr(raw, f) {
			t.Errorf("JSON missing field %s", f)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
