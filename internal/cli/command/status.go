package command

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/repocontext"
	appstate "github.com/your-org/gitdex/internal/app/state"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/state/repo"
)

func newStatusCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var ownerFlag, repoFlag string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show consolidated repository state summary",
		Long:  "Display local Git state, remote divergence, collaboration signals, workflow state, and deployment status for the current repository.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repoRoot := app.RepoRoot
			if repoRoot == "" {
				repoRoot = app.Config.Paths.RepositoryRoot
			}

			owner, repoName := resolveOwnerRepo(ownerFlag, repoFlag, repoRoot)
			if owner == "" || repoName == "" {
				return fmt.Errorf("cannot determine repository owner/name; use --owner and --repo flags or run from a git repository with a remote")
			}
			repoRoot = selectRepoRootForRemote(app, repoRoot, owner, repoName)

			var ghClient *ghclient.Client
			if client, err := newGitHubClientFromApp(app); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: GitHub identity error: %v\n", err)
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Showing local state only. Run 'gitdex init' to configure.\n")
			} else {
				ghClient = client
			}

			assembler := appstate.NewAssembler(ghClient)
			summary, err := assembler.Assemble(context.Background(), owner, repoName, repoRoot)
			if err != nil {
				return fmt.Errorf("failed to assemble repository state: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, summary)
			}
			return renderStatusText(cmd.OutOrStdout(), summary)
		},
	}

	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Repository owner (auto-detected from remote)")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository name (auto-detected from remote)")

	return cmd
}

func resolveOwnerRepo(ownerFlag, repoFlag, repoRoot string) (string, string) {
	if ownerFlag != "" && repoFlag != "" {
		return ownerFlag, repoFlag
	}

	if repoRoot != "" {
		owner, name := repocontext.ResolveOwnerRepoFromLocalPath(context.Background(), repoRoot)
		if ownerFlag == "" {
			ownerFlag = owner
		}
		if repoFlag == "" {
			repoFlag = name
		}
	}
	return ownerFlag, repoFlag
}

func renderStatusText(out io.Writer, s *repo.RepoSummary) error {
	_, _ = fmt.Fprintf(out, "📊 Repository Status: %s/%s\n", s.Owner, s.Repo)
	_, _ = fmt.Fprintf(out, "   Overall: %s\n\n", labelIcon(s.OverallLabel))

	_, _ = fmt.Fprintf(out, "── Local (%s) ──\n", labelIcon(s.Local.Label))
	_, _ = fmt.Fprintf(out, "   Branch:  %s\n", s.Local.Branch)
	_, _ = fmt.Fprintf(out, "   SHA:     %s\n", s.Local.HeadSHA)
	if s.Local.IsDetached {
		_, _ = fmt.Fprintf(out, "   ⚠️  HEAD is detached\n")
	}
	if s.Local.IsClean {
		_, _ = fmt.Fprintf(out, "   Working tree: clean\n")
	} else {
		_, _ = fmt.Fprintf(out, "   Working tree: %d dirty, %d staged\n", s.Local.DirtyCount, s.Local.StagedCount)
	}
	if s.Local.Ahead > 0 || s.Local.Behind > 0 {
		_, _ = fmt.Fprintf(out, "   Divergence:   ↑%d ↓%d\n", s.Local.Ahead, s.Local.Behind)
	}
	if s.Local.Detail != "" {
		_, _ = fmt.Fprintf(out, "   Detail: %s\n", s.Local.Detail)
	}

	_, _ = fmt.Fprintf(out, "\n── Remote (%s) ──\n", labelIcon(s.Remote.Label))
	if s.Remote.FullName != "" {
		_, _ = fmt.Fprintf(out, "   Name:    %s\n", s.Remote.FullName)
		_, _ = fmt.Fprintf(out, "   Default: %s\n", s.Remote.DefaultBranch)
	}
	if s.Remote.Detail != "" {
		_, _ = fmt.Fprintf(out, "   Detail: %s\n", s.Remote.Detail)
	}

	_, _ = fmt.Fprintf(out, "\n── Collaboration (%s) ──\n", labelIcon(s.Collaboration.Label))
	_, _ = fmt.Fprintf(out, "   Open PRs:    %d\n", s.Collaboration.OpenPRCount)
	_, _ = fmt.Fprintf(out, "   Open Issues: %d\n", s.Collaboration.OpenIssueCount)
	if s.Collaboration.Detail != "" {
		_, _ = fmt.Fprintf(out, "   Detail: %s\n", s.Collaboration.Detail)
	}

	_, _ = fmt.Fprintf(out, "\n── Workflows (%s) ──\n", labelIcon(s.Workflows.Label))
	for _, r := range s.Workflows.Runs {
		icon := "✅"
		if r.Conclusion == "failure" || r.Conclusion == "timed_out" {
			icon = "❌"
		} else if r.Status == "in_progress" {
			icon = "🔄"
		}
		_, _ = fmt.Fprintf(out, "   %s %s (%s)\n", icon, r.Name, r.Branch)
	}
	if s.Workflows.Detail != "" {
		_, _ = fmt.Fprintf(out, "   Detail: %s\n", s.Workflows.Detail)
	}

	_, _ = fmt.Fprintf(out, "\n── Deployments (%s) ──\n", labelIcon(s.Deployments.Label))
	for _, d := range s.Deployments.Deployments {
		_, _ = fmt.Fprintf(out, "   %s: %s (ref: %s)\n", d.Environment, d.State, d.Ref)
	}
	if s.Deployments.Detail != "" {
		_, _ = fmt.Fprintf(out, "   Detail: %s\n", s.Deployments.Detail)
	}

	if len(s.Risks) > 0 {
		_, _ = fmt.Fprintf(out, "\n── Risks ──\n")
		for _, r := range s.Risks {
			_, _ = fmt.Fprintf(out, "   [%s] %s\n", strings.ToUpper(string(r.Severity)), r.Description)
			_, _ = fmt.Fprintf(out, "         Evidence: %s\n", r.Evidence)
			_, _ = fmt.Fprintf(out, "         Action:   %s\n", r.Action)
		}
	}

	if len(s.NextActions) > 0 {
		_, _ = fmt.Fprintf(out, "\n── Next Actions ──\n")
		for i, a := range s.NextActions {
			_, _ = fmt.Fprintf(out, "   %d. %s (%s risk)\n", i+1, a.Action, a.RiskLevel)
			_, _ = fmt.Fprintf(out, "      Reason: %s\n", a.Reason)
		}
	}

	return nil
}

func labelIcon(l repo.StateLabel) string {
	switch l {
	case repo.Healthy:
		return "✅ healthy"
	case repo.Drifting:
		return "⚠️  drifting"
	case repo.Blocked:
		return "🚫 blocked"
	case repo.Degraded:
		return "🔶 degraded"
	case repo.Unknown:
		return "❓ unknown"
	default:
		return string(l)
	}
}
