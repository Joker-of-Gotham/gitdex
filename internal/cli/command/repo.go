package command

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/repocontext"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/gitops"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
)

func newRepoGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Repository operations and state management",
	}
	cmd.AddCommand(newRepoInspectCommand(flags, appFn))
	cmd.AddCommand(newRepoCloneCommand(flags, appFn))
	cmd.AddCommand(newRepoSyncCommand(flags, appFn))
	cmd.AddCommand(newRepoHygieneGroupCommand(flags, appFn))
	cmd.AddCommand(newRepoWorktreeGroupCommand(flags, appFn))
	return cmd
}

type repoInspectReport struct {
	Owner           string                     `json:"owner" yaml:"owner"`
	Repo            string                     `json:"repo" yaml:"repo"`
	Remote          *ghclient.RepositoryDetail `json:"remote,omitempty" yaml:"remote,omitempty"`
	LocalPaths      []string                   `json:"local_paths,omitempty" yaml:"local_paths,omitempty"`
	SelectedLocal   string                     `json:"selected_local,omitempty" yaml:"selected_local,omitempty"`
	LocalInspection *gitops.RepoInspection     `json:"local_inspection,omitempty" yaml:"local_inspection,omitempty"`
	Recommendation  *gitops.SyncRecommendation `json:"recommendation,omitempty" yaml:"recommendation,omitempty"`
}

func newRepoInspectCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string
	var pathFlag string

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect repository state, branches, and upstream divergence",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repoRoot := app.RepoRoot
			if repoRoot == "" {
				repoRoot = app.Config.Paths.RepositoryRoot
			}
			owner, repoName := parseRepoFlag(repoFlag, repoRoot)
			if repoFlag != "" && (owner == "" || repoName == "") {
				return fmt.Errorf("invalid --repo %q; use owner/repo", repoFlag)
			}

			report, err := inspectRepoContext(context.Background(), app, repoRoot, owner, repoName, pathFlag)
			if err != nil {
				return err
			}
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, report)
			}
			return renderRepoInspectReport(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo for remote-aware inspection")
	cmd.Flags().StringVar(&pathFlag, "path", "", "Explicit local clone path to inspect")
	return cmd
}

func newRepoCloneCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string
	var branchFlag string
	var depthFlag int

	cmd := &cobra.Command{
		Use:   "clone [target-dir]",
		Short: "Clone a remote repository into a local workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			owner, repoName := parseRepoFlag(repoFlag, "")
			if owner == "" || repoName == "" {
				return fmt.Errorf("--repo must be set as owner/repo")
			}

			targetDir := ""
			if len(args) > 0 {
				targetDir = strings.TrimSpace(args[0])
			}
			if targetDir == "" {
				targetDir = defaultCloneDir(app, repoName)
			}

			remoteURL := fmt.Sprintf("https://%s/%s/%s.git", effectiveGitHubHost(app), owner, repoName)
			rm := gitops.NewRemoteManager(gitops.NewGitExecutor())
			if err := rm.Clone(context.Background(), remoteURL, targetDir, gitops.CloneOptions{
				Branch:       strings.TrimSpace(branchFlag),
				Depth:        depthFlag,
				SingleBranch: strings.TrimSpace(branchFlag) != "",
			}); err != nil {
				return fmt.Errorf("clone failed: %w", err)
			}

			payload := map[string]any{
				"owner":      owner,
				"repo":       repoName,
				"remote_url": remoteURL,
				"target_dir": targetDir,
			}
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, payload)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Cloned %s/%s into %s\n", owner, repoName, targetDir)
			return err
		},
	}
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo to clone (required)")
	cmd.Flags().StringVar(&branchFlag, "branch", "", "Clone only the specified branch")
	cmd.Flags().IntVar(&depthFlag, "depth", 0, "Shallow clone depth")
	return cmd
}

