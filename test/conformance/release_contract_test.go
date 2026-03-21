package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/collaboration"
)

func TestReleaseReadiness_JSONContract(t *testing.T) {
	r := &collaboration.ReleaseReadiness{
		RepoOwner:      "owner",
		RepoName:       "repo",
		Tag:            "v1.0.0",
		Status:         collaboration.ReleaseReady,
		Blockers:       []string{},
		IncludedPRs:    []int{1, 2},
		CheckResults:   []collaboration.CheckResult{{Name: "build", Status: collaboration.CheckPassed, Details: "ok"}},
		ApprovalStatus: "approved",
		Notes:          "test",
		AssessedAt:     time.Now().UTC(),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"repo_owner"`,
		`"repo_name"`,
		`"tag"`,
		`"status"`,
		`"blockers"`,
		`"included_prs"`,
		`"check_results"`,
		`"approval_status"`,
		`"notes"`,
		`"assessed_at"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestReleaseStatus_AllValues(t *testing.T) {
	statuses := []collaboration.ReleaseStatus{
		collaboration.ReleaseReady,
		collaboration.ReleaseBlocked,
		collaboration.ReleasePending,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("release status should not be empty")
		}
	}
}

func TestCheckResult_JSONContract(t *testing.T) {
	c := collaboration.CheckResult{
		Name:    "build",
		Status:  collaboration.CheckPassed,
		Details: "CI passed",
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	for _, f := range []string{`"name"`, `"status"`, `"details"`} {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s", f)
		}
	}
}
