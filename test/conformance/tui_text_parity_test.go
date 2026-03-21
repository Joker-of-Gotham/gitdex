package conformance

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/panes"
	"github.com/your-org/gitdex/internal/tui/presenter"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestTUIAndTextParity_StateLabels(t *testing.T) {
	states := []repo.StateLabel{repo.Healthy, repo.Drifting, repo.Blocked, repo.Degraded, repo.Unknown}

	for _, state := range states {
		summary := buildTestSummary(state)

		var textBuf bytes.Buffer
		err := presenter.RenderTextSummary(&textBuf, summary)
		if err != nil {
			t.Fatalf("state %s: text render error: %v", state, err)
		}
		textOutput := textBuf.String()

		token := theme.TokenForState(string(state))
		if !strings.Contains(textOutput, token.Label) {
			t.Errorf("state %s: text output missing label %q", state, token.Label)
		}
		if !strings.Contains(textOutput, token.Icon) {
			t.Errorf("state %s: text output missing icon %q", state, token.Icon)
		}
	}
}

func TestTUIAndTextParity_DimensionPresence(t *testing.T) {
	summary := buildTestSummary(repo.Healthy)

	var textBuf bytes.Buffer
	err := presenter.RenderTextSummary(&textBuf, summary)
	if err != nil {
		t.Fatalf("text render error: %v", err)
	}
	textOutput := textBuf.String()

	dimensions := []string{"Local", "Remote", "Collaboration", "Workflows", "Deployments"}
	for _, dim := range dimensions {
		if !strings.Contains(textOutput, dim) {
			t.Errorf("text output missing dimension %q", dim)
		}
	}

	th := theme.NewTheme(true)
	s := theme.NewStyles(th)
	tableOutput := panes.RenderDimensionTable(summary, s)
	for _, dim := range dimensions {
		if !strings.Contains(tableOutput, dim) {
			t.Errorf("TUI dimension table missing %q", dim)
		}
	}
}

func TestTUIAndTextParity_RiskPresence(t *testing.T) {
	summary := buildTestSummary(repo.Degraded)
	summary.Risks = []repo.Risk{
		{Severity: repo.RiskHigh, Description: "test risk", Action: "fix it"},
	}
	summary.NextActions = []repo.NextAction{
		{Action: "take action", Reason: "because risk"},
	}

	var textBuf bytes.Buffer
	err := presenter.RenderTextSummary(&textBuf, summary)
	if err != nil {
		t.Fatalf("text render error: %v", err)
	}
	textOutput := textBuf.String()

	if !strings.Contains(textOutput, "test risk") {
		t.Error("text output missing risk description")
	}
	if !strings.Contains(textOutput, "take action") {
		t.Error("text output missing next action")
	}
}

func TestTUIAndTextParity_GracefulDegradation(t *testing.T) {
	summary := buildTestSummary(repo.Unknown)
	summary.Remote.Detail = "GitHub identity not configured"
	summary.Collaboration.Detail = "requires GitHub identity"

	var textBuf bytes.Buffer
	err := presenter.RenderTextSummary(&textBuf, summary)
	if err != nil {
		t.Fatalf("text render error: %v", err)
	}
	textOutput := textBuf.String()

	if !strings.Contains(textOutput, "[unknown]") {
		t.Error("text output should show unknown label for unconfigured state")
	}
}

func buildTestSummary(label repo.StateLabel) *repo.RepoSummary {
	return &repo.RepoSummary{
		Owner:         "test-owner",
		Repo:          "test-repo",
		OverallLabel:  label,
		Timestamp:     time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
		Local:         repo.LocalState{Label: label, Branch: "main", Detail: "test detail"},
		Remote:        repo.RemoteState{Label: label, Detail: "test detail"},
		Collaboration: repo.CollaborationSignals{Label: label, Detail: "test detail"},
		Workflows:     repo.WorkflowState{Label: label, Detail: "test detail"},
		Deployments:   repo.DeploymentState{Label: label, Detail: "test detail"},
	}
}
