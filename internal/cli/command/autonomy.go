package command

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

func newAutonomyGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autonomy",
		Short: "Define autonomy levels and control autonomous tasks",
	}
	cmd.AddCommand(newAutonomyShowCommand(flags, appFn))
	cmd.AddCommand(newAutonomyListCommand(flags, appFn))
	cmd.AddCommand(newAutonomySetCommand(flags, appFn))
	cmd.AddCommand(newAutonomyPauseCommand(flags, appFn))
	cmd.AddCommand(newAutonomyResumeCommand(flags, appFn))
	cmd.AddCommand(newAutonomyCancelCommand(flags, appFn))
	cmd.AddCommand(newAutonomyTakeoverCommand(flags, appFn))
	cmd.AddCommand(newAutonomyRunOnceCommand(flags, appFn))
	return cmd
}

func newAutonomyShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current autonomy config",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			autonomyStore := app.StorageProvider.AutonomyStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			active, err := autonomyStore.GetActiveConfig()
			if err != nil {
				return fmt.Errorf("failed to get active config: %w", err)
			}
			if active == nil {
				if clioutput.IsStructured(format) {
					return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]string{
						"status":  "no_config",
						"message": "No autonomy config configured. Run 'gitdex autonomy set --capability <cap> --level <level>' to set levels.",
					})
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No autonomy config configured. Run 'gitdex autonomy set --capability <cap> --level <level>' to set levels.")
				return nil
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, active)
			}
			return renderAutonomyConfigText(cmd.OutOrStdout(), active)
		},
	}
}

func newAutonomyListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List autonomy configs",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			autonomyStore := app.StorageProvider.AutonomyStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			configs, err := autonomyStore.ListConfigs()
			if err != nil {
				return err
			}

			active, _ := autonomyStore.GetActiveConfig()
			activeID := ""
			if active != nil {
				activeID = active.ConfigID
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"configs":   configs,
					"active_id": activeID,
				})
			}
			return renderAutonomyConfigListText(cmd.OutOrStdout(), configs, activeID)
		},
	}
}

func newAutonomySetCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set autonomy level for a capability",
		Long:  "Set the autonomy level for a capability. Use --capability and --level (manual, supervised, autonomous, full_auto).",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			autonomyStore := app.StorageProvider.AutonomyStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			capability, _ := cmd.Flags().GetString("capability")
			if capability == "" {
				return fmt.Errorf("--capability is required")
			}
			levelStr, _ := cmd.Flags().GetString("level")
			if levelStr == "" {
				return fmt.Errorf("--level is required")
			}
			level := autonomy.AutonomyLevel(strings.ToLower(levelStr))
			switch level {
			case autonomy.LevelManual, autonomy.LevelSupervised, autonomy.LevelAutonomous, autonomy.LevelFullAuto:
				// valid
			default:
				return fmt.Errorf("invalid level %q; use manual, supervised, autonomous, or full_auto", levelStr)
			}

			active, err := autonomyStore.GetActiveConfig()
			if err != nil {
				return fmt.Errorf("failed to get active config: %w", err)
			}
			var cfg *autonomy.AutonomyConfig
			if active != nil {
				cfg, err = autonomyStore.GetConfig(active.ConfigID)
				if err != nil {
					return fmt.Errorf("failed to get config %q: %w", active.ConfigID, err)
				}
			}
			if cfg == nil {
				cfg = &autonomy.AutonomyConfig{
					Name:         "default",
					DefaultLevel: autonomy.LevelManual,
				}
			}

			// Update or add capability
			found := false
			for i := range cfg.CapabilityAutonomies {
				if cfg.CapabilityAutonomies[i].Capability == capability {
					cfg.CapabilityAutonomies[i].Level = level
					found = true
					break
				}
			}
			if !found {
				cfg.CapabilityAutonomies = append(cfg.CapabilityAutonomies, autonomy.CapabilityAutonomy{
					Capability: capability,
					Level:      level,
				})
			}

			if err := autonomyStore.SaveConfig(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			if err := autonomyStore.SetActiveConfig(cfg.ConfigID); err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, cfg)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set %s to %s (config: %s)\n", capability, level, cfg.ConfigID)
			return nil
		},
	}
	cmd.Flags().String("capability", "", "Capability name (required)")
	cmd.Flags().String("level", "", "Autonomy level: manual, supervised, autonomous, full_auto (required)")
	_ = cmd.MarkFlagRequired("capability")
	_ = cmd.MarkFlagRequired("level")
	return cmd
}

