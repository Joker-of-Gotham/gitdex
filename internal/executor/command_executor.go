package executor

import (
	"context"
	"os/exec"
)

// CommandExecutor wraps process execution so runtime checks are injectable and testable.
type CommandExecutor interface {
	CombinedOutput(ctx context.Context, binary string, args []string, dir string) ([]byte, error)
}

type osCommandExecutor struct{}

func (osCommandExecutor) CombinedOutput(ctx context.Context, binary string, args []string, dir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}
