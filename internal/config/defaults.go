package config

import "github.com/spf13/viper"

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		Version:    CurrentConfigVersion,
		Suggestion: SuggestionConfig{Mode: "zen", Language: "auto"},
		LLM: LLMConfig{
			Provider:       DefaultProvider,
			Model:          DefaultModel,
			Endpoint:       DefaultOllamaEndpoint,
			APIKey:         "",
			APIKeyEnv:      "",
			ContextLength:  0,   // 0 = auto-detect from model metadata
			RequestTimeout: 300, // 5 minutes default for LLM generation
			Primary: ModelConfig{
				Provider:  DefaultProvider,
				Model:     DefaultModel,
				Endpoint:  DefaultOllamaEndpoint,
				APIKey:    "",
				APIKeyEnv: "",
				Enabled:   true,
			},
			Secondary: ModelConfig{
				Provider:  DefaultProvider,
				Model:     "",
				Endpoint:  DefaultOllamaEndpoint,
				APIKey:    "",
				APIKeyEnv: "",
				Enabled:   false,
			},
		},
		Sync: SyncConfig{
			AutoFetchInterval: 300,
		},
		Automation: AutomationConfig{
			Mode:               AutomationModeManual,
			Enabled:            true,
			MonitorInterval:    900,
			AutoAnalyze:        true,
			Unattended:         false,
			AutoAcceptSafe:     false,
			TrustedMode:        false,
			MaxAutoSteps:       8,
			Schedules:          nil,
			MaintenanceWindows: nil,
			ApprovalPolicy: AutomationApprovalPolicy{
				RequireForPartial:       true,
				RequireForComposed:      true,
				RequireForAdapterBacked: true,
				RequireForIrreversible:  true,
			},
			TrustPolicy: AutomationTrustPolicy{
				TrustedCapabilities: nil,
				AllowDangerousGit:   false,
			},
			Concurrency: AutomationConcurrencyPolicy{
				Enabled: true,
			},
			Escalation: AutomationEscalationPolicy{
				FailureThreshold: 3,
			},
			DeadLetter: AutomationDeadLetterPolicy{
				PauseAfter: 2,
			},
		},
		Memory: MemoryConfig{
			Episodic: MemoryEpisodicConfig{
				MaxRecentEvents:      40,
				MaxArtifactNotes:     20,
				MaxEpisodes:          24,
				CompressionThreshold: 20,
				MaxPromptEpisodes:    8,
			},
			Semantic: MemorySemanticConfig{
				MaxFacts:       12,
				MaxPromptFacts: 8,
				MaxEvidence:    6,
				MinScore:       0.10,
				DefaultDecay:   0.02,
			},
			Task: MemoryTaskConfig{
				MaxConstraints: 8,
				MaxPending:     8,
			},
		},
		Platform: PlatformConfig{},
		Adapters: AdapterConfig{
			Git: CommandAdapterConfig{
				Enabled: true,
				Binary:  "git",
			},
			GitHub: GitHubAdapterConfig{
				GH: CommandAdapterConfig{
					Enabled: true,
					Binary:  "gh",
				},
				Browser: BrowserAdapterConfig{
					Enabled: false,
					Driver:  "default",
				},
			},
			GitLab: BrowserOnlyAdapterConfig{
				Browser: BrowserAdapterConfig{
					Enabled: false,
					Driver:  "default",
				},
			},
			Bitbucket: BrowserOnlyAdapterConfig{
				Browser: BrowserAdapterConfig{
					Enabled: false,
					Driver:  "default",
				},
			},
		},
		Reports: ReportsConfig{},
		Theme:   ThemeConfig{Name: "catppuccin"},
		I18n:    I18nConfig{Language: "auto"},
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("version", CurrentConfigVersion)
	v.SetDefault("suggestion.mode", "zen")
	v.SetDefault("suggestion.language", "auto")
	v.SetDefault("automation.mode", AutomationModeManual)
	v.SetDefault("llm.provider", DefaultProvider)
	v.SetDefault("llm.model", DefaultModel)
	v.SetDefault("llm.endpoint", DefaultOllamaEndpoint)
	v.SetDefault("llm.api_key", "")
	v.SetDefault("llm.api_key_env", "")
	v.SetDefault("llm.context_length", 0)
	v.SetDefault("llm.request_timeout", 300)
	v.SetDefault("llm.primary.provider", DefaultProvider)
	v.SetDefault("llm.primary.model", DefaultModel)
	v.SetDefault("llm.primary.endpoint", DefaultOllamaEndpoint)
	v.SetDefault("llm.primary.api_key", "")
	v.SetDefault("llm.primary.api_key_env", "")
	v.SetDefault("llm.primary.enabled", true)
	v.SetDefault("llm.secondary.provider", DefaultProvider)
	v.SetDefault("llm.secondary.model", "")
	v.SetDefault("llm.secondary.endpoint", DefaultOllamaEndpoint)
	v.SetDefault("llm.secondary.api_key", "")
	v.SetDefault("llm.secondary.api_key_env", "")
	v.SetDefault("llm.secondary.enabled", false)
	v.SetDefault("sync.auto_fetch_interval", 300)
	v.SetDefault("automation.enabled", true)
	v.SetDefault("automation.monitor_interval", 900)
	v.SetDefault("automation.auto_analyze", true)
	v.SetDefault("automation.unattended", false)
	v.SetDefault("automation.auto_accept_safe", false)
	v.SetDefault("automation.trusted_mode", false)
	v.SetDefault("automation.max_auto_steps", 8)
	v.SetDefault("automation.schedules", []map[string]any{})
	v.SetDefault("automation.maintenance_windows", []map[string]any{})
	v.SetDefault("automation.approval_policy.require_for_partial", true)
	v.SetDefault("automation.approval_policy.require_for_composed", true)
	v.SetDefault("automation.approval_policy.require_for_adapter_backed", true)
	v.SetDefault("automation.approval_policy.require_for_irreversible", true)
	v.SetDefault("automation.trust_policy.trusted_capabilities", []string{})
	v.SetDefault("automation.trust_policy.allow_dangerous_git", false)
	v.SetDefault("automation.concurrency.enabled", true)
	v.SetDefault("automation.escalation.failure_threshold", 3)
	v.SetDefault("automation.dead_letter.pause_after", 2)
	v.SetDefault("memory.episodic.max_recent_events", 40)
	v.SetDefault("memory.episodic.max_artifact_notes", 20)
	v.SetDefault("memory.episodic.max_episodes", 24)
	v.SetDefault("memory.episodic.compression_threshold", 20)
	v.SetDefault("memory.episodic.max_prompt_episodes", 8)
	v.SetDefault("memory.semantic.max_facts", 12)
	v.SetDefault("memory.semantic.max_prompt_facts", 8)
	v.SetDefault("memory.semantic.max_evidence", 6)
	v.SetDefault("memory.semantic.min_score", 0.10)
	v.SetDefault("memory.semantic.default_decay", 0.02)
	v.SetDefault("memory.task.max_constraints", 8)
	v.SetDefault("memory.task.max_pending", 8)
	v.SetDefault("platform.github_token", "")
	v.SetDefault("platform.gitlab_token", "")
	v.SetDefault("platform.bitbucket_token", "")
	v.SetDefault("adapters.git.enabled", true)
	v.SetDefault("adapters.git.binary", "git")
	v.SetDefault("adapters.github.gh.enabled", true)
	v.SetDefault("adapters.github.gh.binary", "gh")
	v.SetDefault("adapters.github.browser.enabled", false)
	v.SetDefault("adapters.github.browser.driver", "default")
	v.SetDefault("adapters.gitlab.browser.enabled", false)
	v.SetDefault("adapters.gitlab.browser.driver", "default")
	v.SetDefault("adapters.bitbucket.browser.enabled", false)
	v.SetDefault("adapters.bitbucket.browser.driver", "default")
	v.SetDefault("reports.export_dir", "")
	v.SetDefault("theme.name", "catppuccin")
	v.SetDefault("i18n.language", "auto")
}
