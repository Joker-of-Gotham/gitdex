package gitops

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// HygieneExecutor performs low-risk repository hygiene tasks.
type HygieneExecutor struct {
	executor *GitExecutor
}

// NewHygieneExecutor returns a new HygieneExecutor.
func NewHygieneExecutor(executor *GitExecutor) *HygieneExecutor {
	return &HygieneExecutor{executor: executor}
}

// Execute runs the specified hygiene action on the repository at repoPath.
func (e *HygieneExecutor) Execute(ctx context.Context, repoPath string, action HygieneAction) (*HygieneResult, error) {
	if !isValidHygieneAction(action) {
		return nil, fmt.Errorf("unsupported hygiene action %q", action)
	}

	select {
	case <-ctx.Done():
		return &HygieneResult{
			Success:      false,
			Action:       action,
			ErrorMessage: ctx.Err().Error(),
			Summary:      "Hygiene run cancelled; no changes made",
		}, nil
	default:
	}

	if strings.TrimSpace(repoPath) == "" {
		return &HygieneResult{
			Success:      false,
			Action:       action,
			ErrorMessage: "repository path is required",
			Summary:      "Hygiene run failed: no repository path provided. Retry with a valid repo path or configure repository_root.",
		}, nil
	}

	task := taskForAction(action)

	switch action {
	case HygienePruneRemoteBranches:
		return e.executePruneRemoteBranches(ctx, repoPath, task)
	case HygieneGCAggressive:
		return e.executeGCAggressive(ctx, repoPath, task)
	case HygieneCleanUntracked:
		return e.executeCleanUntracked(ctx, repoPath, task)
	case HygieneRemoveMergedBranches:
		return e.executeRemoveMergedBranches(ctx, repoPath, task)
	default:
		return nil, fmt.Errorf("unsupported hygiene action %q", action)
	}
}

// DryRun runs the specified hygiene action in dry-run mode.
func (e *HygieneExecutor) DryRun(ctx context.Context, repoPath string, action HygieneAction) (*HygieneResult, error) {
	if !isValidHygieneAction(action) {
		return nil, fmt.Errorf("unsupported hygiene action %q", action)
	}

	if strings.TrimSpace(repoPath) == "" {
		return &HygieneResult{
			Success:      false,
			Action:       action,
			ErrorMessage: "repository path is required",
			Summary:      "Hygiene run failed: no repository path provided.",
		}, nil
	}

	task := taskForAction(action)

	switch action {
	case HygienePruneRemoteBranches:
		return e.dryRunPrune(ctx, repoPath, task)
	case HygieneGCAggressive:
		// gc has no dry-run; return empty result
		return &HygieneResult{
			Success: true,
			Action:  action,
			Summary: fmt.Sprintf("%s has no dry-run; execute to run.", task.Description),
		}, nil
	case HygieneCleanUntracked:
		return e.dryRunClean(ctx, repoPath, task)
	case HygieneRemoveMergedBranches:
		return e.dryRunRemoveMerged(ctx, repoPath, task)
	default:
		return nil, fmt.Errorf("unsupported hygiene action %q", action)
	}
}

func (e *HygieneExecutor) executePruneRemoteBranches(ctx context.Context, repoPath string, task HygieneTask) (*HygieneResult, error) {
	result, err := e.executor.Run(ctx, repoPath, "fetch", "--prune", "--all")
	if err != nil {
		return &HygieneResult{
			Success:      false,
			Action:       HygienePruneRemoteBranches,
			ErrorMessage: err.Error(),
			Summary:      fmt.Sprintf("Prune failed: %s", err),
		}, nil
	}
	count := countPruningRefs(result.Stderr)
	return &HygieneResult{
		Success:          true,
		Action:           HygienePruneRemoteBranches,
		BranchesAffected: count,
		Summary:          fmt.Sprintf("%s completed successfully.", task.Description),
	}, nil
}

var pruningLineRe = regexp.MustCompile(`^\s*-\s+\[deleted\]\s+.*`)

