package gitops

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type Inspector struct {
	executor *GitExecutor
}

func NewInspector(executor *GitExecutor) *Inspector {
	return &Inspector{executor: executor}
}

func (ins *Inspector) Inspect(ctx context.Context, repoPath string) (*RepoInspection, error) {
	if repoPath == "" {
		return nil, fmt.Errorf("repository path is required")
	}

	branch, err := ins.currentBranch(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine current branch: %w", err)
	}

	result := &RepoInspection{
		RepoPath:    repoPath,
		LocalBranch: branch,
	}

	if branch == "HEAD" {
		result.Divergence = DivDetached
		return result, nil
	}

	remote, err := ins.trackingBranch(ctx, repoPath, branch)
	if err != nil || remote == "" {
		result.Divergence = DivNoUpstream
		return result, nil
	}
	result.RemoteBranch = remote

	ahead, behind, err := ins.aheadBehind(ctx, repoPath, branch, remote)
	if err != nil {
		result.Divergence = DivNoUpstream
		return result, nil
	}
	result.Ahead = ahead
	result.Behind = behind

	switch {
	case ahead == 0 && behind == 0:
		result.Divergence = DivSynced
	case ahead > 0 && behind == 0:
		result.Divergence = DivAhead
	case ahead == 0 && behind > 0:
		result.Divergence = DivBehind
	default:
		result.Divergence = DivDiverged
	}

	uncommitted, _ := ins.hasUncommittedChanges(ctx, repoPath)
	result.HasUncommitted = uncommitted

	untracked, _ := ins.hasUntrackedFiles(ctx, repoPath)
	result.HasUntracked = untracked

	return result, nil
}

func (ins *Inspector) Recommend(inspection *RepoInspection) *SyncRecommendation {
	if inspection == nil {
		return &SyncRecommendation{Action: "none", RiskLevel: "low", Description: "No inspection data available"}
	}

	switch inspection.Divergence {
	case DivSynced:
		return &SyncRecommendation{
			Action:      "none",
			RiskLevel:   "low",
			Description: "Repository is up to date with upstream. No sync needed.",
			Previewable: false,
		}
	case DivAhead:
		return &SyncRecommendation{
			Action:      "push",
			RiskLevel:   "low",
			Description: fmt.Sprintf("Local branch is %d commit(s) ahead. Consider pushing to upstream.", inspection.Ahead),
			Previewable: true,
		}
	case DivBehind:
		action := "fast_forward"
		risk := "low"
		if inspection.HasUncommitted {
			action = "stash_and_pull"
			risk = "medium"
		}
		return &SyncRecommendation{
			Action:      action,
			RiskLevel:   risk,
			Description: fmt.Sprintf("Local branch is %d commit(s) behind. Safe to fast-forward.", inspection.Behind),
			Previewable: true,
		}
	case DivDiverged:
		return &SyncRecommendation{
			Action:      "merge_or_rebase",
			RiskLevel:   "high",
			Description: fmt.Sprintf("Branch has diverged: %d ahead, %d behind. Manual review recommended before sync.", inspection.Ahead, inspection.Behind),
			Previewable: true,
		}
	case DivDetached:
		return &SyncRecommendation{
			Action:      "checkout_branch",
			RiskLevel:   "medium",
			Description: "HEAD is detached. Checkout a branch before syncing.",
			Previewable: false,
		}
	case DivNoUpstream:
		return &SyncRecommendation{
			Action:      "set_upstream",
			RiskLevel:   "low",
			Description: "No upstream tracking branch configured. Set upstream before syncing.",
			Previewable: false,
		}
	default:
		return &SyncRecommendation{
			Action:      "unknown",
			RiskLevel:   "medium",
			Description: "Unable to determine recommended action.",
			Previewable: false,
		}
	}
}

func (ins *Inspector) currentBranch(ctx context.Context, repoPath string) (string, error) {
	out, err := ins.gitCmd(ctx, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (ins *Inspector) trackingBranch(ctx context.Context, repoPath, branch string) (string, error) {
	out, err := ins.gitCmd(ctx, repoPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", branch+"@{upstream}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (ins *Inspector) aheadBehind(ctx context.Context, repoPath, local, remote string) (int, int, error) {
	out, err := ins.gitCmd(ctx, repoPath, "rev-list", "--left-right", "--count", local+"..."+remote)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", out)
	}
	ahead, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	behind, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return ahead, behind, nil
}

func (ins *Inspector) hasUncommittedChanges(ctx context.Context, repoPath string) (bool, error) {
	out, err := ins.gitCmd(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "??") {
			return true, nil
		}
	}
	return false, nil
}

func (ins *Inspector) hasUntrackedFiles(ctx context.Context, repoPath string) (bool, error) {
	out, err := ins.gitCmd(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "??") {
			return true, nil
		}
	}
	return false, nil
}

func (ins *Inspector) gitCmd(ctx context.Context, repoPath string, args ...string) (string, error) {
	result, err := ins.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}
