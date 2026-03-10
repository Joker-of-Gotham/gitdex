package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitCLI defines the interface for executing git commands.
type GitCLI interface {
	Exec(ctx context.Context, args ...string) (stdout, stderr string, err error)
	ExecStream(ctx context.Context, args ...string) (<-chan string, error)
	Version() (string, error)
}

// CLIExecutor is a concrete implementation of GitCLI.
type CLIExecutor struct {
	gitPath string
}

// NewCLIExecutor creates a new CLIExecutor by locating the git binary.
func NewCLIExecutor() (*CLIExecutor, error) {
	path, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git not found: %w", err)
	}
	return &CLIExecutor{gitPath: path}, nil
}

// Exec runs a git command with the given args and returns stdout, stderr, and error.
func (c *CLIExecutor) Exec(ctx context.Context, args ...string) (string, string, error) {
	// #nosec G204 -- git path is resolved locally and arguments are intentional Git CLI tokens.
	cmd := exec.CommandContext(ctx, c.gitPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ExecStream runs a git command and returns output as a channel.
func (c *CLIExecutor) ExecStream(ctx context.Context, args ...string) (<-chan string, error) {
	// #nosec G204 -- git path is resolved locally and arguments are intentional Git CLI tokens.
	cmd := exec.CommandContext(ctx, c.gitPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("exec stream: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("exec stream: start: %w", err)
	}
	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case ch <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
		if waitErr := cmd.Wait(); waitErr != nil && ctx.Err() == nil {
			return
		}
	}()
	return ch, nil
}

// Version returns the git version string.
func (c *CLIExecutor) Version() (string, error) {
	stdout, _, err := c.Exec(context.Background(), "version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout), nil
}
