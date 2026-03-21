package command

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/collaboration"
	platformgithub "github.com/your-org/gitdex/internal/platform/github"
)

func newReleaseGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Assess release readiness and manage releases",
	}
	cmd.AddCommand(newReleaseAssessCommand(flags, appFn))
	cmd.AddCommand(newReleaseListCommand(flags, appFn))
	cmd.AddCommand(newReleaseShowCommand(flags, appFn))
	cmd.AddCommand(newReleaseCreateCommand(flags, appFn))
	cmd.AddCommand(newReleaseEditCommand(flags, appFn))
	cmd.AddCommand(newReleasePublishCommand(flags, appFn))
	cmd.AddCommand(newReleaseDeleteCommand(flags, appFn))
	return cmd
}

func newReleaseAssessCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "assess",
		Short: "Assess release readiness for a tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			tag, _ := cmd.Flags().GetString("tag")
			if tag == "" {
				return fmt.Errorf("--tag is required")
			}

			owner, repoName := parseRepoFlag(repoFlag, "")
			if owner == "" || repoName == "" {
				return fmt.Errorf("--repo owner/repo is required")
			}

			ghClient, err := requireGitHubClient(app, "release assess")
			if err != nil {
				return err
			}
			engine := collaboration.NewGitHubReleaseEngine(ghClient)
			result, err := engine.Assess(context.Background(), owner, repoName, tag)
			if err != nil {
				return fmt.Errorf("assess failed: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderReleaseReadiness(cmd.OutOrStdout(), result)
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo (required)")
	_ = cmd.MarkFlagRequired("repo")
	cmd.Flags().String("tag", "", "Release tag (e.g. v1.0.0)")
	_ = cmd.MarkFlagRequired("tag")
	return cmd
}

func newReleaseListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			owner, repoName := parseRepoFlag(repoFlag, "")
			if owner == "" || repoName == "" {
				return fmt.Errorf("--repo owner/repo is required")
			}

			ghClient, err := requireGitHubClient(app, "release list")
			if err != nil {
				return err
			}
			engine := collaboration.NewGitHubReleaseEngine(ghClient)
			releases, err := engine.ListReleases(context.Background(), owner, repoName)
			if err != nil {
				return fmt.Errorf("list failed: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"releases": releases})
			}
			return renderReleaseList(cmd.OutOrStdout(), releases)
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo (required)")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newReleaseShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show a release by tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			tag, _ := cmd.Flags().GetString("tag")
			if tag == "" {
				return fmt.Errorf("--tag is required")
			}
			owner, repoName := parseRepoFlag(repoFlag, "")
			if owner == "" || repoName == "" {
				return fmt.Errorf("--repo owner/repo is required")
			}
			ghClient, err := requireGitHubClient(app, "release show")
			if err != nil {
				return err
			}
			release, err := ghClient.GetReleaseByTag(context.Background(), owner, repoName, tag)
			if err != nil {
				return fmt.Errorf("show failed: %w", err)
			}
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, release)
			}
			return renderReleaseDetail(cmd.OutOrStdout(), release)
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo (required)")
	_ = cmd.MarkFlagRequired("repo")
	cmd.Flags().String("tag", "", "Release tag (e.g. v1.0.0)")
	_ = cmd.MarkFlagRequired("tag")
	return cmd
}

func newReleaseCreateCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a release",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			tag, _ := cmd.Flags().GetString("tag")
			name, _ := cmd.Flags().GetString("name")
			body, _ := cmd.Flags().GetString("notes")
			draft, _ := cmd.Flags().GetBool("draft")
			prerelease, _ := cmd.Flags().GetBool("prerelease")
			if tag == "" {
				return fmt.Errorf("--tag is required")
			}
			if strings.TrimSpace(name) == "" {
				name = tag
			}
			owner, repoName := parseRepoFlag(repoFlag, "")
			if owner == "" || repoName == "" {
				return fmt.Errorf("--repo owner/repo is required")
			}
			ghClient, err := requireGitHubClient(app, "release create")
			if err != nil {
				return err
			}
			release, err := ghClient.CreateRelease(context.Background(), owner, repoName, tag, name, body, draft, prerelease)
			if err != nil {
				return fmt.Errorf("create failed: %w", err)
			}
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, release)
			}
			return renderReleaseDetail(cmd.OutOrStdout(), release)
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo (required)")
	_ = cmd.MarkFlagRequired("repo")
	cmd.Flags().String("tag", "", "Release tag (e.g. v1.0.0)")
	_ = cmd.MarkFlagRequired("tag")
	cmd.Flags().String("name", "", "Release name (defaults to tag)")
	cmd.Flags().String("notes", "", "Release notes/body")
	cmd.Flags().Bool("draft", false, "Create as draft")
	cmd.Flags().Bool("prerelease", false, "Mark as prerelease")
	return cmd
}

func newReleaseEditCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a release by tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			tagFlag, _ := cmd.Flags().GetString("tag")
			newTag, _ := cmd.Flags().GetString("new-tag")
			if tagFlag == "" {
				return fmt.Errorf("--tag is required")
			}
			owner, repoName := parseRepoFlag(repoFlag, "")
			if owner == "" || repoName == "" {
				return fmt.Errorf("--repo owner/repo is required")
			}
			ghClient, err := requireGitHubClient(app, "release edit")
			if err != nil {
				return err
			}
			release, err := ghClient.GetReleaseByTag(context.Background(), owner, repoName, tagFlag)
			if err != nil {
				return fmt.Errorf("lookup failed: %w", err)
			}
			if strings.TrimSpace(newTag) == "" && !cmd.Flags().Changed("name") && !cmd.Flags().Changed("notes") && !cmd.Flags().Changed("draft") && !cmd.Flags().Changed("prerelease") {
				return fmt.Errorf("no changes specified; provide --new-tag, --name, --notes, --draft, or --prerelease")
			}
			outTag := release.TagName
			if strings.TrimSpace(newTag) != "" {
				outTag = newTag
			}
			outName := release.Name
			if cmd.Flags().Changed("name") {
				outName, _ = cmd.Flags().GetString("name")
			}
			outBody := release.Body
			if cmd.Flags().Changed("notes") {
				outBody, _ = cmd.Flags().GetString("notes")
			}
			draft := release.Draft
			if cmd.Flags().Changed("draft") {
				draft, _ = cmd.Flags().GetBool("draft")
			}
			prerelease := release.Prerelease
			if cmd.Flags().Changed("prerelease") {
				prerelease, _ = cmd.Flags().GetBool("prerelease")
			}
			updated, err := ghClient.UpdateRelease(context.Background(), owner, repoName, release.ID, outTag, outName, outBody, draft, prerelease)
			if err != nil {
				return fmt.Errorf("edit failed: %w", err)
			}
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, updated)
			}
			return renderReleaseDetail(cmd.OutOrStdout(), updated)
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo (required)")
	_ = cmd.MarkFlagRequired("repo")
	cmd.Flags().String("tag", "", "Existing release tag")
	_ = cmd.MarkFlagRequired("tag")
	cmd.Flags().String("new-tag", "", "Updated tag")
	cmd.Flags().String("name", "", "Updated release name")
	cmd.Flags().String("notes", "", "Updated release notes/body")
	cmd.Flags().Bool("draft", false, "Set draft state")
	cmd.Flags().Bool("prerelease", false, "Set prerelease state")
	return cmd
}

func newReleasePublishCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish a draft release by tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			tag, _ := cmd.Flags().GetString("tag")
			if tag == "" {
				return fmt.Errorf("--tag is required")
			}
			owner, repoName := parseRepoFlag(repoFlag, "")
			if owner == "" || repoName == "" {
				return fmt.Errorf("--repo owner/repo is required")
			}
			ghClient, err := requireGitHubClient(app, "release publish")
			if err != nil {
				return err
			}
			release, err := ghClient.GetReleaseByTag(context.Background(), owner, repoName, tag)
			if err != nil {
				return fmt.Errorf("lookup failed: %w", err)
			}
			updated, err := ghClient.PublishRelease(context.Background(), owner, repoName, release.ID)
			if err != nil {
				return fmt.Errorf("publish failed: %w", err)
			}
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, updated)
			}
			return renderReleaseDetail(cmd.OutOrStdout(), updated)
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo (required)")
	_ = cmd.MarkFlagRequired("repo")
	cmd.Flags().String("tag", "", "Release tag")
	_ = cmd.MarkFlagRequired("tag")
	return cmd
}

func newReleaseDeleteCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a release by tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			tag, _ := cmd.Flags().GetString("tag")
			if tag == "" {
				return fmt.Errorf("--tag is required")
			}
			owner, repoName := parseRepoFlag(repoFlag, "")
			if owner == "" || repoName == "" {
				return fmt.Errorf("--repo owner/repo is required")
			}
			ghClient, err := requireGitHubClient(app, "release delete")
			if err != nil {
				return err
			}
			release, err := ghClient.GetReleaseByTag(context.Background(), owner, repoName, tag)
			if err != nil {
				return fmt.Errorf("lookup failed: %w", err)
			}
			if err := ghClient.DeleteRelease(context.Background(), owner, repoName, release.ID); err != nil {
				return fmt.Errorf("delete failed: %w", err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted release %s\n", tag)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo (required)")
	_ = cmd.MarkFlagRequired("repo")
	cmd.Flags().String("tag", "", "Release tag")
	_ = cmd.MarkFlagRequired("tag")
	return cmd
}

func requireGitHubClient(app bootstrap.App, action string) (*platformgithub.Client, error) {
	ghClient, err := newGitHubClientFromApp(app)
	if err != nil {
		return nil, fmt.Errorf("github identity error: %w", err)
	}
	if ghClient == nil {
		return nil, fmt.Errorf("github identity is required for %s", action)
	}
	return ghClient, nil
}

func renderReleaseReadiness(out io.Writer, r *collaboration.ReleaseReadiness) error {
	_, _ = fmt.Fprintf(out, "Release Readiness: %s/%s %s\n\n", r.RepoOwner, r.RepoName, r.Tag)
	_, _ = fmt.Fprintf(out, "Status:  %s\n", r.Status)
	_, _ = fmt.Fprintf(out, "Assessed: %s\n", r.AssessedAt.Format("2006-01-02 15:04:05"))

	if len(r.Blockers) > 0 {
		_, _ = fmt.Fprintf(out, "\nBlockers:\n")
		for _, b := range r.Blockers {
			_, _ = fmt.Fprintf(out, "  - %s\n", b)
		}
	}
	if len(r.IncludedPRs) > 0 {
		_, _ = fmt.Fprintf(out, "\nIncluded PRs: %v\n", r.IncludedPRs)
	}
	if len(r.CheckResults) > 0 {
		_, _ = fmt.Fprintf(out, "\nChecks:\n")
		for _, c := range r.CheckResults {
			_, _ = fmt.Fprintf(out, "  %s: %s - %s\n", c.Name, c.Status, c.Details)
		}
	}
	if r.ApprovalStatus != "" {
		_, _ = fmt.Fprintf(out, "\nApproval: %s\n", r.ApprovalStatus)
	}
	if r.Notes != "" {
		_, _ = fmt.Fprintf(out, "\nNotes: %s\n", r.Notes)
	}
	return nil
}

func renderReleaseList(out io.Writer, releases []collaboration.ReleaseInfo) error {
	_, _ = fmt.Fprintf(out, "Recent Releases\n\n")
	for _, r := range releases {
		_, _ = fmt.Fprintf(out, "  %s  %s  (%s)\n", r.Tag, r.Status, r.PublishedAt.Format("2006-01-02"))
	}
	return nil
}

func renderReleaseDetail(out io.Writer, release *platformgithub.Release) error {
	if release == nil {
		_, _ = fmt.Fprintln(out, "Release not found")
		return nil
	}
	_, _ = fmt.Fprintf(out, "Release: %s\n", firstNonEmpty(release.Name, release.TagName))
	_, _ = fmt.Fprintf(out, "Tag: %s\n", release.TagName)
	_, _ = fmt.Fprintf(out, "Draft: %t\n", release.Draft)
	_, _ = fmt.Fprintf(out, "Prerelease: %t\n", release.Prerelease)
	if !release.PublishedAt.IsZero() {
		_, _ = fmt.Fprintf(out, "Published: %s\n", release.PublishedAt.Format("2006-01-02 15:04:05"))
	}
	if release.HTMLURL != "" {
		_, _ = fmt.Fprintf(out, "URL: %s\n", release.HTMLURL)
	}
	if body := strings.TrimSpace(release.Body); body != "" {
		_, _ = fmt.Fprintf(out, "\n%s\n", body)
	}
	return nil
}
