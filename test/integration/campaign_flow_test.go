package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/campaign"
	"github.com/your-org/gitdex/internal/cli/command"
)

func TestCampaignMatrixWithValidID(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID: "camp_mat_test",
		Name:       "Matrix Flow",
		Status:     campaign.StatusDraft,
		TargetRepos: []campaign.RepoTarget{
			{Owner: "a", Repo: "r1", InclusionStatus: campaign.InclusionPending},
			{Owner: "b", Repo: "r2", InclusionStatus: campaign.InclusionIncluded},
		},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"campaign", "matrix", "camp_mat_test"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign matrix failed: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "Campaign Matrix") {
		t.Errorf("expected matrix header, got: %s", output)
	}
	if !strings.Contains(output, "a/r1") {
		t.Errorf("expected repo a/r1 in matrix, got: %s", output)
	}
	if !strings.Contains(output, "b/r2") {
		t.Errorf("expected repo b/r2 in matrix, got: %s", output)
	}
}

func TestCampaignMatrixJSON(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID:  "camp_mat_json",
		Name:        "JSON Matrix",
		Status:      campaign.StatusDraft,
		TargetRepos: []campaign.RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: campaign.InclusionPending}},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"campaign", "matrix", "camp_mat_json", "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign matrix --output json failed: %v", err)
	}
	output := out.String()
	for _, field := range []string{`"campaign_id"`, `"entries"`, `"summary"`} {
		if !strings.Contains(output, field) {
			t.Errorf("JSON output missing %s, got: %s", field, output)
		}
	}
}

func TestCampaignStatusWithValidID(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID:  "camp_stat_test",
		Name:        "Status Flow",
		Status:      campaign.StatusExecuting,
		TargetRepos: []campaign.RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: campaign.InclusionPending}},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"campaign", "status", "camp_stat_test"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign status failed: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "Status Flow") {
		t.Errorf("expected campaign name in status, got: %s", output)
	}
	if !strings.Contains(output, "pending=1") {
		t.Errorf("expected pending=1 in status summary, got: %s", output)
	}
}

func TestCampaignStatusJSON(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID:  "camp_stat_json",
		Name:        "JSON Status",
		Status:      campaign.StatusDraft,
		TargetRepos: []campaign.RepoTarget{},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"campaign", "status", "camp_stat_json", "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign status --output json failed: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, `"total"`) {
		t.Errorf("JSON status output missing 'total' field, got: %s", output)
	}
}

func TestCampaignCreateAddRepoShowFlow(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"campaign", "create", "--name", "E2E Flow", "--description", "test"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign create failed: %v", err)
	}

	list, _ := store.ListCampaigns()
	if len(list) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(list))
	}
	campID := list[0].CampaignID

	out.Reset()
	root.SetArgs([]string{"campaign", "add-repo", campID, "--repo", "org/service"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign add-repo failed: %v", err)
	}

	out.Reset()
	root.SetArgs([]string{"campaign", "show", campID})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign show failed: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "org/service") {
		t.Errorf("show output should include added repo, got: %s", output)
	}
}

func TestCampaignExcludeRuns(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID:  "camp_excl",
		Name:        "Exclude Test",
		Status:      campaign.StatusDraft,
		TargetRepos: []campaign.RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: campaign.InclusionPending}},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"campaign", "exclude", "camp_excl", "--repo", "o/r", "--reason", "deprecated"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign exclude failed: %v", err)
	}
	if !strings.Contains(out.String(), "OK") {
		t.Errorf("expected OK status, got: %s", out.String())
	}

	updated, _ := store.GetCampaign("camp_excl")
	if updated.TargetRepos[0].InclusionStatus != campaign.InclusionExcluded {
		t.Errorf("expected excluded status, got %s", updated.TargetRepos[0].InclusionStatus)
	}
}

func TestCampaignRetryRuns(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID:  "camp_retry",
		Name:        "Retry Test",
		Status:      campaign.StatusExecuting,
		TargetRepos: []campaign.RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: campaign.InclusionExcluded}},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"campaign", "retry", "camp_retry", "--repo", "o/r"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign retry failed: %v", err)
	}
	if !strings.Contains(out.String(), "OK") {
		t.Errorf("expected OK status, got: %s", out.String())
	}
}

func TestCampaignIntervenePauseResume(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID:  "camp_pr",
		Name:        "Pause Resume",
		Status:      campaign.StatusExecuting,
		TargetRepos: []campaign.RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: campaign.InclusionIncluded}},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)

	root.SetArgs([]string{"campaign", "intervene", "camp_pr", "--repo", "o/r", "--action", "pause"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign intervene pause failed: %v", err)
	}
	if !strings.Contains(out.String(), "OK") {
		t.Errorf("expected OK for pause, got: %s", out.String())
	}

	updated, _ := store.GetCampaign("camp_pr")
	if updated.Status != campaign.StatusPaused {
		t.Errorf("expected paused status, got %s", updated.Status)
	}

	out.Reset()
	root.SetArgs([]string{"campaign", "intervene", "camp_pr", "--repo", "o/r", "--action", "resume"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign intervene resume failed: %v", err)
	}

	updated, _ = store.GetCampaign("camp_pr")
	if updated.Status != campaign.StatusExecuting {
		t.Errorf("expected executing status after resume, got %s", updated.Status)
	}
}

func TestCampaignInterveneInvalidAction(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID:  "camp_ia",
		Name:        "Invalid Action",
		Status:      campaign.StatusDraft,
		TargetRepos: []campaign.RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: campaign.InclusionPending}},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	root.SetArgs([]string{"campaign", "intervene", "camp_ia", "--repo", "o/r", "--action", "invalid"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestCampaignRemoveRepoFlow(t *testing.T) {
	store := campaign.NewMemoryCampaignStore()
	c := &campaign.Campaign{
		CampaignID: "camp_rm",
		Name:       "Remove Repo",
		Status:     campaign.StatusDraft,
		TargetRepos: []campaign.RepoTarget{
			{Owner: "a", Repo: "b", InclusionStatus: campaign.InclusionPending},
			{Owner: "c", Repo: "d", InclusionStatus: campaign.InclusionPending},
		},
	}
	_ = store.SaveCampaign(c)
	restore := command.SetCampaignStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"campaign", "remove-repo", "camp_rm", "--repo", "a/b"})
	if err := root.Execute(); err != nil {
		t.Fatalf("campaign remove-repo failed: %v", err)
	}

	updated, _ := store.GetCampaign("camp_rm")
	if len(updated.TargetRepos) != 1 {
		t.Errorf("expected 1 repo after removal, got %d", len(updated.TargetRepos))
	}
}
