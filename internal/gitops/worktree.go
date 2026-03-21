package gitops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WorktreeConfig configures creation of an isolated worktree.
type WorktreeConfig struct {
	RepoPath    string `json:"repo_path" yaml:"repo_path"`
	Branch      string `json:"branch" yaml:"branch"`
	WorktreeDir string `json:"worktree_dir" yaml:"worktree_dir"`
	// StartPoint, when non-empty, creates a new branch with -b from this commit/branch.
	StartPoint string `json:"start_point,omitempty" yaml:"start_point,omitempty"`
}

// WorktreeStatus represents the state of a worktree.
type WorktreeStatus string

const (
	WorktreeStatusActive  WorktreeStatus = "active"
	WorktreeStatusDirty   WorktreeStatus = "dirty"
	WorktreeStatusClean   WorktreeStatus = "clean"
	WorktreeStatusRemoved WorktreeStatus = "removed"
)

// Worktree represents an isolated worktree and its metadata.
type Worktree struct {
	Config         WorktreeConfig `json:"config" yaml:"config"`
	Status         WorktreeStatus `json:"status" yaml:"status"`
	CreatedAt      time.Time      `json:"created_at" yaml:"created_at"`
	DiffSummary    string         `json:"diff_summary,omitempty" yaml:"diff_summary,omitempty"`
	HeadSHA        string         `json:"head_sha,omitempty" yaml:"head_sha,omitempty"`
	IsLocked       bool           `json:"is_locked" yaml:"is_locked"`
	LockReason     string         `json:"lock_reason,omitempty" yaml:"lock_reason,omitempty"`
	UntrackedCount int            `json:"untracked_count" yaml:"untracked_count"`
	ModifiedCount  int            `json:"modified_count" yaml:"modified_count"`
}

// WorktreeManager manages isolated worktrees using real git worktree commands.
type WorktreeManager struct {
	executor *GitExecutor
}

// NewWorktreeManager creates a new WorktreeManager with the given executor.
func NewWorktreeManager(executor *GitExecutor) *WorktreeManager {
	return &WorktreeManager{executor: executor}
}

// Create creates an isolated worktree using git worktree add.
func (m *WorktreeManager) Create(ctx context.Context, config WorktreeConfig) (*Worktree, error) {
	if config.RepoPath == "" {
		return nil, fmt.Errorf("repository path is required")
	}
	if config.Branch == "" {
		return nil, fmt.Errorf("branch is required")
	}
	if config.WorktreeDir == "" {
		config.WorktreeDir = filepath.Join(config.RepoPath, "..", "gitdex-worktree-"+config.Branch)
	}

	var args []string
	if config.StartPoint != "" {
		args = []string{"worktree", "add", "-b", config.Branch, config.WorktreeDir, config.StartPoint}
	} else {
		args = []string{"worktree", "add", config.WorktreeDir, config.Branch}
	}

	_, err := m.executor.Run(ctx, config.RepoPath, args...)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(config.WorktreeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("worktree directory was not created: %s", config.WorktreeDir)
	}

	wt, err := m.Inspect(ctx, config.WorktreeDir)
	if err != nil {
		return nil, err
	}
	wt.Config = config
	wt.CreatedAt = time.Now().UTC()
	return wt, nil
}

