package llmfactory

import (
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	openaillm "github.com/Joker-of-Gotham/gitdex/internal/llm/openai"
)

func Build(cfg config.LLMConfig) (llm.LLMProvider, config.LLMConfig) {
	effective := cfg
	primaryRole := effective.PrimaryRole()
	primaryProvider := BuildRoleProvider(primaryRole, effective)
	var secondaryProvider llm.LLMProvider
	if effective.Secondary.Enabled {
		secondaryRole := effective.SecondaryRole()
		secondaryProvider = BuildRoleProvider(secondaryRole, effective)
		if secondaryProvider == nil {
			effective.Secondary.Enabled = false
			effective.Secondary.Model = ""
		}
	}
	if primaryProvider == nil {
		return nil, effective
	}
	return llm.NewRouter(primaryProvider, secondaryProvider), effective
}

func BuildRoleProvider(role config.ModelConfig, llmCfg config.LLMConfig) llm.LLMProvider {
	model := strings.TrimSpace(role.Model)
	if model == "" {
		return nil
	}

	switch config.RoleProvider(role) {
	case openaillm.ProviderOpenAI, openaillm.ProviderDeepSeek:
		apiKey := config.ResolveRoleAPIKey(role)
		if apiKey == "" {
			return nil
		}
		timeout := time.Duration(llmCfg.RequestTimeout) * time.Second
		return openaillm.NewClient(
			config.RoleProvider(role),
			config.RoleEndpoint(role),
			apiKey,
			model,
			timeout,
		)
	default:
		return ollama.NewClient(config.RoleEndpoint(role), model, llmCfg.ContextLength)
	}
}
