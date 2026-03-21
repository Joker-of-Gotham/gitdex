package autonomyexec

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/repocontext"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/llm/adapter"
)

var providerOverride adapter.Provider

type Request struct {
	RepoRoot          string
	Owner             string
	Repo              string
	Intent            string
	Execute           bool
	AutoThreshold     autonomy.RiskLevel
	ApprovalThreshold autonomy.RiskLevel
}

type Result struct {
	Mode              string                `json:"mode"`
	RepoRoot          string                `json:"repo_root,omitempty"`
	Owner             string                `json:"owner,omitempty"`
	Repo              string                `json:"repo,omitempty"`
	AutoThreshold     string                `json:"auto_threshold"`
	ApprovalThreshold string                `json:"approval_threshold"`
	Plans             []autonomy.ActionPlan `json:"plans"`
	Report            autonomy.CruiseReport `json:"report"`
}

func SetProviderOverrideForTest(p adapter.Provider) func() {
	prev := providerOverride
	providerOverride = p
	return func() {
		providerOverride = prev
	}
}

func Run(ctx context.Context, app bootstrap.App, req Request) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	provider, err := resolveProvider(app)
	if err != nil {
		return Result{}, err
	}

	ghClient, err := newGitHubClientFromApp(app)
	if err != nil {
		return Result{}, err
	}

	registry := buildToolRegistry(req.RepoRoot, ghClient, req.Owner, req.Repo)
	planner := autonomy.NewPlanner(provider, func() string {
		return buildRepoContext(ctx, app, registry, req.RepoRoot, req.Owner, req.Repo)
	})

	var plans []autonomy.ActionPlan
	if strings.TrimSpace(req.Intent) != "" {
		plan, err := planner.PlanFromUserIntent(ctx, req.Intent)
		if err != nil {
			return Result{}, err
		}
		plans = []autonomy.ActionPlan{*plan}
	} else {
		plans, err = planner.AnalyzeAndPlan(ctx)
		if err != nil {
			return Result{}, err
		}
	}

	report := executePlans(ctx, plans, registry, req.Execute, req.AutoThreshold, req.ApprovalThreshold)
	return Result{
		Mode:              runMode(req.Execute),
		RepoRoot:          req.RepoRoot,
		Owner:             req.Owner,
		Repo:              req.Repo,
		AutoThreshold:     req.AutoThreshold.String(),
		ApprovalThreshold: req.ApprovalThreshold.String(),
		Plans:             plans,
		Report:            report,
	}, nil
}

func RenderResult(out io.Writer, result Result) error {
	_, _ = fmt.Fprintf(out, "Autonomy Mode: %s\n", result.Mode)
	if result.Owner != "" && result.Repo != "" {
		_, _ = fmt.Fprintf(out, "Repository:     %s/%s\n", result.Owner, result.Repo)
	}
	if result.RepoRoot != "" {
		_, _ = fmt.Fprintf(out, "Local Clone:    %s\n", result.RepoRoot)
	}
	_, _ = fmt.Fprintf(out, "Auto Threshold: %s\n", result.AutoThreshold)
	_, _ = fmt.Fprintf(out, "Approval Threshold: %s\n", result.ApprovalThreshold)
	return RenderDetails(out, result)
}

func RenderDetails(out io.Writer, result Result) error {
	_, _ = fmt.Fprintf(out, "\nPlans:\n")
	if len(result.Plans) == 0 {
		_, _ = fmt.Fprintf(out, "  - none\n")
	} else {
		for i, plan := range result.Plans {
			risk := plan.RiskLevelStr
			if risk == "" && plan.RiskLevel > 0 {
				risk = plan.RiskLevel.String()
			}
			_, _ = fmt.Fprintf(out, "  %d. %s [%s]\n", i+1, plan.Description, risk)
			if plan.Rationale != "" {
				_, _ = fmt.Fprintf(out, "     %s\n", plan.Rationale)
			}
			for _, step := range plan.Steps {
				_, _ = fmt.Fprintf(out, "     - %d %s\n", step.Order, step.Action)
			}
		}
	}

	if len(result.Report.Executed) > 0 {
		_, _ = fmt.Fprintf(out, "\nExecuted:\n")
		for _, executed := range result.Report.Executed {
			status := "ok"
			if !executed.Result.Success {
				status = "failed"
			}
			_, _ = fmt.Fprintf(out, "  - %s [%s]\n", executed.Plan.Description, status)
			for _, step := range executed.Result.StepResults {
				stepStatus := "ok"
				if !step.Success {
					stepStatus = "failed"
				}
				_, _ = fmt.Fprintf(out, "    * %s [%s]\n", step.Action, stepStatus)
				if step.Output != "" {
					_, _ = fmt.Fprintf(out, "      %s\n", step.Output)
				}
				if step.Error != "" {
					_, _ = fmt.Fprintf(out, "      error: %s\n", step.Error)
				}
			}
		}
	}

	if len(result.Report.Pending) > 0 {
		_, _ = fmt.Fprintf(out, "\nPending:\n")
		for _, pending := range result.Report.Pending {
			risk := pending.RiskLevel.String()
			if pending.RiskLevel == 0 && pending.RiskLevelStr != "" {
				risk = pending.RiskLevelStr
			}
			_, _ = fmt.Fprintf(out, "  - %s [%s]\n", pending.Description, risk)
		}
	}

	if len(result.Report.Blocked) > 0 {
		_, _ = fmt.Fprintf(out, "\nBlocked:\n")
		for _, blocked := range result.Report.Blocked {
			_, _ = fmt.Fprintf(out, "  - %s: %s\n", blocked.Plan.Description, blocked.Reason)
		}
	}

	if len(result.Report.Errors) > 0 {
		_, _ = fmt.Fprintf(out, "\nErrors:\n")
		for _, message := range result.Report.Errors {
			_, _ = fmt.Fprintf(out, "  - %s\n", message)
		}
	}

	return nil
}

func SelectRepoRootForRemote(app bootstrap.App, repoRoot, owner, repoName string) string {
	rc, err := repocontext.Resolve(context.Background(), app, repocontext.ResolveOptions{
		RepoRoot: repoRoot,
		Owner:    owner,
		Repo:     repoName,
	})
	if err != nil || rc == nil {
		return ""
	}
	return strings.TrimSpace(rc.ActiveLocalPath)
}
