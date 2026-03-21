package command

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/api"
	"github.com/your-org/gitdex/internal/app/bootstrap"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

func newExportGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export plans, reports, and handoff artifacts for external reuse",
	}
	cmd.AddCommand(newExportGenerateCommand(flags, appFn))
	cmd.AddCommand(newExportListCommand(flags, appFn))
	return cmd
}

func newExportGenerateCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var exportTypeStr, formatStr string
	var includeEvidence bool
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate export",
		Long:  "Generate plan_report, task_report, campaign_report, audit_report, or handoff_artifact. Use --type and --format json|yaml|markdown.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			fmtOut := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			et := strings.TrimSpace(strings.ToLower(exportTypeStr))
			if et == "" {
				et = "plan_report"
			}
			ft := strings.TrimSpace(strings.ToLower(formatStr))
			if ft == "" {
				ft = "json"
			}

			req := &api.ExportRequest{
				ExportType:      api.ExportType(et),
				Format:          ft,
				IncludeEvidence: includeEvidence,
			}

			if app.StorageProvider == nil {
				return fmt.Errorf("export requires a configured storage provider")
			}
			engine := api.NewProviderExportEngine(app.StorageProvider)
			result, err := engine.Export(context.Background(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(fmtOut) {
				return clioutput.WriteValue(cmd.OutOrStdout(), fmtOut, result)
			}
			return renderExportResultText(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&exportTypeStr, "type", "plan_report", "Export type: plan_report, task_report, campaign_report, audit_report, handoff_artifact")
	cmd.Flags().StringVar(&formatStr, "format", "json", "Format: json, yaml, markdown")
	cmd.Flags().BoolVar(&includeEvidence, "include-evidence", false, "Include evidence in export")
	return cmd
}

func newExportListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available export types",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			fmtOut := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			types := api.ListExportTypes()

			if clioutput.IsStructured(fmtOut) {
				return clioutput.WriteValue(cmd.OutOrStdout(), fmtOut, map[string]any{"export_types": types})
			}
			return renderExportTypesText(cmd.OutOrStdout(), types)
		},
	}
}

func renderExportResultText(out io.Writer, r *api.ExportResult) error {
	_, _ = fmt.Fprintf(out, "Export type: %s\n", r.ExportType)
	_, _ = fmt.Fprintf(out, "Format: %s\n", r.Format)
	_, _ = fmt.Fprintf(out, "Generated: %s\n", r.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"))
	if r.FilePath != "" {
		_, _ = fmt.Fprintf(out, "File: %s\n", r.FilePath)
	}
	if r.Data != "" {
		_, _ = fmt.Fprintf(out, "Data: %s\n", r.Data)
	}
	return nil
}

func renderExportTypesText(out io.Writer, types []api.ExportType) error {
	for _, t := range types {
		_, _ = fmt.Fprintf(out, "%s\n", t)
	}
	return nil
}
