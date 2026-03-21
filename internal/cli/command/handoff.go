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
)

var handoffGeneratorOverride func(app bootstrap.App, taskID string) (*autonomy.HandoffPackage, error)

func SetHandoffGeneratorForTest(generator func(app bootstrap.App, taskID string) (*autonomy.HandoffPackage, error)) func() {
	prev := handoffGeneratorOverride
	handoffGeneratorOverride = generator
	return func() { handoffGeneratorOverride = prev }
}

func newHandoffGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Generate and manage handoff packages for long-running tasks",
	}
	cmd.AddCommand(newHandoffGenerateCommand(flags, appFn))
	cmd.AddCommand(newHandoffShowCommand(flags, appFn))
	cmd.AddCommand(newHandoffListCommand(flags, appFn))
	return cmd
}

func newHandoffGenerateCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "generate <task_id>",
		Short: "Generate a handoff package for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			handoffStore := app.StorageProvider.HandoffStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			var (
				pkg *autonomy.HandoffPackage
				err error
			)
			if handoffGeneratorOverride != nil {
				pkg, err = handoffGeneratorOverride(app, args[0])
			} else {
				pkg, err = autonomy.GenerateHandoffPackageFromStores(
					handoffStore,
					app.StorageProvider.TaskStore(),
					app.StorageProvider.PlanStore(),
					app.StorageProvider.AuditLedger(),
					args[0],
				)
			}
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, pkg)
			}
			return renderHandoffPackage(cmd.OutOrStdout(), pkg)
		},
	}
}

func newHandoffShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <package_id>",
		Short: "Show a handoff package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			handoffStore := app.StorageProvider.HandoffStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			pkg, err := handoffStore.GetPackage(args[0])
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, pkg)
			}
			return renderHandoffPackage(cmd.OutOrStdout(), pkg)
		},
	}
}

func newHandoffListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List handoff packages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			handoffStore := app.StorageProvider.HandoffStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			packages, err := handoffStore.ListPackages()
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"packages": packages,
				})
			}
			return renderHandoffList(cmd.OutOrStdout(), packages)
		},
	}
}

func renderHandoffPackage(out io.Writer, pkg *autonomy.HandoffPackage) error {
	_, _ = fmt.Fprintf(out, "Package ID: %s\n", pkg.PackageID)
	_, _ = fmt.Fprintf(out, "Task ID: %s\n", pkg.TaskID)
	_, _ = fmt.Fprintf(out, "Task summary: %s\n", pkg.TaskSummary)
	_, _ = fmt.Fprintf(out, "Current state: %s\n", pkg.CurrentState)
	_, _ = fmt.Fprintf(out, "Created at: %s\n", pkg.CreatedAt.Format(time.RFC3339))
	if len(pkg.CompletedSteps) > 0 {
		_, _ = fmt.Fprintf(out, "Completed steps: %s\n", strings.Join(pkg.CompletedSteps, ", "))
	}
	if len(pkg.PendingSteps) > 0 {
		_, _ = fmt.Fprintf(out, "Pending steps: %s\n", strings.Join(pkg.PendingSteps, ", "))
	}
	if len(pkg.Artifacts) > 0 {
		_, _ = fmt.Fprintf(out, "Artifacts: %s\n", strings.Join(pkg.Artifacts, ", "))
	}
	if len(pkg.Recommendations) > 0 {
		_, _ = fmt.Fprintf(out, "Recommendations: %s\n", strings.Join(pkg.Recommendations, ", "))
	}
	return nil
}

func renderHandoffList(out io.Writer, packages []*autonomy.HandoffPackage) error {
	if len(packages) == 0 {
		_, _ = fmt.Fprintln(out, "No handoff packages found.")
		return nil
	}

	_, _ = fmt.Fprintf(out, "%-20s %-20s %-25s %s\n",
		"Package ID", "Task ID", "Current State", "Created At")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 85))
	for _, pkg := range packages {
		_, _ = fmt.Fprintf(out, "%-20s %-20s %-25s %s\n",
			pkg.PackageID, pkg.TaskID, pkg.CurrentState,
			pkg.CreatedAt.Format(time.RFC3339))
	}
	return nil
}
