package status

import (
	"context"
	"regexp"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
)

type CommitSummary struct {
	RecentCommits      []string `json:"recent_commits,omitempty"`
	UsesConventional   bool     `json:"uses_conventional_commits"`
	CommitFrequency    string   `json:"commit_frequency,omitempty"`
	LastCommitRelative string   `json:"last_commit_relative,omitempty"`
}

var conventionalPattern = regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+?\))?!?:`)

func enrichCommitSummary(ctx context.Context, gitCLI cli.GitCLI, state *GitState) {
	cs := &CommitSummary{}
	if state.IsInitial {
		state.CommitSummaryInfo = cs
		return
	}

	// Last 10 commits as oneline
	stdout, _, err := gitCLI.Exec(ctx, "log", "--oneline", "-10", "--format=%s")
	if err == nil {
		convCount := 0
		for _, line := range splitLines(stdout) {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			cs.RecentCommits = append(cs.RecentCommits, line)
			if conventionalPattern.MatchString(line) {
				convCount++
			}
		}
		if len(cs.RecentCommits) > 0 && convCount*2 >= len(cs.RecentCommits) {
			cs.UsesConventional = true
		}
	}

	// Last commit relative time
	stdout, _, err = gitCLI.Exec(ctx, "log", "-1", "--format=%cr")
	if err == nil {
		cs.LastCommitRelative = strings.TrimSpace(stdout)
	}

	// Commit frequency (last 7 days)
	stdout, _, err = gitCLI.Exec(ctx, "rev-list", "--count", "--since=7.days", "HEAD")
	if err == nil {
		count := strings.TrimSpace(stdout)
		if count != "" && count != "0" {
			cs.CommitFrequency = count + " commits in last 7 days"
		}
	}

	state.CommitSummaryInfo = cs
}
