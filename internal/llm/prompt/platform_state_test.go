package prompt

import (
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func TestBuildAnalyzeRich_IncludesPlatformPlaybooks(t *testing.T) {
	builder := NewBuilderWithBudget(4096)

	_, user := builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{
			LocalBranch: git.BranchInfo{Name: "main", Upstream: "origin/main"},
			Remotes:     []string{"origin"},
		},
		Mode: "full",
		Session: &SessionContext{
			ActiveGoal: "configure deployment and pages",
		},
		PlatformState: &PlatformState{
			Detected: "github",
			Playbooks: []CapabilityPlaybook{
				{
					ID:      "deployment",
					Label:   "Deployments",
					Inspect: []string{"Inspect deployment environments."},
					Apply:   []string{"Apply rollout automation."},
					Verify:  []string{"Verify deployment health."},
				},
			},
		},
	})

	if !strings.Contains(user, "deployment") {
		t.Fatal("expected platform playbook content in user prompt")
	}
	if !strings.Contains(user, "Apply rollout automation") {
		t.Fatal("expected playbook steps in user prompt")
	}
}
