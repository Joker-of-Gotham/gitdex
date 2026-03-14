package platform

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoliciesForDependabotIncludesSchemaChecks(t *testing.T) {
	policies := PoliciesFor(PlatformGitHub, "dependabot_config", "mutate", "update")
	assert.Contains(t, policies.Validation.ExternalChecks, "schema")
	assert.Equal(t, RollbackReversible, policies.Rollback.Kind)
}

func TestDiagnosePlatformOperationAutoRepairsTokens(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Name: "feature/deps"},
		RepoConfig:  git.RepoConfig{DefaultBranch: "main"},
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}
	set, repaired := DiagnosePlatformOperation(PlatformGitHub, state, &git.PlatformExecInfo{
		CapabilityID: "pages",
		Flow:         "mutate",
		Operation:    "update",
		Payload:      []byte(`{"source":{"branch":"<default_branch>"}}`),
	})
	require.NotNil(t, repaired)
	assert.Equal(t, DiagnosticAutoRepair, set.Decision)
	assert.JSONEq(t, `{"source":{"branch":"main"}}`, string(repaired.Payload))
}

func TestDiagnosePlatformOperationBlocksInspectOnlyMutation(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:owner/repo.git",
		}},
	}
	set, _ := DiagnosePlatformOperation(PlatformGitHub, state, &git.PlatformExecInfo{
		CapabilityID: "code_scanning_tool_settings",
		Flow:         "mutate",
		Operation:    "update",
	})
	assert.Equal(t, DiagnosticBlocked, set.Decision)
	require.NotEmpty(t, set.Items)
	assert.Equal(t, "boundary_inspect_only", set.Items[0].Code)
}
