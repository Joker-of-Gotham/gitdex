package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var cfg *Config

// Config holds the full application configuration.
type Config struct {
	Suggestion SuggestionConfig `mapstructure:"suggestion"`
	LLM        LLMConfig        `mapstructure:"llm"`
	Sync       SyncConfig       `mapstructure:"sync"`
	Platform   PlatformConfig   `mapstructure:"platform"`
	Theme      ThemeConfig      `mapstructure:"theme"`
	I18n       I18nConfig       `mapstructure:"i18n"`
}

// SuggestionConfig controls suggestion behavior.
type SuggestionConfig struct {
	Mode     string `mapstructure:"mode"`     // "zen", "full", "manual"
	Language string `mapstructure:"language"` // "auto", "en", "zh", "ja"
}

// LLMConfig configures the LLM provider.
type LLMConfig struct {
	Provider      string      `mapstructure:"provider"`       // "ollama"
	Model         string      `mapstructure:"model"`          // legacy fallback for primary model
	Endpoint      string      `mapstructure:"endpoint"`       // "http://localhost:11434"
	ContextLength int         `mapstructure:"context_length"` // num_ctx for Ollama; 0 = auto-detect
	Primary       ModelConfig `mapstructure:"primary"`
	Secondary     ModelConfig `mapstructure:"secondary"`
}

type ModelConfig struct {
	Model   string `mapstructure:"model"`
	Enabled bool   `mapstructure:"enabled"`
}

// SyncConfig controls network sync freshness behavior.
type SyncConfig struct {
	// AutoFetchInterval is in seconds. 0 disables background fetch.
	AutoFetchInterval int `mapstructure:"auto_fetch_interval"`
}

// PlatformConfig holds platform API tokens.
type PlatformConfig struct {
	GitHubToken    string `mapstructure:"github_token"`
	GitLabToken    string `mapstructure:"gitlab_token"`
	BitbucketToken string `mapstructure:"bitbucket_token"`
}

// ThemeConfig controls UI theme.
type ThemeConfig struct {
	Name string `mapstructure:"name"` // "dark", "light", "high-contrast"
}

// I18nConfig controls internationalization.
type I18nConfig struct {
	Language string `mapstructure:"language"` // "auto", "en", "zh", "ja"
}

// Load reads configuration with 4-tier priority:
// CLI flags > Env vars (GITDEX_*, legacy GITMANUAL_*) > Project (.gitdexrc, legacy .gitmanualrc)
// > Global (~/.config/gitdex/config.yaml, legacy ~/.config/gitmanual/config.yaml) > Defaults
func Load() (*Config, error) {
	v := viper.New()

	setDefaults(v)

	v.SetConfigName("config")
	v.AddConfigPath(".")
	if dir, err := GlobalConfigDir(); err == nil {
		v.AddConfigPath(dir)
	}
	if dir, err := LegacyGlobalConfigDir(); err == nil {
		v.AddConfigPath(dir)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("config load: read config file: %w", err)
		}
		// Config file not found is ok; we use defaults
	}

	for _, path := range existingProjectConfigFiles() {
		v.SetConfigFile(path)
		v.SetConfigType("yaml")
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("config load: merge %s: %w", path, err)
		}
	}

	bindEnvAliases(v)

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config load: unmarshal: %w", err)
	}
	normalize(&c)

	if err := Validate(&c); err != nil {
		return nil, fmt.Errorf("config load: validate: %w", err)
	}

	cfg = &c
	return cfg, nil
}

// Get returns the loaded config. Must call Load() or Set() first.
func Get() *Config {
	return cfg
}

// Set assigns the config (e.g. when Load fails and defaults are used).
func Set(c *Config) {
	normalize(c)
	cfg = c
}

