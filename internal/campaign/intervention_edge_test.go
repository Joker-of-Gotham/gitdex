package campaign

import (
	"context"
	"testing"
)

func TestDefaultInterventionEngine_RetryRepo(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{
		CampaignID:  "camp_retry",
		Name:        "retry test",
		Status:      StatusExecuting,
		TargetRepos: []RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: InclusionExcluded}},
	}
	_ = store.SaveCampaign(c)
	engine := NewDefaultInterventionEngine(store)
	res, err := engine.Execute(context.Background(), InterventionRequest{
		InterventionType: InterventionRetryRepo,
		CampaignID:       "camp_retry",
		Owner:            "o",
		Repo:             "r",
		Actor:            "cli",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got %s", res.Message)
	}
	if res.NewStatus != string(InclusionPending) {
		t.Errorf("expected new status pending, got %s", res.NewStatus)
	}
}

func TestDefaultInterventionEngine_OverridePlan(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{
		CampaignID:  "camp_override",
		Name:        "override test",
		Status:      StatusDraft,
		TargetRepos: []RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: InclusionPending}},
	}
	_ = store.SaveCampaign(c)
	engine := NewDefaultInterventionEngine(store)
	res, err := engine.Execute(context.Background(), InterventionRequest{
		InterventionType: InterventionOverridePlan,
		CampaignID:       "camp_override",
		Owner:            "o",
		Repo:             "r",
		Actor:            "cli",
		Overrides:        map[string]string{"branch": "hotfix"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got %s", res.Message)
	}

	updated, _ := store.GetCampaign("camp_override")
	if updated.TargetRepos[0].PerRepoOverrides["branch"] != "hotfix" {
		t.Errorf("override not applied: %v", updated.TargetRepos[0].PerRepoOverrides)
	}
}

func TestDefaultInterventionEngine_PauseRepo(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{
		CampaignID:  "camp_pause",
		Name:        "pause test",
		Status:      StatusExecuting,
		TargetRepos: []RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: InclusionIncluded}},
	}
	_ = store.SaveCampaign(c)
	engine := NewDefaultInterventionEngine(store)
	res, err := engine.Execute(context.Background(), InterventionRequest{
		InterventionType: InterventionPauseRepo,
		CampaignID:       "camp_pause",
		Owner:            "o",
		Repo:             "r",
		Actor:            "cli",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got %s", res.Message)
	}

	updated, _ := store.GetCampaign("camp_pause")
	if updated.Status != StatusPaused {
		t.Errorf("expected campaign status paused, got %s", updated.Status)
	}
}

func TestDefaultInterventionEngine_ResumeRepo(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{
		CampaignID:  "camp_resume",
		Name:        "resume test",
		Status:      StatusPaused,
		TargetRepos: []RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: InclusionIncluded}},
	}
	_ = store.SaveCampaign(c)
	engine := NewDefaultInterventionEngine(store)
	res, err := engine.Execute(context.Background(), InterventionRequest{
		InterventionType: InterventionResumeRepo,
		CampaignID:       "camp_resume",
		Owner:            "o",
		Repo:             "r",
		Actor:            "cli",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got %s", res.Message)
	}

	updated, _ := store.GetCampaign("camp_resume")
	if updated.Status != StatusExecuting {
		t.Errorf("expected campaign status executing, got %s", updated.Status)
	}
}

func TestDefaultInterventionEngine_UnknownType(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{
		CampaignID:  "camp_unknown",
		Name:        "unknown test",
		Status:      StatusDraft,
		TargetRepos: []RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: InclusionPending}},
	}
	_ = store.SaveCampaign(c)
	engine := NewDefaultInterventionEngine(store)
	res, err := engine.Execute(context.Background(), InterventionRequest{
		InterventionType: InterventionType("invalid_action"),
		CampaignID:       "camp_unknown",
		Owner:            "o",
		Repo:             "r",
		Actor:            "cli",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Success {
		t.Error("expected failure for unknown intervention type")
	}
}

func TestDefaultInterventionEngine_EmptyCampaignID(t *testing.T) {
	store := NewMemoryCampaignStore()
	engine := NewDefaultInterventionEngine(store)
	res, err := engine.Execute(context.Background(), InterventionRequest{
		InterventionType: InterventionApproveRepo,
		CampaignID:       "",
		Owner:            "o",
		Repo:             "r",
		Actor:            "cli",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Success {
		t.Error("expected failure for empty campaign_id")
	}
}

func TestDefaultInterventionEngine_CampaignNotFound(t *testing.T) {
	store := NewMemoryCampaignStore()
	engine := NewDefaultInterventionEngine(store)
	res, err := engine.Execute(context.Background(), InterventionRequest{
		InterventionType: InterventionApproveRepo,
		CampaignID:       "nonexistent",
		Owner:            "o",
		Repo:             "r",
		Actor:            "cli",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Success {
		t.Error("expected failure for nonexistent campaign")
	}
}
