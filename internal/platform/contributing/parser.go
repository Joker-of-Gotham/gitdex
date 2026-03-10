package contributing

import (
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

// Parse analyzes CONTRIBUTING.md content and extracts a ContributingSpec.
func Parse(content string) *platform.ContributingSpec {
	spec := &platform.ContributingSpec{}
	lower := strings.ToLower(content)

	// Detect commit convention (Conventional Commits, Angular, etc.)
	if strings.Contains(lower, "conventional commits") {
		spec.CommitConvention = "conventional"
	} else if strings.Contains(lower, "angular") && strings.Contains(lower, "commit") {
		spec.CommitConvention = "angular"
	}

	// Detect branch naming
	if strings.Contains(lower, "feature/") || strings.Contains(lower, "develop") && strings.Contains(lower, "main") {
		spec.BranchNaming = "gitflow"
	} else if strings.Contains(lower, "branch") && strings.Contains(lower, "bugfix/") {
		spec.BranchNaming = "gitflow"
	}

	// Detect DCO (Developer Certificate of Origin)
	if strings.Contains(lower, "dco") || strings.Contains(lower, "developer certificate") {
		spec.DCORequired = true
	}

	return spec
}