func inspectRepoContext(ctx context.Context, app bootstrap.App, repoRoot, owner, repoName, explicitPath string) (*repoInspectReport, error) {
	rc, err := repocontext.Resolve(ctx, app, repocontext.ResolveOptions{
		RepoRoot:  repoRoot,
		LocalPath: explicitPath,
		Owner:     owner,
		Repo:      repoName,
	})
	if err != nil {
		return nil, err
	}
	if rc == nil || (rc.Owner == "" && rc.Repo == "" && rc.ActiveLocalPath == "" && len(rc.LocalPaths) == 0) {
		return nil, fmt.Errorf("no repository found; run from a git repository, configure repository_root, or use --repo owner/repo")
	}

	report := &repoInspectReport{
		Owner:      rc.Owner,
		Repo:       rc.Repo,
		LocalPaths: append([]string{}, rc.LocalPaths...),
	}
	localPath := strings.TrimSpace(rc.ActiveLocalPath)

	if rc.Owner != "" && rc.Repo != "" {
		if client, err := newGitHubClientFromApp(app); err == nil && client != nil {
			detail, err := client.GetRepositoryDetail(ctx, rc.Owner, rc.Repo)
			if err != nil {
				return nil, fmt.Errorf("remote inspection failed: %w", err)
			}
			report.Remote = detail
		}
	}

	if localPath == "" {
		return report, nil
	}
	report.LocalPaths = appendUniquePaths(report.LocalPaths, localPath)

	inspector := gitops.NewInspector(gitops.NewGitExecutor())
	inspection, err := inspector.Inspect(ctx, localPath)
	if err != nil {
		if owner == "" || repoName == "" {
			return nil, fmt.Errorf("inspection failed: %w", err)
		}
		return report, nil
	}
	report.SelectedLocal = localPath
	report.LocalInspection = inspection
	report.Recommendation = inspector.Recommend(inspection)
	if len(report.LocalPaths) == 0 {
		report.LocalPaths = []string{localPath}
	}

	return report, nil
}

func discoverLocalClonePaths(ctx context.Context, app bootstrap.App, owner, repoName string) []string {
	rc, err := repocontext.Resolve(ctx, app, repocontext.ResolveOptions{Owner: owner, Repo: repoName})
	if err != nil || rc == nil {
		return nil
	}
	return append([]string{}, rc.LocalPaths...)
}

func selectRepoRootForRemote(app bootstrap.App, repoRoot, owner, repoName string) string {
	rc, err := repocontext.Resolve(context.Background(), app, repocontext.ResolveOptions{
		RepoRoot: repoRoot,
		Owner:    owner,
		Repo:     repoName,
	})
	if err != nil || rc == nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(rc.ActiveLocalPath))
}

func appendUniquePaths(paths []string, extras ...string) []string {
	seen := make(map[string]bool, len(paths)+len(extras))
	out := make([]string, 0, len(paths)+len(extras))
	for _, p := range append(append([]string{}, paths...), extras...) {
		p = filepath.ToSlash(filepath.Clean(strings.TrimSpace(p)))
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

func defaultCloneDir(app bootstrap.App, repoName string) string {
	return repocontext.DefaultCloneDir(app, repoName)
}

func effectiveGitHubHost(app bootstrap.App) string {
	rc, _ := repocontext.Resolve(context.Background(), app, repocontext.ResolveOptions{})
	if rc != nil && strings.TrimSpace(rc.Host) != "" {
		return strings.TrimSpace(rc.Host)
	}
	return "github.com"
}

func parseRemoteURL(repoRoot string) (string, string) {
	return repocontext.ResolveOwnerRepoFromLocalPath(context.Background(), repoRoot)
}

func newRepoSyncCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Preview or execute controlled upstream synchronization",
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)
			preview, _ := c.Flags().GetBool("preview")
			execute, _ := c.Flags().GetBool("execute")

			if !preview && !execute {
				return fmt.Errorf("specify --preview or --execute")
			}

			repoRoot := app.RepoRoot
			if repoRoot == "" {
				repoRoot = app.Config.Paths.RepositoryRoot
			}
			if repoRoot == "" {
				return fmt.Errorf("no repository found; run from a git repository or configure repository_root")
			}

			inspector := gitops.NewInspector(gitops.NewGitExecutor())
			inspection, err := inspector.Inspect(context.Background(), repoRoot)
			if err != nil {
				return fmt.Errorf("inspection failed: %w", err)
			}

			rec := inspector.Recommend(inspection)
			syncer := gitops.NewSyncer(gitops.NewGitExecutor())

			if preview {
				prev, err := syncer.Preview(context.Background(), inspection, rec)
				if err != nil {
					return fmt.Errorf("preview failed: %w", err)
				}
				if clioutput.IsStructured(format) {
					return clioutput.WriteValue(c.OutOrStdout(), format, prev)
				}
				return renderSyncPreview(c.OutOrStdout(), prev)
			}

			result, err := syncer.Execute(context.Background(), inspection, rec)
			if err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, result)
			}
			return renderSyncResult(c.OutOrStdout(), result)
		},
	}
	cmd.Flags().Bool("preview", false, "Preview sync impact without executing")
	cmd.Flags().Bool("execute", false, "Execute the sync action")
	return cmd
}