// Inspect inspects the state of a worktree.
func (m *WorktreeManager) Inspect(ctx context.Context, worktreeDir string) (*Worktree, error) {
	if worktreeDir == "" {
		return nil, fmt.Errorf("worktree directory is required")
	}

	wt := &Worktree{
		Config: WorktreeConfig{
			WorktreeDir: worktreeDir,
			RepoPath:    filepath.Dir(worktreeDir),
		},
	}

	// Branch: git -C <dir> rev-parse --abbrev-ref HEAD
	branchLines, err := m.executor.RunLines(ctx, worktreeDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	if len(branchLines) > 0 && branchLines[0] != "" {
		wt.Config.Branch = branchLines[0]
	}

	// HeadSHA: git -C <dir> log -1 --format=%H
	shaRes, err := m.executor.Run(ctx, worktreeDir, "log", "-1", "--format=%H")
	if err == nil && shaRes.Stdout != "" {
		wt.HeadSHA = strings.TrimSpace(shaRes.Stdout)
	}

	// Status: git -C <dir> status --porcelain
	porcelain, err := m.executor.RunLines(ctx, worktreeDir, "status", "--porcelain")
	if err != nil {
		wt.Status = WorktreeStatusActive
		return wt, nil
	}

	modified := 0
	untracked := 0
	for _, line := range porcelain {
		if line == "" {
			continue
		}
		if len(line) >= 2 {
			xy := line[:2]
			if xy == "??" {
				untracked++
			} else {
				modified++
			}
		}
	}
	wt.ModifiedCount = modified
	wt.UntrackedCount = untracked
	if modified > 0 || untracked > 0 {
		wt.Status = WorktreeStatusDirty
	} else {
		wt.Status = WorktreeStatusClean
	}

	// Lock status: check .git file for locked worktrees (git worktree list --porcelain shows "locked")
	// We run list from repo root to get lock info; for a single dir we don't have lock in status.
	// IsLocked/LockReason come from List parsing; for Inspect we leave them false/empty.
	return wt, nil
}

// Diff returns the diff from the worktree (working tree + staged).
func (m *WorktreeManager) Diff(ctx context.Context, worktreeDir string) (string, error) {
	if worktreeDir == "" {
		return "", fmt.Errorf("worktree directory is required")
	}

	var parts []string

	// Unstaged diff
	unstaged, err := m.executor.Run(ctx, worktreeDir, "diff")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(unstaged.Stdout) != "" {
		parts = append(parts, unstaged.Stdout)
	}

	// Staged diff
	staged, err := m.executor.Run(ctx, worktreeDir, "diff", "--cached")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(staged.Stdout) != "" {
		parts = append(parts, staged.Stdout)
	}

	return strings.Join(parts, "\n"), nil
}

// Discard removes the worktree. Uses --force if the worktree has uncommitted changes.
func (m *WorktreeManager) Discard(ctx context.Context, worktreeDir string) error {
	return m.discardWithForce(ctx, worktreeDir, false)
}

func (m *WorktreeManager) discardWithForce(ctx context.Context, worktreeDir string, force bool) error {
	if worktreeDir == "" {
		return fmt.Errorf("worktree directory is required")
	}

	// Get repo path (parent of worktree's .git)
	gitDirRes, err := m.executor.Run(ctx, worktreeDir, "rev-parse", "--git-dir")
	if err != nil {
		return err
	}
	wtGitDir := strings.TrimSpace(gitDirRes.Stdout)
	repoPath := resolveRepoRootFromWorktreeGitDir(wtGitDir, worktreeDir)

	args := []string{"worktree", "remove", worktreeDir}
	if force {
		args = append(args, "--force")
	}

	_, err = m.executor.Run(ctx, repoPath, args...)
	if err != nil {
		// Retry with --force if remove fails (e.g. dirty worktree)
		if !force {
			return m.discardWithForce(ctx, worktreeDir, true)
		}
		return err
	}
	return nil
}

// List returns all worktrees for the repository.
func (m *WorktreeManager) List(ctx context.Context, repoPath string) ([]*Worktree, error) {
	lines, err := m.executor.RunLines(ctx, repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var result []*Worktree
	var current *Worktree

	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				result = append(result, current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			current = &Worktree{
				Config: WorktreeConfig{
					RepoPath:    repoPath,
					WorktreeDir: path,
				},
			}
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "HEAD ") {
			current.HeadSHA = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
		} else if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch refs/heads/")
			current.Config.Branch = ref
		} else if line == "locked" {
			current.IsLocked = true
		} else if strings.HasPrefix(line, "locked ") {
			current.IsLocked = true
			current.LockReason = strings.TrimPrefix(line, "locked ")
		}
	}
	if current != nil {
		result = append(result, current)
	}

	// Enrich with status for each worktree
	for _, wt := range result {
		insp, err := m.Inspect(ctx, wt.Config.WorktreeDir)
		if err == nil {
			wt.Status = insp.Status
			wt.HeadSHA = insp.HeadSHA
			wt.ModifiedCount = insp.ModifiedCount
			wt.UntrackedCount = insp.UntrackedCount
		}
	}

	return result, nil
}

// Lock locks a worktree with the given reason.
func (m *WorktreeManager) Lock(ctx context.Context, dir, reason string) error {
	if dir == "" {
		return fmt.Errorf("worktree directory is required")
	}
	gitDirRes, err := m.executor.Run(ctx, dir, "rev-parse", "--git-dir")
	if err != nil {
		return err
	}
	wtGitDir := strings.TrimSpace(gitDirRes.Stdout)
	repoPath := resolveRepoRootFromWorktreeGitDir(wtGitDir, dir)

	args := []string{"worktree", "lock"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	args = append(args, dir)
	_, err = m.executor.Run(ctx, repoPath, args...)
	return err
}

// Unlock unlocks a worktree.
func (m *WorktreeManager) Unlock(ctx context.Context, dir string) error {
	if dir == "" {
		return fmt.Errorf("worktree directory is required")
	}
	gitDirRes, err := m.executor.Run(ctx, dir, "rev-parse", "--git-dir")
	if err != nil {
		return err
	}
	wtGitDir := strings.TrimSpace(gitDirRes.Stdout)
	repoPath := resolveRepoRootFromWorktreeGitDir(wtGitDir, dir)
	_, err = m.executor.Run(ctx, repoPath, "worktree", "unlock", dir)
	return err
}

// resolveRepoRootFromWorktreeGitDir returns the main repo root given a worktree's git dir.
// For linked worktrees, git dir is like /repo/.git/worktrees/branch-name; we need /repo.
func resolveRepoRootFromWorktreeGitDir(wtGitDir, worktreeDir string) string {
	// If it's a worktree, path is repo/.git/worktrees/xxx — go up to .git, then to repo.
	if strings.Contains(wtGitDir, filepath.Join(".git", "worktrees")) {
		d := filepath.Dir(wtGitDir) // .git/worktrees
		d = filepath.Dir(d)         // .git
		return filepath.Dir(d)      // repo root
	}
	// Main worktree: git dir is repo/.git
	return filepath.Dir(wtGitDir)
}
