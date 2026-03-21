package command

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/emergency"
)

var controlEngineOverride emergency.ControlEngine

func SetControlEngineForTest(engine emergency.ControlEngine) func() {
	prev := controlEngineOverride
	controlEngineOverride = engine
	return func() { controlEngineOverride = prev }
}

func getControlEngine(app bootstrap.App) emergency.ControlEngine {
	if controlEngineOverride != nil {
		return controlEngineOverride
	}
	return emergency.NewControlEngine(
		app.StorageProvider.TaskStore(),
		app.StorageProvider.AuditLedger(),
		autonomy.NewTaskController(app.StorageProvider.TaskStore(), app.StorageProvider.AuditLedger()),
	)
}

func newEmergencyGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emergency",
		Short: "Trigger emergency controls and containment",
	}
	cmd.AddCommand(newEmergencyPauseCommand(flags, appFn))
	cmd.AddCommand(newEmergencySuspendCommand(flags, appFn))
	cmd.AddCommand(newEmergencyKillCommand(flags, appFn))
	cmd.AddCommand(newEmergencyStatusCommand(flags, appFn))
	return cmd
}

func newEmergencyPauseCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "pause [task_id]",
		Short: "Pause a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			req := emergency.ControlRequest{
				Action:    emergency.ControlPauseTask,
				Scope:     args[0],
				Reason:    "user requested",
				Actor:     "cli",
				Timestamp: time.Now().UTC(),
			}

			result, err := getControlEngine(app).Execute(req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderControlResult(cmd.OutOrStdout(), result)
		},
	}
}

func newEmergencySuspendCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "suspend [scope]",
		Short: "Suspend a scope",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			req := emergency.ControlRequest{
				Action:    emergency.ControlSuspendCapability,
				Scope:     args[0],
				Reason:    "user requested",
				Actor:     "cli",
				Timestamp: time.Now().UTC(),
			}

			result, err := getControlEngine(app).Execute(req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderControlResult(cmd.OutOrStdout(), result)
		},
	}
}

func newEmergencyKillCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "kill",
		Short: "Kill switch for all tasks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			req := emergency.ControlRequest{
				Action:    emergency.ControlKillSwitch,
				Scope:     "*",
				Reason:    "user requested kill switch",
				Actor:     "cli",
				Timestamp: time.Now().UTC(),
			}

			result, err := getControlEngine(app).Execute(req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderControlResult(cmd.OutOrStdout(), result)
		},
	}
}

func newEmergencyStatusCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current emergency controls",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			engine, ok := getControlEngine(app).(*emergency.DefaultControlEngine)
			if !ok {
				return fmt.Errorf("control engine does not support status")
			}
			controls := engine.Status()

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"active_controls": controls,
				})
			}
			return renderEmergencyStatus(cmd.OutOrStdout(), controls)
		},
	}
}

func renderControlResult(out io.Writer, r *emergency.ControlResult) error {
	status := "OK"
	if !r.Success {
		status = "FAILED"
	}
	_, _ = fmt.Fprintf(out, "Status: %s\n", status)
	_, _ = fmt.Fprintf(out, "Message: %s\n", r.Message)
	if len(r.AffectedTasks) > 0 {
		_, _ = fmt.Fprintf(out, "Affected tasks: %s\n", strings.Join(r.AffectedTasks, ", "))
	}
	if len(r.AffectedScopes) > 0 {
		_, _ = fmt.Fprintf(out, "Affected scopes: %s\n", strings.Join(r.AffectedScopes, ", "))
	}
	return nil
}

func renderEmergencyStatus(out io.Writer, controls []emergency.ControlRequest) error {
	if len(controls) == 0 {
		_, _ = fmt.Fprintln(out, "No active emergency controls.")
		return nil
	}

	_, _ = fmt.Fprintln(out, "Active emergency controls:")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 60))
	for _, c := range controls {
		_, _ = fmt.Fprintf(out, "  %s | scope=%s | actor=%s | %s\n",
			c.Action, c.Scope, c.Actor, c.Timestamp.Format(time.RFC3339))
	}
	return nil
}
