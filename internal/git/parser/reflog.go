package parser

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
)

// ReflogEntry represents a parsed reflog entry.
type ReflogEntry struct {
	Hash    string
	Action  string
	Message string
}

// ParseReflog parses git reflog output into ReflogEntry slice.
// Format: hash HEAD@{n}: action: message (e.g. "abc1234 HEAD@{0}: checkout: moving from main to feat")
func ParseReflog(output string) []ReflogEntry {
	if strings.TrimSpace(output) == "" {
		return nil
	}
	// Match: hash optional_head_ref rest
	re := regexp.MustCompile(`^([a-f0-9]{7,40})\s+(?:HEAD@\{\d+\}:\s+)?(.+)$`)
	var entries []ReflogEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := re.FindStringSubmatch(line)
		if len(m) >= 3 {
			rest := strings.TrimSpace(m[2])
			action := rest
			msg := ""
			if idx := strings.Index(rest, ": "); idx >= 0 {
				action = strings.TrimSpace(rest[:idx])
				msg = strings.TrimSpace(rest[idx+2:])
			}
			entries = append(entries, ReflogEntry{
				Hash:    m[1],
				Action:  action,
				Message: msg,
			})
		}
	}
	return entries
}

// LastFetchTime attempts to determine when the remote ref was last fetched
// by parsing git reflog for refs/remotes/<remote>/<branch>.
// Returns zero time if unknown.
func LastFetchTime(ctx context.Context, gitCLI cli.GitCLI, remote, branch string) time.Time {
	if gitCLI == nil || remote == "" || branch == "" {
		return time.Time{}
	}
	ref := "refs/remotes/" + remote + "/" + branch
	// Use log -g to walk reflog; %ci = committer date ISO
	out, _, err := gitCLI.Exec(ctx, "log", "-g", "-1", "--format=%ci", ref)
	if err != nil || out == "" {
		return time.Time{}
	}
	// Format: 2024-01-15 10:30:00 +0000 (RFC3339-like)
	s := strings.TrimSpace(out)
	t, err := time.Parse("2006-01-02 15:04:05 -0700", s)
	if err != nil {
		// Fallback without timezone
		parts := strings.Fields(s)
		if len(parts) >= 2 {
			t, err = time.Parse("2006-01-02 15:04:05", parts[0]+" "+parts[1])
		}
	}
	if err != nil {
		return time.Time{}
	}
	return t
}
