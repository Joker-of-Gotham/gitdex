package executor

import (
	"context"

	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

// GitAdapter abstracts git command execution from runtime orchestration.
type GitAdapter interface {
	ExecGit(ctx context.Context, cmd string) *ExecutionResult
}

// GitHubAdapter abstracts gh command execution from runtime orchestration.
type GitHubAdapter interface {
	ExecGitHub(ctx context.Context, cmd string) *ExecutionResult
}

// FileAdapter abstracts file read/write operations from runtime orchestration.
type FileAdapter interface {
	Write(action planner.ActionSpec) *ExecutionResult
	Read(action planner.ActionSpec) *ExecutionResult
}

type runnerGitAdapter struct{ runner *Runner }

func (a runnerGitAdapter) ExecGit(ctx context.Context, cmd string) *ExecutionResult {
	return a.runner.execGitCommand(ctx, cmd)
}

type runnerGitHubAdapter struct{ runner *Runner }

func (a runnerGitHubAdapter) ExecGitHub(ctx context.Context, cmd string) *ExecutionResult {
	return a.runner.execGitHubOp(ctx, cmd)
}

type runnerFileAdapter struct{ runner *Runner }

func (a runnerFileAdapter) Write(action planner.ActionSpec) *ExecutionResult {
	return a.runner.execFileWrite(action)
}

func (a runnerFileAdapter) Read(action planner.ActionSpec) *ExecutionResult {
	return a.runner.execFileRead(action)
}

