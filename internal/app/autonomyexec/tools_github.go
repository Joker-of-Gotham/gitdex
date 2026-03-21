package autonomyexec

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/your-org/gitdex/internal/autonomy"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
)

func registerGitHubTools(registry *autonomy.ToolRegistry, ghClient *ghclient.Client, owner, repoName string) {
	register := func(name, desc string, params map[string]autonomy.ToolParam, handler autonomy.ActionHandler) {
		registry.Register(autonomy.Tool{
			Name:        name,
			Description: desc,
			Parameters:  params,
			Handler:     handler,
		})
	}

	withRepo := func() (string, string, error) {
		if ghClient == nil {
			return "", "", fmt.Errorf("GitHub identity is not configured")
		}
		if strings.TrimSpace(owner) == "" || strings.TrimSpace(repoName) == "" {
			return "", "", fmt.Errorf("GitHub repo context is unavailable; set owner/repo first")
		}
		return owner, repoName, nil
	}

	register("github.pr.create", "Create a pull request", map[string]autonomy.ToolParam{
		"title": {Name: "title", Type: "string", Description: "Pull request title", Required: true},
		"body":  {Name: "body", Type: "string", Description: "Pull request body", Required: false},
		"head":  {Name: "head", Type: "string", Description: "Head branch", Required: true},
		"base":  {Name: "base", Type: "string", Description: "Base branch", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		pr, err := ghClient.CreatePullRequest(ctx, owner, repoName, strings.TrimSpace(args["title"]), args["body"], strings.TrimSpace(args["head"]), strings.TrimSpace(args["base"]))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("created PR #%d", pr.GetNumber()), nil
	})

	register("github.pr.merge", "Merge a pull request", map[string]autonomy.ToolParam{
		"number":  {Name: "number", Type: "int", Description: "Pull request number", Required: true},
		"message": {Name: "message", Type: "string", Description: "Merge commit message", Required: false},
		"method":  {Name: "method", Type: "string", Description: "merge, squash, or rebase", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		method := strings.TrimSpace(args["method"])
		if method == "" {
			method = "merge"
		}
		if _, err := ghClient.MergePullRequest(ctx, owner, repoName, number, args["message"], method); err != nil {
			return "", err
		}
		return fmt.Sprintf("merged PR #%d", number), nil
	})

	register("github.pr.close", "Close a pull request", map[string]autonomy.ToolParam{
		"number": {Name: "number", Type: "int", Description: "Pull request number", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		if err := ghClient.CloseIssue(ctx, owner, repoName, number); err != nil {
			return "", err
		}
		return fmt.Sprintf("closed PR #%d", number), nil
	})

	register("github.pr.comment", "Comment on a pull request", map[string]autonomy.ToolParam{
		"number": {Name: "number", Type: "int", Description: "Pull request number", Required: true},
		"body":   {Name: "body", Type: "string", Description: "Comment body", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		if _, err := ghClient.CreateComment(ctx, owner, repoName, number, args["body"]); err != nil {
			return "", err
		}
		return fmt.Sprintf("commented on PR #%d", number), nil
	})

	register("github.pr.review", "Submit a pull request review", map[string]autonomy.ToolParam{
		"number": {Name: "number", Type: "int", Description: "Pull request number", Required: true},
		"event":  {Name: "event", Type: "string", Description: "APPROVE, REQUEST_CHANGES, or COMMENT", Required: true},
		"body":   {Name: "body", Type: "string", Description: "Review body", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		event := strings.ToUpper(strings.TrimSpace(args["event"]))
		if _, err := ghClient.SubmitPRReview(ctx, owner, repoName, number, event, args["body"]); err != nil {
			return "", err
		}
		return fmt.Sprintf("review submitted for PR #%d", number), nil
	})

	register("github.issue.create", "Create an issue", map[string]autonomy.ToolParam{
		"title":     {Name: "title", Type: "string", Description: "Issue title", Required: true},
		"body":      {Name: "body", Type: "string", Description: "Issue body", Required: false},
		"labels":    {Name: "labels", Type: "string", Description: "Comma-separated labels", Required: false},
		"assignees": {Name: "assignees", Type: "string", Description: "Comma-separated assignees", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		issue, err := ghClient.CreateIssue(ctx, owner, repoName, strings.TrimSpace(args["title"]), args["body"], splitCSV(args["labels"]), splitCSV(args["assignees"]))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("created issue #%d", issue.GetNumber()), nil
	})

	register("github.issue.close", "Close an issue", map[string]autonomy.ToolParam{
		"number": {Name: "number", Type: "int", Description: "Issue number", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		if err := ghClient.CloseIssue(ctx, owner, repoName, number); err != nil {
			return "", err
		}
		return fmt.Sprintf("closed issue #%d", number), nil
	})

	register("github.issue.reopen", "Reopen an issue", map[string]autonomy.ToolParam{
		"number": {Name: "number", Type: "int", Description: "Issue number", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		if err := ghClient.ReopenIssue(ctx, owner, repoName, number); err != nil {
			return "", err
		}
		return fmt.Sprintf("reopened issue #%d", number), nil
	})

	register("github.issue.comment", "Comment on an issue", map[string]autonomy.ToolParam{
		"number": {Name: "number", Type: "int", Description: "Issue number", Required: true},
		"body":   {Name: "body", Type: "string", Description: "Comment body", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		if _, err := ghClient.CreateComment(ctx, owner, repoName, number, args["body"]); err != nil {
			return "", err
		}
		return fmt.Sprintf("commented on issue #%d", number), nil
	})

	register("github.issue.label", "Add labels to an issue", map[string]autonomy.ToolParam{
		"number": {Name: "number", Type: "int", Description: "Issue number", Required: true},
		"labels": {Name: "labels", Type: "string", Description: "Comma-separated labels", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		labels := splitCSV(args["labels"])
		if len(labels) == 0 {
			return "", fmt.Errorf("labels are required")
		}
		if err := ghClient.AddLabels(ctx, owner, repoName, number, labels); err != nil {
			return "", err
		}
		return fmt.Sprintf("labeled issue #%d", number), nil
	})

	register("github.issue.assign", "Assign users to an issue", map[string]autonomy.ToolParam{
		"number":    {Name: "number", Type: "int", Description: "Issue number", Required: true},
		"assignees": {Name: "assignees", Type: "string", Description: "Comma-separated assignees", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		number, err := intArg(args, "number")
		if err != nil {
			return "", err
		}
		assignees := splitCSV(args["assignees"])
		if len(assignees) == 0 {
			return "", fmt.Errorf("assignees are required")
		}
		if err := ghClient.SetAssignees(ctx, owner, repoName, number, assignees); err != nil {
			return "", err
		}
		return fmt.Sprintf("assigned issue #%d", number), nil
	})

	register("github.release.create", "Create a release", map[string]autonomy.ToolParam{
		"tag":   {Name: "tag", Type: "string", Description: "Release tag", Required: true},
		"name":  {Name: "name", Type: "string", Description: "Release name", Required: false},
		"body":  {Name: "body", Type: "string", Description: "Release notes", Required: false},
		"draft": {Name: "draft", Type: "bool", Description: "Whether the release should be a draft", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		tag := strings.TrimSpace(args["tag"])
		name := strings.TrimSpace(args["name"])
		if name == "" {
			name = tag
		}
		draft, _ := strconv.ParseBool(strings.TrimSpace(args["draft"]))
		prerelease := strings.EqualFold(strings.TrimSpace(args["prerelease"]), "true")
		release, err := ghClient.CreateRelease(ctx, owner, repoName, tag, name, args["body"], draft, prerelease)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("created release %s", release.TagName), nil
	})

	register("github.workflow.trigger", "Dispatch a workflow", map[string]autonomy.ToolParam{
		"workflow_id": {Name: "workflow_id", Type: "int", Description: "Workflow numeric ID", Required: true},
		"ref":         {Name: "ref", Type: "string", Description: "Git ref to run against", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		owner, repoName, err := withRepo()
		if err != nil {
			return "", err
		}
		workflowID, err := int64Arg(args, "workflow_id")
		if err != nil {
			return "", err
		}
		ref := strings.TrimSpace(args["ref"])
		if ref == "" {
			return "", fmt.Errorf("ref is required")
		}
		if err := ghClient.TriggerWorkflow(ctx, owner, repoName, workflowID, ref); err != nil {
			return "", err
		}
		return fmt.Sprintf("triggered workflow %d on %s", workflowID, ref), nil
	})
}

func intArg(args map[string]string, key string) (int, error) {
	raw := strings.TrimSpace(args[key])
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}

func int64Arg(args map[string]string, key string) (int64, error) {
	raw := strings.TrimSpace(args[key])
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
