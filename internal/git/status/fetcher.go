package status

import (
	"context"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git/parser"
)

func (w *StatusWatcher) enrichRemoteBranches(ctx context.Context, state *GitState) {
	stdout, _, err := w.cli.Exec(ctx, "branch", "-r", "--format=%(refname:short) %(upstream:track)")
	if err != nil {
		return
	}
	for _, line := range splitLines(stdout) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		state.RemoteBranches = append(state.RemoteBranches, line)
	}
}

func (w *StatusWatcher) enrichUpstreamCommits(ctx context.Context, state *GitState) {
	if state == nil || strings.TrimSpace(state.LocalBranch.Upstream) == "" || state.IsInitial {
		return
	}

	aheadOut, _, err := w.cli.Exec(ctx, "log", "--oneline", "-n", "5", "@{upstream}..HEAD")
	if err == nil {
		for _, line := range splitLines(aheadOut) {
			line = strings.TrimSpace(line)
			if line != "" {
				state.AheadCommits = append(state.AheadCommits, line)
			}
		}
	}

	behindOut, _, err := w.cli.Exec(ctx, "log", "--oneline", "-n", "5", "HEAD..@{upstream}")
	if err == nil {
		for _, line := range splitLines(behindOut) {
			line = strings.TrimSpace(line)
			if line != "" {
				state.BehindCommits = append(state.BehindCommits, line)
			}
		}
	}
}

func (w *StatusWatcher) enrichLastFetchTime(ctx context.Context, state *GitState) {
	if state == nil {
		return
	}
	remote, branch := parseUpstreamRef(state.LocalBranch.Upstream)
	if remote == "" || branch == "" {
		return
	}
	state.LastFetchTime = parser.LastFetchTime(ctx, w.cli, remote, branch)
}

func (w *StatusWatcher) maybeAutoFetch(state *GitState) {
	if state == nil || w.autoFetchInterval <= 0 {
		return
	}
	if !state.LastFetchTime.IsZero() && time.Since(state.LastFetchTime) < w.autoFetchInterval {
		return
	}

	remote, _ := parseUpstreamRef(state.LocalBranch.Upstream)
	if remote == "" && len(state.Remotes) > 0 {
		remote = strings.TrimSpace(state.Remotes[0])
	}
	if remote == "" {
		return
	}

	w.fetchMu.Lock()
	if w.fetchInFlight {
		w.fetchMu.Unlock()
		return
	}
	w.fetchInFlight = true
	w.fetchMu.Unlock()

	go func() {
		defer func() {
			w.fetchMu.Lock()
			w.fetchInFlight = false
			w.fetchMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, _, _ = w.cli.Exec(ctx, "fetch", "--quiet", "--prune", remote)
	}()
}

func parseUpstreamRef(upstream string) (remote, branch string) {
	upstream = strings.TrimSpace(upstream)
	if upstream == "" {
		return "", ""
	}
	parts := strings.SplitN(upstream, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
