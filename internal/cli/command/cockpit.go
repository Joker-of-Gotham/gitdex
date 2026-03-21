package command

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/repocontext"
	appstate "github.com/your-org/gitdex/internal/app/state"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/state/repo"
	tuiapp "github.com/your-org/gitdex/internal/tui/app"
	"github.com/your-org/gitdex/internal/tui/presenter"
)

func newCockpitCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var noTUI bool
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "cockpit",
		Short: "Open the Gitdex cockpit for repository observation",
		Long:  "Launch the interactive cockpit in rich TUI or text-only mode for repository state viewing, risk inspection, and next-action discovery.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()

			repoRoot := app.RepoRoot
			if repoRoot == "" {
				repoRoot = app.Config.Paths.RepositoryRoot
			}
			owner, repoName := parseRepoFlag(repoFlag, repoRoot)
			if repoFlag != "" && (owner == "" || repoName == "") {
				return fmt.Errorf("invalid --repo %q; use owner/repo", repoFlag)
			}

			outputFmt := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			repoRoot = selectRepoRootForRemote(app, repoRoot, owner, repoName)

			isTTY := term.IsTerminal(int(os.Stdout.Fd()))
			useTextMode := noTUI || !isTTY || clioutput.IsStructured(outputFmt)

			summary := loadCockpitSummary(cmd, app, repoRoot, owner, repoName)

			if useTextMode {
				if clioutput.IsStructured(outputFmt) && summary != nil {
					return clioutput.WriteValue(cmd.OutOrStdout(), outputFmt, summary)
				}
				return presenter.RenderTextSummary(cmd.OutOrStdout(), summary)
			}

			m := tuiapp.New()
			m.SetBootstrapApp(app)
			if summary != nil {
				m.SetSummary(summary)
			}

			p := tea.NewProgram(m)
			_, err := p.Run()
			return err
		},
	}

	cmd.Flags().BoolVar(&noTUI, "no-tui", false, "Force text-only output mode")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo for remote-only or mixed inspection")

	return cmd
}

func loadCockpitSummary(cmd *cobra.Command, app bootstrap.App, repoRoot, owner, repoName string) *repo.RepoSummary {
	if owner == "" || repoName == "" {
		owner, repoName = repocontext.ResolveOwnerRepoFromLocalPath(context.Background(), repoRoot)
	}
	if owner == "" || repoName == "" {
		return nil
	}

	var ghClient *ghclient.Client
	if client, err := newGitHubClientFromApp(app); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: GitHub identity error: %v\n", err)
	} else {
		ghClient = client
	}

	assembler := appstate.NewAssembler(ghClient)
	summary, err := assembler.Assemble(context.Background(), owner, repoName, repoRoot)
	if err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not load repository state: %v\n", err)
		return nil
	}
	return summary
}
