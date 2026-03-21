package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DriftStatus indicates the result of a drift check.
type DriftStatus string

const (
	DriftOK       DriftStatus = "ok"
	DriftDetected DriftStatus = "drift"
	DriftError    DriftStatus = "error"
)

// DriftCheck holds the result of a single drift check.
type DriftCheck struct {
	Name     string
	Status   DriftStatus
	Detail   string
	Severity string
}

// DriftReport aggregates all drift checks for a repository.
type DriftReport struct {
	RepoPath  string
	Timestamp time.Time
	Checks    []DriftCheck
	HasDrift  bool
	Summary   string
}

// RemediationResult holds the outcome of remediation attempts.
type RemediationResult struct {
	Fixed   []string
	Failed  []string
	Skipped []string
}

// ReconciliationController runs drift checks and remediation.
type ReconciliationController struct {
	executor  *GitExecutor
	worktrees *WorktreeManager
	inspector *Inspector
	integrity *IntegrityChecker
}

// NewReconciliationController creates a new ReconciliationController.
func NewReconciliationController(executor *GitExecutor, worktrees *WorktreeManager, inspector *Inspector, integrity *IntegrityChecker) *ReconciliationController {
	return &ReconciliationController{
		executor:  executor,
		worktrees: worktrees,
		inspector: inspector,
		integrity: integrity,
	}
}

// RunFullCheck runs all drift checks and returns a DriftReport.
func (rc *ReconciliationController) RunFullCheck(ctx context.Context, repoPath string) (*DriftReport, error) {
	r := &DriftReport{
		RepoPath:  repoPath,
		Timestamp: time.Now().UTC(),
	}

	checks := []DriftCheck{
		rc.CheckOrphanedWorktrees(ctx, repoPath),
		rc.CheckStaleLocks(ctx, repoPath),
		rc.CheckRefIntegrity(ctx, repoPath),
		rc.CheckBranchTracking(ctx, repoPath),
		rc.CheckStaleRemoteBranches(ctx, repoPath),
	}

	r.Checks = checks
	for _, c := range checks {
		if c.Status == DriftDetected || c.Status == DriftError {
			r.HasDrift = true
			break
		}
	}

	var summaryParts []string
	for _, c := range checks {
		if c.Status != DriftOK {
			summaryParts = append(summaryParts, c.Name+": "+string(c.Status))
		}
	}
	if len(summaryParts) > 0 {
		r.Summary = strings.Join(summaryParts, "; ")
	} else {
		r.Summary = "All checks passed"
	}
	return r, nil
}

// Remediate attempts to auto-fix drift items from the report.
func (rc *ReconciliationController) Remediate(ctx context.Context, report *DriftReport) (*RemediationResult, error) {
	res := &RemediationResult{}
	for _, c := range report.Checks {
		if c.Status != DriftDetected {
			continue
		}
		switch c.Name {
		case "orphaned_worktrees":
			// Skip - requires task context to know which worktrees are orphaned
			res.Skipped = append(res.Skipped, c.Name)
		case "stale_locks":
			// Remove .lock files in .git (risky - only if we're sure they're stale)
			gitDir := filepath.Join(report.RepoPath, ".git")
			removed, err := rc.removeStaleLocks(gitDir)
			if err != nil {
				res.Failed = append(res.Failed, c.Name)
			} else if removed {
				res.Fixed = append(res.Fixed, c.Name)
			} else {
				res.Skipped = append(res.Skipped, c.Name)
			}
		case "ref_integrity":
			// Run fsck and potentially gc
			fsck, err := rc.integrity.Fsck(ctx, report.RepoPath, false)
			if err != nil {
				res.Failed = append(res.Failed, c.Name)
			} else if !fsck.Clean {
				_ = rc.integrity.Maintenance(ctx, report.RepoPath, "gc")
				res.Fixed = append(res.Fixed, c.Name)
			} else {
				res.Skipped = append(res.Skipped, c.Name)
			}
		case "branch_tracking":
			// Skip - manual config
			res.Skipped = append(res.Skipped, c.Name)
		case "stale_remote_branches":
			// Prune remote-tracking refs
			_, err := rc.executor.Run(ctx, report.RepoPath, "remote", "prune", "origin")
			if err != nil {
				res.Failed = append(res.Failed, c.Name)
			} else {
				res.Fixed = append(res.Fixed, c.Name)
			}
		default:
			res.Skipped = append(res.Skipped, c.Name)
		}
	}
	return res, nil
}

