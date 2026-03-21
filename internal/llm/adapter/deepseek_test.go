package adapter

import "testing"

func TestNewDeepSeekProvider_Defaults(t *testing.T) {
	p := NewDeepSeekProvider(OpenAIConfig{})
	if p.config.Endpoint != DeepSeekDefaultEndpoint {
		t.Errorf("endpoint = %q, want %q", p.config.Endpoint, DeepSeekDefaultEndpoint)
	}
	if p.config.Model != DeepSeekDefaultModel {
		t.Errorf("model = %q, want %q", p.config.Model, DeepSeekDefaultModel)
	}
}

func TestNewDeepSeekProvider_CustomOverride(t *testing.T) {
	p := NewDeepSeekProvider(OpenAIConfig{
		Endpoint: "https://custom-ds.example.com/v1",
		Model:    "deepseek-coder",
		APIKey:   "sk-custom",
	})
	if p.config.Endpoint != "https://custom-ds.example.com/v1" {
		t.Errorf("custom endpoint not preserved")
	}
	if p.config.Model != "deepseek-coder" {
		t.Errorf("custom model not preserved")
	}
	if p.config.APIKey != "sk-custom" {
		t.Errorf("custom api key not preserved")
	}
}
