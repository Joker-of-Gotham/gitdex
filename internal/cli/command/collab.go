package command

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/collaboration"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
)

var (
	collabObjectStoreOverride  collaboration.ObjectStore
	collabContextStoreOverride collaboration.ContextStore
)

// SetCollabObjectStoreForTest allows integration tests to inject a custom object store.
func SetCollabObjectStoreForTest(s collaboration.ObjectStore) func() {
	prev := collabObjectStoreOverride
	collabObjectStoreOverride = s
	return func() { collabObjectStoreOverride = prev }
}

// SetCollabContextStoreForTest allows integration tests to inject a custom context store.
func SetCollabContextStoreForTest(s collaboration.ContextStore) func() {
	prev := collabContextStoreOverride
	collabContextStoreOverride = s
	return func() { collabContextStoreOverride = prev }
}

func getCollabObjectStore(appFn func() bootstrap.App) collaboration.ObjectStore {
	if collabObjectStoreOverride != nil {
		return collabObjectStoreOverride
	}
	return appFn().StorageProvider.ObjectStore()
}

func getCollabContextStore(appFn func() bootstrap.App) collaboration.ContextStore {
	if collabContextStoreOverride != nil {
		return collabContextStoreOverride
	}
	return appFn().StorageProvider.ContextStore()
}

func newCollabGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collab",
		Short: "Collaboration triage, linking, and context",
	}
	cmd.AddCommand(newCollabTriageCommand(flags, appFn))
	cmd.AddCommand(newCollabSummaryCommand(flags, appFn))
	cmd.AddCommand(newCollabLinkCommand(flags, appFn))
	cmd.AddCommand(newCollabContextCommand(flags, appFn))
	cmd.AddCommand(newCollabListCommand(flags, appFn))
	cmd.AddCommand(newCollabShowCommand(flags, appFn))
	cmd.AddCommand(newCollabCreateCommand(flags, appFn))
	cmd.AddCommand(newCollabCommentCommand(flags, appFn))
	cmd.AddCommand(newCollabCloseCommand(flags, appFn))
	cmd.AddCommand(newCollabReopenCommand(flags, appFn))
	return cmd
}

func newCollabTriageCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Triage and prioritize collaboration objects",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			collabObjectStore := getCollabObjectStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			owner, repoName := parseRepoFlag(repoFlag, app.RepoRoot)
			if owner == "" || repoName == "" {
				return fmt.Errorf("repository required; use --repo owner/repo or run from a git repository")
			}

			filter := &collaboration.ObjectFilter{
				RepoOwner: owner,
				RepoName:  repoName,
				State:     "open",
			}
			var objects []*collaboration.CollaborationObject
			if ghClient, err := newGitHubClientFromApp(app); err == nil && ghClient != nil {
				objects, err = loadLiveCollaborationObjects(context.Background(), ghClient, filter)
				if err != nil {
					return fmt.Errorf("list live objects: %w", err)
				}
			} else {
				objects, err = collabObjectStore.ListObjects(context.Background(), filter)
				if err != nil {
					return fmt.Errorf("list objects: %w", err)
				}
			}

			engine := collaboration.NewRuleBasedTriageEngine()
			var results []collaboration.TriageResult
			for _, obj := range objects {
				res, err := engine.Triage(context.Background(), obj)
				if err != nil {
					continue
				}
				results = append(results, *res)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"triage_results": results})
			}
			return renderTriageText(cmd.OutOrStdout(), results)
		},
	}
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository (owner/repo)")
	return cmd
}

func newCollabSummaryCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var repoFlag, periodFlag string

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Generate activity summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			collabObjectStore := getCollabObjectStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			owner, repoName := parseRepoFlag(repoFlag, app.RepoRoot)
			if owner == "" || repoName == "" {
				return fmt.Errorf("repository required; use --repo owner/repo or run from a git repository")
			}

			if periodFlag == "" {
				periodFlag = "7d"
			}

			filter := &collaboration.ObjectFilter{
				RepoOwner: owner,
				RepoName:  repoName,
			}
			var objects []*collaboration.CollaborationObject
			if ghClient, err := newGitHubClientFromApp(app); err == nil && ghClient != nil {
				objects, err = loadLiveCollaborationObjects(context.Background(), ghClient, filter)
				if err != nil {
					return fmt.Errorf("list live objects: %w", err)
				}
			} else {
				objects, err = collabObjectStore.ListObjects(context.Background(), filter)
				if err != nil {
					return fmt.Errorf("list objects: %w", err)
				}
			}

			engine := collaboration.NewRuleBasedTriageEngine()
			summary, err := engine.Summarize(context.Background(), objects, periodFlag)
			if err != nil {
				return fmt.Errorf("summarize: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, summary)
			}
			return renderSummaryText(cmd.OutOrStdout(), summary)
		},
	}
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository (owner/repo)")
	cmd.Flags().StringVar(&periodFlag, "period", "7d", "Period (e.g. 7d, 24h)")
	return cmd
}

func newCollabLinkCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var linkTypeFlag string

	cmd := &cobra.Command{
		Use:   "link <source_ref> <target_ref>",
		Short: "Link objects",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			collabContextStore := getCollabContextStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			if linkTypeFlag == "" {
				linkTypeFlag = "relates_to"
			}
			lt := collaboration.LinkType(linkTypeFlag)
			if !lt.Valid() {
				return fmt.Errorf("invalid link type %q; use blocks, blocked_by, relates_to, duplicate_of, parent_of, child_of", linkTypeFlag)
			}

			link := &collaboration.ObjectLink{
				SourceRef: args[0],
				TargetRef: args[1],
				LinkType:  lt,
				CreatedAt: time.Now().UTC(),
			}
			tc, _ := collabContextStore.GetByObjectRef(context.Background(), args[0])
			if tc == nil {
				tc = &collaboration.TaskContext{
					PrimaryObjectRef: args[0],
					LinkedObjects:    []collaboration.ObjectLink{},
				}
			}
			tc.LinkedObjects = append(tc.LinkedObjects, *link)
			if err := collabContextStore.SaveContext(context.Background(), tc); err != nil {
				return fmt.Errorf("save link: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, link)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Linked %s --[%s]--> %s\n", args[0], linkTypeFlag, args[1])
			return err
		},
	}
	cmd.Flags().StringVar(&linkTypeFlag, "type", "relates_to", "Link type: blocks, blocked_by, relates_to, duplicate_of, parent_of, child_of")
	return cmd
}

func newCollabContextCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "context <object_ref>",
		Short: "Show full context for an object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			collabContextStore := getCollabContextStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			ctx, err := collabContextStore.GetByObjectRef(context.Background(), args[0])
			if err != nil || ctx == nil {
				ctx = &collaboration.TaskContext{
					PrimaryObjectRef: args[0],
					LinkedObjects:    nil,
					RelatedTasks:     nil,
					Notes:            "",
				}
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, ctx)
			}
			return renderContextText(cmd.OutOrStdout(), ctx)
		},
	}
}

func parseRepoFlag(repoFlag, repoRoot string) (owner, repo string) {
	if repoFlag != "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
		return "", ""
	}
	if repoRoot != "" {
		owner, name := parseRemoteURL(repoRoot)
		return owner, name
	}
	return "", ""
}

func renderTriageText(out io.Writer, results []collaboration.TriageResult) error {
	_, _ = fmt.Fprintf(out, "═══ Triage Results ═══\n\n")
	for _, r := range results {
		_, _ = fmt.Fprintf(out, "  %s  [%s] %s\n", r.ObjectRef, r.Priority, r.Reason)
		_, _ = fmt.Fprintf(out, "      Action: %s\n", r.SuggestedAction)
		if len(r.Tags) > 0 {
			_, _ = fmt.Fprintf(out, "      Tags: %s\n", strings.Join(r.Tags, ", "))
		}
		_, _ = fmt.Fprintf(out, "\n")
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintf(out, "  No objects to triage.\n")
	}
	return nil
}

