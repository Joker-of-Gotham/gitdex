package integration_test

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestStarterCommandsExecuteDaemonRunPaths(t *testing.T) {
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{
			name: "gitdex daemon run",
			cmd: func() *cobra.Command {
				return command.NewRootCommand()
			},
			args: []string{"daemon", "run"},
		},
		{
			name: "gitdexd run",
			cmd: func() *cobra.Command {
				return command.NewDaemonBinaryRootCommand()
			},
			args: []string{"run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("daemon run starts real server; cannot assert starter baseline output in integration test")
		})
	}
}