func (rc *ReconciliationController) removeStaleLocks(gitDir string) (bool, error) {
	var removed bool
	err := filepath.Walk(gitDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".lock") {
			if err := os.Remove(path); err == nil {
				removed = true
			}
		}
		return nil
	})
	return removed, err
}

// CheckOrphanedWorktrees checks for worktrees without matching active tasks.
func (rc *ReconciliationController) CheckOrphanedWorktrees(ctx context.Context, repoPath string) DriftCheck {
	c := DriftCheck{Name: "orphaned_worktrees", Severity: "low"}
	if rc.worktrees == nil {
		c.Status = DriftOK
		c.Detail = "WorktreeManager not configured"
		return c
	}
	wts, err := rc.worktrees.List(ctx, repoPath)
	if err != nil {
		c.Status = DriftError
		c.Detail = err.Error()
		return c
	}
	// Consider main worktree + 1 = 2 as baseline; extra worktrees may be orphaned
	// Without task context we can only report count
	if len(wts) > 1 {
		c.Status = DriftDetected
		c.Detail = "multiple worktrees present; verify none are orphaned"
		return c
	}
	c.Status = DriftOK
	return c
}

// CheckStaleLocks checks for .lock files in .git.
func (rc *ReconciliationController) CheckStaleLocks(ctx context.Context, repoPath string) DriftCheck {
	c := DriftCheck{Name: "stale_locks", Severity: "medium"}
	gitDir := filepath.Join(repoPath, ".git")
	var lockCount int
	_ = filepath.Walk(gitDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".lock") {
			lockCount++
		}
		return nil
	})
	if lockCount > 0 {
		c.Status = DriftDetected
		c.Detail = "found lock files in .git (possible stale locks)"
	} else {
		c.Status = DriftOK
	}
	return c
}

// CheckRefIntegrity runs git fsck --no-dangling for a quick integrity check.
func (rc *ReconciliationController) CheckRefIntegrity(ctx context.Context, repoPath string) DriftCheck {
	c := DriftCheck{Name: "ref_integrity", Severity: "high"}
	if rc.integrity == nil {
		c.Status = DriftOK
		c.Detail = "IntegrityChecker not configured"
		return c
	}
	fsck, err := rc.integrity.Fsck(ctx, repoPath, false)
	if err != nil {
		c.Status = DriftError
		c.Detail = err.Error()
		return c
	}
	if !fsck.Clean || len(fsck.Missing) > 0 || len(fsck.Corrupt) > 0 {
		c.Status = DriftDetected
		c.Detail = "repository integrity issues detected"
		return c
	}
	c.Status = DriftOK
	return c
}

// CheckBranchTracking checks for branches without upstream.
func (rc *ReconciliationController) CheckBranchTracking(ctx context.Context, repoPath string) DriftCheck {
	c := DriftCheck{Name: "branch_tracking", Severity: "low"}
	lines, err := rc.executor.RunLines(ctx, repoPath, "for-each-ref", "--format=%(refname:short)%x00%(upstream:short)", "refs/heads/")
	if err != nil {
		c.Status = DriftError
		c.Detail = err.Error()
		return c
	}
	var untracked []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 2)
		if len(parts) >= 2 && parts[1] == "" {
			untracked = append(untracked, parts[0])
		}
	}
	if len(untracked) > 0 {
		c.Status = DriftDetected
		c.Detail = "branches without upstream: " + strings.Join(untracked, ", ")
	} else {
		c.Status = DriftOK
	}
	return c
}

// CheckStaleRemoteBranches checks for stale remote-tracking refs via git remote prune --dry-run.
func (rc *ReconciliationController) CheckStaleRemoteBranches(ctx context.Context, repoPath string) DriftCheck {
	c := DriftCheck{Name: "stale_remote_branches", Severity: "low"}
	result, err := rc.executor.Run(ctx, repoPath, "remote", "prune", "origin", "--dry-run")
	if err != nil {
		c.Status = DriftError
		c.Detail = err.Error()
		return c
	}
	// Dry-run output lists refs that would be pruned
	stderr := result.Stderr
	if stderr == "" {
		stderr = result.Stdout
	}
	if strings.Contains(stderr, "Would prune") || strings.Contains(stderr, "Pruning") {
		c.Status = DriftDetected
		c.Detail = "stale remote-tracking branches would be pruned"
	} else {
		c.Status = DriftOK
	}
	return c
}
