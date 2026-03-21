package gitops

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Syncer struct {
	executor *GitExecutor
}

func NewSyncer(executor *GitExecutor) *Syncer {
	return &Syncer{executor: executor}
}

func (s *Syncer) Preview(_ context.Context, inspection *RepoInspection, rec *SyncRecommendation) (*SyncPreview, error) {
	if inspection == nil {
		return nil, fmt.Errorf("inspection data is required for preview")
	}
	if rec == nil || !rec.Previewable {
		return nil, fmt.Errorf("sync action %q is not previewable", rec.Action)
	}

	preview := &SyncPreview{
		MergeStrategy: "fast-forward",
		ConflictRisk:  "none",
	}

	switch rec.Action {
	case "fast_forward":
		preview.AffectedFiles = inspection.Behind
		preview.Description = fmt.Sprintf("Fast-forward merge will apply %d commit(s) from upstream.", inspection.Behind)
	case "push":
		preview.AffectedFiles = inspection.Ahead
		preview.Description = fmt.Sprintf("Push will send %d commit(s) to upstream.", inspection.Ahead)
	case "stash_and_pull":
		preview.AffectedFiles = inspection.Behind
		preview.MergeStrategy = "stash-pull-pop"
		preview.ConflictRisk = "low"
		preview.Description = fmt.Sprintf("Stash local changes, fast-forward %d commit(s), then restore.", inspection.Behind)
	case "merge_or_rebase":
		preview.AffectedFiles = inspection.Ahead + inspection.Behind
		preview.MergeStrategy = "merge"
		preview.ConflictRisk = "high"
		preview.Description = fmt.Sprintf("Branch diverged: %d local, %d remote commits. Merge or rebase required. Conflict risk is high.", inspection.Ahead, inspection.Behind)
	default:
		return nil, fmt.Errorf("unsupported sync action %q for preview", rec.Action)
	}

	return preview, nil
}

func (s *Syncer) Execute(ctx context.Context, inspection *RepoInspection, rec *SyncRecommendation) (*SyncResult, error) {
	if rec == nil {
		return nil, fmt.Errorf("sync recommendation is required")
	}
	if inspection == nil || inspection.RepoPath == "" {
		return nil, fmt.Errorf("inspection with repo path is required for execute")
	}

	repoPath := inspection.RepoPath
	remote := "origin"

	switch rec.Action {
	case "none":
		return &SyncResult{
			Success:     true,
			Description: "No sync action needed.",
		}, nil

	case "fast_forward":
		if err := s.fetchUpstream(ctx, repoPath, remote); err != nil {
			return &SyncResult{
				Success:      false,
				ErrorMessage: err.Error(),
				Description:  "Failed to fetch from upstream.",
			}, nil
		}
		mergeRef := inspection.RemoteBranch
		if mergeRef == "" {
			mergeRef = remote + "/" + inspection.LocalBranch
		}
		_, err := s.executor.Run(ctx, repoPath, "merge", "--ff-only", mergeRef)
		if err != nil {
			return &SyncResult{
				Success:      false,
				ErrorMessage: err.Error(),
				Description:  "Fast-forward merge failed.",
			}, nil
		}
		count, _ := s.countChangedFiles(ctx, repoPath, "HEAD@{1}", "HEAD")
		return &SyncResult{
			Success:      true,
			FilesChanged: count,
			Description:  "Fast-forward merge completed successfully.",
		}, nil

	case "push":
		branch := inspection.LocalBranch
		if branch == "" {
			return &SyncResult{
				Success:      false,
				ErrorMessage: "no local branch to push",
				Description:  "Push failed: no branch name.",
			}, nil
		}
		_, err := s.executor.Run(ctx, repoPath, "push", remote, branch)
		if err != nil {
			return &SyncResult{
				Success:      false,
				ErrorMessage: err.Error(),
				Description:  "Push failed.",
			}, nil
		}
		return &SyncResult{
			Success:     true,
			Description: "Push completed successfully.",
		}, nil

	case "stash_and_pull":
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		stashMsg := "gitdex-sync-" + timestamp
		stashRef := ""

		if inspection.HasUncommitted || inspection.HasUntracked {
			_, err := s.executor.Run(ctx, repoPath, "stash", "push", "-m", stashMsg)
			if err != nil {
				return &SyncResult{
					Success:      false,
					ErrorMessage: err.Error(),
					Description:  "Stash failed.",
				}, nil
			}
			stashRef = "stash@{0}"
		}

		_, err := s.executor.Run(ctx, repoPath, "pull", "--ff-only")
		if err != nil {
			if stashRef != "" {
				return &SyncResult{
					Success:      false,
					StashRef:     stashRef,
					ErrorMessage: err.Error(),
					Description:  "Pull failed; stash preserved.",
				}, nil
			}
			return &SyncResult{
				Success:      false,
				ErrorMessage: err.Error(),
				Description:  "Pull failed.",
			}, nil
		}

		if stashRef != "" {
			_, err = s.executor.Run(ctx, repoPath, "stash", "pop")
			if err != nil {
				return &SyncResult{
					Success:      false,
					Conflicts:    1,
					StashRef:     stashRef,
					ErrorMessage: err.Error(),
					Description:  "Stash pop had conflicts; stash preserved. Resolve manually and run git stash pop.",
				}, nil
			}
		}

		count, _ := s.countChangedFiles(ctx, repoPath, "HEAD@{1}", "HEAD")
		result := &SyncResult{
			Success:      true,
			FilesChanged: count,
			Description:  "Stash-and-pull completed successfully.",
		}
		if stashRef != "" {
			result.StashRef = stashRef
		}
		return result, nil

	case "merge_or_rebase":
		if err := s.fetchUpstream(ctx, repoPath, remote); err != nil {
			return &SyncResult{
				Success:      false,
				ErrorMessage: err.Error(),
				Description:  "Failed to fetch from upstream.",
			}, nil
		}
		mergeRef := inspection.RemoteBranch
		if mergeRef == "" {
			mergeRef = remote + "/" + inspection.LocalBranch
		}
		_, err := s.executor.Run(ctx, repoPath, "merge", mergeRef)
		if err != nil {
			conflictFiles, _ := s.executor.RunLines(ctx, repoPath, "diff", "--name-only", "--diff-filter=U")
			_, _ = s.executor.Run(ctx, repoPath, "merge", "--abort")
			return &SyncResult{
				Success:      false,
				Conflicts:    len(conflictFiles),
				ErrorMessage: err.Error(),
				Description:  "Merge conflicted. Resolve manually or use 'gitdex plan compile' to create a governed merge plan.",
			}, nil
		}
		count, _ := s.countChangedFiles(ctx, repoPath, "HEAD@{1}", "HEAD")
		return &SyncResult{
			Success:      true,
			FilesChanged: count,
			Description:  "Merge completed successfully.",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported sync action %q", rec.Action)
	}
}

func (s *Syncer) fetchUpstream(ctx context.Context, repoPath, remote string) error {
	_, err := s.executor.Run(ctx, repoPath, "fetch", remote)
	return err
}

func (s *Syncer) countChangedFiles(ctx context.Context, repoPath, fromRef, toRef string) (int, error) {
	lines, err := s.executor.RunLines(ctx, repoPath, "diff", "--name-only", fromRef+".."+toRef)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}
