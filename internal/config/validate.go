package config

import (
	"fmt"
	"strings"
)

// Validate checks that config values are within allowed sets.
func Validate(cfg *Config) error {
	validModes := map[string]bool{"zen": true, "full": true, "manual": true}
	if !validModes[strings.ToLower(cfg.Suggestion.Mode)] {
		return fmt.Errorf("suggestion.mode must be one of zen, full, manual; got %q", cfg.Suggestion.Mode)
	}
	validLanguages := map[string]bool{"auto": true, "en": true, "zh": true, "ja": true}
	suggestionLang := strings.ToLower(strings.TrimSpace(cfg.Suggestion.Language))
	if suggestionLang == "" {
		suggestionLang = "auto"
	}
	if !validLanguages[suggestionLang] {
		return fmt.Errorf("suggestion.language must be one of auto, en, zh, ja; got %q", cfg.Suggestion.Language)
	}

	validThemes := map[string]bool{"dark": true, "light": true, "high-contrast": true}
	if !validThemes[strings.ToLower(cfg.Theme.Name)] {
		return fmt.Errorf("theme.name must be one of dark, light, high-contrast; got %q", cfg.Theme.Name)
	}
	i18nLang := strings.ToLower(strings.TrimSpace(cfg.I18n.Language))
	if i18nLang == "" {
		i18nLang = "auto"
	}
	if !validLanguages[i18nLang] {
		return fmt.Errorf("i18n.language must be one of auto, en, zh, ja; got %q", cfg.I18n.Language)
	}

	if strings.ToLower(cfg.LLM.Provider) != "ollama" {
		return fmt.Errorf("llm.provider must be ollama; got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.Primary.Enabled && strings.TrimSpace(cfg.LLM.Primary.Model) == "" {
		return fmt.Errorf("llm.primary.model must be set when llm.primary.enabled=true")
	}
	if cfg.LLM.Secondary.Enabled && strings.TrimSpace(cfg.LLM.Secondary.Model) == "" {
		return fmt.Errorf("llm.secondary.model must be set when llm.secondary.enabled=true")
	}
	if cfg.Sync.AutoFetchInterval < 0 {
		return fmt.Errorf("sync.auto_fetch_interval must be >= 0; got %d", cfg.Sync.AutoFetchInterval)
	}

	return nil
}
