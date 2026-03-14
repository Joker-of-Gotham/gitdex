package prompt

import (
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func TestBuildAnalyzeRichIncludesWorkflowOrchestration(t *testing.T) {
	builder := NewBuilderWithBudget(4096)

	_, user := builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{
			LocalBranch: git.BranchInfo{Name: "main", Upstream: "origin/main"},
			Remotes:     []string{"origin"},
		},
		Mode: "full",
		Session: &SessionContext{
			ActiveGoal: "Configure Pages and deployment",
		},
		Workflow: &WorkflowOrchestration{
			WorkflowID:    "pages_setup",
			WorkflowLabel: "Pages / Static site",
			Capabilities:  []string{"pages", "environments"},
			Steps: []WorkflowOrchestrationStep{
				{
					Title:      "Pages: inspect latest build",
					Capability: "pages",
					Flow:       "inspect",
					Query:      map[string]string{"view": "latest_build"},
				},
				{
					Title:      "Environments: inspect",
					Capability: "environments",
					Flow:       "inspect",
				},
			},
		},
	})

	for _, fragment := range []string{
		"Available platform operations",
		"workflow: pages_setup",
		"Pages: inspect latest build",
	} {
		if !strings.Contains(user, fragment) {
			t.Fatalf("expected prompt to contain %q", fragment)
		}
	}
}
