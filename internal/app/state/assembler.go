package state

import (
	"context"
	"fmt"
	"time"

	gitstate "github.com/your-org/gitdex/internal/platform/git"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/state/repo"
)

type Assembler struct {
	ghClient *ghclient.Client
}

func NewAssembler(ghClient *ghclient.Client) *Assembler {
	return &Assembler{ghClient: ghClient}
}

func (a *Assembler) Assemble(ctx context.Context, owner, repoName, repoPath string) (*repo.RepoSummary, error) {
	summary := &repo.RepoSummary{
		Owner:     owner,
		Repo:      repoName,
		Timestamp: time.Now().UTC(),
	}

	if repoPath != "" {
		local, err := gitstate.ReadLocalState(repoPath)
		if err != nil {
			summary.Local = repo.LocalState{Label: repo.Unknown, Detail: fmt.Sprintf("cannot read local state: %v", err)}
		} else {
			summary.Local = assembleLocal(local)
		}
	} else {
		summary.Local = repo.LocalState{Label: repo.Unknown, Detail: "no local repository path available"}
	}

	if a.ghClient != nil {
		a.assembleRemote(ctx, owner, repoName, summary)
	} else {
		summary.Remote = repo.RemoteState{Label: repo.Unknown, Detail: "GitHub identity not configured. Run 'gitdex init' to set up GitHub App credentials."}
		summary.Collaboration = repo.CollaborationSignals{Label: repo.Unknown, Detail: "requires GitHub identity"}
		summary.Workflows = repo.WorkflowState{Label: repo.Unknown, Detail: "requires GitHub identity"}
		summary.Deployments = repo.DeploymentState{Label: repo.Unknown, Detail: "requires GitHub identity"}
	}

	summary.OverallLabel = repo.WorstLabel(
		summary.Local.Label,
		summary.Remote.Label,
		summary.Collaboration.Label,
		summary.Workflows.Label,
		summary.Deployments.Label,
	)

	summary.Risks = assembleRisks(summary)
	summary.NextActions = assembleNextActions(summary)

	return summary, nil
}

func assembleLocal(g *gitstate.LocalGitState) repo.LocalState {
	local := repo.LocalState{
		Branch:        g.Branch,
		HeadSHA:       g.HeadSHA,
		IsDetached:    g.IsDetached,
		IsClean:       g.IsClean,
		StagedCount:   g.StagedCount,
		DirtyCount:    g.DirtyCount,
		Ahead:         g.Ahead,
		Behind:        g.Behind,
		DefaultRemote: g.DefaultRemote,
	}

	switch {
	case g.IsDetached:
		local.Label = repo.Degraded
		local.Detail = "HEAD is detached"
	case !g.IsClean && g.Behind > 0:
		local.Label = repo.Drifting
		local.Detail = fmt.Sprintf("uncommitted changes (%d dirty) and %d commits behind upstream", g.DirtyCount, g.Behind)
	case !g.IsClean:
		local.Label = repo.Drifting
		local.Detail = fmt.Sprintf("uncommitted changes: %d dirty files", g.DirtyCount)
	case g.Behind > 0:
		local.Label = repo.Drifting
		local.Detail = fmt.Sprintf("%d commits behind upstream", g.Behind)
	case g.Ahead > 0:
		local.Label = repo.Drifting
		local.Detail = fmt.Sprintf("%d commits ahead of upstream (unpushed)", g.Ahead)
	default:
		local.Label = repo.Healthy
		local.Detail = "clean working tree, up to date"
	}
	return local
}

func (a *Assembler) assembleRemote(ctx context.Context, owner, repoName string, summary *repo.RepoSummary) {
	remote, err := a.ghClient.GetRepository(ctx, owner, repoName)
	if err != nil {
		summary.Remote = repo.RemoteState{Label: repo.Degraded, Detail: fmt.Sprintf("failed to read: %v", err)}
	} else {
		summary.Remote = *remote
		summary.Remote.Label = repo.Healthy
	}

	prs, err := a.ghClient.ListOpenPullRequests(ctx, owner, repoName)
	if err != nil {
		summary.Collaboration = repo.CollaborationSignals{Label: repo.Degraded, Detail: fmt.Sprintf("failed to read PRs: %v", err)}
	} else {
		summary.Collaboration = assembleCollaboration(prs)
	}

	issueCount, err := a.ghClient.EstimateOpenIssueCount(ctx, owner, repoName)
	if err == nil {
		summary.Collaboration.OpenIssueCount = issueCount
	}

	runs, err := a.ghClient.ListWorkflowRuns(ctx, owner, repoName)
	if err != nil {
		summary.Workflows = repo.WorkflowState{Label: repo.Degraded, Detail: fmt.Sprintf("failed to read: %v", err)}
	} else {
		summary.Workflows = assembleWorkflows(runs)
	}

	deps, err := a.ghClient.ListDeployments(ctx, owner, repoName)
	if err != nil {
		summary.Deployments = repo.DeploymentState{Label: repo.Degraded, Detail: fmt.Sprintf("failed to read: %v", err)}
	} else {
		summary.Deployments = assembleDeployments(deps)
	}
}