func countPruningRefs(stderr string) int {
	count := 0
	for _, line := range strings.Split(stderr, "\n") {
		if pruningLineRe.MatchString(strings.TrimSpace(line)) {
			count++
		}
	}
	return count
}

func (e *HygieneExecutor) executeGCAggressive(ctx context.Context, repoPath string, task HygieneTask) (*HygieneResult, error) {
	sizeBefore := ""
	if before, err := e.executor.Run(ctx, repoPath, "count-objects", "-v"); err == nil {
		sizeBefore = parseSizePack(before.Stdout)
	}

	_, err := e.executor.Run(ctx, repoPath, "gc", "--aggressive", "--prune=now")
	if err != nil {
		return &HygieneResult{
			Success:      false,
			Action:       HygieneGCAggressive,
			ErrorMessage: err.Error(),
			Summary:      fmt.Sprintf("GC failed: %s", err),
		}, nil
	}

	diskReclaimed := ""
	if after, err := e.executor.Run(ctx, repoPath, "count-objects", "-v"); err == nil {
		sizeAfter := parseSizePack(after.Stdout)
		if sizeBefore != "" && sizeAfter != "" {
			diskReclaimed = fmt.Sprintf("%s -> %s", sizeBefore, sizeAfter)
		}
	}

	return &HygieneResult{
		Success:       true,
		Action:        HygieneGCAggressive,
		DiskReclaimed: diskReclaimed,
		Summary:       fmt.Sprintf("%s completed successfully.", task.Description),
	}, nil
}

var sizePackRe = regexp.MustCompile(`size-pack:\s*(\d+)`)

func parseSizePack(stdout string) string {
	m := sizePackRe.FindStringSubmatch(stdout)
	if len(m) >= 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return fmt.Sprintf("%d KB", n)
		}
	}
	return ""
}

func (e *HygieneExecutor) executeCleanUntracked(ctx context.Context, repoPath string, task HygieneTask) (*HygieneResult, error) {
	result, err := e.executor.Run(ctx, repoPath, "clean", "-fd")
	if err != nil {
		return &HygieneResult{
			Success:      false,
			Action:       HygieneCleanUntracked,
			ErrorMessage: err.Error(),
			Summary:      fmt.Sprintf("Clean failed: %s", err),
		}, nil
	}
	files := parseRemovingLines(result.Stdout)
	return &HygieneResult{
		Success:       true,
		Action:        HygieneCleanUntracked,
		FilesAffected: len(files),
		DeletedFiles:  files,
		Summary:       fmt.Sprintf("%s completed successfully.", task.Description),
	}, nil
}

var removingRe = regexp.MustCompile(`^Removing\s+(.+)$`)

func parseRemovingLines(stdout string) []string {
	var files []string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if m := removingRe.FindStringSubmatch(line); len(m) >= 2 {
			files = append(files, m[1])
		}
	}
	return files
}

func (e *HygieneExecutor) executeRemoveMergedBranches(ctx context.Context, repoPath string, task HygieneTask) (*HygieneResult, error) {
	merged, err := e.executor.RunLines(ctx, repoPath, "branch", "--merged")
	if err != nil {
		return &HygieneResult{
			Success:      false,
			Action:       HygieneRemoveMergedBranches,
			ErrorMessage: err.Error(),
			Summary:      fmt.Sprintf("List merged branches failed: %s", err),
		}, nil
	}

	currentLines, _ := e.executor.RunLines(ctx, repoPath, "branch", "--show-current")
	currentBranch := ""
	if len(currentLines) > 0 {
		currentBranch = strings.TrimSpace(currentLines[0])
	}

	protected := map[string]bool{"main": true, "master": true, "develop": true, currentBranch: true}

	var toDelete []string
	for _, b := range merged {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		b = strings.TrimPrefix(b, "* ")
		if protected[b] {
			continue
		}
		toDelete = append(toDelete, b)
	}

	var deleted []string
	for _, b := range toDelete {
		_, err := e.executor.Run(ctx, repoPath, "branch", "-d", b)
		if err == nil {
			deleted = append(deleted, b)
		}
	}

	return &HygieneResult{
		Success:          true,
		Action:           HygieneRemoveMergedBranches,
		BranchesAffected: len(deleted),
		DeletedBranches:  deleted,
		Summary:          fmt.Sprintf("%s completed successfully.", task.Description),
	}, nil
}