func renderSummaryText(out io.Writer, s *collaboration.ActivitySummary) error {
	_, _ = fmt.Fprintf(out, "═══ Activity Summary: %s/%s ═══\n\n", s.RepoOwner, s.RepoName)
	_, _ = fmt.Fprintf(out, "Period: %s | Total: %d\n\n", s.Period, s.TotalObjects)
	if len(s.ByType) > 0 {
		_, _ = fmt.Fprintf(out, "By type:\n")
		for t, c := range s.ByType {
			_, _ = fmt.Fprintf(out, "  %s: %d\n", t, c)
		}
		_, _ = fmt.Fprintf(out, "\n")
	}
	if len(s.ByPriority) > 0 {
		_, _ = fmt.Fprintf(out, "By priority:\n")
		for p, c := range s.ByPriority {
			_, _ = fmt.Fprintf(out, "  %s: %d\n", p, c)
		}
		_, _ = fmt.Fprintf(out, "\n")
	}
	if len(s.TopItems) > 0 {
		_, _ = fmt.Fprintf(out, "Top items:\n")
		for _, r := range s.TopItems {
			_, _ = fmt.Fprintf(out, "  %s [%s] %s\n", r.ObjectRef, r.Priority, r.Reason)
		}
	}
	_, _ = fmt.Fprintf(out, "\nGenerated: %s\n", s.GeneratedAt.Format("2006-01-02 15:04:05 MST"))
	return nil
}

func renderContextText(out io.Writer, c *collaboration.TaskContext) error {
	_, _ = fmt.Fprintf(out, "═══ Context: %s ═══\n\n", c.PrimaryObjectRef)
	if len(c.LinkedObjects) > 0 {
		_, _ = fmt.Fprintf(out, "Linked objects:\n")
		for _, l := range c.LinkedObjects {
			_, _ = fmt.Fprintf(out, "  %s --[%s]--> %s\n", l.SourceRef, l.LinkType, l.TargetRef)
		}
		_, _ = fmt.Fprintf(out, "\n")
	}
	if len(c.RelatedTasks) > 0 {
		_, _ = fmt.Fprintf(out, "Related tasks: %s\n\n", strings.Join(c.RelatedTasks, ", "))
	}
	if c.Notes != "" {
		_, _ = fmt.Fprintf(out, "Notes: %s\n", c.Notes)
	}
	if len(c.LinkedObjects) == 0 && len(c.RelatedTasks) == 0 && c.Notes == "" {
		_, _ = fmt.Fprintf(out, "  No context found.\n")
	}
	return nil
}

// --- Story 4.1/4.2: list, show, create, comment, close, reopen ---

func newCollabListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List collaboration objects",
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			collabObjectStore := getCollabObjectStore(appFn)
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			typeFlag, _ := c.Flags().GetString("type")
			stateFlag, _ := c.Flags().GetString("state")
			repoFlag, _ := c.Flags().GetString("repo")
			labelFlag, _ := c.Flags().GetString("label")

			filter := &collaboration.ObjectFilter{
				State:       stateFlag,
				Labels:      []string{},
				SearchQuery: "",
			}
			if typeFlag != "" {
				ot := parseCollabObjectType(typeFlag)
				if ot == "" {
					return fmt.Errorf("invalid --type %q; use issue, pr, or discussion", typeFlag)
				}
				filter.ObjectType = ot
			}
			if labelFlag != "" {
				filter.Labels = []string{labelFlag}
			}
			if repoFlag != "" {
				owner, repoName := parseRepoSpec(repoFlag)
				if owner == "" || repoName == "" {
					return fmt.Errorf("invalid --repo %q; use owner/repo", repoFlag)
				}
				filter.RepoOwner = owner
				filter.RepoName = repoName
			}

			var (
				objects []*collaboration.CollaborationObject
				err     error
			)
			if ghClient, ghErr := newGitHubClientFromApp(app); ghErr == nil && ghClient != nil && filter.RepoOwner != "" && filter.RepoName != "" {
				objects, err = loadLiveCollaborationObjects(context.Background(), ghClient, filter)
				if err != nil {
					return fmt.Errorf("list failed: %w", err)
				}
			} else {
				objects, err = collabObjectStore.ListObjects(context.Background(), filter)
				if err != nil {
					return fmt.Errorf("list failed: %w", err)
				}
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, map[string]any{"objects": objects})
			}
			return renderCollabListText(c.OutOrStdout(), objects)
		},
	}
	cmd.Flags().String("type", "", "Object type: issue, pr, discussion")
	cmd.Flags().String("state", "open", "State filter: open, closed, all")
	cmd.Flags().String("repo", "", "Repository (owner/repo)")
	cmd.Flags().String("label", "", "Filter by label")
	return cmd
}

func newCollabShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <owner/repo#number>",
		Short: "Show object details with body",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			collabObjectStore := getCollabObjectStore(appFn)
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			owner, repoName, number, err := parseObjectRef(args[0])
			if err != nil {
				return err
			}

			var obj *collaboration.CollaborationObject
			if ghClient, ghErr := newGitHubClientFromApp(app); ghErr == nil && ghClient != nil {
				obj, err = fetchLiveCollaborationObject(context.Background(), ghClient, owner, repoName, number)
			} else {
				obj, err = collabObjectStore.GetByRepoAndNumber(context.Background(), owner, repoName, number)
			}
			if err != nil {
				return fmt.Errorf("object not found: %s/%s#%d", owner, repoName, number)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, obj)
			}
			return renderCollabShowText(c.OutOrStdout(), obj)
		},
	}
}

func newCollabCreateCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a collaboration object",
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			collabObjectStore := getCollabObjectStore(appFn)
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			typeFlag, _ := c.Flags().GetString("type")
			repoFlag, _ := c.Flags().GetString("repo")
			titleFlag, _ := c.Flags().GetString("title")
			bodyFlag, _ := c.Flags().GetString("body")

			if typeFlag == "" {
				return fmt.Errorf("--type is required")
			}
			if repoFlag == "" {
				return fmt.Errorf("--repo is required")
			}
			if titleFlag == "" {
				return fmt.Errorf("--title is required")
			}

			ot := parseCollabObjectType(typeFlag)
			if ot == "" {
				return fmt.Errorf("invalid --type %q; use issue, pr, or discussion", typeFlag)
			}

			owner, repoName := parseRepoSpec(repoFlag)
			if owner == "" || repoName == "" {
				return fmt.Errorf("invalid --repo %q; use owner/repo", repoFlag)
			}

			req := &collaboration.MutationRequest{
				MutationType: collaboration.MutationCreate,
				ObjectType:   ot,
				RepoOwner:    owner,
				RepoName:     repoName,
				Title:        titleFlag,
				Body:         bodyFlag,
			}

			ghClient, ghErr := newGitHubClientFromApp(app)
			if ghErr != nil || ghClient == nil {
				return fmt.Errorf("GitHub authentication required; run 'gitdex setup' to configure")
			}

			var result *collaboration.MutationResult
			var err error
			if ot == collaboration.ObjectTypeDiscussion {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				discussion, derr := ghClient.CreateDiscussion(ctx, owner, repoName, titleFlag, bodyFlag)
				if derr != nil {
					return fmt.Errorf("create failed: %w", derr)
				}
				obj := liveObjectFromDiscussion(discussion, owner, repoName)
				result = &collaboration.MutationResult{
					Request: *req,
					Success: true,
					Object:  obj,
					Message: "created",
				}
			} else {
				collabMutationEng := collaboration.NewGitHubMutationEngine(ghClient)
				result, err = collabMutationEng.Execute(context.Background(), req)
				if err != nil {
					return fmt.Errorf("create failed: %w", err)
				}
			}
			if result.Object != nil {
				_ = collabObjectStore.SaveObject(context.Background(), result.Object)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, result)
			}
			return renderMutationResultText(c.OutOrStdout(), result)
		},
	}
	cmd.Flags().String("type", "", "Object type: issue, pr, discussion (required)")
	cmd.Flags().String("repo", "", "Repository owner/repo (required)")
	cmd.Flags().String("title", "", "Title (required)")
	cmd.Flags().String("body", "", "Body content")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newCollabCommentCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment <owner/repo#number> --body \"comment\"",
		Short: "Add a comment to an object",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			collabObjectStore := getCollabObjectStore(appFn)
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			owner, repoName, number, err := parseObjectRef(args[0])
			if err != nil {
				return err
			}

			bodyFlag, _ := c.Flags().GetString("body")
			if bodyFlag == "" {
				return fmt.Errorf("--body is required")
			}

			req := &collaboration.MutationRequest{
				MutationType: collaboration.MutationComment,
				RepoOwner:    owner,
				RepoName:     repoName,
				Number:       &number,
				Body:         bodyFlag,
			}

			var collabMutationEng collaboration.MutationEngine
			var ghClient *ghclient.Client
			if client, err := newGitHubClientFromApp(app); err == nil && client != nil {
				ghClient = client
				if obj, fetchErr := fetchLiveCollaborationObject(context.Background(), client, owner, repoName, number); fetchErr == nil && obj != nil {
					req.ObjectType = obj.ObjectType
				}
				collabMutationEng = collaboration.NewGitHubMutationEngine(client)
			} else {
				return fmt.Errorf("GitHub authentication required; run 'gitdex setup' to configure")
			}

			result, err := collabMutationEng.Execute(context.Background(), req)
			if err != nil {
				return fmt.Errorf("comment failed: %w", err)
			}
			if ghClient != nil {
				if obj, fetchErr := fetchLiveCollaborationObject(context.Background(), ghClient, owner, repoName, number); fetchErr == nil && obj != nil {
					result.Object = obj
					_ = collabObjectStore.SaveObject(context.Background(), obj)
				}
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, result)
			}
			return renderMutationResultText(c.OutOrStdout(), result)
		},
	}
	cmd.Flags().String("body", "", "Comment body (required)")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func newCollabCloseCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "close <owner/repo#number>",
		Short: "Close an object",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			collabObjectStore := getCollabObjectStore(appFn)
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			owner, repoName, number, err := parseObjectRef(args[0])
			if err != nil {
				return err
			}

			req := &collaboration.MutationRequest{
				MutationType: collaboration.MutationClose,
				RepoOwner:    owner,
				RepoName:     repoName,
				Number:       &number,
			}

			var collabMutationEng collaboration.MutationEngine
			var ghClient *ghclient.Client
			if client, err := newGitHubClientFromApp(app); err == nil && client != nil {
				ghClient = client
				if obj, fetchErr := fetchLiveCollaborationObject(context.Background(), client, owner, repoName, number); fetchErr == nil && obj != nil {
					req.ObjectType = obj.ObjectType
				}
				collabMutationEng = collaboration.NewGitHubMutationEngine(client)
			} else {
				return fmt.Errorf("GitHub authentication required; run 'gitdex setup' to configure")
			}

			result, err := collabMutationEng.Execute(context.Background(), req)
			if err != nil {
				return fmt.Errorf("close failed: %w", err)
			}
			if ghClient != nil {
				if obj, fetchErr := fetchLiveCollaborationObject(context.Background(), ghClient, owner, repoName, number); fetchErr == nil && obj != nil {
					result.Object = obj
					_ = collabObjectStore.SaveObject(context.Background(), obj)
				}
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, result)
			}
			return renderMutationResultText(c.OutOrStdout(), result)
		},
	}
}

func newCollabReopenCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <owner/repo#number>",
		Short: "Reopen an object",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			collabObjectStore := getCollabObjectStore(appFn)
			format := effectiveOutputFormat(c, *flags, app.Config.Output)

			owner, repoName, number, err := parseObjectRef(args[0])
			if err != nil {
				return err
			}

			req := &collaboration.MutationRequest{
				MutationType: collaboration.MutationReopen,
				RepoOwner:    owner,
				RepoName:     repoName,
				Number:       &number,
			}

			var collabMutationEng collaboration.MutationEngine
			var ghClient *ghclient.Client
			if client, err := newGitHubClientFromApp(app); err == nil && client != nil {
				ghClient = client
				if obj, fetchErr := fetchLiveCollaborationObject(context.Background(), client, owner, repoName, number); fetchErr == nil && obj != nil {
					req.ObjectType = obj.ObjectType
				}
				collabMutationEng = collaboration.NewGitHubMutationEngine(client)
			} else {
				return fmt.Errorf("GitHub authentication required; run 'gitdex setup' to configure")
			}

			result, err := collabMutationEng.Execute(context.Background(), req)
			if err != nil {
				return fmt.Errorf("reopen failed: %w", err)
			}
			if ghClient != nil {
				if obj, fetchErr := fetchLiveCollaborationObject(context.Background(), ghClient, owner, repoName, number); fetchErr == nil && obj != nil {
					result.Object = obj
					_ = collabObjectStore.SaveObject(context.Background(), obj)
				}
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, result)
			}
			return renderMutationResultText(c.OutOrStdout(), result)
		},
	}
}

