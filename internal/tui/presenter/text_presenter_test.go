package presenter

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/state/repo"
)

func TestRenderTextSummary_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := RenderTextSummary(&buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No repository data") {
		t.Errorf("expected no-data message, got: %s", buf.String())
	}
}

func TestRenderTextSummary_Healthy(t *testing.T) {
	summary := &repo.RepoSummary{
		Owner:        "test-owner",
		Repo:         "test-repo",
		OverallLabel: repo.Healthy,
		Timestamp:    time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
		Local: repo.LocalState{
			Label:  repo.Healthy,
			Branch: "main",
			Detail: "clean working tree",
		},
		Remote:        repo.RemoteState{Label: repo.Healthy},
		Collaboration: repo.CollaborationSignals{Label: repo.Healthy},
		Workflows:     repo.WorkflowState{Label: repo.Healthy},
		Deployments:   repo.DeploymentState{Label: repo.Healthy},
	}

	var buf bytes.Buffer
	err := RenderTextSummary(&buf, summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	required := []string{
		"test-owner/test-repo",
		"[healthy]",
		"Local",
		"Remote",
		"Collaboration",
		"Workflows",
		"Deployments",
		"No material risks",
	}
	for _, r := range required {
		if !strings.Contains(output, r) {
			t.Errorf("expected output to contain %q, output:\n%s", r, output)
		}
	}
}

func TestRenderTextSummary_WithRisks(t *testing.T) {
	summary := &repo.RepoSummary{
		Owner:         "org",
		Repo:          "repo",
		OverallLabel:  repo.Degraded,
		Timestamp:     time.Now(),
		Local:         repo.LocalState{Label: repo.Drifting, Detail: "dirty files"},
		Remote:        repo.RemoteState{Label: repo.Healthy},
		Collaboration: repo.CollaborationSignals{Label: repo.Healthy},
		Workflows:     repo.WorkflowState{Label: repo.Degraded, Detail: "CI failure"},
		Deployments:   repo.DeploymentState{Label: repo.Healthy},
		Risks: []repo.Risk{
			{Severity: repo.RiskHigh, Description: "CI pipeline broken", Action: "fix CI"},
		},
		NextActions: []repo.NextAction{
			{Action: "investigate CI", Reason: "failures detected"},
		},
	}

	var buf bytes.Buffer
	err := RenderTextSummary(&buf, summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CI pipeline broken") {
		t.Error("expected risk in output")
	}
	if !strings.Contains(output, "investigate CI") {
		t.Error("expected next action in output")
	}
}

func TestRenderTextSummary_AllDimensions(t *testing.T) {
	states := []repo.StateLabel{repo.Healthy, repo.Drifting, repo.Blocked, repo.Degraded, repo.Unknown}
	for _, state := range states {
		summary := &repo.RepoSummary{
			Owner:         "o",
			Repo:          "r",
			OverallLabel:  state,
			Timestamp:     time.Now(),
			Local:         repo.LocalState{Label: state},
			Remote:        repo.RemoteState{Label: state},
			Collaboration: repo.CollaborationSignals{Label: state},
			Workflows:     repo.WorkflowState{Label: state},
			Deployments:   repo.DeploymentState{Label: state},
		}

		var buf bytes.Buffer
		err := RenderTextSummary(&buf, summary)
		if err != nil {
			t.Fatalf("state %s: unexpected error: %v", state, err)
		}
		if buf.Len() == 0 {
			t.Errorf("state %s: empty output", state)
		}
	}
}