func normalize(c *Config) {
	if c == nil {
		return
	}
	c.Suggestion.Mode = strings.TrimSpace(c.Suggestion.Mode)
	c.Suggestion.Language = strings.TrimSpace(c.Suggestion.Language)
	c.LLM.Provider = strings.TrimSpace(c.LLM.Provider)
	c.LLM.Model = strings.TrimSpace(c.LLM.Model)
	c.LLM.Endpoint = strings.TrimSpace(c.LLM.Endpoint)
	c.LLM.Primary.Model = strings.TrimSpace(c.LLM.Primary.Model)
	c.LLM.Secondary.Model = strings.TrimSpace(c.LLM.Secondary.Model)
	c.Theme.Name = strings.TrimSpace(c.Theme.Name)
	c.I18n.Language = strings.TrimSpace(c.I18n.Language)
	if strings.TrimSpace(c.LLM.Primary.Model) == "" {
		c.LLM.Primary.Model = c.LLM.Model
	}
	if strings.TrimSpace(c.LLM.Primary.Model) == "" {
		c.LLM.Primary.Model = "qwen2.5:3b"
	}
	if !c.LLM.Primary.Enabled {
		c.LLM.Primary.Enabled = true
	}
	// Keep legacy field in sync for older call sites and tests.
	if strings.TrimSpace(c.LLM.Model) == "" {
		c.LLM.Model = c.LLM.Primary.Model
	}
	if c.Suggestion.Language == "" {
		c.Suggestion.Language = "auto"
	}
	if c.I18n.Language == "" {
		c.I18n.Language = "auto"
	}
	if c.Theme.Name == "" {
		c.Theme.Name = "dark"
	}
}

func bindEnvAliases(v *viper.Viper) {
	envs := map[string][]string{
		"suggestion.mode":          {"GITDEX_SUGGESTION_MODE", "GITMANUAL_SUGGESTION_MODE"},
		"suggestion.language":      {"GITDEX_SUGGESTION_LANGUAGE", "GITMANUAL_SUGGESTION_LANGUAGE"},
		"llm.provider":             {"GITDEX_LLM_PROVIDER", "GITMANUAL_LLM_PROVIDER"},
		"llm.model":                {"GITDEX_LLM_MODEL", "GITMANUAL_LLM_MODEL"},
		"llm.endpoint":             {"GITDEX_LLM_ENDPOINT", "GITMANUAL_LLM_ENDPOINT"},
		"llm.context_length":       {"GITDEX_LLM_CONTEXT_LENGTH", "GITMANUAL_LLM_CONTEXT_LENGTH"},
		"llm.primary.model":        {"GITDEX_LLM_PRIMARY_MODEL", "GITMANUAL_LLM_PRIMARY_MODEL"},
		"llm.primary.enabled":      {"GITDEX_LLM_PRIMARY_ENABLED", "GITMANUAL_LLM_PRIMARY_ENABLED"},
		"llm.secondary.model":      {"GITDEX_LLM_SECONDARY_MODEL", "GITMANUAL_LLM_SECONDARY_MODEL"},
		"llm.secondary.enabled":    {"GITDEX_LLM_SECONDARY_ENABLED", "GITMANUAL_LLM_SECONDARY_ENABLED"},
		"sync.auto_fetch_interval": {"GITDEX_SYNC_AUTO_FETCH_INTERVAL", "GITMANUAL_SYNC_AUTO_FETCH_INTERVAL"},
		"platform.github_token":    {"GITDEX_PLATFORM_GITHUB_TOKEN", "GITMANUAL_PLATFORM_GITHUB_TOKEN"},
		"platform.gitlab_token":    {"GITDEX_PLATFORM_GITLAB_TOKEN", "GITMANUAL_PLATFORM_GITLAB_TOKEN"},
		"platform.bitbucket_token": {"GITDEX_PLATFORM_BITBUCKET_TOKEN", "GITMANUAL_PLATFORM_BITBUCKET_TOKEN"},
		"theme.name":               {"GITDEX_THEME_NAME", "GITMANUAL_THEME_NAME"},
		"i18n.language":            {"GITDEX_I18N_LANGUAGE", "GITMANUAL_I18N_LANGUAGE"},
	}
	for key, aliases := range envs {
		args := append([]string{key}, aliases...)
		_ = v.BindEnv(args...)
	}
}