func parseCollabObjectType(s string) collaboration.ObjectType {
	switch strings.ToLower(s) {
	case "issue":
		return collaboration.ObjectTypeIssue
	case "pr", "pull_request":
		return collaboration.ObjectTypePullRequest
	case "discussion":
		return collaboration.ObjectTypeDiscussion
	case "release":
		return collaboration.ObjectTypeRelease
	case "check_run":
		return collaboration.ObjectTypeCheckRun
	default:
		return ""
	}
}

func parseRepoSpec(s string) (owner, repo string) {
	parts := strings.SplitN(strings.TrimSpace(s), "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func parseObjectRef(s string) (owner, repo string, number int, err error) {
	hash := strings.Index(s, "#")
	if hash < 0 {
		return "", "", 0, fmt.Errorf("invalid ref %q; use owner/repo#number", s)
	}
	owner, repo = parseRepoSpec(s[:hash])
	if owner == "" || repo == "" {
		return "", "", 0, fmt.Errorf("invalid ref %q; use owner/repo#number", s)
	}
	numStr := strings.TrimSpace(s[hash+1:])
	if numStr == "" {
		return "", "", 0, fmt.Errorf("invalid ref %q; use owner/repo#number", s)
	}
	n, e := strconv.Atoi(numStr)
	if e != nil || n < 1 {
		return "", "", 0, fmt.Errorf("invalid number in %q; use owner/repo#number", s)
	}
	return owner, repo, n, nil
}

func liveObjectFromIssue(issue *gh.Issue, owner, repo string) *collaboration.CollaborationObject {
	if issue == nil {
		return nil
	}
	labels := make([]string, 0, len(issue.Labels))
	for _, l := range issue.Labels {
		labels = append(labels, l.GetName())
	}
	assignees := make([]string, 0, len(issue.Assignees))
	for _, a := range issue.Assignees {
		assignees = append(assignees, a.GetLogin())
	}
	objectType := collaboration.ObjectTypeIssue
	if issue.PullRequestLinks != nil {
		objectType = collaboration.ObjectTypePullRequest
	}
	var createdAt, updatedAt time.Time
	if issue.CreatedAt != nil {
		createdAt = issue.CreatedAt.Time
	}
	if issue.UpdatedAt != nil {
		updatedAt = issue.UpdatedAt.Time
	}
	return &collaboration.CollaborationObject{
		ObjectID:      issue.GetNodeID(),
		ObjectType:    objectType,
		RepoOwner:     owner,
		RepoName:      repo,
		Number:        issue.GetNumber(),
		Title:         issue.GetTitle(),
		State:         issue.GetState(),
		Author:        issue.GetUser().GetLogin(),
		Assignees:     assignees,
		Labels:        labels,
		Body:          issue.GetBody(),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		CommentsCount: issue.GetComments(),
		URL:           issue.GetHTMLURL(),
	}
}

func liveObjectFromPullRequest(pr *gh.PullRequest, owner, repo string) *collaboration.CollaborationObject {
	if pr == nil {
		return nil
	}
	labels := make([]string, 0, len(pr.Labels))
	for _, l := range pr.Labels {
		labels = append(labels, l.GetName())
	}
	assignees := make([]string, 0, len(pr.Assignees))
	for _, a := range pr.Assignees {
		assignees = append(assignees, a.GetLogin())
	}
	var createdAt, updatedAt time.Time
	if pr.CreatedAt != nil {
		createdAt = pr.CreatedAt.Time
	}
	if pr.UpdatedAt != nil {
		updatedAt = pr.UpdatedAt.Time
	}
	return &collaboration.CollaborationObject{
		ObjectID:      pr.GetNodeID(),
		ObjectType:    collaboration.ObjectTypePullRequest,
		RepoOwner:     owner,
		RepoName:      repo,
		Number:        pr.GetNumber(),
		Title:         pr.GetTitle(),
		State:         pr.GetState(),
		Author:        pr.GetUser().GetLogin(),
		Assignees:     assignees,
		Labels:        labels,
		Body:          pr.GetBody(),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		CommentsCount: pr.GetComments(),
		URL:           pr.GetHTMLURL(),
	}
}

func liveObjectFromDiscussion(discussion *ghclient.Discussion, owner, repo string) *collaboration.CollaborationObject {
	if discussion == nil {
		return nil
	}
	createdAt, _ := time.Parse(time.RFC3339, discussion.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, discussion.UpdatedAt)
	return &collaboration.CollaborationObject{
		ObjectID:      discussion.ID,
		ObjectType:    collaboration.ObjectTypeDiscussion,
		RepoOwner:     owner,
		RepoName:      repo,
		Number:        discussion.Number,
		Title:         discussion.Title,
		State:         discussion.State,
		Author:        discussion.Author,
		Body:          discussion.Body,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		CommentsCount: discussion.CommentsCount,
		URL:           discussion.URL,
		Milestone:     discussion.Category,
	}
}

func matchesLiveFilter(obj *collaboration.CollaborationObject, filter *collaboration.ObjectFilter) bool {
	if obj == nil || filter == nil {
		return true
	}
	if filter.ObjectType != "" && obj.ObjectType != filter.ObjectType {
		return false
	}
	if filter.State != "" && filter.State != "all" && obj.State != filter.State {
		return false
	}
	if filter.RepoOwner != "" && obj.RepoOwner != filter.RepoOwner {
		return false
	}
	if filter.RepoName != "" && obj.RepoName != filter.RepoName {
		return false
	}
	if filter.Author != "" && !strings.EqualFold(obj.Author, filter.Author) {
		return false
	}
	if filter.Assignee != "" {
		found := false
		for _, a := range obj.Assignees {
			if strings.EqualFold(a, filter.Assignee) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(filter.Labels) > 0 {
		for _, want := range filter.Labels {
			found := false
			for _, have := range obj.Labels {
				if strings.EqualFold(have, want) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	if filter.SearchQuery != "" {
		query := strings.ToLower(strings.TrimSpace(filter.SearchQuery))
		text := strings.ToLower(obj.Title + "\n" + obj.Body)
		if !strings.Contains(text, query) {
			return false
		}
	}
	return true
}

func loadLiveCollaborationObjects(ctx context.Context, client *ghclient.Client, filter *collaboration.ObjectFilter) ([]*collaboration.CollaborationObject, error) {
	if client == nil {
		return nil, fmt.Errorf("github client unavailable")
	}
	if filter == nil || filter.RepoOwner == "" || filter.RepoName == "" {
		return nil, fmt.Errorf("repository filter is required for live collaboration objects")
	}

	state := strings.TrimSpace(filter.State)
	if state == "" {
		state = "open"
	}

	var objects []*collaboration.CollaborationObject
	if filter.ObjectType == "" || filter.ObjectType == collaboration.ObjectTypeIssue {
		issues, err := client.ListIssues(ctx, filter.RepoOwner, filter.RepoName, state)
		if err != nil {
			return nil, err
		}
		for _, issue := range issues {
			if issue.PullRequestLinks != nil {
				continue
			}
			obj := liveObjectFromIssue(issue, filter.RepoOwner, filter.RepoName)
			if matchesLiveFilter(obj, filter) {
				objects = append(objects, obj)
			}
		}
	}
	if filter.ObjectType == "" || filter.ObjectType == collaboration.ObjectTypePullRequest {
		prs, err := client.ListPullRequests(ctx, filter.RepoOwner, filter.RepoName, state)
		if err != nil {
			return nil, err
		}
		for _, pr := range prs {
			obj := liveObjectFromPullRequest(pr, filter.RepoOwner, filter.RepoName)
			if matchesLiveFilter(obj, filter) {
				objects = append(objects, obj)
			}
		}
	}
	if filter.ObjectType == "" || filter.ObjectType == collaboration.ObjectTypeDiscussion {
		discussions, err := client.ListDiscussions(ctx, filter.RepoOwner, filter.RepoName)
		if err != nil {
			return nil, err
		}
		for _, discussion := range discussions {
			obj := liveObjectFromDiscussion(&discussion, filter.RepoOwner, filter.RepoName)
			if matchesLiveFilter(obj, filter) {
				objects = append(objects, obj)
			}
		}
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].UpdatedAt.After(objects[j].UpdatedAt)
	})
	return objects, nil
}

func fetchLiveCollaborationObject(ctx context.Context, client *ghclient.Client, owner, repo string, number int) (*collaboration.CollaborationObject, error) {
	if client == nil {
		return nil, fmt.Errorf("github client unavailable")
	}
	issue, err := client.GetIssue(ctx, owner, repo, number)
	if err == nil && issue != nil {
		return liveObjectFromIssue(issue, owner, repo), nil
	}
	discussion, discussionErr := client.GetDiscussion(ctx, owner, repo, number)
	if discussionErr != nil {
		return nil, err
	}
	return liveObjectFromDiscussion(discussion, owner, repo), nil
}

func renderCollabListText(out io.Writer, objects []*collaboration.CollaborationObject) error {
	if len(objects) == 0 {
		_, _ = fmt.Fprintln(out, "No collaboration objects found.")
		return nil
	}
	_, _ = fmt.Fprintf(out, "%-6s %-10s %-8s %-30s %s\n", "#", "Type", "State", "Title", "Repo")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 70))
	for _, o := range objects {
		title := o.Title
		if len(title) > 28 {
			title = title[:25] + "..."
		}
		_, _ = fmt.Fprintf(out, "%-6d %-10s %-8s %-30s %s/%s\n",
			o.Number, o.ObjectType, o.State, title, o.RepoOwner, o.RepoName)
	}
	return nil
}

func renderCollabShowText(out io.Writer, obj *collaboration.CollaborationObject) error {
	if obj == nil {
		return nil
	}
	_, _ = fmt.Fprintf(out, "═══ %s #%d ═══\n\n", obj.ObjectType, obj.Number)
	_, _ = fmt.Fprintf(out, "Title:   %s\n", obj.Title)
	_, _ = fmt.Fprintf(out, "State:   %s\n", obj.State)
	_, _ = fmt.Fprintf(out, "Author:  %s\n", obj.Author)
	_, _ = fmt.Fprintf(out, "Repo:    %s/%s\n", obj.RepoOwner, obj.RepoName)
	_, _ = fmt.Fprintf(out, "URL:     %s\n", obj.URL)
	_, _ = fmt.Fprintf(out, "Created: %s\n", obj.CreatedAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(out, "Updated: %s\n", obj.UpdatedAt.Format(time.RFC3339))
	if len(obj.Labels) > 0 {
		_, _ = fmt.Fprintf(out, "Labels:  %s\n", strings.Join(obj.Labels, ", "))
	}
	if len(obj.Assignees) > 0 {
		_, _ = fmt.Fprintf(out, "Assignees: %s\n", strings.Join(obj.Assignees, ", "))
	}
	_, _ = fmt.Fprintf(out, "Comments: %d\n\n", obj.CommentsCount)
	if obj.Body != "" {
		_, _ = fmt.Fprintf(out, "── Body ──\n%s\n", obj.Body)
	}
	return nil
}

func renderMutationResultText(out io.Writer, r *collaboration.MutationResult) error {
	if r.Success {
		_, _ = fmt.Fprintf(out, "%s\n", r.Message)
		if r.Object != nil {
			_, _ = fmt.Fprintf(out, "  %s/%s#%d - %s\n", r.Object.RepoOwner, r.Object.RepoName, r.Object.Number, r.Object.Title)
		}
	} else {
		_, _ = fmt.Fprintf(out, "Error: %s\n", r.Message)
	}
	return nil
}
