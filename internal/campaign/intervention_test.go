package campaign

import (
	"context"
	"testing"
)

func TestDefaultInterventionEngine_ApproveRepo(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{
		CampaignID: "camp_int",
		Name:       "int test",
		Status:     StatusDraft,
		TargetRepos: []RepoTarget{
			{Owner: "o", Repo: "r", InclusionStatus: InclusionPending},
		},
	}
	_ = store.SaveCampaign(c)
	engine := NewDefaultInterventionEngine(store)
	req := InterventionRequest{
		InterventionType: InterventionApproveRepo,
		CampaignID:       "camp_int",
		Owner:            "o",
		Repo:             "r",
		Actor:            "cli",
	}
	res, err := engine.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got %s", res.Message)
	}
	updated, _ := store.GetCampaign("camp_int")
	if updated.TargetRepos[0].InclusionStatus != InclusionIncluded {
		t.Errorf("expected included, got %s", updated.TargetRepos[0].InclusionStatus)
	}
}

func TestDefaultInterventionEngine_ExcludeRepo(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{
		CampaignID: "camp_ex",
		Name:       "ex test",
		Status:     StatusDraft,
		TargetRepos: []RepoTarget{
			{Owner: "o", Repo: "r", InclusionStatus: InclusionPending},
		},
	}
	_ = store.SaveCampaign(c)
	engine := NewDefaultInterventionEngine(store)
	req := InterventionRequest{
		InterventionType: InterventionExcludeRepo,
		CampaignID:       "camp_ex",
		Owner:            "o",
		Repo:             "r",
		Reason:           "test",
		Actor:            "cli",
	}
	res, err := engine.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got %s", res.Message)
	}
	updated, _ := store.GetCampaign("camp_ex")
	if updated.TargetRepos[0].InclusionStatus != InclusionExcluded {
		t.Errorf("expected excluded, got %s", updated.TargetRepos[0].InclusionStatus)
	}
}

func TestDefaultInterventionEngine_RepoNotFound(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{CampaignID: "camp_nf", Name: "x", Status: StatusDraft, TargetRepos: []RepoTarget{}}
	_ = store.SaveCampaign(c)
	engine := NewDefaultInterventionEngine(store)
	req := InterventionRequest{
		InterventionType: InterventionApproveRepo,
		CampaignID:       "camp_nf",
		Owner:            "none",
		Repo:             "found",
		Actor:            "cli",
	}
	res, err := engine.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Success {
		t.Error("expected failure for unknown repo")
	}
}
