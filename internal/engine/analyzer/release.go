package analyzer

import (
	"regexp"
	"strconv"
	"strings"
)

// ReleaseAnalyzer analyzes commits and tags for version suggestion.
type ReleaseAnalyzer struct{}

// NewReleaseAnalyzer creates a new ReleaseAnalyzer.
func NewReleaseAnalyzer() *ReleaseAnalyzer {
	return &ReleaseAnalyzer{}
}

// SuggestVersion parses the current semver tag, analyzes commits for "feat:", "fix:",
// "BREAKING CHANGE:", and returns the suggested next version.
func (a *ReleaseAnalyzer) SuggestVersion(currentTag string, commits []string) string {
	major, minor, patch := parseSemver(currentTag)
	hasBreaking := false
	hasFeat := false
	hasFix := false

	for _, msg := range commits {
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "breaking change") || strings.HasPrefix(lower, "break:") || strings.Contains(msg, "BREAKING CHANGE:") {
			hasBreaking = true
			break
		}
		if strings.HasPrefix(lower, "feat:") || strings.HasPrefix(lower, "feat(") {
			hasFeat = true
		}
		if strings.HasPrefix(lower, "fix:") || strings.HasPrefix(lower, "fix(") {
			hasFix = true
		}
	}

	if hasBreaking {
		return formatSemver(major+1, 0, 0)
	}
	if hasFeat {
		return formatSemver(major, minor+1, 0)
	}
	if hasFix {
		return formatSemver(major, minor, patch+1)
	}
	return formatSemver(major, minor, patch+1)
}

var semverRe = regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?`)

func parseSemver(tag string) (major, minor, patch int) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return 0, 0, 1
	}
	matches := semverRe.FindStringSubmatch(tag)
	if len(matches) < 2 {
		return 0, 0, 1
	}
	major, _ = strconv.Atoi(matches[1])
	if len(matches) >= 3 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	if len(matches) >= 4 && matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}
	return major, minor, patch
}

func formatSemver(major, minor, patch int) string {
	return "v" + strconv.Itoa(major) + "." + strconv.Itoa(minor) + "." + strconv.Itoa(patch)
}
