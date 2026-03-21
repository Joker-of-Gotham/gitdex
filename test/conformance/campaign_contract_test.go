package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/campaign"
)

func TestCampaign_JSONContract(t *testing.T) {
	c := &campaign.Campaign{
		CampaignID:     "camp_abc123",
		Name:           "Test",
		Description:    "Desc",
		Status:         campaign.StatusDraft,
		TargetRepos:    []campaign.RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: campaign.InclusionPending}},
		PlanTemplate:   "tpl",
		PolicyBundleID: "bundle_1",
		CreatedBy:      "user",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{
		`"campaign_id"`, `"name"`, `"description"`, `"status"`,
		`"target_repos"`, `"plan_template"`, `"policy_bundle_id"`,
		`"created_by"`, `"created_at"`, `"updated_at"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestRepoTarget_JSONContract(t *testing.T) {
	rt := campaign.RepoTarget{
		Owner:            "owner",
		Repo:             "repo",
		InclusionStatus:  campaign.InclusionIncluded,
		PerRepoOverrides: map[string]string{"k": "v"},
	}
	data, err := json.Marshal(rt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{`"owner"`, `"repo"`, `"inclusion_status"`, `"per_repo_overrides"`}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s", f)
		}
	}
}

func TestCampaignStatus_AllValues(t *testing.T) {
	statuses := []campaign.CampaignStatus{
		campaign.StatusDraft, campaign.StatusPlanning, campaign.StatusExecuting,
		campaign.StatusPaused, campaign.StatusCompleted, campaign.StatusCancelled,
	}
	seen := make(map[campaign.CampaignStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Error("status should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate status: %s", s)
		}
		seen[s] = true
	}
}
