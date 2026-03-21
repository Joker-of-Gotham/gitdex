package adapter

import (
	"fmt"
	"strings"
)

var SupportedProviders = []string{"openai", "deepseek", "ollama"}

func NewProviderFromConfig(provider, model, apiKey, endpoint string) (Provider, error) {
	cfg := OpenAIConfig{
		APIKey:   apiKey,
		Endpoint: endpoint,
		Model:    model,
	}
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai", "":
		return NewOpenAIProvider(cfg), nil
	case "deepseek":
		return NewDeepSeekProvider(cfg), nil
	case "ollama":
		return NewOllamaProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider %q; supported: %s", provider, strings.Join(SupportedProviders, ", "))
	}
}

func DefaultModelForProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "deepseek":
		return DeepSeekDefaultModel
	case "ollama":
		return OllamaDefaultModel
	default:
		return "gpt-4o-mini"
	}
}

func DefaultEndpointForProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "deepseek":
		return DeepSeekDefaultEndpoint
	case "ollama":
		return OllamaDefaultEndpoint
	default:
		return "https://api.openai.com/v1"
	}
}
