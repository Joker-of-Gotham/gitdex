package command

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/autonomyexec"
	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/llm/adapter"
)

type autonomyRunResult = autonomyexec.Result
type autonomyRunRequest = autonomyexec.Request

func setAutonomyProviderForTest(p adapter.Provider) func() {
	return autonomyexec.SetProviderOverrideForTest(p)
}

func newAutonomyRunOnceCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string
	var pathFlag string
	var intentFlag string
	var execute bool
	var autoThreshold string
	var approvalThreshold string

	cmd := &cobra.Command{
		Use:   "run-once [intent]",
		Short: "Plan and optionally execute a single autonomous cycle",
		Long:  "Generate autonomous plans from repository context or user intent, then optionally execute plans within configured risk thresholds.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			intent := intentFlag
			if intent == "" && len(args) > 0 {
				intent = strings.TrimSpace(strings.Join(args, " "))
			}

			repoRoot := firstNonEmpty(pathFlag, app.RepoRoot, app.Config.Paths.RepositoryRoot)
			owner, repoName := parseRepoFlag(repoFlag, repoRoot)
			if repoFlag != "" && (owner == "" || repoName == "") {
				return fmt.Errorf("invalid --repo %q; use owner/repo", repoFlag)
			}
			if owner == "" || repoName == "" {
				owner, repoName = resolveOwnerRepo("", "", repoRoot)
			}
			repoRoot = autonomyexec.SelectRepoRootForRemote(app, repoRoot, owner, repoName)

			result, err := runAutonomyCycle(cmd.Context(), cmd, app, autonomyRunRequest{
				RepoRoot:          repoRoot,
				Owner:             owner,
				Repo:              repoName,
				Intent:            intent,
				Execute:           execute,
				AutoThreshold:     autonomy.ParseRiskLevel(autoThreshold),
				ApprovalThreshold: autonomy.ParseRiskLevel(approvalThreshold),
			})
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderAutonomyRunResult(cmd.OutOrStdout(), result)
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo for remote-aware planning")
	cmd.Flags().StringVar(&pathFlag, "path", "", "Explicit local clone path to use for Git and file actions")
	cmd.Flags().StringVar(&intentFlag, "intent", "", "Explicit user intent to plan for")
	cmd.Flags().BoolVar(&execute, "execute", false, "Execute plans at or below the auto threshold; otherwise preview only")
	cmd.Flags().StringVar(&autoThreshold, "auto-threshold", autonomy.RiskLow.String(), "Auto-execution threshold: low, medium, high, critical")
	cmd.Flags().StringVar(&approvalThreshold, "approval-threshold", autonomy.RiskMedium.String(), "Threshold above which plans stay pending")
	return cmd
}

func runAutonomyCycle(ctx context.Context, _ *cobra.Command, app bootstrap.App, req autonomyRunRequest) (autonomyRunResult, error) {
	return autonomyexec.Run(ctx, app, req)
}

func renderAutonomyRunResult(out io.Writer, result autonomyRunResult) error {
	return autonomyexec.RenderResult(out, result)
}

func renderAutonomyRunDetails(out io.Writer, result autonomyRunResult) error {
	return autonomyexec.RenderDetails(out, result)
}
