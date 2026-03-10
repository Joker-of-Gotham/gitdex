package parser

import (
	"regexp"
	"strings"
)

// CommitEntry represents a parsed commit from git log.
type CommitEntry struct {
	Hash      string
	ShortHash string
	Author    string
	Date      string
	Message   string
	Branch    string
}

// LogFormat is the format string for git log --format.
// Uses placeholder that produces: HASH|SHORTHASH|AUTHOR|DATE|MESSAGE|BRANCH
const LogFormat = "%H|%h|%an|%ai|%s|%D"

// ParseLog parses `git log --oneline --format=...` style output into CommitEntry slice.
func ParseLog(output string) []CommitEntry {
	if strings.TrimSpace(output) == "" {
		return nil
	}
	var entries []CommitEntry
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		e := parseLogLine(line)
		if e.Hash != "" {
			entries = append(entries, e)
		}
	}
	return entries
}

// parseLogLine parses a single line in format: HASH|SHORTHASH|AUTHOR|DATE|MESSAGE|BRANCH
func parseLogLine(line string) CommitEntry {
	parts := strings.SplitN(line, "|", 6)
	var e CommitEntry
	if len(parts) >= 1 {
		e.Hash = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		e.ShortHash = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		e.Author = strings.TrimSpace(parts[2])
	}
	if len(parts) >= 4 {
		e.Date = strings.TrimSpace(parts[3])
	}
	if len(parts) >= 5 {
		e.Message = strings.TrimSpace(parts[4])
	}
	if len(parts) >= 6 {
		e.Branch = extractBranch(parts[5])
	}
	if e.ShortHash == "" && e.Hash != "" {
		if len(e.Hash) >= 7 {
			e.ShortHash = e.Hash[:7]
		} else {
			e.ShortHash = e.Hash
		}
	}
	return e
}

// extractBranch parses refs from %D output (e.g. "HEAD -> main, origin/main")
func extractBranch(s string) string {
	s = strings.TrimSpace(s)
	// Prefer "HEAD -> branch"
	if idx := strings.Index(s, "HEAD -> "); idx >= 0 {
		rest := s[idx+8:]
		if comma := strings.Index(rest, ","); comma >= 0 {
			return strings.TrimSpace(rest[:comma])
		}
		return strings.TrimSpace(rest)
	}
	// Fallback: first ref before comma
	if comma := strings.Index(s, ","); comma >= 0 {
		return strings.TrimSpace(s[:comma])
	}
	return s
}

// ParseLogOneline parses simple --oneline output (shortHash message).
func ParseLogOneline(output string) []CommitEntry {
	if strings.TrimSpace(output) == "" {
		return nil
	}
	re := regexp.MustCompile(`^([a-f0-9]{7,40})\s+(.*)$`)
	var entries []CommitEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := re.FindStringSubmatch(line)
		if len(m) >= 3 {
			entries = append(entries, CommitEntry{
				ShortHash: m[1],
				Hash:      m[1],
				Message:   m[2],
			})
		}
	}
	return entries
}
