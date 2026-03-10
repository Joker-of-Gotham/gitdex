package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SaveGlobal writes the current config into the new global config location.
func SaveGlobal(c *Config) error {
	if c == nil {
		return fmt.Errorf("config save: nil config")
	}
	normalize(c)
	if err := Validate(c); err != nil {
		return fmt.Errorf("config save: validate: %w", err)
	}

	path, err := GlobalConfigPath()
	if err != nil {
		return fmt.Errorf("config save: resolve path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config save: mkdir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config save: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("config save: write: %w", err)
	}

	Set(c)
	return nil
}

// UpdateLanguagePreference persists both UI and suggestion language preferences.
func UpdateLanguagePreference(lang string) error {
	current := Get()
	if current == nil {
		current = DefaultConfig()
	}
	next := *current
	next.Suggestion = current.Suggestion
	next.LLM = current.LLM
	next.Sync = current.Sync
	next.Platform = current.Platform
	next.Theme = current.Theme
	next.I18n = current.I18n
	next.Suggestion.Language = lang
	next.I18n.Language = lang
	return SaveGlobal(&next)
}
