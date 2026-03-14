package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/spf13/viper"
)

var (
	cfg   *Config
	cfgMu sync.RWMutex
)

// Config holds the full application configuration.
type Config struct {
	Suggestion SuggestionConfig `mapstructure:"suggestion" yaml:"suggestion"`
	LLM        LLMConfig        `mapstructure:"llm" yaml:"llm"`
	Sync       SyncConfig       `mapstructure:"sync" yaml:"sync"`
	Automation AutomationConfig `mapstructure:"automation" yaml:"automation"`
	Memory     MemoryConfig     `mapstructure:"memory" yaml:"memory"`
	Platform   PlatformConfig   `mapstructure:"platform" yaml:"platform"`
	Adapters   AdapterConfig    `mapstructure:"adapters" yaml:"adapters"`
	Reports    ReportsConfig    `mapstructure:"reports" yaml:"reports"`
	Theme      ThemeConfig      `mapstructure:"theme" yaml:"theme"`
	I18n       I18nConfig       `mapstructure:"i18n" yaml:"i18n"`
}

// SuggestionConfig controls suggestion behavior.
type SuggestionConfig struct {
	Mode     string `mapstructure:"mode" yaml:"mode"`         // "zen", "full", "manual"
	Language string `mapstructure:"language" yaml:"language"` // "auto", "en", "zh", "ja"
}

// LLMConfig configures the LLM provider.
type LLMConfig struct {
	Provider       string      `mapstructure:"provider" yaml:"provider"`               // "ollama", "openai", "deepseek"
	Model          string      `mapstructure:"model" yaml:"model"`                     // legacy fallback for primary model
	Endpoint       string      `mapstructure:"endpoint" yaml:"endpoint"`               // legacy/default endpoint for both roles
	APIKey         string      `mapstructure:"api_key" yaml:"api_key"`                 // legacy/default API key for both roles
	APIKeyEnv      string      `mapstructure:"api_key_env" yaml:"api_key_env"`         // env var that resolves the API key at runtime
	ContextLength  int         `mapstructure:"context_length" yaml:"context_length"`   // num_ctx for Ollama; 0 = auto-detect
	RequestTimeout int         `mapstructure:"request_timeout" yaml:"request_timeout"` // seconds; 0 = provider default
	Primary        ModelConfig `mapstructure:"primary" yaml:"primary"`
	Secondary      ModelConfig `mapstructure:"secondary" yaml:"secondary"`
}

type ModelConfig struct {
	Provider  string `mapstructure:"provider" yaml:"provider"`
	Model     string `mapstructure:"model" yaml:"model"`
	Endpoint  string `mapstructure:"endpoint" yaml:"endpoint"`
	APIKey    string `mapstructure:"api_key" yaml:"api_key"`
	APIKeyEnv string `mapstructure:"api_key_env" yaml:"api_key_env"`
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
}

// SyncConfig controls network sync freshness behavior.
type SyncConfig struct {
	// AutoFetchInterval is in seconds. 0 disables background fetch.
	AutoFetchInterval int `mapstructure:"auto_fetch_interval" yaml:"auto_fetch_interval"`
}

type AutomationConfig struct {
	Mode               string                        `mapstructure:"mode" yaml:"mode"`
	Enabled            bool                          `mapstructure:"enabled" yaml:"enabled"`
	MonitorInterval    int                           `mapstructure:"monitor_interval" yaml:"monitor_interval"`
	AutoAnalyze        bool                          `mapstructure:"auto_analyze" yaml:"auto_analyze"`
	Unattended         bool                          `mapstructure:"unattended" yaml:"unattended"`
	AutoAcceptSafe     bool                          `mapstructure:"auto_accept_safe" yaml:"auto_accept_safe"`
	TrustedMode        bool                          `mapstructure:"trusted_mode" yaml:"trusted_mode"`
	MaxAutoSteps       int                           `mapstructure:"max_auto_steps" yaml:"max_auto_steps"`
	Schedules          []AutomationSchedule          `mapstructure:"schedules" yaml:"schedules"`
	MaintenanceWindows []AutomationMaintenanceWindow `mapstructure:"maintenance_windows" yaml:"maintenance_windows"`
	ApprovalPolicy     AutomationApprovalPolicy      `mapstructure:"approval_policy" yaml:"approval_policy"`
	TrustPolicy        AutomationTrustPolicy         `mapstructure:"trust_policy" yaml:"trust_policy"`
	Concurrency        AutomationConcurrencyPolicy   `mapstructure:"concurrency" yaml:"concurrency"`
	Escalation         AutomationEscalationPolicy    `mapstructure:"escalation" yaml:"escalation"`
	DeadLetter         AutomationDeadLetterPolicy    `mapstructure:"dead_letter" yaml:"dead_letter"`
}

