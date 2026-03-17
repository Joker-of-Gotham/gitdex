package collector

import (
	"context"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

// GitCollector wraps StatusWatcher and writes results to .gitdex/maintain/git-content.txt.
type GitCollector struct {
	watcher *status.StatusWatcher
	store   *dotgitdex.Manager
}

// NewGitCollector creates a new GitCollector.
func NewGitCollector(watcher *status.StatusWatcher, store *dotgitdex.Manager) *GitCollector {
	return &GitCollector{watcher: watcher, store: store}
}

// Collect runs a full git status scan and returns the state.
func (c *GitCollector) Collect(ctx context.Context) (*status.GitState, error) {
	return c.watcher.GetStatus(ctx)
}

// Refresh collects the git state and writes it to git-content.txt.
func (c *GitCollector) Refresh(ctx context.Context) (*status.GitState, error) {
	state, err := c.watcher.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.store.WriteGitContent(state); err != nil {
		return state, err
	}
	return state, nil
}