func (e *HygieneExecutor) dryRunPrune(ctx context.Context, repoPath string, task HygieneTask) (*HygieneResult, error) {
	result, err := e.executor.Run(ctx, repoPath, "remote", "prune", "--dry-run", "origin")
	if err != nil {
		return &HygieneResult{
			Success:      false,
			Action:       HygienePruneRemoteBranches,
			ErrorMessage: err.Error(),
			Summary:      fmt.Sprintf("Dry-run prune failed: %s", err),
		}, nil
	}
	count := countPruningRefs(result.Stderr)
	return &HygieneResult{
		Success:          true,
		Action:           HygienePruneRemoteBranches,
		BranchesAffected: count,
		Summary:          fmt.Sprintf("Dry-run: %d ref(s) would be pruned.", count),
	}, nil
}

func (e *HygieneExecutor) dryRunClean(ctx context.Context, repoPath string, task HygieneTask) (*HygieneResult, error) {
	result, err := e.executor.Run(ctx, repoPath, "clean", "-nd")
	if err != nil {
		return &HygieneResult{
			Success:      false,
			Action:       HygieneCleanUntracked,
			ErrorMessage: err.Error(),
			Summary:      fmt.Sprintf("Dry-run clean failed: %s", err),
		}, nil
	}
	// git clean -nd outputs "Would remove <path>" lines
	files := parseWouldRemoveLines(result.Stdout)
	return &HygieneResult{
		Success:       true,
		Action:        HygieneCleanUntracked,
		FilesAffected: len(files),
		DeletedFiles:  files,
		Summary:       fmt.Sprintf("Dry-run: %d file(s) would be removed.", len(files)),
	}, nil
}

var wouldRemoveRe = regexp.MustCompile(`^Would remove\s+(.+)$`)

func parseWouldRemoveLines(stdout string) []string {
	var files []string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if m := wouldRemoveRe.FindStringSubmatch(line); len(m) >= 2 {
			files = append(files, m[1])
		}
	}
	return files
}

func (e *HygieneExecutor) dryRunRemoveMerged(ctx context.Context, repoPath string, task HygieneTask) (*HygieneResult, error) {
	merged, err := e.executor.RunLines(ctx, repoPath, "branch", "--merged")
	if err != nil {
		return &HygieneResult{
			Success:      false,
			Action:       HygieneRemoveMergedBranches,
			ErrorMessage: err.Error(),
			Summary:      fmt.Sprintf("Dry-run list failed: %s", err),
		}, nil
	}

	currentLines, _ := e.executor.RunLines(ctx, repoPath, "branch", "--show-current")
	currentBranch := ""
	if len(currentLines) > 0 {
		currentBranch = strings.TrimSpace(currentLines[0])
	}

	protected := map[string]bool{"main": true, "master": true, "develop": true, currentBranch: true}

	var wouldDelete []string
	for _, b := range merged {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		b = strings.TrimPrefix(b, "* ")
		if protected[b] {
			continue
		}
		wouldDelete = append(wouldDelete, b)
	}

	return &HygieneResult{
		Success:          true,
		Action:           HygieneRemoveMergedBranches,
		BranchesAffected: len(wouldDelete),
		DeletedBranches:  wouldDelete,
		Summary:          fmt.Sprintf("Dry-run: %d branch(es) would be deleted.", len(wouldDelete)),
	}, nil
}

func isValidHygieneAction(a HygieneAction) bool {
	for _, t := range SupportedHygieneTasks() {
		if t.Action == a {
			return true
		}
	}
	return false
}

func taskForAction(a HygieneAction) HygieneTask {
	for _, t := range SupportedHygieneTasks() {
		if t.Action == a {
			return t
		}
	}
	return HygieneTask{}
}
