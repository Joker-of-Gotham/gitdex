package engine

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestParseRisk_SafeValues(t *testing.T) {
	for _, input := range []string{"safe", "Safe", "SAFE", "low", "none", ""} {
		assert.Equal(t, git.RiskSafe, parseRisk(input), "input=%q should be RiskSafe", input)
	}
}

func TestParseRisk_CautionValues(t *testing.T) {
	for _, input := range []string{"caution", "warning", "medium", "Caution", "WARNING"} {
		assert.Equal(t, git.RiskCaution, parseRisk(input), "input=%q should be RiskCaution", input)
	}
}

func TestParseRisk_DangerousValues(t *testing.T) {
	for _, input := range []string{"dangerous", "danger", "high", "DANGEROUS", "High"} {
		assert.Equal(t, git.RiskDangerous, parseRisk(input), "input=%q should be RiskDangerous", input)
	}
}

func TestParseRisk_UnknownDefaultsToSafe(t *testing.T) {
	assert.Equal(t, git.RiskSafe, parseRisk("unknown"))
	assert.Equal(t, git.RiskSafe, parseRisk("foobar"))
}
