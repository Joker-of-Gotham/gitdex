package config

import "github.com/spf13/viper"

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		Suggestion: SuggestionConfig{Mode: "zen", Language: "auto"},
		LLM: LLMConfig{
			Provider:      "ollama",
			Model:         "qwen2.5:3b",
			Endpoint:      "http://localhost:11434",
			ContextLength: 0, // 0 = auto-detect from model metadata
			Primary: ModelConfig{
				Model:   "qwen2.5:3b",
				Enabled: true,
			},
			Secondary: ModelConfig{
				Model:   "",
				Enabled: false,
			},
		},
		Sync: SyncConfig{
			AutoFetchInterval: 300,
		},
		Platform: PlatformConfig{},
		Theme:    ThemeConfig{Name: "dark"},
		I18n:     I18nConfig{Language: "auto"},
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("suggestion.mode", "zen")
	v.SetDefault("suggestion.language", "auto")
	v.SetDefault("llm.provider", "ollama")
	v.SetDefault("llm.model", "qwen2.5:3b")
	v.SetDefault("llm.endpoint", "http://localhost:11434")
	v.SetDefault("llm.context_length", 0)
	v.SetDefault("llm.primary.model", "qwen2.5:3b")
	v.SetDefault("llm.primary.enabled", true)
	v.SetDefault("llm.secondary.model", "")
	v.SetDefault("llm.secondary.enabled", false)
	v.SetDefault("sync.auto_fetch_interval", 300)
	v.SetDefault("platform.github_token", "")
	v.SetDefault("platform.gitlab_token", "")
	v.SetDefault("platform.bitbucket_token", "")
	v.SetDefault("theme.name", "dark")
	v.SetDefault("i18n.language", "auto")
}
