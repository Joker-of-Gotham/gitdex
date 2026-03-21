package command

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

var recoveryEngineOverride autonomy.RecoveryEngine

// SetRecoveryEngineForTest allows integration tests to inject a custom recovery engine.
func SetRecoveryEngineForTest(eng autonomy.RecoveryEngine) func() {
	prev := recoveryEngineOverride
	recoveryEngineOverride = eng
	return func() { recoveryEngineOverride = prev }
}

func getRecoveryEngine() autonomy.RecoveryEngine {
	if recoveryEngineOverride != nil {
		return recoveryEngineOverride
	}
	return nil
}

func getRecoveryEngineForApp(app bootstrap.App) autonomy.RecoveryEngine {
	if recoveryEngineOverride != nil {
		return recoveryEngineOverride
	}
	return autonomy.NewRecoveryEngine(
		app.StorageProvider.TaskStore(),
		app.StorageProvider.PlanStore(),
		app.StorageProvider.AuditLedger(),
		app.StorageProvider.HandoffStore(),
	)
}

func newRecoveryGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recovery",
		Short: "Assess and recover from blocked, failed, or drifted tasks",
	}
	cmd.AddCommand(newRecoveryAssessCommand(flags, appFn))
	cmd.AddCommand(newRecoveryExecuteCommand(flags, appFn))
	cmd.AddCommand(newRecoveryHistoryCommand(flags, appFn))
	return cmd
}

func newRecoveryAssessCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "assess <task_id>",
		Short: "Assess recovery options for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			assessment, err := getRecoveryEngineForApp(app).Assess(context.Background(), args[0])
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, assessment)
			}
			return renderRecoveryAssessment(cmd.OutOrStdout(), assessment)
		},
	}
}

func newRecoveryExecuteCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var strategy string
	var maxRetries int
	var reason string
	cmd := &cobra.Command{
		Use:   "execute <task_id>",
		Short: "Execute recovery for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			s := autonomy.RecoveryStrategy(strings.ToLower(strings.TrimSpace(strategy)))
			if s == "" {
				s = autonomy.RecoveryRetry
			}
			valid := map[autonomy.RecoveryStrategy]struct{}{
				autonomy.RecoveryRetry:              {},
				autonomy.RecoveryRollback:           {},
				autonomy.RecoveryEscalate:           {},
				autonomy.RecoverySkip:               {},
				autonomy.RecoveryManualIntervention: {},
			}
			if _, ok := valid[s]; !ok {
				return fmt.Errorf("invalid strategy %q; use retry, rollback, escalate, skip, or manual_intervention", strategy)
			}

			req := autonomy.RecoveryRequest{
				TaskID:     args[0],
				Strategy:   s,
				MaxRetries: maxRetries,
				Reason:     reason,
				Actor:      "cli",
			}
			if req.Reason == "" {
				req.Reason = "user requested"
			}

			result, err := getRecoveryEngineForApp(app).Execute(context.Background(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderRecoveryResult(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&strategy, "strategy", "retry", "Recovery strategy: retry, rollback, escalate, skip, manual_intervention")
	cmd.Flags().IntVar(&maxRetries, "max-retries", 1, "Maximum retry attempts")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for recovery")
	return cmd
}

func newRecoveryHistoryCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var taskID string
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show recovery history",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			history, err := loadRecoveryHistory(app, taskID)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"history": history,
				})
			}
			return renderRecoveryHistory(cmd.OutOrStdout(), history)
		},
	}
	cmd.Flags().StringVar(&taskID, "task_id", "", "Filter by task ID")
	return cmd
}

func loadRecoveryHistory(app bootstrap.App, taskID string) ([]autonomy.RecoveryResult, error) {
	if override := getRecoveryEngine(); override != nil {
		if eng, ok := override.(*autonomy.DefaultRecoveryEngine); ok {
			return eng.History(taskID), nil
		}
		return nil, fmt.Errorf("recovery engine does not support history")
	}
	entries, err := app.StorageProvider.AuditLedger().Query(audit.AuditFilter{
		TaskID:    taskID,
		EventType: audit.EventRecovery,
	})
	if err != nil {
		return nil, err
	}
	results := make([]autonomy.RecoveryResult, 0, len(entries))
	for _, entry := range entries {
		results = append(results, autonomy.RecoveryResult{
			Request: autonomy.RecoveryRequest{
				TaskID:   entry.TaskID,
				Strategy: autonomy.RecoveryStrategy(entry.Action),
				Actor:    entry.Actor,
			},
			Success:     true,
			Attempts:    1,
			FinalStatus: entry.PolicyResult,
			Message:     entry.Action,
			RecoveredAt: entry.Timestamp,
		})
	}
	return results, nil
}

func renderRecoveryAssessment(out io.Writer, a *autonomy.RecoveryAssessment) error {
	_, _ = fmt.Fprintf(out, "Task ID: %s\n", a.TaskID)
	_, _ = fmt.Fprintf(out, "Failure type: %s\n", a.FailureType)
	_, _ = fmt.Fprintf(out, "Recommended strategy: %s\n", a.RecommendedStrategy)
	_, _ = fmt.Fprintf(out, "Risk level: %s\n", a.RiskLevel)
	_, _ = fmt.Fprintf(out, "Details: %s\n", a.Details)
	return nil
}

func renderRecoveryResult(out io.Writer, r *autonomy.RecoveryResult) error {
	status := "OK"
	if !r.Success {
		status = "FAILED"
	}
	_, _ = fmt.Fprintf(out, "Status: %s\n", status)
	_, _ = fmt.Fprintf(out, "Message: %s\n", r.Message)
	_, _ = fmt.Fprintf(out, "Final status: %s\n", r.FinalStatus)
	_, _ = fmt.Fprintf(out, "Attempts: %d\n", r.Attempts)
	_, _ = fmt.Fprintf(out, "Recovered at: %s\n", r.RecoveredAt.Format(time.RFC3339))
	return nil
}

func renderRecoveryHistory(out io.Writer, history []autonomy.RecoveryResult) error {
	if len(history) == 0 {
		_, _ = fmt.Fprintln(out, "No recovery history.")
		return nil
	}

	_, _ = fmt.Fprintf(out, "%-20s %-12s %-15s %-12s %s\n",
		"Task ID", "Strategy", "Final Status", "Attempts", "Recovered At")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 75))
	for _, r := range history {
		_, _ = fmt.Fprintf(out, "%-20s %-12s %-15s %-12d %s\n",
			r.Request.TaskID, r.Request.Strategy, r.FinalStatus, r.Attempts,
			r.RecoveredAt.Format(time.RFC3339))
	}
	return nil
}