func renderInspection(out io.Writer, insp *gitops.RepoInspection, rec *gitops.SyncRecommendation) error {
	_, _ = fmt.Fprintf(out, "═══ Repository Inspection ═══\n\n")
	_, _ = fmt.Fprintf(out, "Path:     %s\n", insp.RepoPath)
	_, _ = fmt.Fprintf(out, "Branch:   %s\n", insp.LocalBranch)
	if insp.RemoteBranch != "" {
		_, _ = fmt.Fprintf(out, "Upstream: %s\n", insp.RemoteBranch)
	}
	_, _ = fmt.Fprintf(out, "State:    %s\n", insp.Divergence)

	if insp.Ahead > 0 || insp.Behind > 0 {
		_, _ = fmt.Fprintf(out, "Ahead:    %d  Behind: %d\n", insp.Ahead, insp.Behind)
	}
	if insp.HasUncommitted {
		_, _ = fmt.Fprintf(out, "Warning:  uncommitted changes detected\n")
	}
	if insp.HasUntracked {
		_, _ = fmt.Fprintf(out, "Warning:  untracked files detected\n")
	}

	if rec != nil {
		_, _ = fmt.Fprintf(out, "\n── Recommended Action ──\n")
		_, _ = fmt.Fprintf(out, "  Action: %s\n", rec.Action)
		_, _ = fmt.Fprintf(out, "  Risk:   %s\n", rec.RiskLevel)
		_, _ = fmt.Fprintf(out, "  %s\n", rec.Description)
		if rec.Previewable {
			_, _ = fmt.Fprintf(out, "\n  → gitdex repo sync --preview    (preview impact)\n")
			_, _ = fmt.Fprintf(out, "  → gitdex repo sync --execute    (execute sync)\n")
		}
	}

	return nil
}

func renderRepoInspectReport(out io.Writer, report *repoInspectReport) error {
	if report == nil {
		_, err := fmt.Fprintln(out, "No repository data available")
		return err
	}

	if report.Owner != "" && report.Repo != "" {
		_, _ = fmt.Fprintf(out, "Remote Repository: %s/%s\n", report.Owner, report.Repo)
		if report.Remote != nil {
			_, _ = fmt.Fprintf(out, "  Default branch: %s\n", report.Remote.DefaultBranch)
			_, _ = fmt.Fprintf(out, "  Language:       %s\n", report.Remote.Language)
			_, _ = fmt.Fprintf(out, "  Stars/Forks:    %d / %d\n", report.Remote.Stars, report.Remote.Forks)
			_, _ = fmt.Fprintf(out, "  Open issues:    %d\n", report.Remote.OpenIssues)
			if report.Remote.Description != "" {
				_, _ = fmt.Fprintf(out, "  Description:    %s\n", report.Remote.Description)
			}
		}
		if len(report.LocalPaths) > 0 {
			_, _ = fmt.Fprintf(out, "  Local clones:   %s\n", strings.Join(report.LocalPaths, ", "))
		} else {
			_, _ = fmt.Fprintf(out, "  Local clones:   none detected\n")
			_, _ = fmt.Fprintf(out, "  Next step:      gitdex repo clone --repo %s/%s\n", report.Owner, report.Repo)
		}
		_, _ = fmt.Fprintln(out)
	}

	if report.LocalInspection != nil {
		return renderInspection(out, report.LocalInspection, report.Recommendation)
	}

	_, err := fmt.Fprintln(out, "No local clone selected for branch/divergence inspection.")
	return err
}

func renderSyncPreview(out io.Writer, p *gitops.SyncPreview) error {
	_, _ = fmt.Fprintf(out, "═══ Sync Preview ═══\n\n")
	_, _ = fmt.Fprintf(out, "Strategy:      %s\n", p.MergeStrategy)
	_, _ = fmt.Fprintf(out, "Affected files: %d\n", p.AffectedFiles)
	_, _ = fmt.Fprintf(out, "Conflict risk: %s\n", p.ConflictRisk)
	_, _ = fmt.Fprintf(out, "Description:   %s\n", p.Description)
	return nil
}

func newRepoHygieneGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hygiene",
		Short: "Run low-risk repository hygiene and maintenance tasks",
	}
	cmd.AddCommand(newRepoHygieneListCommand(flags, appFn))
	cmd.AddCommand(newRepoHygieneRunCommand(flags, appFn))
	return cmd
}

func newRepoHygieneListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available hygiene tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			tasks := gitops.SupportedHygieneTasks()

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"tasks": tasks})
			}
			return renderHygieneTaskList(cmd.OutOrStdout(), tasks)
		},
	}
}

func newRepoHygieneRunCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "run [action]",
		Short: "Execute a hygiene task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			action := gitops.HygieneAction(args[0])

			repoRoot := app.RepoRoot
			if repoRoot == "" {
				repoRoot = app.Config.Paths.RepositoryRoot
			}
			if repoRoot == "" {
				return fmt.Errorf("no repository found; run from a git repository or configure repository_root")
			}

			executor := gitops.NewHygieneExecutor(gitops.NewGitExecutor())
			result, err := executor.Execute(context.Background(), repoRoot, action)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderHygieneResult(cmd.OutOrStdout(), result)
		},
	}
}

func renderHygieneTaskList(out io.Writer, tasks []gitops.HygieneTask) error {
	_, _ = fmt.Fprintf(out, "═══ Available Hygiene Tasks ═══\n\n")
	for _, t := range tasks {
		_, _ = fmt.Fprintf(out, "  %-24s  %s (risk: %s)\n", string(t.Action), t.Description, t.RiskLevel)
		_, _ = fmt.Fprintf(out, "    Reversible: %v | %s\n\n", t.Reversible, t.EstimatedImpact)
	}
	return nil
}

func renderHygieneResult(out io.Writer, r *gitops.HygieneResult) error {
	if r.Success {
		_, _ = fmt.Fprintf(out, "Hygiene task completed successfully.\n")
		_, _ = fmt.Fprintf(out, "Action:   %s\n", string(r.Action))
		_, _ = fmt.Fprintf(out, "Files:    %d\n", r.FilesAffected)
		_, _ = fmt.Fprintf(out, "Branches: %d\n", r.BranchesAffected)
		_, _ = fmt.Fprintf(out, "%s\n", r.Summary)
	} else {
		_, _ = fmt.Fprintf(out, "Hygiene task failed.\n")
		_, _ = fmt.Fprintf(out, "Action: %s\n", string(r.Action))
		if r.ErrorMessage != "" {
			_, _ = fmt.Fprintf(out, "Error: %s\n", r.ErrorMessage)
		}
		_, _ = fmt.Fprintf(out, "%s\n", r.Summary)
	}
	return nil
}

func newRepoWorktreeGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "Manage isolated worktrees for controlled local file modifications",
	}
	cmd.AddCommand(newRepoWorktreeCreateCommand(flags, appFn))
	cmd.AddCommand(newRepoWorktreeInspectCommand(flags, appFn))
	cmd.AddCommand(newRepoWorktreeDiffCommand(flags, appFn))
	cmd.AddCommand(newRepoWorktreeDiscardCommand(flags, appFn))
	return cmd
}

func newRepoWorktreeCreateCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an isolated worktree",
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			repoRoot := app.RepoRoot
			if repoRoot == "" {
				repoRoot = app.Config.Paths.RepositoryRoot
			}
			if repoRoot == "" {
				return fmt.Errorf("no repository found; run from a git repository or configure repository_root")
			}

			branch, _ := c.Flags().GetString("branch")
			if branch == "" {
				return fmt.Errorf("--branch is required")
			}

			worktreeDir, _ := c.Flags().GetString("worktree-dir")
			if worktreeDir == "" {
				worktreeDir = filepath.Join(repoRoot, "..", "gitdex-worktree-"+branch)
			}

			config := gitops.WorktreeConfig{
				RepoPath:    repoRoot,
				Branch:      branch,
				WorktreeDir: worktreeDir,
			}
			mgr := gitops.NewWorktreeManager(gitops.NewGitExecutor())
			wt, err := mgr.Create(context.Background(), config)
			if err != nil {
				return fmt.Errorf("create worktree failed: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, wt)
			}
			return renderWorktree(c.OutOrStdout(), wt)
		},
	}
	cmd.Flags().String("branch", "", "Branch for the worktree (required)")
	cmd.Flags().String("worktree-dir", "", "Directory for the worktree (default: ../gitdex-worktree-<branch>)")
	return cmd
}

func newRepoWorktreeInspectCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect worktree state",
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			worktreeDir, _ := c.Flags().GetString("worktree-dir")
			if worktreeDir == "" {
				worktreeDir = app.Config.Paths.WorkingDir
			}
			if worktreeDir == "" {
				return fmt.Errorf("no worktree directory; specify --worktree-dir or run from within a worktree")
			}

			mgr := gitops.NewWorktreeManager(gitops.NewGitExecutor())
			wt, err := mgr.Inspect(context.Background(), worktreeDir)
			if err != nil {
				return fmt.Errorf("inspect worktree failed: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, wt)
			}
			return renderWorktree(c.OutOrStdout(), wt)
		},
	}
	cmd.Flags().String("worktree-dir", "", "Path to the worktree directory")
	return cmd
}

func newRepoWorktreeDiffCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff from worktree",
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			worktreeDir, _ := c.Flags().GetString("worktree-dir")
			if worktreeDir == "" {
				worktreeDir = app.Config.Paths.WorkingDir
			}
			if worktreeDir == "" {
				return fmt.Errorf("no worktree directory; specify --worktree-dir or run from within a worktree")
			}

			mgr := gitops.NewWorktreeManager(gitops.NewGitExecutor())
			diff, err := mgr.Diff(context.Background(), worktreeDir)
			if err != nil {
				return fmt.Errorf("diff failed: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, map[string]string{"diff": diff})
			}
			_, err = fmt.Fprint(c.OutOrStdout(), diff)
			return err
		},
	}
	cmd.Flags().String("worktree-dir", "", "Path to the worktree directory")
	return cmd
}

func newRepoWorktreeDiscardCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discard",
		Short: "Remove worktree safely",
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			worktreeDir, _ := c.Flags().GetString("worktree-dir")
			if worktreeDir == "" {
				worktreeDir = app.Config.Paths.WorkingDir
			}
			if worktreeDir == "" {
				return fmt.Errorf("no worktree directory; specify --worktree-dir or run from within a worktree")
			}

			mgr := gitops.NewWorktreeManager(gitops.NewGitExecutor())
			if err := mgr.Discard(context.Background(), worktreeDir); err != nil {
				return fmt.Errorf("discard worktree failed: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, map[string]string{"status": "discarded"})
			}
			_, err := fmt.Fprintf(c.OutOrStdout(), "Worktree discarded: %s\n", worktreeDir)
			return err
		},
	}
	cmd.Flags().String("worktree-dir", "", "Path to the worktree directory")
	return cmd
}

func renderWorktree(out io.Writer, wt *gitops.Worktree) error {
	_, _ = fmt.Fprintf(out, "═══ Worktree ═══\n\n")
	_, _ = fmt.Fprintf(out, "Repo:       %s\n", wt.Config.RepoPath)
	_, _ = fmt.Fprintf(out, "Branch:     %s\n", wt.Config.Branch)
	_, _ = fmt.Fprintf(out, "Worktree:   %s\n", wt.Config.WorktreeDir)
	_, _ = fmt.Fprintf(out, "Status:     %s\n", wt.Status)
	_, _ = fmt.Fprintf(out, "Created:    %s\n", wt.CreatedAt.Format("2006-01-02 15:04:05 MST"))
	if wt.DiffSummary != "" {
		_, _ = fmt.Fprintf(out, "\nDiff summary:\n%s\n", wt.DiffSummary)
	}
	return nil
}

func renderSyncResult(out io.Writer, r *gitops.SyncResult) error {
	if r.Success {
		_, _ = fmt.Fprintf(out, "Sync completed successfully.\n")
		_, _ = fmt.Fprintf(out, "Files changed: %d\n", r.FilesChanged)
		_, _ = fmt.Fprintf(out, "%s\n", r.Description)
	} else {
		_, _ = fmt.Fprintf(out, "Sync blocked.\n")
		if r.ErrorMessage != "" {
			_, _ = fmt.Fprintf(out, "Error: %s\n", r.ErrorMessage)
		}
		if r.Conflicts > 0 {
			_, _ = fmt.Fprintf(out, "Conflicts: %d\n", r.Conflicts)
		}
		_, _ = fmt.Fprintf(out, "%s\n", r.Description)
		_, _ = fmt.Fprintf(out, "\n── Next Step ──\n")
		_, _ = fmt.Fprintf(out, "  Resolve conflicts manually or create a governed merge plan:\n")
		_, _ = fmt.Fprintf(out, "  → gitdex plan compile \"merge upstream changes\"\n")
	}
	return nil
}
