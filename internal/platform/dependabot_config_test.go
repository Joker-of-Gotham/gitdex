package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependabotConfigRoundTripIsDeterministic(t *testing.T) {
	input := `version: 2
registries:
  private-npm:
    type: npm-registry
    url: https://registry.example.com
updates:
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
      day: monday
    labels:
      - dependencies
    groups:
      security:
        patterns:
          - "*"
        applies-to: security-updates
  - package-ecosystem: github-actions
    directory: /docs
    schedule:
      interval: weekly
      day: monday
    labels:
      - dependencies
    groups:
      security:
        patterns:
          - "*"
        applies-to: security-updates
`
	cfg, err := ParseDependabotConfigYAML(input)
	require.NoError(t, err)
	require.Len(t, cfg.Updates, 1)
	assert.Equal(t, []string{"/", "/docs"}, cfg.Updates[0].Directories)
	assert.Contains(t, cfg.Updates[0].SecurityUpdates.GroupedUpdates, "security")

	rendered, err := RenderDependabotConfigYAML(cfg)
	require.NoError(t, err)
	reparsed, err := ParseDependabotConfigYAML(rendered)
	require.NoError(t, err)
	renderedAgain, err := RenderDependabotConfigYAML(reparsed)
	require.NoError(t, err)
	assert.Equal(t, rendered, renderedAgain)
}

func TestValidateDependabotConfigRequiresStructuredFields(t *testing.T) {
	err := ValidateDependabotConfig(DependabotConfig{
		Version: 2,
		Updates: []DependabotUpdate{{
			Directories: []string{"/"},
			Schedule: DependabotSchedule{
				Interval: "weekly",
			},
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ecosystem")
}
