package llmfactory

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
)

func TestBuildDefaultOllamaProvider(t *testing.T) {
	provider, effective := Build(config.DefaultConfig().LLM)
	if provider == nil {
		t.Fatal("expected provider")
	}
	if effective.PrimaryRole().Model == "" {
		t.Fatal("expected primary model")
	}
}

func TestBuildCloudProviderWithoutKeyDisablesSecondary(t *testing.T) {
	cfg := config.DefaultConfig().LLM
	cfg.Primary.Provider = "openai"
	cfg.Primary.Model = "gpt-4.1-mini"
	cfg.Primary.APIKey = ""
	cfg.Provider = "openai"
	cfg.Model = "gpt-4.1-mini"
	if provider, _ := Build(cfg); provider != nil {
		t.Fatal("expected nil provider without api key")
	}
}
