package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoliciesForDependabotConfigExposeStructuredValidationChecks(t *testing.T) {
	policies := PoliciesFor(PlatformGitHub, "dependabot_config", "mutate", "update")

	assert.Equal(t, "subset_match", policies.Validation.Strategy)
	assert.Contains(t, policies.Validation.ExternalChecks, "schema")
	assert.Contains(t, policies.Validation.ExternalChecks, "deterministic_reencode")
	assert.Contains(t, policies.Validation.ExternalChecks, "no_op_diff")
	assert.Equal(t, RollbackReversible, policies.Rollback.Kind)
	assert.False(t, policies.Compensation.Required)
}

func TestPoliciesForInspectOnlyFlowDisableRollback(t *testing.T) {
	policies := PoliciesFor(PlatformGitHub, "code_scanning_tool_settings", "inspect", "")

	assert.False(t, policies.Validation.RequiresTarget)
	assert.Equal(t, RollbackNotSupported, policies.Rollback.Kind)
	assert.False(t, policies.Rollback.RequiresBeforeSnapshot)
	assert.Contains(t, policies.Rollback.Note, "do not mutate")
}

func TestPoliciesForCompensatingSurfaceRequireOperatorCompensation(t *testing.T) {
	policies := PoliciesFor(PlatformGitHub, "release", "mutate", "asset_delete")

	assert.Equal(t, RollbackCompensating, policies.Rollback.Kind)
	assert.True(t, policies.Rollback.RequiresBeforeSnapshot)
	assert.True(t, policies.Compensation.Required)
	assert.Equal(t, "compensating", policies.Compensation.Kind)
	assert.True(t, policies.Compensation.OperatorRequired)
}

func TestPoliciesForNonReversibleSurfaceRequireManualRestore(t *testing.T) {
	policies := PoliciesFor(PlatformGitHub, "copilot_autofix", "mutate", "update")

	assert.Equal(t, RollbackNotSupported, policies.Rollback.Kind)
	assert.True(t, policies.Compensation.Required)
	assert.Equal(t, "manual_restore_required", policies.Compensation.Kind)
	assert.True(t, policies.Compensation.OperatorRequired)
}
