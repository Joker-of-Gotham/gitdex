package command

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	appstate "github.com/your-org/gitdex/internal/app/state"
	"github.com/your-org/gitdex/internal/autonomy"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	repostate "github.com/your-org/gitdex/internal/state/repo"
)

var monitorStoreOverride autonomy.MonitorStore

// SetMonitorStoreForTest allows integration tests to inject a custom store.
func SetMonitorStoreForTest(s autonomy.MonitorStore) func() {
	prev := monitorStoreOverride
	monitorStoreOverride = s
	return func() { monitorStoreOverride = prev }
}

func getMonitorStore(appFn func() bootstrap.App) autonomy.MonitorStore {
	if monitorStoreOverride != nil {
		return monitorStoreOverride
	}
	return appFn().StorageProvider.MonitorStore()
}

func newMonitorGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitor authorized repositories continuously or on schedule",
	}
	cmd.AddCommand(newMonitorAddCommand(flags, appFn))
	cmd.AddCommand(newMonitorListCommand(flags, appFn))
	cmd.AddCommand(newMonitorEventsCommand(flags, appFn))
	cmd.AddCommand(newMonitorRemoveCommand(flags, appFn))
	cmd.AddCommand(newMonitorCheckCommand(flags, appFn))
	return cmd
}

func newMonitorAddCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a repository monitor",
		Long:  "Add a monitor for a repository. Use --repo owner/repo and --interval (e.g. 5m).",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			monitorStore := getMonitorStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				return fmt.Errorf("--repo is required (format: owner/repo)")
			}
			parts := strings.SplitN(repo, "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("--repo must be in format owner/repo")
			}
			owner, name := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if owner == "" || name == "" {
				return fmt.Errorf("--repo must be in format owner/repo")
			}

			interval, _ := cmd.Flags().GetString("interval")
			if interval == "" {
				interval = "5m"
			}

			cfg := &autonomy.MonitorConfig{
				RepoOwner: owner,
				RepoName:  name,
				Interval:  interval,
				Checks:    []string{},
				Enabled:   true,
			}
			if err := monitorStore.SaveMonitorConfig(cfg); err != nil {
				return fmt.Errorf("failed to add monitor: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, cfg)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Monitor added: %s/%s (id: %s, interval: %s)\n", owner, name, cfg.MonitorID, interval)
			return nil
		},
	}
	cmd.Flags().String("repo", "", "Repository in owner/repo format (required)")
	cmd.Flags().String("interval", "5m", "Check interval (e.g. 5m, 1h)")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newMonitorListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List repository monitors",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			monitorStore := getMonitorStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			configs, err := monitorStore.ListMonitorConfigs()
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"monitors": configs})
			}
			return renderMonitorListText(cmd.OutOrStdout(), configs)
		},
	}
}

func newMonitorEventsCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var limit int
	var repo string
	cmd := &cobra.Command{
		Use:   "events",
		Short: "List recent monitor events",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			monitorStore := getMonitorStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			filter := autonomy.MonitorEventFilter{Limit: limit}
			if repo != "" {
				parts := strings.SplitN(repo, "/", 2)
				if len(parts) == 2 {
					filter.RepoOwner = strings.TrimSpace(parts[0])
					filter.RepoName = strings.TrimSpace(parts[1])
				}
			}
			if filter.Limit <= 0 {
				filter.Limit = 20
			}

			events, err := monitorStore.ListEvents(filter)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"events": events})
			}
			return renderMonitorEventsText(cmd.OutOrStdout(), events)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Filter by repo (owner/repo)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of events")
	return cmd
}

func newMonitorRemoveCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <monitor_id>",
		Short: "Remove a monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			monitorStore := getMonitorStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			monitorID := args[0]
			if err := monitorStore.RemoveMonitorConfig(monitorID); err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]string{"status": "removed", "monitor_id": monitorID})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Monitor %s removed.\n", monitorID)
			return nil
		},
	}
}

func newMonitorCheckCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var monitorID string
	var repoFlag string
	var execute bool
	var intent string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run an immediate monitor check and append monitor events",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			monitorStore := getMonitorStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			targets, err := resolveMonitorTargets(monitorStore, monitorID, repoFlag)
			if err != nil {
				return err
			}

			results := make([]map[string]any, 0, len(targets))
			for _, cfg := range targets {
				res, err := runMonitorCheck(cmd.Context(), cmd, app, monitorStore, cfg, execute, intent)
				if err != nil {
					return err
				}
				results = append(results, res)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"results": results})
			}
			return renderMonitorCheckResults(cmd.OutOrStdout(), results)
		},
	}
	cmd.Flags().StringVar(&monitorID, "monitor", "", "Run a specific monitor by ID")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Run checks for a repo (owner/repo)")
	cmd.Flags().BoolVar(&execute, "execute", false, "If health issues are found, ask autonomy to propose and execute a remediation cycle")
	cmd.Flags().StringVar(&intent, "intent", "", "Override the default remediation intent")
	return cmd
}

func resolveMonitorTargets(store autonomy.MonitorStore, monitorID, repoFlag string) ([]*autonomy.MonitorConfig, error) {
	if strings.TrimSpace(monitorID) != "" {
		cfg, err := store.GetMonitorConfig(strings.TrimSpace(monitorID))
		if err != nil {
			return nil, err
		}
		return []*autonomy.MonitorConfig{cfg}, nil
	}

	configs, err := store.ListMonitorConfigs()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(repoFlag) == "" {
		return enabledMonitors(configs), nil
	}

	owner, repoName := parseRepoFlag(repoFlag, "")
	if owner == "" || repoName == "" {
		return nil, fmt.Errorf("--repo must be in format owner/repo")
	}

	var result []*autonomy.MonitorConfig
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		if cfg.RepoOwner == owner && cfg.RepoName == repoName {
			result = append(result, cfg)
		}
	}
	if len(result) == 0 {
		result = append(result, &autonomy.MonitorConfig{
			MonitorID: "ad-hoc",
			RepoOwner: owner,
			RepoName:  repoName,
			Interval:  "manual",
			Enabled:   true,
		})
	}
	return result, nil
}

func enabledMonitors(configs []*autonomy.MonitorConfig) []*autonomy.MonitorConfig {
	result := make([]*autonomy.MonitorConfig, 0, len(configs))
	for _, cfg := range configs {
		if cfg.Enabled {
			result = append(result, cfg)
		}
	}
	return result
}

func runMonitorCheck(ctx context.Context, cmd *cobra.Command, app bootstrap.App, store autonomy.MonitorStore, cfg *autonomy.MonitorConfig, execute bool, intent string) (map[string]any, error) {
	repoRoot := selectRepoRootForRemote(app, firstNonEmpty(app.RepoRoot, app.Config.Paths.RepositoryRoot), cfg.RepoOwner, cfg.RepoName)
	ghClient, err := newGitHubClientFromApp(app)
	if err != nil {
		return nil, err
	}

	assembler := appstate.NewAssembler(ghClient)
	summary, err := assembler.Assemble(ctx, cfg.RepoOwner, cfg.RepoName, repoRoot)
	if err != nil {
		return nil, err
	}

	events := buildMonitorEvents(cfg, summary)
	for _, ev := range events {
		if err := store.AppendEvent(ev); err != nil {
			return nil, err
		}
	}

	result := map[string]any{
		"monitor_id": cfg.MonitorID,
		"repo":       cfg.RepoOwner + "/" + cfg.RepoName,
		"repo_root":  repoRoot,
		"events":     events,
		"summary":    summary,
	}

	if execute && shouldAutoRemediate(summary) {
		runResult, err := runAutonomyCycle(ctx, cmd, app, autonomyRunRequest{
			RepoRoot:          repoRoot,
			Owner:             cfg.RepoOwner,
			Repo:              cfg.RepoName,
			Intent:            firstNonEmpty(strings.TrimSpace(intent), defaultMonitorIntent(summary)),
			Execute:           true,
			AutoThreshold:     autonomy.RiskLow,
			ApprovalThreshold: autonomy.RiskMedium,
		})
		if err != nil {
			return nil, err
		}
		result["autonomy"] = runResult
	}

	return result, nil
}

