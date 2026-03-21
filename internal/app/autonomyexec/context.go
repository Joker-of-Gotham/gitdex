package autonomyexec

import (
	"context"
	"encoding/json"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/repocontext"
	appstate "github.com/your-org/gitdex/internal/app/state"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/gitops"
)

func buildRepoContext(ctx context.Context, app bootstrap.App, registry *autonomy.ToolRegistry, repoRoot, owner, repoName string) string {
	payload := map[string]any{
		"repo_root": repoRoot,
		"owner":     owner,
		"repo":      repoName,
		"summary":   loadSummary(ctx, app, repoRoot, owner, repoName),
		"inspect":   inspectRepoContext(ctx, app, repoRoot, owner, repoName),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return registry.GenerateToolPrompt()
	}
	return string(data) + "\n\n" + registry.GenerateToolPrompt()
}

func loadSummary(ctx context.Context, app bootstrap.App, repoRoot, owner, repoName string) any {
	if owner == "" || repoName == "" {
		owner, repoName = repocontext.ResolveOwnerRepoFromLocalPath(ctx, repoRoot)
	}
	if owner == "" || repoName == "" {
		return nil
	}

	client, err := newGitHubClientFromApp(app)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	assembler := appstate.NewAssembler(client)
	summary, err := assembler.Assemble(ctx, owner, repoName, repoRoot)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return summary
}

func inspectRepoContext(ctx context.Context, app bootstrap.App, repoRoot, owner, repoName string) map[string]any {
	rc, _ := repocontext.Resolve(ctx, app, repocontext.ResolveOptions{
		RepoRoot: repoRoot,
		Owner:    owner,
		Repo:     repoName,
	})
	report := map[string]any{
		"owner": owner,
		"repo":  repoName,
	}
	if rc != nil {
		report["host"] = rc.Host
		report["access_mode"] = rc.AccessMode
		report["canonical_remote"] = rc.CanonicalRemote
		report["local_paths"] = rc.LocalPaths
		report["remote_topology"] = rc.Topology
		if owner == "" {
			owner = rc.Owner
			report["owner"] = rc.Owner
		}
		if repoName == "" {
			repoName = rc.Repo
			report["repo"] = rc.Repo
		}
	}
	localPath := repoRoot
	if rc != nil && rc.ActiveLocalPath != "" {
		localPath = rc.ActiveLocalPath
	}
	if owner != "" && repoName != "" {
		if client, err := newGitHubClientFromApp(app); err == nil && client != nil {
			if detail, err := client.GetRepositoryDetail(ctx, owner, repoName); err == nil {
				report["remote"] = detail
			} else {
				report["remote_error"] = err.Error()
			}
		}
	}
	if localPath != "" {
		report["selected_local"] = localPath
		inspector := gitops.NewInspector(gitops.NewGitExecutor())
		if inspection, err := inspector.Inspect(ctx, localPath); err == nil {
			report["local_inspection"] = inspection
			report["recommendation"] = inspector.Recommend(inspection)
		} else {
			report["local_error"] = err.Error()
		}
	}

	return report
}
