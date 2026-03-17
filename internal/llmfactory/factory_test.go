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

func TestBuild_FallbackToSecondaryWhenPrimaryUnavailable(t *testing.T) {
	cfg := config.DefaultConfig().LLM
	cfg.Provider = "deepseek"
	cfg.Model = "deepseek-chat"
	cfg.Primary.Provider = "deepseek"
	cfg.Primary.Model = "deepseek-chat"
	cfg.Primary.APIKey = ""
	cfg.Primary.APIKeyEnv = "NO_SUCH_DEEPSEEK_ENV_VAR"
	cfg.Primary.Enabled = true

	cfg.Secondary.Provider = "ollama"
	cfg.Secondary.Model = "qwen2.5:3b"
	cfg.Secondary.Endpoint = config.DefaultOllamaEndpoint
	cfg.Secondary.Enabled = true

	provider, effective := Build(cfg)
	if provider == nil {
		t.Fatal("expected fallback provider from secondary role")
	}
	if effective.Primary.Provider != "ollama" {
		t.Fatalf("expected effective primary provider=ollama, got %q", effective.Primary.Provider)
	}
	if effective.Primary.Model != "qwen2.5:3b" {
		t.Fatalf("expected effective primary model=qwen2.5:3b, got %q", effective.Primary.Model)
	}
	if effective.Secondary.Enabled {
		t.Fatal("expected secondary to be disabled after promotion")
	}
}

func TestBuildWithDiagnostics_ReportsMissingAPIKey(t *testing.T) {
	cfg := config.DefaultConfig().LLM
	cfg.Primary.Provider = "deepseek"
	cfg.Primary.Model = "deepseek-chat"
	cfg.Primary.APIKey = ""
	cfg.Primary.APIKeyEnv = "MISSING_ENV"
	cfg.Primary.Enabled = true
	cfg.Secondary.Enabled = false

	provider, _, diag := BuildWithDiagnostics(cfg)
	if provider != nil {
		t.Fatal("expected nil provider when primary key missing and no secondary")
	}
	if diag.Primary.Code != DiagAPIKeyMissing {
		t.Fatalf("expected diag code %q, got %q", DiagAPIKeyMissing, diag.Primary.Code)
	}
}
