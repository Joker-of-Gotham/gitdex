package gitops

import (
	"fmt"
	"strings"
)

type GitError struct {
	Command  string
	Args     []string
	ExitCode int
	Stderr   string
	Kind     GitErrorKind
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %s failed (exit %d, %s): %s", e.Command, e.ExitCode, e.Kind, e.Stderr)
}

type GitErrorKind string

const (
	ErrKindConflict    GitErrorKind = "conflict"
	ErrKindAuth        GitErrorKind = "auth_failed"
	ErrKindLock        GitErrorKind = "lock_failed"
	ErrKindRefNotFound GitErrorKind = "ref_not_found"
	ErrKindDirtyTree   GitErrorKind = "dirty_worktree"
	ErrKindNotARepo    GitErrorKind = "not_a_repo"
	ErrKindUnknown     GitErrorKind = "unknown"
)

func ClassifyGitError(stderr string, exitCode int) GitErrorKind {
	s := strings.ToLower(stderr)

	if strings.Contains(s, "conflict") || strings.Contains(s, "merge conflict") {
		return ErrKindConflict
	}
	if strings.Contains(s, "authentication failed") || strings.Contains(s, "could not read username") {
		return ErrKindAuth
	}
	if strings.Contains(s, "unable to create") && strings.Contains(s, ".lock") {
		return ErrKindLock
	}
	if strings.Contains(s, "not a git repository") {
		return ErrKindNotARepo
	}
	if (strings.Contains(s, "pathspec") && strings.Contains(s, "did not match")) ||
		strings.Contains(s, "unknown revision") {
		return ErrKindRefNotFound
	}

	return ErrKindUnknown
}
