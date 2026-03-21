package gitops

import (
	"context"
	"strconv"
	"strings"
)

type BranchInfo struct {
	Name       string
	SHA        string
	Upstream   string
	Ahead      int
	Behind     int
	LastCommit string
}

type MergeOptions struct {
	Strategy string
	FFOnly   bool
	NoFF     bool
	NoCommit bool
	Squash   bool
}

type MergeResult struct {
	Success     bool
	FastForward bool
	Conflicts   []ConflictFile
	MergeCommit string
}

type ConflictFile struct {
	Path   string
	Status string
}

type RebaseOptions struct {
	Onto       string
	Autosquash bool
}

type BranchManager struct {
	executor *GitExecutor
}

func NewBranchManager(executor *GitExecutor) *BranchManager {
	return &BranchManager{executor: executor}
}

func (bm *BranchManager) ListBranches(ctx context.Context, repoPath string, remotes bool) ([]BranchInfo, error) {
	args := []string{"branch"}
	if remotes {
		args = append(args, "--remotes")
	}
	args = append(args, "--format=%(refname:short)%x00%(objectname)%x00%(upstream:short)%x00%(upstream:track)%x00%(contents:subject)")

	lines, err := bm.executor.RunLines(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}

	var branches []BranchInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.ReplaceAll(line, "%x00", "\x00")
		parts := strings.SplitN(line, "\x00", 5)
		bi := BranchInfo{Name: sanitizeBranchField(parts[0])}
		if len(parts) > 1 {
			bi.SHA = sanitizeBranchField(parts[1])
		}
		if len(parts) > 2 && parts[2] != "" {
			bi.Upstream = sanitizeBranchField(parts[2])
		}
		if len(parts) > 3 {
			bi.Ahead, bi.Behind = parseBranchTrack(parts[3])
		}
		if len(parts) > 4 {
			bi.LastCommit = sanitizeBranchField(parts[4])
		}
		if bi.LastCommit == "" && bi.Name != "" {
			if result, err := bm.executor.Run(ctx, repoPath, "log", "-1", "--format=%s", bi.Name); err == nil {
				bi.LastCommit = sanitizeBranchField(strings.TrimSpace(result.Stdout))
			}
		}
		branches = append(branches, bi)
	}
	return branches, nil
}

func (bm *BranchManager) CreateBranch(ctx context.Context, repoPath string, name string, startPoint string) error {
	args := []string{"branch"}
	if startPoint != "" {
		args = append(args, name, startPoint)
	} else {
		args = append(args, name)
	}
	_, err := bm.executor.Run(ctx, repoPath, args...)
	return err
}

func (bm *BranchManager) DeleteBranch(ctx context.Context, repoPath string, name string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, name)
	_, err := bm.executor.Run(ctx, repoPath, args...)
	return err
}

func (bm *BranchManager) RenameBranch(ctx context.Context, repoPath string, oldName string, newName string) error {
	args := []string{"branch", "-m"}
	if oldName != "" {
		args = append(args, oldName, newName)
	} else {
		args = append(args, newName)
	}
	_, err := bm.executor.Run(ctx, repoPath, args...)
	return err
}

func (bm *BranchManager) SwitchBranch(ctx context.Context, repoPath string, branch string) error {
	_, err := bm.executor.Run(ctx, repoPath, "checkout", branch)
	return err
}

func (bm *BranchManager) MergeBranch(ctx context.Context, repoPath string, source string, opts *MergeOptions) (*MergeResult, error) {
	args := []string{"merge"}
	if opts != nil {
		if opts.FFOnly {
			args = append(args, "--ff-only")
		}
		if opts.NoFF {
			args = append(args, "--no-ff")
		}
		if opts.NoCommit {
			args = append(args, "--no-commit")
		}
		if opts.Squash {
			args = append(args, "--squash")
		}
		if opts.Strategy != "" {
			args = append(args, "-s", opts.Strategy)
		}
	}
	args = append(args, source)

	result, err := bm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		gerr, ok := err.(*GitError)
		if ok && gerr.Kind == ErrKindConflict {
			conflicts, _ := bm.getConflictFiles(ctx, repoPath)
			return &MergeResult{Success: false, Conflicts: conflicts}, nil
		}
		if ok && strings.Contains(strings.ToLower(gerr.Stderr), "conflict") {
			conflicts, _ := bm.getConflictFiles(ctx, repoPath)
			return &MergeResult{Success: false, Conflicts: conflicts}, nil
		}
		return nil, err
	}

	mergeCommit := ""
	fastForward := strings.Contains(result.Stdout, "Fast-forward") || strings.Contains(result.Stderr, "Fast-forward")
	if !fastForward {
		commitResult, _ := bm.executor.Run(ctx, repoPath, "log", "-1", "--format=%H")
		if commitResult != nil {
			mergeCommit = strings.TrimSpace(commitResult.Stdout)
		}
	}
	return &MergeResult{Success: true, FastForward: fastForward, MergeCommit: mergeCommit}, nil
}

