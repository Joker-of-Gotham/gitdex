package engine

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func TestParseLLMResponsePlatformExecSuggestion(t *testing.T) {
	state := &status.GitState{}
	text := `{
  "analysis": "inspect pages configuration",
  "suggestions": [
    {
      "action": "Inspect Pages site",
      "reason": "Need the current Pages source before mutating it",
      "risk": "safe",
      "interaction": "platform_exec",
      "capability_id": "pages",
      "flow": "inspect",
      "scope": {"scope": "repo"},
      "query": {"include": "latest_build"}
    }
  ]
}`

	parsed, err := parseLLMResponse(state, text)
	if err != nil {
		t.Fatalf("parseLLMResponse returned error: %v", err)
	}
	if len(parsed.suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(parsed.suggestions))
	}
	got := parsed.suggestions[0]
	if got.PlatformOp == nil {
		t.Fatalf("expected platform op metadata")
	}
	if got.PlatformOp.CapabilityID != "pages" || got.PlatformOp.Flow != "inspect" {
		t.Fatalf("unexpected platform op: %#v", got.PlatformOp)
	}
	if got.PlatformOp.Query["include"] != "latest_build" {
		t.Fatalf("expected query to be preserved")
	}
}

func TestConvertSuggestionPlatformExecInputs(t *testing.T) {
	s, err := convertSuggestion(nil, 0, llmSuggestionJSON{
		Action:       "Create webhook",
		Reason:       "Need a delivery endpoint",
		Risk:         "caution",
		Interaction:  "platform_exec",
		CapabilityID: "webhooks",
		Flow:         "mutate",
		Operation:    "create",
		Payload:      []byte(`{"config":{"url":"<endpoint>"}}`),
		Inputs: []llmInputJSON{{
			Key:         "endpoint",
			Label:       "Webhook URL",
			Placeholder: "https://example.com/hook",
			ArgIndex:    -1,
		}},
	})
	if err != nil {
		t.Fatalf("convertSuggestion returned error: %v", err)
	}
	if len(s.Inputs) != 1 || s.Inputs[0].ArgIndex != -1 {
		t.Fatalf("expected platform input metadata, got %#v", s.Inputs)
	}
	if s.PlatformOp == nil || string(s.PlatformOp.Payload) == "" {
		t.Fatalf("expected payload to be preserved")
	}
}
