package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithDefaults(t *testing.T) {
	configureConfigEnv(t)

	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, "zen", loaded.Suggestion.Mode)
	assert.Equal(t, "auto", loaded.Suggestion.Language)
	assert.Equal(t, "ollama", loaded.LLM.Provider)
	assert.Equal(t, "qwen2.5:3b", loaded.LLM.Model)
	assert.Equal(t, "qwen2.5:3b", loaded.LLM.Primary.Model)
	assert.True(t, loaded.LLM.Primary.Enabled)
	assert.False(t, loaded.LLM.Secondary.Enabled)
	assert.Equal(t, "http://localhost:11434", loaded.LLM.Endpoint)
	assert.Equal(t, 300, loaded.Sync.AutoFetchInterval)
	assert.True(t, loaded.Automation.Enabled)
	assert.Equal(t, 900, loaded.Automation.MonitorInterval)
	assert.Equal(t, 8, loaded.Automation.MaxAutoSteps)
	assert.Empty(t, loaded.Automation.Schedules)
	assert.True(t, loaded.Automation.ApprovalPolicy.RequireForPartial)
	assert.True(t, loaded.Automation.ApprovalPolicy.RequireForComposed)
	assert.True(t, loaded.Automation.Concurrency.Enabled)
	assert.Equal(t, 3, loaded.Automation.Escalation.FailureThreshold)
	assert.Equal(t, 2, loaded.Automation.DeadLetter.PauseAfter)
	assert.True(t, loaded.Adapters.Git.Enabled)
	assert.Equal(t, "git", loaded.Adapters.Git.Binary)
	assert.True(t, loaded.Adapters.GitHub.GH.Enabled)
	assert.Equal(t, "gh", loaded.Adapters.GitHub.GH.Binary)
	assert.Equal(t, "default", loaded.Adapters.GitHub.Browser.Driver)
	assert.Equal(t, "default", loaded.Adapters.GitLab.Browser.Driver)
	assert.Equal(t, "default", loaded.Adapters.Bitbucket.Browser.Driver)
	assert.Equal(t, "catppuccin", loaded.Theme.Name)
	assert.Equal(t, "auto", loaded.I18n.Language)
}

func TestLoad_ProjectBrowserAdapterConfig(t *testing.T) {
	configureConfigEnv(t)

	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ProjectConfigName), []byte(`
adapters:
  gitlab:
    browser:
      enabled: true
      driver: "playwright"
  bitbucket:
    browser:
      enabled: true
      driver: "selenium"
`), 0o600))

	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workDir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)
	assert.True(t, loaded.Adapters.GitLab.Browser.Enabled)
	assert.Equal(t, "playwright", loaded.Adapters.GitLab.Browser.Driver)
	assert.True(t, loaded.Adapters.Bitbucket.Browser.Enabled)
	assert.Equal(t, "selenium", loaded.Adapters.Bitbucket.Browser.Driver)
}

func TestLoad_ProjectAutomationSchedules(t *testing.T) {
	configureConfigEnv(t)

	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ProjectConfigName), []byte(`
automation:
  schedules:
    - id: "advanced-security-hourly"
      enabled: true
      workflow_id: "advanced_security"
      interval: 3600
`), 0o600))

	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workDir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)
	require.Len(t, loaded.Automation.Schedules, 1)
	assert.Equal(t, "advanced-security-hourly", loaded.Automation.Schedules[0].ID)
	assert.Equal(t, "advanced_security", loaded.Automation.Schedules[0].WorkflowID)
	assert.Equal(t, 3600, loaded.Automation.Schedules[0].Interval)
}

func TestValidate_MaintenanceWindowRequiresValidTimeAndDay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Automation.MaintenanceWindows = []AutomationMaintenanceWindow{{
		Days:  []string{"mon", "friyay"},
		Start: "09:00",
		End:   "18:00",
	}}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid day")
}

func TestLoad_GlobalConfig(t *testing.T) {
	configureConfigEnv(t)

	globalDir, err := GlobalConfigDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(`
suggestion:
  mode: "full"
`), 0o600))

	workDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workDir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "full", loaded.Suggestion.Mode)
}

func TestLoad_LegacyGlobalConfigFallback(t *testing.T) {
	configureConfigEnv(t)

	globalDir, err := LegacyGlobalConfigDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(`
suggestion:
  mode: "manual"
`), 0o600))

	workDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workDir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "manual", loaded.Suggestion.Mode)
}

