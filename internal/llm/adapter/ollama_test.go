package adapter

import "testing"

func TestNewOllamaProvider_Defaults(t *testing.T) {
	p := NewOllamaProvider(OpenAIConfig{})
	if p.config.Endpoint != OllamaDefaultEndpoint {
		t.Errorf("endpoint = %q, want %q", p.config.Endpoint, OllamaDefaultEndpoint)
	}
	if p.config.Model != OllamaDefaultModel {
		t.Errorf("model = %q, want %q", p.config.Model, OllamaDefaultModel)
	}
}

func TestNewOllamaProvider_CustomOverride(t *testing.T) {
	p := NewOllamaProvider(OpenAIConfig{
		Endpoint: "http://192.168.1.100:11434/v1",
		Model:    "mistral",
	})
	if p.config.Endpoint != "http://192.168.1.100:11434/v1" {
		t.Errorf("custom endpoint not preserved")
	}
	if p.config.Model != "mistral" {
		t.Errorf("custom model not preserved")
	}
}

func TestNewOllamaProvider_NoAPIKey(t *testing.T) {
	p := NewOllamaProvider(OpenAIConfig{})
	if p.config.APIKey != "" {
		t.Errorf("ollama should not require API key, got %q", p.config.APIKey)
	}
}
