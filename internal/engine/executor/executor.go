package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
)

// CommandExecutor executes git commands via GitCLI.
type CommandExecutor struct {
	cli cli.GitCLI
}

// New returns a new CommandExecutor.
func New(gitCLI cli.GitCLI) *CommandExecutor {
	return &CommandExecutor{cli: gitCLI}
}

// NewCommandExecutor is an alias for New.
func NewCommandExecutor(gitCLI cli.GitCLI) *CommandExecutor {
	return New(gitCLI)
}

// Commit runs `git commit -m <msg>`.
func (e *CommandExecutor) Commit(ctx context.Context, msg string) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "commit", "-m", msg)
	return &git.ExecutionResult{
		Command: []string{"git", "commit", "-m", msg},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// Push runs `git push`.
func (e *CommandExecutor) Push(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "push")
	return &git.ExecutionResult{
		Command: []string{"git", "push"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// Execute runs each command via GitCLI and returns a combined result.
// Each command string is parsed as space-separated args (leading "git" is stripped if present).
func (e *CommandExecutor) Execute(ctx context.Context, commands []string) (*git.ExecutionResult, error) {
	if len(commands) == 0 {
		return &git.ExecutionResult{Success: true}, nil
	}
	var combinedStdout, combinedStderr strings.Builder
	var allCommands []string
	success := true
	exitCode := 0
	for _, cmd := range commands {
		args := parseCommand(cmd)
		if len(args) == 0 {
			continue
		}
		args = sanitizeArgs(append([]string{}, args...))
		fullCmd := append([]string{"git"}, args...)
		allCommands = append(allCommands, strings.Join(fullCmd, " "))
		stdout, stderr, err := e.cli.Exec(ctx, args...)
		combinedStdout.WriteString(stdout)
		if stdout != "" && !strings.HasSuffix(stdout, "\n") {
			combinedStdout.WriteString("\n")
		}
		combinedStderr.WriteString(stderr)
		if stderr != "" && !strings.HasSuffix(stderr, "\n") {
			combinedStderr.WriteString("\n")
		}
		if err != nil {
			success = false
			exitCode = extractExitCode(err)
		}
	}
	return &git.ExecutionResult{
		Command:  allCommands,
		Stdout:   strings.TrimSuffix(combinedStdout.String(), "\n"),
		Stderr:   strings.TrimSuffix(combinedStderr.String(), "\n"),
		ExitCode: exitCode,
		Success:  success,
	}, nil
}

// ExecuteTokenized runs a pre-tokenized command (e.g. ["git","commit","-m","msg with spaces"]).
// This preserves argument boundaries and handles spaces in args correctly.
func (e *CommandExecutor) ExecuteTokenized(ctx context.Context, tokens []string) (*git.ExecutionResult, error) {
	if len(tokens) == 0 {
		return &git.ExecutionResult{Success: true}, nil
	}
	args := sanitizeArgs(tokens)
	if len(args) > 0 && strings.EqualFold(args[0], "git") {
		args = args[1:]
	}
	if len(args) == 0 {
		return &git.ExecutionResult{Success: true}, nil
	}
	stdout, stderr, err := e.cli.Exec(ctx, args...)
	exitCode := 0
	if err != nil {
		exitCode = extractExitCode(err)
	}
	return &git.ExecutionResult{
		Command:  tokens,
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Success:  err == nil,
	}, err
}

// sanitizeArgs strips null bytes and control characters from arguments
// to prevent Windows CreateProcess "invalid argument" errors.
func sanitizeArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = sanitizeArg(a)
	}
	return out
}

func sanitizeArg(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == 0 {
			continue
		}
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func extractExitCode(err error) int {
	if err == nil {
		return 0
	}
	s := err.Error()
	if strings.HasPrefix(s, "exit status ") {
		code := 0
		if _, scanErr := fmt.Sscanf(s, "exit status %d", &code); scanErr == nil {
			return code
		}
	}
	return 1
}

// parseCommand splits a command string into args, dropping leading "git" if present.
func parseCommand(cmd string) []string {
	parts := shellSplit(cmd)
	if len(parts) > 0 && strings.EqualFold(parts[0], "git") {
		parts = parts[1:]
	}
	return parts
}

// shellSplit splits a command string respecting double and single quotes.
func shellSplit(s string) []string {
	var args []string
	var current strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == ' ' && !inSingle && !inDouble:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// StageAll runs `git add .`
func (e *CommandExecutor) StageAll(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "add", ".")
	return &git.ExecutionResult{
		Command: []string{"git", "add", "."},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// Init runs `git init` in the given directory (or current dir if empty).
func (e *CommandExecutor) Init(ctx context.Context, dir string) (*git.ExecutionResult, error) {
	args := []string{"init"}
	if dir != "" {
		args = append(args, dir)
	}
	stdout, stderr, err := e.cli.Exec(ctx, args...)
	return &git.ExecutionResult{
		Command: append([]string{"git"}, args...),
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// StageFiles runs `git add <files...>`
func (e *CommandExecutor) StageFiles(ctx context.Context, files []string) (*git.ExecutionResult, error) {
	if len(files) == 0 {
		return &git.ExecutionResult{Success: true}, nil
	}
	args := append([]string{"add"}, files...)
	stdout, stderr, err := e.cli.Exec(ctx, args...)
	fullCmd := append([]string{"git"}, args...)
	return &git.ExecutionResult{
		Command: fullCmd,
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// CreateTag runs `git tag -a <name> -m <message>`.
func (e *CommandExecutor) CreateTag(ctx context.Context, name, message string) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "tag", "-a", name, "-m", message)
	return &git.ExecutionResult{
		Command: []string{"git", "tag", "-a", name, "-m", message},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// ListTags runs `git tag -l --sort=-v:refname` and returns tag names.
func (e *CommandExecutor) ListTags(ctx context.Context) ([]string, error) {
	stdout, _, err := e.cli.Exec(ctx, "tag", "-l", "--sort=-v:refname")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	var tags []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

// DeleteTag runs `git tag -d <name>`.
func (e *CommandExecutor) DeleteTag(ctx context.Context, name string) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "tag", "-d", name)
	return &git.ExecutionResult{
		Command: []string{"git", "tag", "-d", name},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// DiffStaged runs `git diff --staged` and returns the diff output.
func (e *CommandExecutor) DiffStaged(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "diff", "--staged")
	return &git.ExecutionResult{
		Command: []string{"git", "diff", "--staged"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// DiffStagedStat returns a short stat summary of staged changes.
func (e *CommandExecutor) DiffStagedStat(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "diff", "--staged", "--stat")
	return &git.ExecutionResult{
		Command: []string{"git", "diff", "--staged", "--stat"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// RestoreStaged unstages files: git restore --staged <files...>
func (e *CommandExecutor) RestoreStaged(ctx context.Context, files []string) (*git.ExecutionResult, error) {
	if len(files) == 0 {
		return &git.ExecutionResult{Success: true}, nil
	}
	args := append([]string{"restore", "--staged"}, files...)
	stdout, stderr, err := e.cli.Exec(ctx, args...)
	return &git.ExecutionResult{
		Command: append([]string{"git"}, args...),
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// RestoreWorktree discards working tree changes: git restore <files...>
func (e *CommandExecutor) RestoreWorktree(ctx context.Context, files []string) (*git.ExecutionResult, error) {
	if len(files) == 0 {
		return &git.ExecutionResult{Success: true}, nil
	}
	args := append([]string{"restore"}, files...)
	stdout, stderr, err := e.cli.Exec(ctx, args...)
	return &git.ExecutionResult{
		Command: append([]string{"git"}, args...),
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// MergeAbort runs `git merge --abort`.
func (e *CommandExecutor) MergeAbort(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "merge", "--abort")
	return &git.ExecutionResult{
		Command: []string{"git", "merge", "--abort"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// MergeContinue runs `git merge --continue`.
func (e *CommandExecutor) MergeContinue(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "merge", "--continue")
	return &git.ExecutionResult{
		Command: []string{"git", "merge", "--continue"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// RebaseAbort runs `git rebase --abort`.
func (e *CommandExecutor) RebaseAbort(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "rebase", "--abort")
	return &git.ExecutionResult{
		Command: []string{"git", "rebase", "--abort"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// RebaseContinue runs `git rebase --continue`.
func (e *CommandExecutor) RebaseContinue(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "rebase", "--continue")
	return &git.ExecutionResult{
		Command: []string{"git", "rebase", "--continue"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// CherryPickAbort runs `git cherry-pick --abort`.
func (e *CommandExecutor) CherryPickAbort(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "cherry-pick", "--abort")
	return &git.ExecutionResult{
		Command: []string{"git", "cherry-pick", "--abort"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// CherryPickContinue runs `git cherry-pick --continue`.
func (e *CommandExecutor) CherryPickContinue(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "cherry-pick", "--continue")
	return &git.ExecutionResult{
		Command: []string{"git", "cherry-pick", "--continue"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// ResetSoft runs `git reset --soft <target>`.
func (e *CommandExecutor) ResetSoft(ctx context.Context, target string) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "reset", "--soft", target)
	return &git.ExecutionResult{
		Command: []string{"git", "reset", "--soft", target},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// Revert runs `git revert <commit>`.
func (e *CommandExecutor) Revert(ctx context.Context, commit string) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "revert", commit)
	return &git.ExecutionResult{
		Command: []string{"git", "revert", commit},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// DeleteBranch runs `git branch -d <name>`.
func (e *CommandExecutor) DeleteBranch(ctx context.Context, name string) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "branch", "-d", name)
	return &git.ExecutionResult{
		Command: []string{"git", "branch", "-d", name},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// MergedBranches runs `git branch --merged` and returns branch names.
func (e *CommandExecutor) MergedBranches(ctx context.Context) ([]string, error) {
	stdout, _, err := e.cli.Exec(ctx, "branch", "--merged")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		name := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if name != "" {
			branches = append(branches, name)
		}
	}
	return branches, nil
}

// RemotePrune runs `git remote prune <remote>`.
func (e *CommandExecutor) RemotePrune(ctx context.Context, remote string) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "remote", "prune", remote)
	return &git.ExecutionResult{
		Command: []string{"git", "remote", "prune", remote},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// Reflog returns recent reflog entries.
func (e *CommandExecutor) Reflog(ctx context.Context, count int) (*git.ExecutionResult, error) {
	args := []string{"reflog", "--oneline", fmt.Sprintf("-n%d", count)}
	stdout, stderr, err := e.cli.Exec(ctx, args...)
	return &git.ExecutionResult{
		Command: append([]string{"git"}, args...),
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// CommitAmend runs `git commit --amend --no-edit`.
func (e *CommandExecutor) CommitAmend(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "commit", "--amend", "--no-edit")
	return &git.ExecutionResult{
		Command: []string{"git", "commit", "--amend", "--no-edit"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// Stash runs `git stash push -m <message>`.
func (e *CommandExecutor) Stash(ctx context.Context, message string) (*git.ExecutionResult, error) {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}
	stdout, stderr, err := e.cli.Exec(ctx, args...)
	return &git.ExecutionResult{
		Command: append([]string{"git"}, args...),
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// StashPop runs `git stash pop`.
func (e *CommandExecutor) StashPop(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "stash", "pop")
	return &git.ExecutionResult{
		Command: []string{"git", "stash", "pop"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}

// FetchPrune runs `git fetch --prune`.
func (e *CommandExecutor) FetchPrune(ctx context.Context) (*git.ExecutionResult, error) {
	stdout, stderr, err := e.cli.Exec(ctx, "fetch", "--prune")
	return &git.ExecutionResult{
		Command: []string{"git", "fetch", "--prune"},
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}, err
}
