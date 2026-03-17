package llmfactory

import (
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	openaillm "github.com/Joker-of-Gotham/gitdex/internal/llm/openai"
)

type ProviderHealth string

const (
	ProviderUp       ProviderHealth = "up"
	ProviderDegraded ProviderHealth = "degraded"
	ProviderDown     ProviderHealth = "down"
)

type DiagnosticCode string

const (
	DiagOK                     DiagnosticCode = "ok"
	DiagModelMissing           DiagnosticCode = "model_missing"
	DiagAPIKeyMissing          DiagnosticCode = "api_key_missing"
	DiagProviderInitFailed     DiagnosticCode = "provider_init_failed"
	DiagSecondaryPromoted      DiagnosticCode = "secondary_promoted"
	DiagSecondaryDisabled      DiagnosticCode = "secondary_disabled"
	DiagSecondaryNotConfigured DiagnosticCode = "secondary_not_configured"
)

type RoleDiagnostics struct {
	Role     string         `json:"role"`
	Provider string         `json:"provider"`
	Model    string         `json:"model"`
	Health   ProviderHealth `json:"health"`
	Code     DiagnosticCode `json:"code"`
	Reason   string         `json:"reason,omitempty"`
}

type BuildDiagnostics struct {
	Primary            RoleDiagnostics `json:"primary"`
	Secondary          RoleDiagnostics `json:"secondary"`
	FallbackPromoted   bool            `json:"fallback_promoted"`
	EffectivePrimaryBy string          `json:"effective_primary_by,omitempty"` // primary|secondary
}

func Build(cfg config.LLMConfig) (llm.LLMProvider, config.LLMConfig) {
	provider, effective, _ := BuildWithDiagnostics(cfg)
	return provider, effective
}

func BuildWithDiagnostics(cfg config.LLMConfig) (llm.LLMProvider, config.LLMConfig, BuildDiagnostics) {
	effective := cfg
	report := BuildDiagnostics{
		Primary: RoleDiagnostics{
			Role:     "primary",
			Provider: config.RoleProvider(effective.PrimaryRole()),
			Model:    strings.TrimSpace(effective.PrimaryRole().Model),
			Health:   ProviderDown,
			Code:     DiagProviderInitFailed,
		},
		Secondary: RoleDiagnostics{
			Role:     "secondary",
			Provider: config.RoleProvider(effective.SecondaryRole()),
			Model:    strings.TrimSpace(effective.SecondaryRole().Model),
			Health:   ProviderDown,
			Code:     DiagSecondaryNotConfigured,
		},
	}

	primaryRole := effective.PrimaryRole()
	primaryProvider, primaryDiag := buildRoleProviderWithDiag(primaryRole, effective, "primary")
	report.Primary = primaryDiag
	var secondaryProvider llm.LLMProvider
	if effective.Secondary.Enabled {
		secondaryRole := effective.SecondaryRole()
		secondaryProvider, report.Secondary = buildRoleProviderWithDiag(secondaryRole, effective, "secondary")
		if secondaryProvider == nil {
			effective.Secondary.Enabled = false
			effective.Secondary.Model = ""
			report.Secondary.Health = ProviderDown
			if report.Secondary.Code == DiagOK {
				report.Secondary.Code = DiagSecondaryDisabled
			}
		}
	} else {
		report.Secondary.Code = DiagSecondaryNotConfigured
		report.Secondary.Health = ProviderDown
	}
	// Resilience fallback: if primary is unavailable but secondary works,
	// promote secondary so the app keeps running instead of hard-failing.
	if primaryProvider == nil && secondaryProvider != nil {
		effective.Primary = effective.Secondary
		effective.Provider = effective.Primary.Provider
		effective.Model = effective.Primary.Model
		effective.Endpoint = effective.Primary.Endpoint
		effective.APIKey = effective.Primary.APIKey
		effective.APIKeyEnv = effective.Primary.APIKeyEnv
		effective.Secondary.Enabled = false
		effective.Secondary.Model = ""
		report.FallbackPromoted = true
		report.EffectivePrimaryBy = "secondary"
		report.Primary.Health = ProviderDegraded
		report.Primary.Code = DiagSecondaryPromoted
		report.Primary.Reason = "primary unavailable; promoted secondary provider"
		return llm.NewRouter(secondaryProvider, nil), effective, report
	}
	if primaryProvider == nil {
		report.EffectivePrimaryBy = ""
		return nil, effective, report
	}
	report.EffectivePrimaryBy = "primary"
	if report.Primary.Code == "" {
		report.Primary.Code = DiagOK
	}
	if report.Primary.Health == "" {
		report.Primary.Health = ProviderUp
	}
	if effective.Secondary.Enabled && report.Secondary.Code == "" {
		report.Secondary.Code = DiagOK
		report.Secondary.Health = ProviderUp
	}
	return llm.NewRouter(primaryProvider, secondaryProvider), effective, report
}

func BuildRoleProvider(role config.ModelConfig, llmCfg config.LLMConfig) llm.LLMProvider {
	p, _ := buildRoleProviderWithDiag(role, llmCfg, "")
	return p
}

func buildRoleProviderWithDiag(role config.ModelConfig, llmCfg config.LLMConfig, roleName string) (llm.LLMProvider, RoleDiagnostics) {
	diag := RoleDiagnostics{
		Role:     roleName,
		Provider: config.RoleProvider(role),
		Model:    strings.TrimSpace(role.Model),
		Health:   ProviderDown,
		Code:     DiagProviderInitFailed,
	}
	model := strings.TrimSpace(role.Model)
	if model == "" {
		diag.Code = DiagModelMissing
		diag.Reason = "model is empty"
		return nil, diag
	}

	switch config.RoleProvider(role) {
	case openaillm.ProviderOpenAI, openaillm.ProviderDeepSeek:
		apiKey := config.ResolveRoleAPIKey(role)
		if apiKey == "" {
			diag.Code = DiagAPIKeyMissing
			diag.Reason = "api key is missing"
			return nil, diag
		}
		timeout := time.Duration(llmCfg.RequestTimeout) * time.Second
		diag.Health = ProviderUp
		diag.Code = DiagOK
		return openaillm.NewClient(
			config.RoleProvider(role),
			config.RoleEndpoint(role),
			apiKey,
			model,
			timeout,
		), diag
	default:
		client := ollama.NewClient(config.RoleEndpoint(role), model, llmCfg.ContextLength)
		if llmCfg.RequestTimeout > 0 {
			client.SetGenerateTimeout(time.Duration(llmCfg.RequestTimeout) * time.Second)
		}
		diag.Health = ProviderUp
		diag.Code = DiagOK
		return client, diag
	}
}
