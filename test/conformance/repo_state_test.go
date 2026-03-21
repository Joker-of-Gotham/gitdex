package conformance_test

import (
	"testing"

	"github.com/your-org/gitdex/internal/state/repo"
)

func TestAC3_AllStateLabelsExist(t *testing.T) {
	required := []repo.StateLabel{
		repo.Healthy,
		repo.Drifting,
		repo.Blocked,
		repo.Degraded,
		repo.Unknown,
	}

	for _, label := range required {
		if string(label) == "" {
			t.Errorf("state label has empty string value")
		}
	}

	expectedValues := map[repo.StateLabel]string{
		repo.Healthy:  "healthy",
		repo.Drifting: "drifting",
		repo.Blocked:  "blocked",
		repo.Degraded: "degraded",
		repo.Unknown:  "unknown",
	}
	for label, expected := range expectedValues {
		if string(label) != expected {
			t.Errorf("label %v = %q, want %q", label, string(label), expected)
		}
	}
}

func TestAC3_StateSeverityOrdering(t *testing.T) {
	if !repo.Blocked.WorseThan(repo.Degraded) {
		t.Error("blocked should be worse than degraded")
	}
	if !repo.Degraded.WorseThan(repo.Drifting) {
		t.Error("degraded should be worse than drifting")
	}
	if !repo.Drifting.WorseThan(repo.Unknown) {
		t.Error("drifting should be worse than unknown")
	}
	if !repo.Unknown.WorseThan(repo.Healthy) {
		t.Error("unknown should be worse than healthy")
	}
}

func TestAC1_RepoSummaryDimensions(t *testing.T) {
	s := repo.RepoSummary{
		Owner:         "owner",
		Repo:          "repo",
		Local:         repo.LocalState{Label: repo.Healthy},
		Remote:        repo.RemoteState{Label: repo.Healthy},
		Collaboration: repo.CollaborationSignals{Label: repo.Healthy},
		Workflows:     repo.WorkflowState{Label: repo.Healthy},
		Deployments:   repo.DeploymentState{Label: repo.Healthy},
	}

	if s.Local.Label == "" {
		t.Error("local state label is empty")
	}
	if s.Remote.Label == "" {
		t.Error("remote state label is empty")
	}
	if s.Collaboration.Label == "" {
		t.Error("collaboration label is empty")
	}
	if s.Workflows.Label == "" {
		t.Error("workflows label is empty")
	}
	if s.Deployments.Label == "" {
		t.Error("deployments label is empty")
	}
}

func TestAC2_RiskAndNextActionStructure(t *testing.T) {
	r := repo.Risk{
		Severity:    repo.RiskHigh,
		Description: "test",
		Evidence:    "evidence",
		Action:      "fix it",
	}
	if r.Severity == "" || r.Description == "" || r.Evidence == "" || r.Action == "" {
		t.Error("risk fields should be populated")
	}

	a := repo.NextAction{
		Action:    "do something",
		Reason:    "because",
		RiskLevel: "low",
	}
	if a.Action == "" || a.Reason == "" || a.RiskLevel == "" {
		t.Error("next action fields should be populated")
	}
}

func TestAC5_GracefulDegradation_UnknownRemote(t *testing.T) {
	s := repo.RepoSummary{
		Local:         repo.LocalState{Label: repo.Healthy},
		Remote:        repo.RemoteState{Label: repo.Unknown, Detail: "not configured"},
		Collaboration: repo.CollaborationSignals{Label: repo.Unknown},
		Workflows:     repo.WorkflowState{Label: repo.Unknown},
		Deployments:   repo.DeploymentState{Label: repo.Unknown},
	}
	overall := repo.WorstLabel(
		s.Local.Label, s.Remote.Label, s.Collaboration.Label,
		s.Workflows.Label, s.Deployments.Label,
	)

	if overall != repo.Unknown {
		t.Errorf("overall = %q, want %q when remote is unknown", overall, repo.Unknown)
	}
	if s.Remote.Detail == "" {
		t.Error("remote should have detail explaining why it's unknown")
	}
}
