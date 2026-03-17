package executor

import (
	"context"
	"os/exec"
	"strings"
)

// CmdObj is a builder for constructing and executing OS commands.
// Inspired by lazygit's oscommands.CmdObj pattern.
type CmdObj struct {
	binary  string
	args    []string
	workDir string
	env     []string
	dontLog bool
	stream  bool
}

// NewCmdObj creates a CmdObj with the given binary and arguments.
func NewCmdObj(binary string, args ...string) *CmdObj {
	return &CmdObj{
		binary: binary,
		args:   args,
	}
}

// SetWd sets the working directory for the command.
func (c *CmdObj) SetWd(dir string) *CmdObj {
	c.workDir = dir
	return c
}

// AddEnv adds an environment variable to the command.
func (c *CmdObj) AddEnv(key, value string) *CmdObj {
	c.env = append(c.env, key+"="+value)
	return c
}

// AddEnvSlice appends multiple environment variables.
func (c *CmdObj) AddEnvSlice(envs []string) *CmdObj {
	c.env = append(c.env, envs...)
	return c
}

// DontLog marks this command as not to be logged (for sensitive operations).
func (c *CmdObj) DontLog() *CmdObj {
	c.dontLog = true
	return c
}

// StreamOutput marks this command for streaming output.
func (c *CmdObj) StreamOutput() *CmdObj {
	c.stream = true
	return c
}

// ShouldLog returns true if the command should be logged.
func (c *CmdObj) ShouldLog() bool { return !c.dontLog }

// String returns a human-readable representation of the command.
func (c *CmdObj) String() string {
	parts := make([]string, 0, 1+len(c.args))
	parts = append(parts, c.binary)
	parts = append(parts, c.args...)
	return strings.Join(parts, " ")
}

// Run executes the command and returns stdout, stderr, and any error.
func (c *CmdObj) Run(ctx context.Context) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, c.binary, c.args...)
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}
	if len(c.env) > 0 {
		cmd.Env = append(cmd.Environ(), c.env...)
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// CombinedRun executes and returns combined output.
func (c *CmdObj) CombinedRun(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, c.binary, c.args...)
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}
	if len(c.env) > 0 {
		cmd.Env = append(cmd.Environ(), c.env...)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
