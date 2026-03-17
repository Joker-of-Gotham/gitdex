package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate checks that config values are within allowed sets.
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
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

	validThemes := map[string]bool{
		"catppuccin": true, "dracula": true, "tokyonight": true,
		"gruvbox": true, "nord": true, "dark": true, "light": true,
	}
	if !validThemes[strings.ToLower(cfg.Theme.Name)] {
		return fmt.Errorf("theme.name must be one of catppuccin, dracula, tokyonight, gruvbox, nord, dark, light; got %q", cfg.Theme.Name)
	}
	i18nLang := strings.ToLower(strings.TrimSpace(cfg.I18n.Language))
	if i18nLang == "" {
		i18nLang = "auto"
	}
	if !validLanguages[i18nLang] {
		return fmt.Errorf("i18n.language must be one of auto, en, zh, ja; got %q", cfg.I18n.Language)
	}

	validProviders := map[string]bool{"ollama": true, "openai": true, "deepseek": true}
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	if !validProviders[provider] {
		return fmt.Errorf("llm.provider must be one of ollama, openai, deepseek; got %q", cfg.LLM.Provider)
	}
	if primaryProvider := strings.ToLower(strings.TrimSpace(cfg.LLM.Primary.Provider)); primaryProvider != "" && !validProviders[primaryProvider] {
		return fmt.Errorf("llm.primary.provider must be one of ollama, openai, deepseek; got %q", cfg.LLM.Primary.Provider)
	}
	if secondaryProvider := strings.ToLower(strings.TrimSpace(cfg.LLM.Secondary.Provider)); secondaryProvider != "" && !validProviders[secondaryProvider] {
		return fmt.Errorf("llm.secondary.provider must be one of ollama, openai, deepseek; got %q", cfg.LLM.Secondary.Provider)
	}
	if cfg.LLM.Primary.Enabled && strings.TrimSpace(cfg.LLM.Primary.Model) == "" {
		return fmt.Errorf("llm.primary.model must be set when llm.primary.enabled=true")
	}
	if cfg.LLM.Secondary.Enabled && strings.TrimSpace(cfg.LLM.Secondary.Model) == "" {
		return fmt.Errorf("llm.secondary.model must be set when llm.secondary.enabled=true")
	}
	if cfg.LLM.ContextLength < 0 {
		return fmt.Errorf("llm.context_length must be >= 0; got %d", cfg.LLM.ContextLength)
	}
	if cfg.LLM.RequestTimeout < 0 {
		return fmt.Errorf("llm.request_timeout must be >= 0; got %d", cfg.LLM.RequestTimeout)
	}
	if cfg.Sync.AutoFetchInterval < 0 {
		return fmt.Errorf("sync.auto_fetch_interval must be >= 0; got %d", cfg.Sync.AutoFetchInterval)
	}
	mode := NormalizeAutomationMode(firstNonEmpty(strings.TrimSpace(cfg.Automation.Mode), AutomationModeFromFlags(cfg.Automation)))
	validAutomationModes := map[string]bool{
		AutomationModeManual: true,
		AutomationModeAuto:   true,
		AutomationModeCruise: true,
	}
	if !validAutomationModes[mode] {
		return fmt.Errorf("automation.mode must be one of manual, auto, cruise; got %q", cfg.Automation.Mode)
	}
	if cfg.Automation.MonitorInterval < 0 {
		return fmt.Errorf("automation.monitor_interval must be >= 0; got %d", cfg.Automation.MonitorInterval)
	}
	if cfg.Automation.MaxAutoSteps < 0 {
		return fmt.Errorf("automation.max_auto_steps must be >= 0; got %d", cfg.Automation.MaxAutoSteps)
	}
	if cfg.Automation.Escalation.FailureThreshold < 0 {
		return fmt.Errorf("automation.escalation.failure_threshold must be >= 0; got %d", cfg.Automation.Escalation.FailureThreshold)
	}
	if cfg.Automation.DeadLetter.PauseAfter < 0 {
		return fmt.Errorf("automation.dead_letter.pause_after must be >= 0; got %d", cfg.Automation.DeadLetter.PauseAfter)
	}
	if cfg.Memory.Episodic.MaxRecentEvents <= 0 {
		return fmt.Errorf("memory.episodic.max_recent_events must be > 0; got %d", cfg.Memory.Episodic.MaxRecentEvents)
	}
	if cfg.Memory.Episodic.MaxArtifactNotes <= 0 {
		return fmt.Errorf("memory.episodic.max_artifact_notes must be > 0; got %d", cfg.Memory.Episodic.MaxArtifactNotes)
	}
	if cfg.Memory.Episodic.MaxEpisodes <= 0 {
		return fmt.Errorf("memory.episodic.max_episodes must be > 0; got %d", cfg.Memory.Episodic.MaxEpisodes)
	}
	if cfg.Memory.Episodic.CompressionThreshold <= 0 {
		return fmt.Errorf("memory.episodic.compression_threshold must be > 0; got %d", cfg.Memory.Episodic.CompressionThreshold)
	}
	if cfg.Memory.Episodic.MaxPromptEpisodes <= 0 {
		return fmt.Errorf("memory.episodic.max_prompt_episodes must be > 0; got %d", cfg.Memory.Episodic.MaxPromptEpisodes)
	}
	if cfg.Memory.Semantic.MaxFacts <= 0 {
		return fmt.Errorf("memory.semantic.max_facts must be > 0; got %d", cfg.Memory.Semantic.MaxFacts)
	}
	if cfg.Memory.Semantic.MaxPromptFacts <= 0 {
		return fmt.Errorf("memory.semantic.max_prompt_facts must be > 0; got %d", cfg.Memory.Semantic.MaxPromptFacts)
	}
	if cfg.Memory.Semantic.MaxEvidence <= 0 {
		return fmt.Errorf("memory.semantic.max_evidence must be > 0; got %d", cfg.Memory.Semantic.MaxEvidence)
	}
	if cfg.Memory.Semantic.MinScore < 0 {
		return fmt.Errorf("memory.semantic.min_score must be >= 0; got %f", cfg.Memory.Semantic.MinScore)
	}
	if cfg.Memory.Semantic.DefaultDecay <= 0 {
		return fmt.Errorf("memory.semantic.default_decay must be > 0; got %f", cfg.Memory.Semantic.DefaultDecay)
	}
	if cfg.Memory.Task.MaxConstraints <= 0 {
		return fmt.Errorf("memory.task.max_constraints must be > 0; got %d", cfg.Memory.Task.MaxConstraints)
	}
	if cfg.Memory.Task.MaxPending <= 0 {
		return fmt.Errorf("memory.task.max_pending must be > 0; got %d", cfg.Memory.Task.MaxPending)
	}
	validDays := map[string]bool{
		"mon": true, "monday": true,
		"tue": true, "tuesday": true,
		"wed": true, "wednesday": true,
		"thu": true, "thursday": true,
		"fri": true, "friday": true,
		"sat": true, "saturday": true,
		"sun": true, "sunday": true,
	}
	for idx, window := range cfg.Automation.MaintenanceWindows {
		if strings.TrimSpace(window.Start) == "" || strings.TrimSpace(window.End) == "" {
			return fmt.Errorf("automation.maintenance_windows[%d] start/end must both be set", idx)
		}
		if _, err := time.Parse("15:04", strings.TrimSpace(window.Start)); err != nil {
			return fmt.Errorf("automation.maintenance_windows[%d].start must be HH:MM; got %q", idx, window.Start)
		}
		if _, err := time.Parse("15:04", strings.TrimSpace(window.End)); err != nil {
			return fmt.Errorf("automation.maintenance_windows[%d].end must be HH:MM; got %q", idx, window.End)
		}
		for dayIdx, day := range window.Days {
			if !validDays[strings.ToLower(strings.TrimSpace(day))] {
				return fmt.Errorf("automation.maintenance_windows[%d].days[%d] invalid day %q", idx, dayIdx, day)
			}
		}
	}
	if cfg.Adapters.GitHub.GH.Enabled && strings.TrimSpace(cfg.Adapters.GitHub.GH.Binary) == "" {
		return fmt.Errorf("adapters.github.gh.binary must be set when adapters.github.gh.enabled=true")
	}
	if cfg.Adapters.Git.Enabled && strings.TrimSpace(cfg.Adapters.Git.Binary) == "" {
		return fmt.Errorf("adapters.git.binary must be set when adapters.git.enabled=true")
	}
	if strings.TrimSpace(cfg.Adapters.GitHub.Browser.Driver) == "" {
		return fmt.Errorf("adapters.github.browser.driver must not be empty")
	}
	if strings.TrimSpace(cfg.Adapters.GitLab.Browser.Driver) == "" {
		return fmt.Errorf("adapters.gitlab.browser.driver must not be empty")
	}
	if strings.TrimSpace(cfg.Adapters.Bitbucket.Browser.Driver) == "" {
		return fmt.Errorf("adapters.bitbucket.browser.driver must not be empty")
	}

	return nil
}
