package conformance

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/campaign"
)

func TestInterventionRequest_JSONContract(t *testing.T) {
	req := campaign.InterventionRequest{
		InterventionType: campaign.InterventionApproveRepo,
		CampaignID:       "camp_abc",
		Owner:            "owner",
		Repo:             "repo",
		Reason:           "approved for deployment",
		Actor:            "cli",
		Overrides:        map[string]string{"branch": "main"},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{
		`"intervention_type"`, `"campaign_id"`, `"owner"`,
		`"repo"`, `"reason"`, `"actor"`, `"overrides"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}

	var decoded campaign.InterventionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.InterventionType != campaign.InterventionApproveRepo {
		t.Errorf("InterventionType = %q, want approve_repo", decoded.InterventionType)
	}
}

func TestInterventionResult_JSONContract(t *testing.T) {
	res := campaign.InterventionResult{
		Request: campaign.InterventionRequest{
			InterventionType: campaign.InterventionExcludeRepo,
			CampaignID:       "camp_abc",
			Owner:            "o",
			Repo:             "r",
			Actor:            "cli",
		},
		Success:        true,
		PreviousStatus: "pending",
		NewStatus:      "excluded",
		Message:        "excluded o/r",
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{
		`"request"`, `"success"`, `"previous_status"`,
		`"new_status"`, `"message"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}

	var decoded campaign.InterventionResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decoded.Success {
		t.Error("decoded Success should be true")
	}
	if decoded.NewStatus != "excluded" {
		t.Errorf("NewStatus = %q, want excluded", decoded.NewStatus)
	}
}

func TestInterventionType_AllValues(t *testing.T) {
	types := []campaign.InterventionType{
		campaign.InterventionApproveRepo,
		campaign.InterventionExcludeRepo,
		campaign.InterventionRetryRepo,
		campaign.InterventionOverridePlan,
		campaign.InterventionPauseRepo,
		campaign.InterventionResumeRepo,
	}
	seen := make(map[campaign.InterventionType]bool)
	for _, tt := range types {
		if tt == "" {
			t.Error("intervention type should not be empty")
		}
		if seen[tt] {
			t.Errorf("duplicate intervention type: %s", tt)
		}
		seen[tt] = true
	}
	if len(types) != 6 {
		t.Errorf("expected 6 intervention types, got %d", len(types))
	}
}

func TestMatrixEntry_JSONContract(t *testing.T) {
	entry := campaign.MatrixEntry{
		Owner:  "o",
		Repo:   "r",
		Status: campaign.RepoStatusAwaitingApproval,
		PlanID: "plan_abc",
		TaskID: "task_abc",
		Notes:  "pending review",
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{
		`"owner"`, `"repo"`, `"status"`,
		`"plan_id"`, `"task_id"`, `"last_updated"`, `"notes"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestCampaignMatrix_JSONContract(t *testing.T) {
	mat := campaign.CampaignMatrix{
		CampaignID: "camp_abc",
		Entries: []campaign.MatrixEntry{
			{Owner: "o", Repo: "r", Status: campaign.RepoStatusSucceeded},
		},
		Summary: campaign.MatrixSummary{Total: 1, Succeeded: 1},
	}
	data, err := json.Marshal(mat)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{`"campaign_id"`, `"entries"`, `"summary"`}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}

	var decoded campaign.CampaignMatrix
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Summary.Total != 1 {
		t.Errorf("Summary.Total = %d, want 1", decoded.Summary.Total)
	}
}

func TestRepoStatus_AllValues(t *testing.T) {
	statuses := []campaign.RepoStatus{
		campaign.RepoStatusNotStarted, campaign.RepoStatusPlanning,
		campaign.RepoStatusAwaitingApproval, campaign.RepoStatusApproved,
		campaign.RepoStatusExecuting, campaign.RepoStatusSucceeded,
		campaign.RepoStatusFailed, campaign.RepoStatusExcluded,
	}
	seen := make(map[campaign.RepoStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Error("repo status should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate repo status: %s", s)
		}
		seen[s] = true
	}
	if len(statuses) != 8 {
		t.Errorf("expected 8 repo statuses, got %d", len(statuses))
	}
}
