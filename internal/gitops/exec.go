package gitops

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type GitResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

type GitExecutor struct {
	gitBinary  string
	defaultEnv []string
	timeout    time.Duration
}

func NewGitExecutor() *GitExecutor {
	return &GitExecutor{
		gitBinary: "git",
		defaultEnv: []string{
			"GIT_TERMINAL_PROMPT=0",
			"GIT_ASKPASS=echo",
			"LANG=C",
		},
		timeout: 60 * time.Second,
	}
}

func NewGitExecutorWithConfig(gitBinary string, timeout time.Duration) *GitExecutor {
	e := NewGitExecutor()
	if gitBinary != "" {
		e.gitBinary = gitBinary
	}
	if timeout > 0 {
		e.timeout = timeout
	}
	return e
}

func (e *GitExecutor) Run(ctx context.Context, repoPath string, args ...string) (*GitResult, error) {
	return e.run(ctx, repoPath, nil, args...)
}

func (e *GitExecutor) RunWithInput(ctx context.Context, repoPath string, stdin io.Reader, args ...string) (*GitResult, error) {
	return e.run(ctx, repoPath, stdin, args...)
}

func (e *GitExecutor) run(ctx context.Context, repoPath string, stdin io.Reader, args ...string) (*GitResult, error) {
	runCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(runCtx, e.gitBinary, args...)
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	cmd.Env = append(os.Environ(), e.defaultEnv...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if stdin != nil {
		cmd.Stdin = stdin
	}

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Context cancelled or other error
			if runCtx.Err() != nil {
				return nil, runCtx.Err()
			}
			return nil, err
		}
		kind := ClassifyGitError(stderr, exitCode)
		return nil, &GitError{
			Command:  e.gitBinary,
			Args:     args,
			ExitCode: exitCode,
			Stderr:   stderr,
			Kind:     kind,
		}
	}

	return &GitResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

func (e *GitExecutor) RunLines(ctx context.Context, repoPath string, args ...string) ([]string, error) {
	result, err := e.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(result.Stdout, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.TrimSpace(line))
	}
	return out, nil
}
