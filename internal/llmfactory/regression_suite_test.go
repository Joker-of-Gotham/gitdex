package llmfactory

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
)

func TestRegression_ProviderUnavailable_HasDiagnosticCode(t *testing.T) {
	cfg := config.DefaultConfig().LLM
	cfg.Primary.Provider = "deepseek"
	cfg.Primary.Model = "deepseek-chat"
	cfg.Primary.APIKey = ""
	cfg.Primary.APIKeyEnv = "MISSING_DEEPSEEK_KEY"
	cfg.Secondary.Enabled = false

	provider, _, diag := BuildWithDiagnostics(cfg)
	if provider != nil {
		t.Fatal("expected nil provider")
	}
	if diag.Primary.Code == DiagOK {
		t.Fatalf("expected non-OK diagnostic code when provider unavailable")
	}
}

