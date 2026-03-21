package repo

import (
	"encoding/json"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func TestStateLabel_WorseThan(t *testing.T) {
	tests := []struct {
		a, b StateLabel
		want bool
	}{
		{Blocked, Healthy, true},
		{Degraded, Drifting, true},
		{Drifting, Unknown, true},
		{Unknown, Healthy, true},
		{Healthy, Blocked, false},
		{Healthy, Healthy, false},
		{Drifting, Drifting, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.a)+"_vs_"+string(tt.b), func(t *testing.T) {
			if got := tt.a.WorseThan(tt.b); got != tt.want {
				t.Errorf("%s.WorseThan(%s) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestWorstLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels []StateLabel
		want   StateLabel
	}{
		{"all_healthy", []StateLabel{Healthy, Healthy}, Healthy},
		{"one_blocked", []StateLabel{Healthy, Drifting, Blocked}, Blocked},
		{"degraded_wins", []StateLabel{Unknown, Degraded, Drifting}, Degraded},
		{"unknown_only", []StateLabel{Unknown}, Unknown},
		{"empty", nil, Healthy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorstLabel(tt.labels...)
			if got != tt.want {
				t.Errorf("WorstLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRepoSummary_JSONRoundTrip(t *testing.T) {
	s := RepoSummary{
		Owner:        "test-org",
		Repo:         "test-repo",
		OverallLabel: Drifting,
		Timestamp:    time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
		Local: LocalState{
			Label:   Healthy,
			Branch:  "main",
			IsClean: true,
		},
		Remote: RemoteState{
			Label:    Unknown,
			FullName: "test-org/test-repo",
		},
		Collaboration: CollaborationSignals{Label: Unknown},
		Workflows:     WorkflowState{Label: Unknown},
		Deployments:   DeploymentState{Label: Unknown},
		Risks: []Risk{
			{Severity: RiskMedium, Description: "test risk", Evidence: "e1", Action: "a1"},
		},
		NextActions: []NextAction{
			{Action: "sync upstream", Reason: "behind", RiskLevel: "low"},
		},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}

	var decoded RepoSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if decoded.Owner != s.Owner {
		t.Errorf("owner = %q, want %q", decoded.Owner, s.Owner)
	}
	if decoded.OverallLabel != s.OverallLabel {
		t.Errorf("overall_label = %q, want %q", decoded.OverallLabel, s.OverallLabel)
	}
	if len(decoded.Risks) != 1 {
		t.Fatalf("risks count = %d, want 1", len(decoded.Risks))
	}
	if decoded.Risks[0].Severity != RiskMedium {
		t.Errorf("risk severity = %q, want %q", decoded.Risks[0].Severity, RiskMedium)
	}
}

func TestRepoSummary_YAMLRoundTrip(t *testing.T) {
	s := RepoSummary{
		Owner:        "org",
		Repo:         "repo",
		OverallLabel: Blocked,
		Timestamp:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Local:        LocalState{Label: Healthy, Branch: "dev"},
		Remote:       RemoteState{Label: Blocked},
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		t.Fatalf("yaml marshal: %v", err)
	}

	var decoded RepoSummary
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if decoded.OverallLabel != Blocked {
		t.Errorf("overall_label = %q, want %q", decoded.OverallLabel, Blocked)
	}
}

func TestAllStateLabels_StringValues(t *testing.T) {
	labels := []StateLabel{Healthy, Drifting, Blocked, Degraded, Unknown}
	expected := []string{"healthy", "drifting", "blocked", "degraded", "unknown"}

	for i, l := range labels {
		if string(l) != expected[i] {
			t.Errorf("label %d = %q, want %q", i, l, expected[i])
		}
	}
}
