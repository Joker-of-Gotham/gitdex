package status

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
)

// StatusWatcher runs git status and returns parsed GitState.
type StatusWatcher struct {
	cli cli.GitCLI

	autoFetchInterval time.Duration
	fetchMu           sync.Mutex
	fetchInFlight     bool
}

// NewStatusWatcher creates a StatusWatcher that uses the given GitCLI.
func NewStatusWatcher(gitCLI cli.GitCLI) *StatusWatcher {
	return &StatusWatcher{
		cli:               gitCLI,
		autoFetchInterval: 5 * time.Minute,
	}
}

// SetAutoFetchInterval configures background fetch cadence.
// Zero or negative disables background fetch.
func (w *StatusWatcher) SetAutoFetchInterval(interval time.Duration) {
	w.autoFetchInterval = interval
}

// GetStatus runs `git status --porcelain=v2 --branch` and enriches with
// remote, branch, stash, tag, and repository state information.
func (w *StatusWatcher) GetStatus(ctx context.Context) (*GitState, error) {
	stdout, _, err := w.cli.Exec(ctx, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return nil, err
	}
	state, err := ParseStatusV2(stdout)
	if err != nil {
		return nil, err
	}

	state.IsInitial = state.HeadRef == ""
	w.enrichRemotes(ctx, state)
	w.enrichBranches(ctx, state)
	w.enrichRemoteBranches(ctx, state)
	w.enrichStash(ctx, state)
	w.enrichTags(ctx, state)
	w.enrichRepoState(ctx, state)
	w.enrichCommitCount(ctx, state)
	w.enrichUpstreamCommits(ctx, state)
	w.enrichLastFetchTime(ctx, state)
	w.maybeAutoFetch(state)
	state.HasGitIgnore = fileExists(".gitignore")

	enrichFileInspection(ctx, w.cli, state)
	enrichCommitSummary(ctx, w.cli, state)
	enrichConfigState(ctx, w.cli, state)

	return state, nil
}

func (w *StatusWatcher) enrichRemotes(ctx context.Context, state *GitState) {
	stdout, _, err := w.cli.Exec(ctx, "remote", "-v")
	if err != nil {
		return
	}
	state.RemoteURLs = make(map[string]string)
	byName := make(map[string]*git.RemoteInfo)
	for _, line := range splitLines(stdout) {
		// Format: "origin	https://github.com/user/repo.git (fetch)"
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		name := parts[0]
		remoteURL := parts[1]
		kind := strings.Trim(parts[2], "()")

		info, ok := byName[name]
		if !ok {
			info = &git.RemoteInfo{Name: name}
			byName[name] = info
			state.Remotes = append(state.Remotes, name)
		}

		switch kind {
		case "fetch":
			info.FetchURL = remoteURL
		case "push":
			info.PushURL = remoteURL
		}
	}

	for _, name := range state.Remotes {
		info := byName[name]
		if info == nil {
			continue
		}
		if info.FetchURL == "" {
			info.FetchURL = info.PushURL
		}
		if info.PushURL == "" {
			info.PushURL = info.FetchURL
		}
		info.FetchURLValid = isLikelyRemoteURL(info.FetchURL)
		info.PushURLValid = isLikelyRemoteURL(info.PushURL)
		info.URL = preferredRemoteURL(*info)
		if info.URL != "" {
			state.RemoteURLs[name] = info.URL
		}
		if info.FetchURL != "" || info.PushURL != "" {
			info.ReachabilityChecked = false
			info.Reachable = false
		}
		if (info.FetchURL != "" || info.PushURL != "") && !info.FetchURLValid && !info.PushURLValid {
			info.LastError = "remote URL looks invalid or placeholder-like"
		}
		state.RemoteInfos = append(state.RemoteInfos, *info)
	}

	if len(state.RemoteInfos) > 0 {
		state.RemoteState = state.RemoteInfos[0]
	}
}

func preferredRemoteURL(info git.RemoteInfo) string {
	if info.PushURL != "" {
		return info.PushURL
	}
	return info.FetchURL
}

func isLikelyRemoteURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	lower := strings.ToLower(raw)
	if strings.ContainsAny(raw, "<>") {
		return false
	}
	if looksLikePlaceholderURL(lower) {
		return false
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return false
		}
		return u.Scheme != "" && (u.Host != "" || u.Scheme == "file")
	}
	if strings.HasPrefix(raw, "git@") && strings.Contains(raw, ":") {
		return true
	}
	if strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") ||
		strings.HasPrefix(raw, ".\\") || strings.HasPrefix(raw, "..\\") ||
		strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "\\") {
		return true
	}
	return len(raw) >= 2 && raw[1] == ':'
}