func TestLoad_NewGlobalConfigOverridesLegacy(t *testing.T) {
	configureConfigEnv(t)

	newDir, err := GlobalConfigDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(newDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "config.yaml"), []byte(`
suggestion:
  mode: "full"
`), 0o600))

	legacyDir, err := LegacyGlobalConfigDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte(`
suggestion:
  mode: "manual"
`), 0o600))

	workDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workDir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "full", loaded.Suggestion.Mode)
}

func TestLoad_EnvOverrides(t *testing.T) {
	configureConfigEnv(t)

	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(oldWd) }()

	t.Setenv("GITDEX_SUGGESTION_MODE", "manual")

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "manual", loaded.Suggestion.Mode)
}

func TestLoad_ProjectOverridesGlobal(t *testing.T) {
	configureConfigEnv(t)

	globalDir, err := GlobalConfigDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(`
suggestion:
  mode: "full"
`), 0o600))

	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ProjectConfigName), []byte(`
suggestion:
  mode: "zen"
`), 0o600))

	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workDir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "zen", loaded.Suggestion.Mode)
}

func TestSaveGlobal_PersistsLanguage(t *testing.T) {
	configureConfigEnv(t)

	defaultCfg := DefaultConfig()
	defaultCfg.I18n.Language = "zh"
	defaultCfg.Suggestion.Language = "zh"

	require.NoError(t, SaveGlobal(defaultCfg))

	path, err := GlobalConfigPath()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "language: zh")
}

func TestSaveGlobal_PersistsAPIKeysAndReloads(t *testing.T) {
	configureConfigEnv(t)

	cfg := DefaultConfig()
	cfg.LLM.Provider = "openai"
	cfg.LLM.Model = "gpt-4.1-mini"
	cfg.LLM.Endpoint = "https://api.openai.com/v1"
	cfg.LLM.APIKey = "sk-test-primary"
	cfg.LLM.APIKeyEnv = ""
	cfg.LLM.Primary.Provider = "openai"
	cfg.LLM.Primary.Model = "gpt-4.1-mini"
	cfg.LLM.Primary.Endpoint = "https://api.openai.com/v1"
	cfg.LLM.Primary.APIKey = "sk-test-primary"
	cfg.LLM.Primary.APIKeyEnv = ""
	cfg.LLM.Secondary.Provider = "deepseek"
	cfg.LLM.Secondary.Model = "deepseek-chat"
	cfg.LLM.Secondary.Endpoint = "https://api.deepseek.com"
	cfg.LLM.Secondary.APIKey = "sk-test-secondary"
	cfg.LLM.Secondary.APIKeyEnv = ""
	cfg.LLM.Secondary.Enabled = true

	require.NoError(t, SaveGlobal(cfg))

	workDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workDir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "openai", loaded.LLM.Primary.Provider)
	assert.Equal(t, "gpt-4.1-mini", loaded.LLM.Primary.Model)
	assert.Equal(t, "sk-test-primary", loaded.LLM.Primary.APIKey)
	assert.Equal(t, "deepseek", loaded.LLM.Secondary.Provider)
	assert.Equal(t, "deepseek-chat", loaded.LLM.Secondary.Model)
	assert.Equal(t, "sk-test-secondary", loaded.LLM.Secondary.APIKey)
	assert.True(t, loaded.LLM.Secondary.Enabled)
}

func TestLoad_ProjectConfigDoesNotWipeSavedAPIKey(t *testing.T) {
	configureConfigEnv(t)

	cfg := DefaultConfig()
	cfg.LLM.Provider = "openai"
	cfg.LLM.Model = "gpt-4.1-mini"
	cfg.LLM.APIKey = "sk-global"
	cfg.LLM.APIKeyEnv = ""
	cfg.LLM.Primary.Provider = "openai"
	cfg.LLM.Primary.Model = "gpt-4.1-mini"
	cfg.LLM.Primary.APIKey = "sk-global"
	cfg.LLM.Primary.APIKeyEnv = ""
	require.NoError(t, SaveGlobal(cfg))

	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ProjectConfigName), []byte(`
suggestion:
  mode: "full"
`), 0o600))

	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workDir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "full", loaded.Suggestion.Mode)
	assert.Equal(t, "sk-global", loaded.LLM.Primary.APIKey)
}

func TestGet_BeforeLoad(t *testing.T) {
	cfg = nil
	assert.Nil(t, Get())
}

