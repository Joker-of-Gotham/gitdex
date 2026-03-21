package autonomyexec

import (
	"context"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/repocontext"
	"github.com/your-org/gitdex/internal/autonomy"
)

// NewCruiseEngineForDaemon wires planner, tool registry, guardrails, and policy gate
// for long-running autonomous cruise cycles alongside the HTTP daemon.
func NewCruiseEngineForDaemon(app bootstrap.App) (*autonomy.CruiseEngine, error) {
	provider, err := resolveProvider(app)
	if err != nil {
		return nil, err
	}

	ghClient, err := newGitHubClientFromApp(app)
	if err != nil {
		return nil, err
	}

	repoRoot := firstNonEmpty(app.RepoRoot, app.Config.Paths.RepositoryRoot)
	owner, repoName := repocontext.ResolveOwnerRepoFromLocalPath(context.Background(), repoRoot)
	repoRoot = SelectRepoRootForRemote(app, repoRoot, owner, repoName)

	registry := buildToolRegistry(repoRoot, ghClient, owner, repoName)
	guard := autonomy.NewGuardrails()
	exec := registry.AsExecutor(guard)

	planner := autonomy.NewPlanner(provider, func() string {
		return buildRepoContext(context.Background(), app, registry, repoRoot, owner, repoName)
	})

	reporter := autonomy.NewReporter(50)
	policyGate := autonomy.NewPolicyGate()

	cfg := autonomy.DefaultCruiseConfig()
	cfg.Enabled = true

	return autonomy.NewCruiseEngine(cfg, planner, guard, exec, reporter, autonomy.WithPolicyGate(policyGate)), nil
}