func looksLikePlaceholderURL(lower string) bool {
	exactPlaceholders := []string{
		"your-username", "your-repo", "your_username", "your_repo",
		"your-user", "youruser", "yourrepo",
		"example.com", "example.org", "example.net",
		"__input_", "placeholder",
		"<url>", "<remote>", "<repo>",
	}
	for _, token := range exactPlaceholders {
		if strings.Contains(lower, token) {
			return true
		}
	}
	pathPlaceholders := []string{
		"user/repo.git",
		"user/repo ",
		"username/repo",
		"user-name/repo-name",
	}
	for _, p := range pathPlaceholders {
		if strings.HasSuffix(lower, p) || strings.Contains(lower, p+".") {
			return true
		}
	}
	if strings.Contains(lower, ":user/repo") || strings.Contains(lower, "/user/repo") {
		after := ""
		if idx := strings.Index(lower, ":user/repo"); idx >= 0 {
			after = lower[idx+len(":user/repo"):]
		} else if idx := strings.Index(lower, "/user/repo"); idx >= 0 {
			after = lower[idx+len("/user/repo"):]
		}
		if after == "" || after == ".git" || after == "/" {
			return true
		}
	}
	return false
}

func (w *StatusWatcher) enrichBranches(ctx context.Context, state *GitState) {
	stdout, _, err := w.cli.Exec(ctx, "branch", "--list", "--format=%(refname:short)")
	if err != nil {
		return
	}
	for _, line := range splitLines(stdout) {
		if line != "" {
			state.LocalBranches = append(state.LocalBranches, line)
		}
	}
}

func (w *StatusWatcher) enrichStash(ctx context.Context, state *GitState) {
	stdout, _, err := w.cli.Exec(ctx, "stash", "list")
	if err != nil {
		return
	}
	for i, line := range splitLines(stdout) {
		if line != "" {
			state.StashStack = append(state.StashStack, git.StashEntry{Index: i, Message: line})
		}
	}
}

func (w *StatusWatcher) enrichTags(ctx context.Context, state *GitState) {
	stdout, _, err := w.cli.Exec(ctx, "tag", "-l", "--sort=-v:refname")
	if err != nil {
		return
	}
	for _, line := range splitLines(stdout) {
		if line != "" {
			state.Tags = append(state.Tags, line)
			if len(state.Tags) >= 10 {
				break
			}
		}
	}
}

func (w *StatusWatcher) enrichRepoState(ctx context.Context, state *GitState) {
	gitDir, _, err := w.cli.Exec(ctx, "rev-parse", "--git-dir")
	if err != nil {
		return
	}
	gitDir = strings.TrimSpace(gitDir)
	state.MergeInProgress = fileExists(gitDir + "/MERGE_HEAD")
	state.RebaseInProgress = dirExists(gitDir+"/rebase-merge") || dirExists(gitDir+"/rebase-apply")
	state.CherryInProgress = fileExists(gitDir + "/CHERRY_PICK_HEAD")
	state.BisectInProgress = fileExists(gitDir + "/BISECT_LOG")
}

func (w *StatusWatcher) enrichCommitCount(ctx context.Context, state *GitState) {
	state.CommitCount = -1
	if state.IsInitial {
		state.CommitCount = 0
		return
	}
	stdout, _, err := w.cli.Exec(ctx, "rev-list", "--count", "HEAD")
	if err != nil {
		return
	}
	if n, err := strconv.Atoi(strings.TrimSpace(stdout)); err == nil {
		state.CommitCount = n
	}
}

// GetStashCount returns the number of stash entries.
func (w *StatusWatcher) GetStashCount(ctx context.Context) (int, error) {
	stdout, stderr, err := w.cli.Exec(ctx, "stash", "list")
	if err != nil {
		msg := err.Error()
		if stderr != "" {
			msg = stderr
		}
		if strings.Contains(strings.ToLower(msg), "not a git repository") {
			return 0, nil
		}
		return 0, err
	}
	lines := splitLines(stdout)
	count := 0
	for _, l := range lines {
		if l != "" {
			count++
		}
	}
	return count, nil
}

func splitLines(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
