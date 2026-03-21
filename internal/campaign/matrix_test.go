package campaign

import (
	"context"
	"testing"
)

func TestDefaultMatrixEngine_Build(t *testing.T) {
	engine := NewDefaultMatrixEngine()
	c := &Campaign{
		CampaignID: "camp_test",
		Name:       "test",
		Status:     StatusDraft,
		TargetRepos: []RepoTarget{
			{Owner: "a", Repo: "r1", InclusionStatus: InclusionPending},
			{Owner: "b", Repo: "r2", InclusionStatus: InclusionExcluded},
			{Owner: "c", Repo: "r3", InclusionStatus: InclusionIncluded},
		},
	}
	mat, err := engine.Build(context.Background(), c)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if mat.CampaignID != "camp_test" {
		t.Errorf("CampaignID = %q, want camp_test", mat.CampaignID)
	}
	if mat.Summary.Total != 3 {
		t.Errorf("Summary.Total = %d, want 3", mat.Summary.Total)
	}
	if mat.Summary.Excluded != 1 {
		t.Errorf("Summary.Excluded = %d, want 1", mat.Summary.Excluded)
	}
	if mat.Summary.Succeeded != 1 {
		t.Errorf("Summary.Succeeded = %d, want 1", mat.Summary.Succeeded)
	}
	if mat.Summary.Pending != 1 {
		t.Errorf("Summary.Pending = %d, want 1", mat.Summary.Pending)
	}
	if len(mat.Entries) != 3 {
		t.Errorf("Entries len = %d, want 3", len(mat.Entries))
	}
}

func TestDefaultMatrixEngine_BuildEmpty(t *testing.T) {
	engine := NewDefaultMatrixEngine()
	c := &Campaign{CampaignID: "camp_empty", TargetRepos: []RepoTarget{}}
	mat, err := engine.Build(context.Background(), c)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if mat.Summary.Total != 0 || len(mat.Entries) != 0 {
		t.Errorf("expected empty matrix, got total=%d entries=%d", mat.Summary.Total, len(mat.Entries))
	}
}