func (bm *BranchManager) getConflictFiles(ctx context.Context, repoPath string) ([]ConflictFile, error) {
	result, err := bm.executor.Run(ctx, repoPath, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	var conflicts []ConflictFile
	for _, line := range strings.Split(result.Stdout, "\n") {
		path := strings.TrimSpace(line)
		if path != "" {
			conflicts = append(conflicts, ConflictFile{Path: path, Status: "U"})
		}
	}
	return conflicts, nil
}

func (bm *BranchManager) AbortMerge(ctx context.Context, repoPath string) error {
	_, err := bm.executor.Run(ctx, repoPath, "merge", "--abort")
	return err
}

func (bm *BranchManager) RebaseBranch(ctx context.Context, repoPath string, onto string, opts *RebaseOptions) error {
	args := []string{"rebase"}
	if opts != nil {
		if opts.Onto != "" {
			args = append(args, "--onto", opts.Onto)
		}
		if opts.Autosquash {
			args = append(args, "--autosquash")
		}
	}
	if onto != "" {
		args = append(args, onto)
	}
	_, err := bm.executor.Run(ctx, repoPath, args...)
	return err
}

func (bm *BranchManager) AbortRebase(ctx context.Context, repoPath string) error {
	_, err := bm.executor.Run(ctx, repoPath, "rebase", "--abort")
	return err
}

func (bm *BranchManager) ContinueRebase(ctx context.Context, repoPath string) error {
	_, err := bm.executor.Run(ctx, repoPath, "rebase", "--continue")
	return err
}

func (bm *BranchManager) CherryPick(ctx context.Context, repoPath string, commit string) error {
	_, err := bm.executor.Run(ctx, repoPath, "cherry-pick", commit)
	return err
}

func (bm *BranchManager) AbortCherryPick(ctx context.Context, repoPath string) error {
	_, err := bm.executor.Run(ctx, repoPath, "cherry-pick", "--abort")
	return err
}

func (bm *BranchManager) ListTags(ctx context.Context, repoPath string) ([]string, error) {
	lines, err := bm.executor.RunLines(ctx, repoPath, "tag", "-l")
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, line := range lines {
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

func (bm *BranchManager) CreateTag(ctx context.Context, repoPath string, name string, object string, annotate bool, message string) error {
	args := []string{"tag"}
	if annotate {
		args = append(args, "-a")
		if message != "" {
			args = append(args, "-m", message)
		}
	}
	args = append(args, name)
	if object != "" {
		args = append(args, object)
	}
	_, err := bm.executor.Run(ctx, repoPath, args...)
	return err
}

func (bm *BranchManager) DeleteTag(ctx context.Context, repoPath string, name string) error {
	_, err := bm.executor.Run(ctx, repoPath, "tag", "-d", name)
	return err
}

func (bm *BranchManager) MergeBase(ctx context.Context, repoPath string, a string, b string) (string, error) {
	result, err := bm.executor.Run(ctx, repoPath, "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

func sanitizeBranchField(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\x00' || r < 0x20 && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	out = strings.ReplaceAll(out, "%00", "")
	out = strings.ReplaceAll(out, "%x00", "")
	return strings.TrimSpace(out)
}

func parseBranchTrack(s string) (ahead int, behind int) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" || s == "gone" {
		return 0, 0
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "ahead "):
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(part, "ahead "))); err == nil {
				ahead = n
			}
		case strings.HasPrefix(part, "behind "):
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(part, "behind "))); err == nil {
				behind = n
			}
		}
	}
	return ahead, behind
}
