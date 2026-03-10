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
	assert.Equal(t, "dark", loaded.Theme.Name)
	assert.Equal(t, "auto", loaded.I18n.Language)
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

func TestGet_BeforeLoad(t *testing.T) {
	cfg = nil
	assert.Nil(t, Get())
}

func TestValidate_InvalidMode(t *testing.T) {
	c := &Config{
		Suggestion: SuggestionConfig{Mode: "invalid", Language: "auto"},
		I18n:       I18nConfig{Language: "auto"},
		Theme:      ThemeConfig{Name: "dark"},
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

func configureConfigEnv(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
}