type AutomationSchedule struct {
	ID         string `mapstructure:"id" yaml:"id"`
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`
	WorkflowID string `mapstructure:"workflow_id" yaml:"workflow_id"`
	Goal       string `mapstructure:"goal" yaml:"goal"`
	Interval   int    `mapstructure:"interval" yaml:"interval"`
}

type AutomationMaintenanceWindow struct {
	Days  []string `mapstructure:"days" yaml:"days"`
	Start string   `mapstructure:"start" yaml:"start"`
	End   string   `mapstructure:"end" yaml:"end"`
}

type AutomationApprovalPolicy struct {
	RequireForPartial       bool `mapstructure:"require_for_partial" yaml:"require_for_partial"`
	RequireForComposed      bool `mapstructure:"require_for_composed" yaml:"require_for_composed"`
	RequireForAdapterBacked bool `mapstructure:"require_for_adapter_backed" yaml:"require_for_adapter_backed"`
	RequireForIrreversible  bool `mapstructure:"require_for_irreversible" yaml:"require_for_irreversible"`
}

type AutomationTrustPolicy struct {
	TrustedCapabilities []string `mapstructure:"trusted_capabilities" yaml:"trusted_capabilities"`
	AllowDangerousGit   bool     `mapstructure:"allow_dangerous_git" yaml:"allow_dangerous_git"`
}

type AutomationConcurrencyPolicy struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

type AutomationEscalationPolicy struct {
	FailureThreshold int `mapstructure:"failure_threshold" yaml:"failure_threshold"`
}

type AutomationDeadLetterPolicy struct {
	PauseAfter int `mapstructure:"pause_after" yaml:"pause_after"`
}

type MemoryConfig struct {
	Episodic MemoryEpisodicConfig `mapstructure:"episodic" yaml:"episodic"`
	Semantic MemorySemanticConfig `mapstructure:"semantic" yaml:"semantic"`
	Task     MemoryTaskConfig     `mapstructure:"task" yaml:"task"`
}

type MemoryEpisodicConfig struct {
	MaxRecentEvents      int `mapstructure:"max_recent_events" yaml:"max_recent_events"`
	MaxArtifactNotes     int `mapstructure:"max_artifact_notes" yaml:"max_artifact_notes"`
	MaxEpisodes          int `mapstructure:"max_episodes" yaml:"max_episodes"`
	CompressionThreshold int `mapstructure:"compression_threshold" yaml:"compression_threshold"`
	MaxPromptEpisodes    int `mapstructure:"max_prompt_episodes" yaml:"max_prompt_episodes"`
}

type MemorySemanticConfig struct {
	MaxFacts       int     `mapstructure:"max_facts" yaml:"max_facts"`
	MaxPromptFacts int     `mapstructure:"max_prompt_facts" yaml:"max_prompt_facts"`
	MaxEvidence    int     `mapstructure:"max_evidence" yaml:"max_evidence"`
	MinScore       float64 `mapstructure:"min_score" yaml:"min_score"`
	DefaultDecay   float64 `mapstructure:"default_decay" yaml:"default_decay"`
}

type MemoryTaskConfig struct {
	MaxConstraints int `mapstructure:"max_constraints" yaml:"max_constraints"`
	MaxPending     int `mapstructure:"max_pending" yaml:"max_pending"`
}

type AdapterConfig struct {
	GitHub    GitHubAdapterConfig      `mapstructure:"github" yaml:"github"`
	GitLab    BrowserOnlyAdapterConfig `mapstructure:"gitlab" yaml:"gitlab"`
	Bitbucket BrowserOnlyAdapterConfig `mapstructure:"bitbucket" yaml:"bitbucket"`
}

type GitHubAdapterConfig struct {
	GH      CommandAdapterConfig `mapstructure:"gh" yaml:"gh"`
	Browser BrowserAdapterConfig `mapstructure:"browser" yaml:"browser"`
}

type BrowserOnlyAdapterConfig struct {
	Browser BrowserAdapterConfig `mapstructure:"browser" yaml:"browser"`
}

type CommandAdapterConfig struct {
	Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
	Binary  string `mapstructure:"binary" yaml:"binary"`
}

type BrowserAdapterConfig struct {
	Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
	Driver  string `mapstructure:"driver" yaml:"driver"`
}

// PlatformConfig holds platform API tokens.
type PlatformConfig struct {
	GitHubToken    string `mapstructure:"github_token" yaml:"github_token"`
	GitLabToken    string `mapstructure:"gitlab_token" yaml:"gitlab_token"`
	BitbucketToken string `mapstructure:"bitbucket_token" yaml:"bitbucket_token"`
}

type ReportsConfig struct {
	ExportDir string `mapstructure:"export_dir" yaml:"export_dir"`
}

// ThemeConfig controls UI theme.
type ThemeConfig struct {
	Name string `mapstructure:"name" yaml:"name"` // "dark", "light", "high-contrast"
}

// I18nConfig controls internationalization.
type I18nConfig struct {
	Language string `mapstructure:"language" yaml:"language"` // "auto", "en", "zh", "ja"
}

// Load reads configuration with 4-tier priority:
// CLI flags > Env vars (GITDEX_*, legacy GITMANUAL_*) > Project (.gitdexrc, legacy .gitmanualrc)
// > Global (~/.config/gitdex/config.yaml, legacy ~/.config/gitmanual/config.yaml) > Defaults
func Load() (*Config, error) {
	v := viper.New()

	setDefaults(v)
	bindEnvAliases(v)

	for _, path := range candidateConfigFiles() {
		if err := mergeConfigFile(v, path); err != nil {
			return nil, fmt.Errorf("config load: merge %s: %w", path, err)
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config load: unmarshal: %w", err)
	}
	normalize(&c)

	if err := Validate(&c); err != nil {
		return nil, fmt.Errorf("config load: validate: %w", err)
	}

	cfgMu.Lock()
	cfg = &c
	cfgMu.Unlock()
	return &c, nil
}

func candidateConfigFiles() []string {
	var files []string

	if path, err := LegacyGlobalConfigPath(); err == nil {
		files = append(files, path)
	}
	if path, err := GlobalConfigPath(); err == nil {
		files = append(files, path)
	}
	files = append(files, existingProjectConfigFiles()...)

	return files
}

func mergeConfigFile(v *viper.Viper, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	return v.MergeInConfig()
}

// Get returns the loaded config. Must call Load() or Set() first.
func Get() *Config {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return cfg
}

// Set assigns the config (e.g. when Load fails and defaults are used).
func Set(c *Config) {
	normalize(c)
	cfgMu.Lock()
	cfg = c
	cfgMu.Unlock()
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
	c.LLM.APIKey = strings.TrimSpace(c.LLM.APIKey)
	c.LLM.APIKeyEnv = strings.TrimSpace(c.LLM.APIKeyEnv)
	c.LLM.Primary.Model = strings.TrimSpace(c.LLM.Primary.Model)
	c.LLM.Primary.Provider = strings.TrimSpace(c.LLM.Primary.Provider)
	c.LLM.Primary.Endpoint = strings.TrimSpace(c.LLM.Primary.Endpoint)
	c.LLM.Primary.APIKey = strings.TrimSpace(c.LLM.Primary.APIKey)
	c.LLM.Primary.APIKeyEnv = strings.TrimSpace(c.LLM.Primary.APIKeyEnv)
	c.LLM.Secondary.Model = strings.TrimSpace(c.LLM.Secondary.Model)
	c.LLM.Secondary.Provider = strings.TrimSpace(c.LLM.Secondary.Provider)
	c.LLM.Secondary.Endpoint = strings.TrimSpace(c.LLM.Secondary.Endpoint)
	c.LLM.Secondary.APIKey = strings.TrimSpace(c.LLM.Secondary.APIKey)
	c.LLM.Secondary.APIKeyEnv = strings.TrimSpace(c.LLM.Secondary.APIKeyEnv)
	c.Theme.Name = strings.TrimSpace(c.Theme.Name)
	c.I18n.Language = strings.TrimSpace(c.I18n.Language)
	c.Automation.Mode = strings.TrimSpace(c.Automation.Mode)
	c.Adapters.GitHub.GH.Binary = strings.TrimSpace(c.Adapters.GitHub.GH.Binary)
	c.Adapters.GitHub.Browser.Driver = strings.TrimSpace(c.Adapters.GitHub.Browser.Driver)
	c.Adapters.GitLab.Browser.Driver = strings.TrimSpace(c.Adapters.GitLab.Browser.Driver)
	c.Adapters.Bitbucket.Browser.Driver = strings.TrimSpace(c.Adapters.Bitbucket.Browser.Driver)
	c.Reports.ExportDir = strings.TrimSpace(c.Reports.ExportDir)
	if strings.TrimSpace(c.LLM.Provider) == "" {
		c.LLM.Provider = "ollama"
	}
	if strings.TrimSpace(c.LLM.Endpoint) == "" {
		c.LLM.Endpoint = defaultEndpointForProvider(c.LLM.Provider)
	}
	if strings.TrimSpace(c.LLM.APIKeyEnv) == "" {
		c.LLM.APIKeyEnv = defaultAPIKeyEnvForProvider(c.LLM.Provider)
	}
	if strings.TrimSpace(c.LLM.Primary.Model) == "" {
		c.LLM.Primary.Model = c.LLM.Model
	}
	if strings.TrimSpace(c.LLM.Primary.Provider) == "" {
		c.LLM.Primary.Provider = c.LLM.Provider
	}
	if strings.TrimSpace(c.LLM.Primary.Endpoint) == "" {
		c.LLM.Primary.Endpoint = c.LLM.Endpoint
	}
	if strings.TrimSpace(c.LLM.Primary.APIKey) == "" {
		c.LLM.Primary.APIKey = c.LLM.APIKey
	}
	if strings.TrimSpace(c.LLM.Primary.APIKeyEnv) == "" {
		c.LLM.Primary.APIKeyEnv = c.LLM.APIKeyEnv
	}
	if strings.TrimSpace(c.LLM.Primary.Model) == "" && RoleProvider(c.LLM.Primary) == "ollama" {
		c.LLM.Primary.Model = "qwen2.5:3b"
	}
	if !c.LLM.Primary.Enabled {
		c.LLM.Primary.Enabled = true
	}
	if strings.TrimSpace(c.LLM.Secondary.Provider) == "" {
		c.LLM.Secondary.Provider = c.LLM.Provider
	}
	if strings.TrimSpace(c.LLM.Secondary.Endpoint) == "" {
		if c.LLM.Secondary.Provider == c.LLM.Primary.Provider {
			c.LLM.Secondary.Endpoint = c.LLM.Primary.Endpoint
		} else {
			c.LLM.Secondary.Endpoint = defaultEndpointForProvider(c.LLM.Secondary.Provider)
		}
	}
	if strings.TrimSpace(c.LLM.Secondary.APIKey) == "" && c.LLM.Secondary.Provider == c.LLM.Provider {
		c.LLM.Secondary.APIKey = c.LLM.APIKey
	}
	if strings.TrimSpace(c.LLM.Secondary.APIKeyEnv) == "" {
		if c.LLM.Secondary.Provider == c.LLM.Primary.Provider {
			c.LLM.Secondary.APIKeyEnv = c.LLM.Primary.APIKeyEnv
		} else {
			c.LLM.Secondary.APIKeyEnv = defaultAPIKeyEnvForProvider(c.LLM.Secondary.Provider)
		}
	}
	// Keep legacy field in sync for older call sites and tests.
	if strings.TrimSpace(c.LLM.Model) == "" && RoleProvider(c.LLM.Primary) == "ollama" {
		c.LLM.Model = c.LLM.Primary.Model
	}
	if strings.TrimSpace(c.LLM.Endpoint) == "" {
		c.LLM.Endpoint = c.LLM.Primary.Endpoint
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
	if c.Automation.MonitorInterval < 0 {
		c.Automation.MonitorInterval = 0
	}
	if c.Automation.MonitorInterval == 0 {
		c.Automation.MonitorInterval = 900
	}
	if c.Automation.MaxAutoSteps <= 0 {
		c.Automation.MaxAutoSteps = 8
	}
	ApplyAutomationMode(&c.Automation)
	if len(c.Automation.MaintenanceWindows) == 0 {
		c.Automation.MaintenanceWindows = nil
	}
	if c.Automation.Escalation.FailureThreshold <= 0 {
		c.Automation.Escalation.FailureThreshold = 3
	}
	if c.Automation.DeadLetter.PauseAfter <= 0 {
		c.Automation.DeadLetter.PauseAfter = 2
	}
	for i := range c.Automation.Schedules {
		c.Automation.Schedules[i].ID = strings.TrimSpace(c.Automation.Schedules[i].ID)
		c.Automation.Schedules[i].WorkflowID = strings.TrimSpace(c.Automation.Schedules[i].WorkflowID)
		c.Automation.Schedules[i].Goal = strings.TrimSpace(c.Automation.Schedules[i].Goal)
		if c.Automation.Schedules[i].Interval < 0 {
			c.Automation.Schedules[i].Interval = 0
		}
	}
	for i := range c.Automation.MaintenanceWindows {
		c.Automation.MaintenanceWindows[i].Start = strings.TrimSpace(c.Automation.MaintenanceWindows[i].Start)
		c.Automation.MaintenanceWindows[i].End = strings.TrimSpace(c.Automation.MaintenanceWindows[i].End)
		for j := range c.Automation.MaintenanceWindows[i].Days {
			c.Automation.MaintenanceWindows[i].Days[j] = strings.TrimSpace(c.Automation.MaintenanceWindows[i].Days[j])
		}
	}
	for i := range c.Automation.TrustPolicy.TrustedCapabilities {
		c.Automation.TrustPolicy.TrustedCapabilities[i] = strings.TrimSpace(c.Automation.TrustPolicy.TrustedCapabilities[i])
	}
	if c.Adapters.GitHub.GH.Binary == "" {
		c.Adapters.GitHub.GH.Binary = "gh"
	}
	if c.Adapters.GitHub.Browser.Driver == "" {
		c.Adapters.GitHub.Browser.Driver = "default"
	}
	if c.Adapters.GitLab.Browser.Driver == "" {
		c.Adapters.GitLab.Browser.Driver = "default"
	}
	if c.Adapters.Bitbucket.Browser.Driver == "" {
		c.Adapters.Bitbucket.Browser.Driver = "default"
	}
}

func bindEnvAliases(v *viper.Viper) {
	envs := map[string][]string{
		"suggestion.mode":                                       {"GITDEX_SUGGESTION_MODE", "GITMANUAL_SUGGESTION_MODE"},
		"suggestion.language":                                   {"GITDEX_SUGGESTION_LANGUAGE", "GITMANUAL_SUGGESTION_LANGUAGE"},
		"llm.provider":                                          {"GITDEX_LLM_PROVIDER", "GITMANUAL_LLM_PROVIDER"},
		"llm.model":                                             {"GITDEX_LLM_MODEL", "GITMANUAL_LLM_MODEL"},
		"llm.endpoint":                                          {"GITDEX_LLM_ENDPOINT", "GITMANUAL_LLM_ENDPOINT"},
		"llm.api_key":                                           {"GITDEX_LLM_API_KEY", "GITMANUAL_LLM_API_KEY"},
		"llm.api_key_env":                                       {"GITDEX_LLM_API_KEY_ENV", "GITMANUAL_LLM_API_KEY_ENV"},
		"llm.context_length":                                    {"GITDEX_LLM_CONTEXT_LENGTH", "GITMANUAL_LLM_CONTEXT_LENGTH"},
		"llm.request_timeout":                                   {"GITDEX_LLM_REQUEST_TIMEOUT", "GITMANUAL_LLM_REQUEST_TIMEOUT"},
		"llm.primary.provider":                                  {"GITDEX_LLM_PRIMARY_PROVIDER", "GITMANUAL_LLM_PRIMARY_PROVIDER"},
		"llm.primary.model":                                     {"GITDEX_LLM_PRIMARY_MODEL", "GITMANUAL_LLM_PRIMARY_MODEL"},
		"llm.primary.endpoint":                                  {"GITDEX_LLM_PRIMARY_ENDPOINT", "GITMANUAL_LLM_PRIMARY_ENDPOINT"},
		"llm.primary.api_key":                                   {"GITDEX_LLM_PRIMARY_API_KEY", "GITMANUAL_LLM_PRIMARY_API_KEY"},
		"llm.primary.api_key_env":                               {"GITDEX_LLM_PRIMARY_API_KEY_ENV", "GITMANUAL_LLM_PRIMARY_API_KEY_ENV"},
		"llm.primary.enabled":                                   {"GITDEX_LLM_PRIMARY_ENABLED", "GITMANUAL_LLM_PRIMARY_ENABLED"},
		"llm.secondary.provider":                                {"GITDEX_LLM_SECONDARY_PROVIDER", "GITMANUAL_LLM_SECONDARY_PROVIDER"},
		"llm.secondary.model":                                   {"GITDEX_LLM_SECONDARY_MODEL", "GITMANUAL_LLM_SECONDARY_MODEL"},
		"llm.secondary.endpoint":                                {"GITDEX_LLM_SECONDARY_ENDPOINT", "GITMANUAL_LLM_SECONDARY_ENDPOINT"},
		"llm.secondary.api_key":                                 {"GITDEX_LLM_SECONDARY_API_KEY", "GITMANUAL_LLM_SECONDARY_API_KEY"},
		"llm.secondary.api_key_env":                             {"GITDEX_LLM_SECONDARY_API_KEY_ENV", "GITMANUAL_LLM_SECONDARY_API_KEY_ENV"},
		"llm.secondary.enabled":                                 {"GITDEX_LLM_SECONDARY_ENABLED", "GITMANUAL_LLM_SECONDARY_ENABLED"},
		"sync.auto_fetch_interval":                              {"GITDEX_SYNC_AUTO_FETCH_INTERVAL", "GITMANUAL_SYNC_AUTO_FETCH_INTERVAL"},
		"automation.enabled":                                    {"GITDEX_AUTOMATION_ENABLED", "GITMANUAL_AUTOMATION_ENABLED"},
		"automation.monitor_interval":                           {"GITDEX_AUTOMATION_MONITOR_INTERVAL", "GITMANUAL_AUTOMATION_MONITOR_INTERVAL"},
		"automation.auto_analyze":                               {"GITDEX_AUTOMATION_AUTO_ANALYZE", "GITMANUAL_AUTOMATION_AUTO_ANALYZE"},
		"automation.unattended":                                 {"GITDEX_AUTOMATION_UNATTENDED", "GITMANUAL_AUTOMATION_UNATTENDED"},
		"automation.auto_accept_safe":                           {"GITDEX_AUTOMATION_AUTO_ACCEPT_SAFE", "GITMANUAL_AUTOMATION_AUTO_ACCEPT_SAFE"},
		"automation.trusted_mode":                               {"GITDEX_AUTOMATION_TRUSTED_MODE", "GITMANUAL_AUTOMATION_TRUSTED_MODE"},
		"automation.max_auto_steps":                             {"GITDEX_AUTOMATION_MAX_AUTO_STEPS", "GITMANUAL_AUTOMATION_MAX_AUTO_STEPS"},
		"automation.approval_policy.require_for_partial":        {"GITDEX_AUTOMATION_APPROVAL_REQUIRE_FOR_PARTIAL", "GITMANUAL_AUTOMATION_APPROVAL_REQUIRE_FOR_PARTIAL"},
		"automation.approval_policy.require_for_composed":       {"GITDEX_AUTOMATION_APPROVAL_REQUIRE_FOR_COMPOSED", "GITMANUAL_AUTOMATION_APPROVAL_REQUIRE_FOR_COMPOSED"},
		"automation.approval_policy.require_for_adapter_backed": {"GITDEX_AUTOMATION_APPROVAL_REQUIRE_FOR_ADAPTER_BACKED", "GITMANUAL_AUTOMATION_APPROVAL_REQUIRE_FOR_ADAPTER_BACKED"},
		"automation.approval_policy.require_for_irreversible":   {"GITDEX_AUTOMATION_APPROVAL_REQUIRE_FOR_IRREVERSIBLE", "GITMANUAL_AUTOMATION_APPROVAL_REQUIRE_FOR_IRREVERSIBLE"},
		"automation.trust_policy.allow_dangerous_git":           {"GITDEX_AUTOMATION_TRUST_ALLOW_DANGEROUS_GIT", "GITMANUAL_AUTOMATION_TRUST_ALLOW_DANGEROUS_GIT"},
		"automation.concurrency.enabled":                        {"GITDEX_AUTOMATION_CONCURRENCY_ENABLED", "GITMANUAL_AUTOMATION_CONCURRENCY_ENABLED"},
		"automation.escalation.failure_threshold":               {"GITDEX_AUTOMATION_ESCALATION_FAILURE_THRESHOLD", "GITMANUAL_AUTOMATION_ESCALATION_FAILURE_THRESHOLD"},
		"automation.dead_letter.pause_after":                    {"GITDEX_AUTOMATION_DEAD_LETTER_PAUSE_AFTER", "GITMANUAL_AUTOMATION_DEAD_LETTER_PAUSE_AFTER"},
		"platform.github_token":                                 {"GITDEX_PLATFORM_GITHUB_TOKEN", "GITMANUAL_PLATFORM_GITHUB_TOKEN"},
		"platform.gitlab_token":                                 {"GITDEX_PLATFORM_GITLAB_TOKEN", "GITMANUAL_PLATFORM_GITLAB_TOKEN"},
		"platform.bitbucket_token":                              {"GITDEX_PLATFORM_BITBUCKET_TOKEN", "GITMANUAL_PLATFORM_BITBUCKET_TOKEN"},
		"adapters.github.gh.enabled":                            {"GITDEX_ADAPTERS_GITHUB_GH_ENABLED", "GITMANUAL_ADAPTERS_GITHUB_GH_ENABLED"},
		"adapters.github.gh.binary":                             {"GITDEX_ADAPTERS_GITHUB_GH_BINARY", "GITMANUAL_ADAPTERS_GITHUB_GH_BINARY"},
		"adapters.github.browser.enabled":                       {"GITDEX_ADAPTERS_GITHUB_BROWSER_ENABLED", "GITMANUAL_ADAPTERS_GITHUB_BROWSER_ENABLED"},
		"adapters.github.browser.driver":                        {"GITDEX_ADAPTERS_GITHUB_BROWSER_DRIVER", "GITMANUAL_ADAPTERS_GITHUB_BROWSER_DRIVER"},
		"adapters.gitlab.browser.enabled":                       {"GITDEX_ADAPTERS_GITLAB_BROWSER_ENABLED", "GITMANUAL_ADAPTERS_GITLAB_BROWSER_ENABLED"},
		"adapters.gitlab.browser.driver":                        {"GITDEX_ADAPTERS_GITLAB_BROWSER_DRIVER", "GITMANUAL_ADAPTERS_GITLAB_BROWSER_DRIVER"},
		"adapters.bitbucket.browser.enabled":                    {"GITDEX_ADAPTERS_BITBUCKET_BROWSER_ENABLED", "GITMANUAL_ADAPTERS_BITBUCKET_BROWSER_ENABLED"},
		"adapters.bitbucket.browser.driver":                     {"GITDEX_ADAPTERS_BITBUCKET_BROWSER_DRIVER", "GITMANUAL_ADAPTERS_BITBUCKET_BROWSER_DRIVER"},
		"reports.export_dir":                                    {"GITDEX_REPORTS_EXPORT_DIR", "GITMANUAL_REPORTS_EXPORT_DIR"},
		"theme.name":                                            {"GITDEX_THEME_NAME", "GITMANUAL_THEME_NAME"},
		"i18n.language":                                         {"GITDEX_I18N_LANGUAGE", "GITMANUAL_I18N_LANGUAGE"},
	}
	for key, aliases := range envs {
		args := append([]string{key}, aliases...)
		_ = v.BindEnv(args...)
	}
}

func defaultEndpointForProvider(provider string) string {
	return llm.ProviderSpecFor(provider).DefaultBaseURL
}

func defaultAPIKeyEnvForProvider(provider string) string {
	return llm.ProviderSpecFor(provider).APIKeyEnv
}