func assembleCollaboration(prs []repo.PullRequestSummary) repo.CollaborationSignals {
	cs := repo.CollaborationSignals{
		OpenPRCount:  len(prs),
		PullRequests: prs,
	}

	hasStale := false
	hasReviewNeeded := false
	for _, pr := range prs {
		if pr.StaleDays > 14 {
			hasStale = true
		}
		if pr.NeedsReview {
			hasReviewNeeded = true
		}
	}

	switch {
	case hasStale:
		cs.Label = repo.Degraded
		cs.Detail = "stale pull requests detected (>14 days without update)"
	case hasReviewNeeded:
		cs.Label = repo.Drifting
		cs.Detail = "pull requests awaiting review"
	case len(prs) == 0:
		cs.Label = repo.Healthy
		cs.Detail = "no open pull requests"
	default:
		cs.Label = repo.Healthy
		cs.Detail = fmt.Sprintf("%d open pull requests", len(prs))
	}
	return cs
}

func assembleWorkflows(runs []repo.WorkflowRunSummary) repo.WorkflowState {
	ws := repo.WorkflowState{Runs: runs}

	if len(runs) == 0 {
		ws.Label = repo.Unknown
		ws.Detail = "no workflow runs found"
		return ws
	}

	hasFailure := false
	for _, r := range runs {
		if r.Conclusion == "failure" || r.Conclusion == "timed_out" {
			hasFailure = true
			break
		}
	}

	if hasFailure {
		ws.Label = repo.Degraded
		ws.Detail = "recent workflow failures detected"
	} else {
		ws.Label = repo.Healthy
		ws.Detail = "all recent workflows succeeded"
	}
	return ws
}

func assembleDeployments(deps []repo.DeploymentSummary) repo.DeploymentState {
	ds := repo.DeploymentState{Deployments: deps}

	if len(deps) == 0 {
		ds.Label = repo.Unknown
		ds.Detail = "no deployments found"
		return ds
	}

	hasFailure := false
	for _, d := range deps {
		if d.State == "failure" || d.State == "error" {
			hasFailure = true
			break
		}
	}

	if hasFailure {
		ds.Label = repo.Degraded
		ds.Detail = "deployment failures detected"
	} else {
		ds.Label = repo.Healthy
		ds.Detail = "deployments healthy"
	}
	return ds
}

func assembleRisks(s *repo.RepoSummary) []repo.Risk {
	var risks []repo.Risk

	if s.Local.Label == repo.Degraded || s.Local.Label == repo.Blocked {
		risks = append(risks, repo.Risk{
			Severity:    repo.RiskMedium,
			Description: "local repository in degraded state",
			Evidence:    s.Local.Detail,
			Action:      "inspect local working tree and resolve issues",
		})
	}
	if s.Local.Behind > 0 {
		risks = append(risks, repo.Risk{
			Severity:    repo.RiskLow,
			Description: "local branch is behind upstream",
			Evidence:    fmt.Sprintf("%d commits behind", s.Local.Behind),
			Action:      "run 'git pull' or 'gitdex sync' to update",
		})
	}

	if s.Workflows.Label == repo.Degraded {
		risks = append(risks, repo.Risk{
			Severity:    repo.RiskHigh,
			Description: "CI/CD pipeline failures",
			Evidence:    s.Workflows.Detail,
			Action:      "investigate failing workflow runs",
		})
	}

	if s.Deployments.Label == repo.Degraded {
		risks = append(risks, repo.Risk{
			Severity:    repo.RiskHigh,
			Description: "deployment failures",
			Evidence:    s.Deployments.Detail,
			Action:      "investigate deployment errors",
		})
	}

	if s.Collaboration.Label == repo.Degraded {
		risks = append(risks, repo.Risk{
			Severity:    repo.RiskMedium,
			Description: "stale collaboration objects",
			Evidence:    s.Collaboration.Detail,
			Action:      "review and close stale pull requests",
		})
	}

	return risks
}

func assembleNextActions(s *repo.RepoSummary) []repo.NextAction {
	var actions []repo.NextAction

	if s.Local.Behind > 0 {
		actions = append(actions, repo.NextAction{
			Action:    "sync with upstream",
			Reason:    fmt.Sprintf("local branch is %d commits behind", s.Local.Behind),
			RiskLevel: "low",
		})
	}
	if !s.Local.IsClean && s.Local.Label != repo.Unknown {
		actions = append(actions, repo.NextAction{
			Action:    "review and commit local changes",
			Reason:    fmt.Sprintf("%d dirty files in working tree", s.Local.DirtyCount),
			RiskLevel: "low",
		})
	}

	for _, pr := range s.Collaboration.PullRequests {
		if pr.NeedsReview {
			actions = append(actions, repo.NextAction{
				Action:       fmt.Sprintf("review PR #%d: %s", pr.Number, pr.Title),
				Reason:       "pull request awaiting review",
				RiskLevel:    "low",
				EvidenceRefs: fmt.Sprintf("PR #%d by %s", pr.Number, pr.Author),
			})
		}
	}

	if s.Workflows.Label == repo.Degraded {
		actions = append(actions, repo.NextAction{
			Action:    "investigate CI failures",
			Reason:    s.Workflows.Detail,
			RiskLevel: "medium",
		})
	}

	return actions
}
