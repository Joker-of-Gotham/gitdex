package command

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/audit"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

var auditLedgerOverride audit.AuditLedger

// SetAuditLedgerForTest allows integration tests to inject a custom ledger.
// Returns a restore function. For testing only.
func SetAuditLedgerForTest(l audit.AuditLedger) func() {
	prev := auditLedgerOverride
	auditLedgerOverride = l
	return func() { auditLedgerOverride = prev }
}

func getAuditLedger(appFn func() bootstrap.App) audit.AuditLedger {
	if auditLedgerOverride != nil {
		return auditLedgerOverride
	}
	return appFn().StorageProvider.AuditLedger()
}

func newAuditGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Inspect audit history, evidence, and task lineage",
	}
	cmd.AddCommand(newAuditLogCommand(flags, appFn))
	cmd.AddCommand(newAuditShowCommand(flags, appFn))
	cmd.AddCommand(newAuditTraceCommand(flags, appFn))
	return cmd
}

func newAuditLogCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show recent audit entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			auditLedger := getAuditLedger(appFn)

			entries, err := auditLedger.Query(audit.AuditFilter{})
			if err != nil {
				return err
			}

			// Apply limit, most recent first
			if limit < 0 {
				limit = 0
			}
			if len(entries) > limit {
				entries = entries[len(entries)-limit:]
			}
			for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
				entries[i], entries[j] = entries[j], entries[i]
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"entries": entries})
			}
			return renderAuditLog(cmd.OutOrStdout(), entries)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of entries to show")
	return cmd
}

func newAuditShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show [entry_id]",
		Short: "Show entry details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			auditLedger := getAuditLedger(appFn)

			entry, ok := auditLedger.GetByEntryID(args[0])
			if !ok {
				return fmt.Errorf("audit entry not found: %s", args[0])
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, entry)
			}
			return renderAuditEntry(cmd.OutOrStdout(), entry)
		},
	}
}

func newAuditTraceCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "trace [correlation_id]",
		Short: "Show full lineage for a correlation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			auditLedger := getAuditLedger(appFn)

			entries, err := auditLedger.GetByCorrelation(args[0])
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"correlation_id": args[0],
					"entries":        entries,
				})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Trace for correlation: %s\n\n", args[0])
			return renderAuditLog(cmd.OutOrStdout(), entries)
		},
	}
}

func renderAuditLog(out io.Writer, entries []*audit.AuditEntry) error {
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(out, "No audit entries found.")
		return nil
	}

	_, _ = fmt.Fprintf(out, "%-20s %-18s %-12s %-15s %s\n",
		"Entry ID", "Event Type", "Task", "Actor", "Timestamp")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 85))
	for _, e := range entries {
		_, _ = fmt.Fprintf(out, "%-20s %-18s %-12s %-15s %s\n",
			e.EntryID, e.EventType, truncate(e.TaskID, 12), truncate(e.Actor, 15),
			e.Timestamp.Format(time.RFC3339))
	}
	return nil
}

func renderAuditEntry(out io.Writer, e *audit.AuditEntry) error {
	_, _ = fmt.Fprintf(out, "Entry ID:       %s\n", e.EntryID)
	_, _ = fmt.Fprintf(out, "Correlation ID: %s\n", e.CorrelationID)
	_, _ = fmt.Fprintf(out, "Task ID:        %s\n", e.TaskID)
	_, _ = fmt.Fprintf(out, "Plan ID:        %s\n", e.PlanID)
	_, _ = fmt.Fprintf(out, "Event Type:     %s\n", e.EventType)
	_, _ = fmt.Fprintf(out, "Actor:          %s\n", e.Actor)
	_, _ = fmt.Fprintf(out, "Action:         %s\n", e.Action)
	_, _ = fmt.Fprintf(out, "Target:         %s\n", e.Target)
	if e.PolicyResult != "" {
		_, _ = fmt.Fprintf(out, "Policy Result:  %s\n", e.PolicyResult)
	}
	if len(e.EvidenceRefs) > 0 {
		_, _ = fmt.Fprintf(out, "Evidence Refs:  %s\n", strings.Join(e.EvidenceRefs, ", "))
	}
	_, _ = fmt.Fprintf(out, "Timestamp:      %s\n", e.Timestamp.Format(time.RFC3339))
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
