package conformance

import (
	"encoding/json"
	"testing"

	"github.com/your-org/gitdex/internal/gitops"
)

func TestWorktreeConfig_RoundTrip(t *testing.T) {
	original := &gitops.WorktreeConfig{
		RepoPath:    "/repo",
		Branch:      "main",
		WorktreeDir: "/worktree-main",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded gitops.WorktreeConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.RepoPath != original.RepoPath {
		t.Errorf("RepoPath: got %q, want %q", decoded.RepoPath, original.RepoPath)
	}
	if decoded.Branch != original.Branch {
		t.Errorf("Branch: got %q, want %q", decoded.Branch, original.Branch)
	}
	if decoded.WorktreeDir != original.WorktreeDir {
		t.Errorf("WorktreeDir: got %q, want %q", decoded.WorktreeDir, original.WorktreeDir)
	}
}

func TestWorktree_RoundTrip(t *testing.T) {
	original := &gitops.Worktree{
		Config: gitops.WorktreeConfig{
			RepoPath:    "/test/repo",
			Branch:      "feature",
			WorktreeDir: "/test/worktree-feature",
		},
		Status:      gitops.WorktreeStatusDirty,
		DiffSummary: "diff summary text",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded gitops.Worktree
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Status != original.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, original.Status)
	}
	if decoded.DiffSummary != original.DiffSummary {
		t.Errorf("DiffSummary: got %q, want %q", decoded.DiffSummary, original.DiffSummary)
	}
	if decoded.Config.RepoPath != original.Config.RepoPath {
		t.Errorf("Config.RepoPath: got %q, want %q", decoded.Config.RepoPath, original.Config.RepoPath)
	}
}

func TestWorktreeStatus_JSONValues(t *testing.T) {
	statuses := []gitops.WorktreeStatus{
		gitops.WorktreeStatusActive,
		gitops.WorktreeStatusDirty,
		gitops.WorktreeStatusClean,
		gitops.WorktreeStatusRemoved,
	}

	for _, s := range statuses {
		data, err := json.Marshal(s)
		if err != nil {
			t.Errorf("marshal %q: %v", s, err)
			continue
		}
		var decoded gitops.WorktreeStatus
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("unmarshal %q: %v", string(data), err)
			continue
		}
		if decoded != s {
			t.Errorf("roundtrip: got %q, want %q", decoded, s)
		}
	}
}
