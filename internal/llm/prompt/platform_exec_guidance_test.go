package prompt

import (
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func TestBuildAnalyzeRichIncludesPlatformExecGuidance(t *testing.T) {
	builder := NewBuilderWithBudget(4096)

	_, user := builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{
			LocalBranch: git.BranchInfo{Name: "main", Upstream: "origin/main"},
			Remotes:     []string{"origin"},
		},
		Mode: "full",
		Session: &SessionContext{
			ActiveGoal: "发布 release 并补上 release assets 与 release notes",
		},
		PlatformState: &PlatformState{
			Detected: "github",
			Playbooks: []CapabilityPlaybook{
				{ID: "release", Label: "Release management"},
			},
		},
	})

	required := []string{
		"Platform executor schema hints",
		"[release]",
		"Platform API boundaries",
		"mode=partial_mutate",
		"mutate ops: create | update | delete | publish_draft | generate_notes | asset_upload | asset_delete",
		"tag_name",
	}
	for _, fragment := range required {
		if !strings.Contains(user, fragment) {
			t.Fatalf("expected prompt to contain %q", fragment)
		}
	}
}
