package llm

import (
	"log"
	"strings"
)

type ProviderUIKind string

const (
	ProviderUILocalModels ProviderUIKind = "local_models"
	ProviderUICloudConfig ProviderUIKind = "cloud_config"
)

type ProviderSpec struct {
	ID                string
	Label             string
	Kind              ProviderUIKind
	DefaultBaseURL    string
	APIKeyEnv         string
	DocsURL           string
	RecommendedModels []string
}

func ProviderSpecs() []ProviderSpec {
	return []ProviderSpec{
		{
			ID:             "ollama",
			Label:          "Ollama",
			Kind:           ProviderUILocalModels,
			DefaultBaseURL: "http://localhost:11434",
			DocsURL:        "https://ollama.com/blog/thinking",
			RecommendedModels: []string{
				"qwen2.5:3b",
				"gemma3:12b",
				"llama3.2:3b",
			},
		},
		{
			ID:             "openai",
			Label:          "OpenAI",
			Kind:           ProviderUICloudConfig,
			DefaultBaseURL: "https://api.openai.com/v1",
			APIKeyEnv:      "OPENAI_API_KEY",
			DocsURL:        "https://platform.openai.com/docs/api-reference/responses",
			RecommendedModels: []string{
				"gpt-4.1-mini",
				"gpt-5-mini",
				"o4-mini",
			},
		},
		{
			ID:             "deepseek",
			Label:          "DeepSeek",
			Kind:           ProviderUICloudConfig,
			DefaultBaseURL: "https://api.deepseek.com",
			APIKeyEnv:      "DEEPSEEK_API_KEY",
			DocsURL:        "https://api-docs.deepseek.com/",
			RecommendedModels: []string{
				"deepseek-chat",
				"deepseek-reasoner",
			},
		},
	}
}

func ProviderSpecFor(id string) ProviderSpec {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, spec := range ProviderSpecs() {
		if spec.ID == id {
			return spec
		}
	}
	if id != "" {
		log.Printf("WARNING: unknown LLM provider %q, falling back to %s", id, ProviderSpecs()[0].ID)
	}
	return ProviderSpecs()[0]
}
