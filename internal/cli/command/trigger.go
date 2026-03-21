package command

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

var triggerStoreOverride autonomy.TriggerStore

// SetTriggerStoreForTest allows integration tests to inject a custom store.
func SetTriggerStoreForTest(s autonomy.TriggerStore) func() {
	prev := triggerStoreOverride
	triggerStoreOverride = s
	return func() { triggerStoreOverride = prev }
}

func getTriggerStore(appFn func() bootstrap.App) autonomy.TriggerStore {
	if triggerStoreOverride != nil {
		return triggerStoreOverride
	}
	return appFn().StorageProvider.TriggerStore()
}

func newTriggerGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Start governed tasks from events, schedules, APIs, or operators",
	}
	cmd.AddCommand(newTriggerAddCommand(flags, appFn))
	cmd.AddCommand(newTriggerListCommand(flags, appFn))
	cmd.AddCommand(newTriggerEnableCommand(flags, appFn))
	cmd.AddCommand(newTriggerDisableCommand(flags, appFn))
	cmd.AddCommand(newTriggerEventsCommand(flags, appFn))
	cmd.AddCommand(newTriggerFireCommand(flags, appFn))
	return cmd
}

func newTriggerAddCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a trigger",
		Long:  "Add a trigger. Use --type (event|schedule|api|operator), --name, --pattern (cron for schedule), --action.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			triggerStore := getTriggerStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			typStr, _ := cmd.Flags().GetString("type")
			if typStr == "" {
				return fmt.Errorf("--type is required (event, schedule, api, operator)")
			}
			typ := autonomy.TriggerType(strings.ToLower(typStr))
			switch typ {
			case autonomy.TriggerTypeEvent, autonomy.TriggerSchedule, autonomy.TriggerAPI, autonomy.TriggerOperator:
				// valid
			default:
				return fmt.Errorf("invalid type %q; use event, schedule, api, or operator", typStr)
			}

			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			pattern, _ := cmd.Flags().GetString("pattern")
			action, _ := cmd.Flags().GetString("action")
			source, _ := cmd.Flags().GetString("source")

			cfg := &autonomy.TriggerConfig{
				TriggerType:    typ,
				Name:           name,
				Source:         source,
				Pattern:        pattern,
				ActionTemplate: action,
				Enabled:        true,
			}
			if err := triggerStore.SaveTrigger(cfg); err != nil {
				return fmt.Errorf("failed to add trigger: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, cfg)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Trigger added: %s (id: %s, type: %s)\n", name, cfg.TriggerID, typ)
			return nil
		},
	}
	cmd.Flags().String("type", "", "Trigger type: event, schedule, api, operator (required)")
	cmd.Flags().String("name", "", "Trigger name (required)")
	cmd.Flags().String("pattern", "", "Cron expression or event pattern")
	cmd.Flags().String("action", "", "Action template (e.g. repo sync)")
	cmd.Flags().String("source", "", "Source identifier")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newTriggerListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List triggers",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			triggerStore := getTriggerStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			configs, err := triggerStore.ListTriggers()
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"triggers": configs})
			}
			return renderTriggerListText(cmd.OutOrStdout(), configs)
		},
	}
}

func newTriggerEnableCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <trigger_id>",
		Short: "Enable a trigger",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			triggerStore := getTriggerStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			triggerID := args[0]
			if err := triggerStore.EnableTrigger(triggerID); err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]string{"status": "enabled", "trigger_id": triggerID})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Trigger %s enabled.\n", triggerID)
			return nil
		},
	}
}

func newTriggerDisableCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "disable <trigger_id>",
		Short: "Disable a trigger",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			triggerStore := getTriggerStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			triggerID := args[0]
			if err := triggerStore.DisableTrigger(triggerID); err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]string{"status": "disabled", "trigger_id": triggerID})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Trigger %s disabled.\n", triggerID)
			return nil
		},
	}
}

func newTriggerEventsCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var limit int
	var triggerID string
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Show trigger history",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			triggerStore := getTriggerStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			if limit <= 0 {
				limit = 20
			}

			events, err := triggerStore.ListTriggerEvents(triggerID, limit)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"events": events})
			}
			return renderTriggerEventsText(cmd.OutOrStdout(), events)
		},
	}
	cmd.Flags().StringVar(&triggerID, "trigger", "", "Filter by trigger ID")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of events")
	return cmd
}

func newTriggerFireCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string
	var pathFlag string
	var intentFlag string
	var execute bool
	var autoThreshold string
	var approvalThreshold string

	cmd := &cobra.Command{
		Use:   "fire <trigger_id>",
		Short: "Fire a trigger immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			triggerStore := getTriggerStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			triggerID := strings.TrimSpace(args[0])
			cfg, err := triggerStore.GetTrigger(triggerID)
			if err != nil {
				return err
			}
			if !cfg.Enabled {
				return fmt.Errorf("trigger %s is disabled", triggerID)
			}

			repoRoot := firstNonEmpty(pathFlag, app.RepoRoot, app.Config.Paths.RepositoryRoot)
			owner, repoName := resolveTriggerRepo(cfg, repoFlag, repoRoot)
			if repoFlag != "" && (owner == "" || repoName == "") {
				return fmt.Errorf("invalid --repo %q; use owner/repo", repoFlag)
			}
			repoRoot = selectRepoRootForRemote(app, repoRoot, owner, repoName)

			result, err := runAutonomyCycle(cmd.Context(), cmd, app, autonomyRunRequest{
				RepoRoot:          repoRoot,
				Owner:             owner,
				Repo:              repoName,
				Intent:            firstNonEmpty(strings.TrimSpace(intentFlag), strings.TrimSpace(cfg.ActionTemplate), "respond to trigger"),
				Execute:           execute,
				AutoThreshold:     autonomy.ParseRiskLevel(autoThreshold),
				ApprovalThreshold: autonomy.ParseRiskLevel(approvalThreshold),
			})
			if err != nil {
				return err
			}

			ev := &autonomy.TriggerEvent{
				TriggerID:       cfg.TriggerID,
				TriggerType:     cfg.TriggerType,
				SourceEvent:     firstNonEmpty(strings.TrimSpace(cfg.Source), "manual.fire"),
				ResultingTaskID: result.Report.CycleID,
			}
			if err := triggerStore.AppendTriggerEvent(ev); err != nil {
				return err
			}

			payload := map[string]any{
				"trigger":   cfg,
				"owner":     owner,
				"repo":      repoName,
				"repo_root": repoRoot,
				"event":     ev,
				"autonomy":  result,
			}
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, payload)
			}
			return renderTriggerFireText(cmd.OutOrStdout(), cfg, ev, result)
		},
	}
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo override for trigger execution")
	cmd.Flags().StringVar(&pathFlag, "path", "", "Explicit local clone path to use for Git and file actions")
	cmd.Flags().StringVar(&intentFlag, "intent", "", "Override trigger action template with an explicit intent")
	cmd.Flags().BoolVar(&execute, "execute", false, "Execute plans at or below the auto threshold; otherwise preview only")
	cmd.Flags().StringVar(&autoThreshold, "auto-threshold", autonomy.RiskLow.String(), "Auto-execution threshold: low, medium, high, critical")
	cmd.Flags().StringVar(&approvalThreshold, "approval-threshold", autonomy.RiskMedium.String(), "Threshold above which plans stay pending")
	return cmd
}

func resolveTriggerRepo(cfg *autonomy.TriggerConfig, repoFlag, repoRoot string) (string, string) {
	if owner, repoName := parseRepoFlag(repoFlag, repoRoot); owner != "" && repoName != "" {
		return owner, repoName
	}
	if cfg != nil {
		if owner, repoName := parseRepoFlag(strings.TrimSpace(cfg.Source), repoRoot); owner != "" && repoName != "" {
			return owner, repoName
		}
	}
	return parseRepoFlag("", repoRoot)
}

func renderTriggerFireText(out io.Writer, cfg *autonomy.TriggerConfig, ev *autonomy.TriggerEvent, result autonomyRunResult) error {
	_, _ = fmt.Fprintf(out, "Trigger fired: %s (%s)\n", cfg.Name, cfg.TriggerID)
	if result.Owner != "" && result.Repo != "" {
		_, _ = fmt.Fprintf(out, "Repository:    %s/%s\n", result.Owner, result.Repo)
	}
	if result.RepoRoot != "" {
		_, _ = fmt.Fprintf(out, "Local clone:   %s\n", result.RepoRoot)
	}
	_, _ = fmt.Fprintf(out, "Mode:          %s\n", result.Mode)
	_, _ = fmt.Fprintf(out, "Cycle ID:      %s\n", result.Report.CycleID)
	_, _ = fmt.Fprintf(out, "Event ID:      %s\n", ev.EventID)
	return nil
}

func renderTriggerListText(out io.Writer, configs []*autonomy.TriggerConfig) error {
	if len(configs) == 0 {
		_, _ = fmt.Fprintln(out, "No triggers configured.")
		return nil
	}
	_, _ = fmt.Fprintf(out, "%-20s %-20s %-12s %-30s %s\n", "Trigger ID", "Name", "Type", "Pattern", "Enabled")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 100))
	for _, c := range configs {
		enabled := "yes"
		if !c.Enabled {
			enabled = "no"
		}
		_, _ = fmt.Fprintf(out, "%-20s %-20s %-12s %-30s %s\n", c.TriggerID, c.Name, c.TriggerType, c.Pattern, enabled)
	}
	return nil
}

func renderTriggerEventsText(out io.Writer, events []*autonomy.TriggerEvent) error {
	if len(events) == 0 {
		_, _ = fmt.Fprintln(out, "No trigger events found.")
		return nil
	}
	_, _ = fmt.Fprintf(out, "%-20s %-20s %-12s %-30s %s\n", "Event ID", "Trigger ID", "Type", "Task ID", "Timestamp")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 100))
	for _, e := range events {
		_, _ = fmt.Fprintf(out, "%-20s %-20s %-12s %-30s %s\n", e.EventID, e.TriggerID, e.TriggerType, e.ResultingTaskID, e.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
	}
	return nil
}
