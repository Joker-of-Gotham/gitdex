package autonomyexec

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/repocontext"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/llm/adapter"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
)

func resolveProvider(app bootstrap.App) (adapter.Provider, error) {
	if providerOverride != nil {
		return providerOverride, nil
	}

	providerName := firstNonEmpty(os.Getenv("GITDEX_LLM_PROVIDER"), app.Config.LLM.Provider)
	apiKey := firstNonEmpty(os.Getenv("GITDEX_LLM_API_KEY"), app.Config.LLM.APIKey)
	endpoint := firstNonEmpty(os.Getenv("GITDEX_LLM_ENDPOINT"), app.Config.LLM.Endpoint)
	model := firstNonEmpty(os.Getenv("GITDEX_LLM_MODEL"), app.Config.LLM.Model)
	if providerName == "" {
		providerName = "openai"
	}
	if !strings.EqualFold(providerName, "ollama") && strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("autonomy execution requires a configured LLM provider; set llm.provider/model/api_key first")
	}

	provider, err := adapter.NewProviderFromConfig(providerName, model, apiKey, endpoint)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func newGitHubClientFromApp(app bootstrap.App) (*ghclient.Client, error) {
	return repocontext.NewGitHubClient(app)
}

func runMode(execute bool) string {
	if execute {
		return "execute"
	}
	return "plan"
}

func executePlans(ctx context.Context, plans []autonomy.ActionPlan, registry *autonomy.ToolRegistry, execute bool, autoThreshold, approvalThreshold autonomy.RiskLevel) autonomy.CruiseReport {
	guard := autonomy.NewGuardrails()
	exec := registry.AsExecutor(guard)
	report := autonomy.CruiseReport{
		CycleID:   fmt.Sprintf("manual-%d", time.Now().UTC().UnixNano()),
		StartTime: time.Now().UTC(),
	}

	for _, plan := range plans {
		if plan.RiskLevel == 0 {
			plan.RiskLevel = guard.EvaluateRisk(plan)
			plan.RiskLevelStr = plan.RiskLevel.String()
		}

		if !execute {
			report.Pending = append(report.Pending, plan)
			continue
		}

		if plan.RiskLevel <= autoThreshold {
			result := exec.Execute(ctx, plan)
			report.Executed = append(report.Executed, autonomy.ExecutedAction{
				Plan:   plan,
				Result: result,
			})
			if !result.Success && result.Error != "" {
				report.Errors = append(report.Errors, result.Error)
			}
			continue
		}

		allowed, reason := guard.CheckPolicy(plan)
		if !allowed {
			report.Blocked = append(report.Blocked, autonomy.BlockedAction{
				Plan:   plan,
				Reason: reason,
			})
			continue
		}

		if plan.RiskLevel <= approvalThreshold {
			report.Pending = append(report.Pending, plan)
			continue
		}

		report.Pending = append(report.Pending, plan)
	}

	report.EndTime = time.Now().UTC()
	return report
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
