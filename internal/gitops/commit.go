package gitops

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type CommitOptions struct {
	AllowEmpty bool
	Author     string
	Date       string
	Signoff    bool
	NoVerify   bool
	GPGSign    bool
}

type CommitResult struct {
	SHA     string
	Short   string
	Summary string
	Author  string
	Date    time.Time
}

type StashEntry struct {
	Index   int
	Message string
	SHA     string
}

type CommitManager struct {
	executor *GitExecutor
}

func NewCommitManager(executor *GitExecutor) *CommitManager {
	return &CommitManager{executor: executor}
}

func (cm *CommitManager) Add(ctx context.Context, repoPath string, paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) AddAll(ctx context.Context, repoPath string) error {
	_, err := cm.executor.Run(ctx, repoPath, "add", "-A")
	return err
}

func (cm *CommitManager) Reset(ctx context.Context, repoPath string, paths ...string) error {
	args := append([]string{"reset"}, paths...)
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) ResetHard(ctx context.Context, repoPath string, revision string) error {
	args := []string{"reset", "--hard"}
	if revision != "" {
		args = append(args, revision)
	}
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) ResetSoft(ctx context.Context, repoPath string, revision string) error {
	args := []string{"reset", "--soft"}
	if revision != "" {
		args = append(args, revision)
	}
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) Restore(ctx context.Context, repoPath string, paths ...string) error {
	args := append([]string{"restore"}, paths...)
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) RestoreStaged(ctx context.Context, repoPath string, paths ...string) error {
	args := append([]string{"restore", "--staged"}, paths...)
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) Remove(ctx context.Context, repoPath string, paths ...string) error {
	args := append([]string{"rm"}, paths...)
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) Move(ctx context.Context, repoPath string, source string, dest string) error {
	_, err := cm.executor.Run(ctx, repoPath, "mv", source, dest)
	return err
}

func (cm *CommitManager) Commit(ctx context.Context, repoPath string, message string, opts *CommitOptions) (*CommitResult, error) {
	args := []string{"commit", "-m", message}
	if opts != nil {
		if opts.AllowEmpty {
			args = append(args, "--allow-empty")
		}
		if opts.Author != "" {
			args = append(args, "--author", opts.Author)
		}
		if opts.Date != "" {
			args = append(args, "--date", opts.Date)
		}
		if opts.Signoff {
			args = append(args, "--signoff")
		}
		if opts.NoVerify {
			args = append(args, "--no-verify")
		}
		if opts.GPGSign {
			args = append(args, "-S")
		}
	}
	_, err := cm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	return cm.getLastCommit(ctx, repoPath)
}

func (cm *CommitManager) CommitAmend(ctx context.Context, repoPath string, message string, opts *CommitOptions) (*CommitResult, error) {
	args := []string{"commit", "--amend"}
	if message != "" {
		args = append(args, "-m", message)
	}
	if opts != nil {
		if opts.AllowEmpty {
			args = append(args, "--allow-empty")
		}
		if opts.Author != "" {
			args = append(args, "--author", opts.Author)
		}
		if opts.Date != "" {
			args = append(args, "--date", opts.Date)
		}
		if opts.NoVerify {
			args = append(args, "--no-verify")
		}
		if opts.GPGSign {
			args = append(args, "-S")
		}
	}
	_, err := cm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	return cm.getLastCommit(ctx, repoPath)
}

func (cm *CommitManager) getLastCommit(ctx context.Context, repoPath string) (*CommitResult, error) {
	format := "%H%x00%h%x00%an%x00%aI%x00%s"
	result, err := cm.executor.Run(ctx, repoPath, "log", "-1", "--format="+format)
	if err != nil {
		return nil, err
	}
	return cm.parseCommitLine(result.Stdout), nil
}

func (cm *CommitManager) parseCommitLine(output string) *CommitResult {
	parts := strings.SplitN(strings.TrimSpace(output), "\x00", 5)
	cr := &CommitResult{}
	if len(parts) >= 1 {
		cr.SHA = parts[0]
	}
	if len(parts) >= 2 {
		cr.Short = parts[1]
	}
	if len(parts) >= 3 {
		cr.Author = parts[2]
	}
	if len(parts) >= 4 {
		if t, err := time.Parse(time.RFC3339, parts[3]); err == nil {
			cr.Date = t
		}
	}
	if len(parts) >= 5 {
		cr.Summary = parts[4]
	}
	return cr
}

func (cm *CommitManager) Revert(ctx context.Context, repoPath string, commit string, opts *CommitOptions) error {
	args := []string{"revert"}
	if opts != nil && opts.NoVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, commit)
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) AbortRevert(ctx context.Context, repoPath string) error {
	_, err := cm.executor.Run(ctx, repoPath, "revert", "--abort")
	return err
}

func (cm *CommitManager) StashPush(ctx context.Context, repoPath string, message string, includeUntracked bool) error {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}
	if includeUntracked {
		args = append(args, "-u")
	}
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) StashPop(ctx context.Context, repoPath string, index int) error {
	args := []string{"stash", "pop"}
	if index >= 0 {
		args = append(args, "stash@{"+strconv.Itoa(index)+"}")
	}
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) StashApply(ctx context.Context, repoPath string, index int) error {
	args := []string{"stash", "apply"}
	if index >= 0 {
		args = append(args, "stash@{"+strconv.Itoa(index)+"}")
	}
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) StashDrop(ctx context.Context, repoPath string, index int) error {
	args := []string{"stash", "drop"}
	if index >= 0 {
		args = append(args, "stash@{"+strconv.Itoa(index)+"}")
	}
	_, err := cm.executor.Run(ctx, repoPath, args...)
	return err
}

func (cm *CommitManager) StashList(ctx context.Context, repoPath string) ([]StashEntry, error) {
	format := "%gd%x00%s%x00%H"
	lines, err := cm.executor.RunLines(ctx, repoPath, "stash", "list", "--format="+format)
	if err != nil {
		return nil, err
	}
	var entries []StashEntry
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 3)
		e := StashEntry{Index: i}
		if len(parts) >= 1 {
			ref := parts[0]
			if idx := strings.TrimSuffix(strings.TrimPrefix(ref, "stash@{"), "}"); idx != ref {
				if n, err := strconv.Atoi(idx); err == nil {
					e.Index = n
				}
			}
		}
		if len(parts) >= 2 {
			e.Message = parts[1]
		}
		if len(parts) >= 3 {
			e.SHA = parts[2]
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (cm *CommitManager) StashShow(ctx context.Context, repoPath string, index int) (string, error) {
	args := []string{"stash", "show", "-p"}
	if index >= 0 {
		args = append(args, "stash@{"+strconv.Itoa(index)+"}")
	}
	result, err := cm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func (cm *CommitManager) StashClear(ctx context.Context, repoPath string) error {
	_, err := cm.executor.Run(ctx, repoPath, "stash", "clear")
	return err
}