func TestValidate_InvalidMode(t *testing.T) {
	c := &Config{
		Suggestion: SuggestionConfig{Mode: "invalid", Language: "auto"},
		I18n:       I18nConfig{Language: "auto"},
		Theme:      ThemeConfig{Name: "catppuccin"},
	}
	err := Validate(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "suggestion.mode")
}

func TestValidate_InvalidTheme(t *testing.T) {
	c := &Config{
		Suggestion: SuggestionConfig{Mode: "zen", Language: "auto"},
		I18n:       I18nConfig{Language: "auto"},
		Theme:      ThemeConfig{Name: "invalid"},
	}
	err := Validate(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "theme.name")
}

func TestValidate_AllowsCloudProviders(t *testing.T) {
	c := DefaultConfig()
	c.LLM.Provider = "openai"
	c.LLM.Primary.Provider = "openai"
	c.LLM.Primary.Model = "gpt-4.1-mini"
	c.LLM.Primary.Endpoint = "https://api.openai.com/v1"
	c.LLM.Secondary.Provider = "deepseek"
	c.LLM.Secondary.Model = "deepseek-reasoner"
	c.LLM.Secondary.Endpoint = "https://api.deepseek.com"
	c.LLM.Secondary.Enabled = true

	err := Validate(c)
	assert.NoError(t, err)
}

func TestValidate_InvalidAutomation(t *testing.T) {
	c := DefaultConfig()
	c.Automation.MonitorInterval = -1
	err := Validate(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "automation.monitor_interval")
}

func TestValidate_InvalidAdapterBinary(t *testing.T) {
	c := DefaultConfig()
	c.Adapters.GitHub.GH.Enabled = true
	c.Adapters.GitHub.GH.Binary = ""
	err := Validate(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "adapters.github.gh.binary")
}

func TestValidate_InvalidGitAdapterBinary(t *testing.T) {
	c := DefaultConfig()
	c.Adapters.Git.Enabled = true
	c.Adapters.Git.Binary = ""
	err := Validate(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "adapters.git.binary")
}

func TestResolveRoleAPIKey_FromDefaultEnv(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "secret-key")

	role := ModelConfig{
		Provider: "deepseek",
	}

	assert.Equal(t, "secret-key", ResolveRoleAPIKey(role))
}

func TestResolveRoleAPIKey_FromAPIKeyEnvLiteralFallback(t *testing.T) {
	role := ModelConfig{
		Provider:  "deepseek",
		APIKeyEnv: "sk-7888b720f4214e448ee7f8808719eb55",
	}

	assert.Equal(t, "sk-7888b720f4214e448ee7f8808719eb55", ResolveRoleAPIKey(role))
}

func TestResolveRoleAPIKey_APIKeyEnvNameWithoutEnvValue(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "")
	role := ModelConfig{
		Provider:  "deepseek",
		APIKeyEnv: "DEEPSEEK_API_KEY",
	}

	assert.Equal(t, "", ResolveRoleAPIKey(role))
}

func TestResolveRoleAPIKey_StripsQuotedEnvName(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "quoted-secret")
	role := ModelConfig{
		Provider:  "deepseek",
		APIKeyEnv: "\"DEEPSEEK_API_KEY\"",
	}

	assert.Equal(t, "quoted-secret", ResolveRoleAPIKey(role))
}

func TestResolveRoleAPIKey_DeepSeekDskDashLiteralFallback(t *testing.T) {
	role := ModelConfig{
		Provider:  "deepseek",
		APIKeyEnv: "dsk-1234567890abcdefghijklmnopqrstuvwxyz",
	}

	assert.Equal(t, "dsk-1234567890abcdefghijklmnopqrstuvwxyz", ResolveRoleAPIKey(role))
}

func configureConfigEnv(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
}

func TestLoad_SetsVersionAndTrace(t *testing.T) {
	configureConfigEnv(t)
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(oldWd) }()

	loaded, err := Load()
	require.NoError(t, err)
	require.Equal(t, CurrentConfigVersion, loaded.Version)

	trace := LastLoadTrace()
	require.NotNil(t, trace)
	assert.True(t, trace.DefaultsApplied)
}

func TestSensitiveFieldWarnings(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LLM.Primary.APIKey = "sk-test"
	cfg.LLM.Secondary.APIKeyEnv = "sk-literal-1234567890abcdefghijklmnopqrstuvwxyz"
	warns := SensitiveFieldWarnings(cfg)
	require.NotEmpty(t, warns)
}
