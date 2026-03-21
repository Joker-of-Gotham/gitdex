package adapter

import (
	"strings"
	"testing"
)

func TestNewProviderFromConfig_OpenAI(t *testing.T) {
	p, err := NewProviderFromConfig("openai", "gpt-4", "sk-test", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oai, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatal("expected *OpenAIProvider")
	}
	if oai.config.Model != "gpt-4" {
		t.Errorf("model = %q, want gpt-4", oai.config.Model)
	}
	if oai.config.APIKey != "sk-test" {
		t.Errorf("api key mismatch")
	}
}

func TestNewProviderFromConfig_EmptyDefaultsToOpenAI(t *testing.T) {
	p, err := NewProviderFromConfig("", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oai, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatal("expected *OpenAIProvider for empty provider")
	}
	if oai.config.Model != "gpt-4o-mini" {
		t.Errorf("default model = %q, want gpt-4o-mini", oai.config.Model)
	}
}

func TestNewProviderFromConfig_DeepSeek(t *testing.T) {
	p, err := NewProviderFromConfig("deepseek", "", "sk-ds", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oai := p.(*OpenAIProvider)
	if oai.config.Model != DeepSeekDefaultModel {
		t.Errorf("model = %q, want %q", oai.config.Model, DeepSeekDefaultModel)
	}
	if oai.config.Endpoint != DeepSeekDefaultEndpoint {
		t.Errorf("endpoint = %q, want %q", oai.config.Endpoint, DeepSeekDefaultEndpoint)
	}
}

func TestNewProviderFromConfig_Ollama(t *testing.T) {
	p, err := NewProviderFromConfig("ollama", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oai := p.(*OpenAIProvider)
	if oai.config.Model != OllamaDefaultModel {
		t.Errorf("model = %q, want %q", oai.config.Model, OllamaDefaultModel)
	}
	if oai.config.Endpoint != OllamaDefaultEndpoint {
		t.Errorf("endpoint = %q, want %q", oai.config.Endpoint, OllamaDefaultEndpoint)
	}
}

func TestNewProviderFromConfig_CaseInsensitive(t *testing.T) {
	for _, name := range []string{"OPENAI", "OpenAI", "Deepseek", "OLLAMA"} {
		_, err := NewProviderFromConfig(name, "", "", "")
		if err != nil {
			t.Errorf("NewProviderFromConfig(%q) error: %v", name, err)
		}
	}
}

func TestNewProviderFromConfig_UnsupportedProvider(t *testing.T) {
	_, err := NewProviderFromConfig("claude", "", "", "")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error = %q, want to contain 'unsupported'", err.Error())
	}
}

func TestNewProviderFromConfig_CustomEndpoint(t *testing.T) {
	p, err := NewProviderFromConfig("openai", "gpt-4", "", "http://custom:8080/v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oai := p.(*OpenAIProvider)
	if oai.config.Endpoint != "http://custom:8080/v1" {
		t.Errorf("endpoint = %q, want custom", oai.config.Endpoint)
	}
}

func TestDefaultModelForProvider(t *testing.T) {
	tests := map[string]string{
		"openai":   "gpt-4o-mini",
		"deepseek": DeepSeekDefaultModel,
		"ollama":   OllamaDefaultModel,
		"unknown":  "gpt-4o-mini",
	}
	for provider, want := range tests {
		if got := DefaultModelForProvider(provider); got != want {
			t.Errorf("DefaultModelForProvider(%q) = %q, want %q", provider, got, want)
		}
	}
}

func TestDefaultEndpointForProvider(t *testing.T) {
	tests := map[string]string{
		"openai":   "https://api.openai.com/v1",
		"deepseek": DeepSeekDefaultEndpoint,
		"ollama":   OllamaDefaultEndpoint,
		"unknown":  "https://api.openai.com/v1",
	}
	for provider, want := range tests {
		if got := DefaultEndpointForProvider(provider); got != want {
			t.Errorf("DefaultEndpointForProvider(%q) = %q, want %q", provider, got, want)
		}
	}
}

func TestSupportedProviders(t *testing.T) {
	if len(SupportedProviders) != 3 {
		t.Errorf("SupportedProviders = %d, want 3", len(SupportedProviders))
	}
	expected := map[string]bool{"openai": true, "deepseek": true, "ollama": true}
	for _, p := range SupportedProviders {
		if !expected[p] {
			t.Errorf("unexpected provider in SupportedProviders: %q", p)
		}
	}
}
