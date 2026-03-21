package campaign

import (
	"context"
	"testing"
)

func TestDefaultMatrixEngine_BuildNilCampaign(t *testing.T) {
	engine := NewDefaultMatrixEngine()
	_, err := engine.Build(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when building matrix with nil campaign")
	}
}

func TestDefaultMatrixEngine_BuildSingleRepoStatuses(t *testing.T) {
	engine := NewDefaultMatrixEngine()

	tests := []struct {
		name            string
		inclusionStatus InclusionStatus
		wantRepoStatus  RepoStatus
	}{
		{"pending", InclusionPending, RepoStatusAwaitingApproval},
		{"included", InclusionIncluded, RepoStatusSucceeded},
		{"excluded", InclusionExcluded, RepoStatusExcluded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Campaign{
				CampaignID:  "camp_" + tt.name,
				TargetRepos: []RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: tt.inclusionStatus}},
			}
			mat, err := engine.Build(context.Background(), c)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if len(mat.Entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(mat.Entries))
			}
			if mat.Entries[0].Status != tt.wantRepoStatus {
				t.Errorf("expected status %s, got %s", tt.wantRepoStatus, mat.Entries[0].Status)
			}
		})
	}
}

func TestDefaultMatrixEngine_BuildSummaryAccuracy(t *testing.T) {
	engine := NewDefaultMatrixEngine()
	c := &Campaign{
		CampaignID: "camp_sum",
		TargetRepos: []RepoTarget{
			{Owner: "a", Repo: "r1", InclusionStatus: InclusionPending},
			{Owner: "b", Repo: "r2", InclusionStatus: InclusionPending},
			{Owner: "c", Repo: "r3", InclusionStatus: InclusionExcluded},
			{Owner: "d", Repo: "r4", InclusionStatus: InclusionIncluded},
			{Owner: "e", Repo: "r5", InclusionStatus: InclusionIncluded},
		},
	}
	mat, err := engine.Build(context.Background(), c)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	s := mat.Summary
	if s.Total != 5 {
		t.Errorf("Total = %d, want 5", s.Total)
	}
	if s.Pending != 2 {
		t.Errorf("Pending = %d, want 2", s.Pending)
	}
	if s.Excluded != 1 {
		t.Errorf("Excluded = %d, want 1", s.Excluded)
	}
	if s.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", s.Succeeded)
	}
	if s.Failed != 0 {
		t.Errorf("Failed = %d, want 0", s.Failed)
	}
}