func buildMonitorEvents(cfg *autonomy.MonitorConfig, summary *repostate.RepoSummary) []*autonomy.MonitorEvent {
	if summary == nil {
		return []*autonomy.MonitorEvent{{
			MonitorID: cfg.MonitorID,
			RepoOwner: cfg.RepoOwner,
			RepoName:  cfg.RepoName,
			CheckName: "summary",
			Status:    "critical",
			Message:   "repository summary unavailable",
		}}
	}

	checks := []struct {
		name   string
		label  repostate.StateLabel
		detail string
	}{
		{name: "local", label: summary.Local.Label, detail: summary.Local.Detail},
		{name: "remote", label: summary.Remote.Label, detail: summary.Remote.Detail},
		{name: "collaboration", label: summary.Collaboration.Label, detail: summary.Collaboration.Detail},
		{name: "workflows", label: summary.Workflows.Label, detail: summary.Workflows.Detail},
		{name: "deployments", label: summary.Deployments.Label, detail: summary.Deployments.Detail},
	}

	events := make([]*autonomy.MonitorEvent, 0, len(checks))
	for _, check := range checks {
		events = append(events, &autonomy.MonitorEvent{
			MonitorID: cfg.MonitorID,
			RepoOwner: cfg.RepoOwner,
			RepoName:  cfg.RepoName,
			CheckName: check.name,
			Status:    monitorStatusForLabel(check.label),
			Message:   firstNonEmpty(strings.TrimSpace(check.detail), string(check.label)),
		})
	}
	return events
}

func monitorStatusForLabel(label repostate.StateLabel) string {
	switch label {
	case repostate.Healthy:
		return "ok"
	case repostate.Unknown:
		return "warning"
	case repostate.Drifting:
		return "warning"
	case repostate.Degraded, repostate.Blocked:
		return "critical"
	default:
		return "warning"
	}
}

func shouldAutoRemediate(summary *repostate.RepoSummary) bool {
	if summary == nil {
		return false
	}
	return len(summary.Risks) > 0 || summary.OverallLabel == repostate.Drifting || summary.OverallLabel == repostate.Degraded || summary.OverallLabel == repostate.Blocked
}

func defaultMonitorIntent(summary *repostate.RepoSummary) string {
	if summary == nil {
		return "stabilize repository health"
	}
	if len(summary.NextActions) > 0 {
		return summary.NextActions[0].Action
	}
	return "stabilize repository health issues"
}

func renderMonitorCheckResults(out io.Writer, results []map[string]any) error {
	if len(results) == 0 {
		_, _ = fmt.Fprintln(out, "No enabled monitors to check.")
		return nil
	}
	for _, result := range results {
		repo := result["repo"]
		monitorID := result["monitor_id"]
		_, _ = fmt.Fprintf(out, "Monitor %v (%v)\n", monitorID, repo)
		if repoRoot, ok := result["repo_root"].(string); ok && strings.TrimSpace(repoRoot) != "" {
			_, _ = fmt.Fprintf(out, "  Local clone: %s\n", repoRoot)
		}
		if events, ok := result["events"].([]*autonomy.MonitorEvent); ok {
			for _, ev := range events {
				_, _ = fmt.Fprintf(out, "  - %s [%s] %s\n", ev.CheckName, ev.Status, ev.Message)
			}
		}
		if auto, ok := result["autonomy"].(autonomyRunResult); ok {
			_, _ = fmt.Fprintf(out, "  Autonomy: %s, cycle %s\n", auto.Mode, auto.Report.CycleID)
		}
	}
	return nil
}

func renderMonitorListText(out io.Writer, configs []*autonomy.MonitorConfig) error {
	if len(configs) == 0 {
		_, _ = fmt.Fprintln(out, "No monitors configured.")
		return nil
	}
	_, _ = fmt.Fprintf(out, "%-20s %-30s %-10s %s\n", "Monitor ID", "Repo", "Interval", "Enabled")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 80))
	for _, c := range configs {
		enabled := "yes"
		if !c.Enabled {
			enabled = "no"
		}
		_, _ = fmt.Fprintf(out, "%-20s %-30s %-10s %s\n", c.MonitorID, c.RepoOwner+"/"+c.RepoName, c.Interval, enabled)
	}
	return nil
}

func renderMonitorEventsText(out io.Writer, events []*autonomy.MonitorEvent) error {
	if len(events) == 0 {
		_, _ = fmt.Fprintln(out, "No events found.")
		return nil
	}
	_, _ = fmt.Fprintf(out, "%-20s %-30s %-12s %-10s %s\n", "Event ID", "Repo", "Check", "Status", "Message")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 100))
	for _, e := range events {
		repo := e.RepoOwner + "/" + e.RepoName
		_, _ = fmt.Fprintf(out, "%-20s %-30s %-12s %-10s %s\n", e.EventID, repo, e.CheckName, e.Status, e.Message)
	}
	return nil
}