func renderAutonomyConfigText(out io.Writer, cfg *autonomy.AutonomyConfig) error {
	_, _ = fmt.Fprintf(out, "═══ Active Autonomy Config ═══\n\n")
	_, _ = fmt.Fprintf(out, "Config ID:     %s\n", cfg.ConfigID)
	_, _ = fmt.Fprintf(out, "Name:          %s\n", cfg.Name)
	_, _ = fmt.Fprintf(out, "Default Level: %s\n", cfg.DefaultLevel)
	_, _ = fmt.Fprintf(out, "Created:       %s\n", cfg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))

	if len(cfg.CapabilityAutonomies) > 0 {
		_, _ = fmt.Fprintf(out, "\nCapability Levels:\n")
		for _, ca := range cfg.CapabilityAutonomies {
			_, _ = fmt.Fprintf(out, "  - %s: %s", ca.Capability, ca.Level)
			if ca.RequiresApproval {
				_, _ = fmt.Fprintf(out, " (requires approval)")
			}
			_, _ = fmt.Fprintf(out, "\n")
		}
	}
	return nil
}

func renderAutonomyConfigListText(out io.Writer, configs []*autonomy.AutonomyConfig, activeID string) error {
	if len(configs) == 0 {
		_, _ = fmt.Fprintln(out, "No autonomy configs found.")
		return nil
	}
	_, _ = fmt.Fprintf(out, "%-24s %-20s %-12s %s\n", "Config ID", "Name", "Default", "Active")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 70))
	for _, c := range configs {
		active := ""
		if c.ConfigID == activeID {
			active = "*"
		}
		_, _ = fmt.Fprintf(out, "%-24s %-20s %-12s %s\n", c.ConfigID, c.Name, c.DefaultLevel, active)
	}
	return nil
}

func newAutonomyPauseCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "pause <task_id>",
		Short: "Pause a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			taskController := autonomy.NewTaskController(app.StorageProvider.TaskStore(), app.StorageProvider.AuditLedger())
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			req := autonomy.TaskControlRequest{
				Action:    autonomy.TaskControlPause,
				TaskID:    args[0],
				Reason:    reason,
				Actor:     "cli",
				Timestamp: time.Now().UTC(),
			}
			if req.Reason == "" {
				req.Reason = "user requested"
			}

			result, err := taskController.Execute(context.Background(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderTaskControlResult(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for pausing")
	return cmd
}

func newAutonomyResumeCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "resume <task_id>",
		Short: "Resume a paused task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			taskController := autonomy.NewTaskController(app.StorageProvider.TaskStore(), app.StorageProvider.AuditLedger())
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			req := autonomy.TaskControlRequest{
				Action:    autonomy.TaskControlResume,
				TaskID:    args[0],
				Actor:     "cli",
				Timestamp: time.Now().UTC(),
			}

			result, err := taskController.Execute(context.Background(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderTaskControlResult(cmd.OutOrStdout(), result)
		},
	}
}

func newAutonomyCancelCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "cancel <task_id>",
		Short: "Cancel a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			taskController := autonomy.NewTaskController(app.StorageProvider.TaskStore(), app.StorageProvider.AuditLedger())
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			req := autonomy.TaskControlRequest{
				Action:    autonomy.TaskControlCancel,
				TaskID:    args[0],
				Reason:    reason,
				Actor:     "cli",
				Timestamp: time.Now().UTC(),
			}
			if req.Reason == "" {
				req.Reason = "user requested"
			}

			result, err := taskController.Execute(context.Background(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderTaskControlResult(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for cancelling")
	return cmd
}

func newAutonomyTakeoverCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "takeover <task_id>",
		Short: "Take over autonomous task to manual",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			taskController := autonomy.NewTaskController(app.StorageProvider.TaskStore(), app.StorageProvider.AuditLedger())
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			req := autonomy.TaskControlRequest{
				Action:    autonomy.TaskControlTakeover,
				TaskID:    args[0],
				Actor:     "cli",
				Timestamp: time.Now().UTC(),
			}

			result, err := taskController.Execute(context.Background(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderTaskControlResult(cmd.OutOrStdout(), result)
		},
	}
}

func renderTaskControlResult(out io.Writer, r *autonomy.TaskControlResult) error {
	status := "OK"
	if !r.Success {
		status = "FAILED"
	}
	_, _ = fmt.Fprintf(out, "Status: %s\n", status)
	_, _ = fmt.Fprintf(out, "Message: %s\n", r.Message)
	if r.PreviousStatus != "" {
		_, _ = fmt.Fprintf(out, "Previous status: %s\n", r.PreviousStatus)
	}
	if r.NewStatus != "" {
		_, _ = fmt.Fprintf(out, "New status: %s\n", r.NewStatus)
	}
	return nil
}
