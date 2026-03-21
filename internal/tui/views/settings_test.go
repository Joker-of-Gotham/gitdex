package views

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/platform/config"
	"github.com/your-org/gitdex/internal/tui/theme"
)

var ansiSeqRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiSeqRe.ReplaceAllString(s, "")
}

func makeTheme() *theme.Theme {
	t := theme.NewTheme(true)
	return &t
}

func TestSettingsView_ID(t *testing.T) {
	v := NewSettingsView(makeTheme())
	if v.ID() != ViewSettings {
		t.Errorf("ID() = %q, want %q", v.ID(), ViewSettings)
	}
	if v.Title() != "Settings" {
		t.Errorf("Title() = %q, want Settings", v.Title())
	}
}

func TestSettingsView_AutoSaveOnOptionChange(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 40)
	v.sectionIdx = 0
	v.fieldIdx = 0

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd == nil {
		t.Fatal("changing an option should return a ConfigSaveMsg command")
	}
	msg := cmd()
	saveMsg, ok := msg.(ConfigSaveMsg)
	if !ok {
		t.Fatalf("save command returned %T, want ConfigSaveMsg", msg)
	}
	if saveMsg.Target != SaveTargetGlobal {
		t.Errorf("Target = %q, want %q", saveMsg.Target, SaveTargetGlobal)
	}
	if len(saveMsg.DirtyKeys) == 0 {
		t.Error("DirtyKeys should include edited fields")
	}
}

func TestSettingsView_LoadFromConfig(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.LoadFromConfig(map[string]string{
		"llm.provider": "deepseek",
		"llm.model":    "deepseek-chat",
	})

	if got := v.fieldValue("llm.provider"); got != "deepseek" {
		t.Errorf("provider = %q, want deepseek", got)
	}
	if got := v.fieldValue("llm.model"); got != "deepseek-chat" {
		t.Errorf("model = %q, want deepseek-chat", got)
	}
}

func TestSettingsView_LoadRuntimeConfigTracksSources(t *testing.T) {
	v := NewSettingsView(makeTheme())
	cfg := config.Config{
		FileConfig: config.FileConfig{
			Output: "json",
			LLM: config.LLMConfig{
				Provider: "ollama",
				Model:    "qwen2.5-coder",
			},
			Identity: config.IdentityConfig{
				Mode:      "token",
				GitHubPAT: "ghp-secret",
				GitHubApp: config.GitHubAppConfig{Host: "ghe.example.test"},
			},
			Git: config.GitConfig{
				WorkspaceRoots: []string{"/workspace/a", "/workspace/b"},
			},
		},
		Paths: config.ConfigPaths{
			GlobalConfig:       "/tmp/global.yaml",
			RepoConfig:         "/repo/.gitdex/config.yaml",
			RepositoryDetected: true,
			ActiveFiles:        []string{"/tmp/global.yaml", "/repo/.gitdex/config.yaml"},
		},
		Sources: map[string]config.ValueSource{
			"output":                   config.SourceGlobal,
			"llm.provider":             config.SourceRepo,
			"llm.model":                config.SourceRepo,
			"identity.mode":            config.SourceGlobal,
			"identity.github_pat":      config.SourceEnv,
			"identity.github_app.host": config.SourceGlobal,
		},
	}

	v.LoadRuntimeConfig(cfg)

	if got := v.fieldValue("llm.provider"); got != "ollama" {
		t.Errorf("provider = %q, want ollama", got)
	}
	if got := v.fieldValue("git.workspace_roots"); got != "/workspace/a, /workspace/b" {
		t.Errorf("workspace roots = %q, want serialized list", got)
	}
	if v.saveTarget != SaveTargetRepo {
		t.Errorf("saveTarget = %q, want repo", v.saveTarget)
	}
	field := v.selectedField()
	if field == nil {
		t.Fatal("selectedField should not be nil")
	}
	if data := v.InspectorData(); !data.RepositoryDetected {
		t.Error("InspectorData should reflect repository context")
	}
}

func TestSettingsView_IdentityModeHidesIrrelevantFields(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.setFieldValue("identity.mode", "token", true)

	visible := v.visibleFieldIndices(v.sections[2])
	keys := make([]string, 0, len(visible))
	for _, idx := range visible {
		keys = append(keys, v.fields[idx].Key)
	}

	if !contains(keys, "identity.github_pat") {
		t.Error("token mode should show personal token")
	}
	if contains(keys, "identity.github_app.app_id") {
		t.Error("token mode should hide app-specific fields")
	}
}

func TestSettingsView_RuntimeFallbackAvoidsFalseBlockingTokenIssue(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_env_fallback")

	v := NewSettingsView(makeTheme())
	v.setFieldValue("identity.mode", "token", true)
	v.setFieldValue("identity.github_pat", "", true)
	v.setFieldValue("identity.github_app.host", "github.com22321321123", true)

	blocking, advisory := v.sectionIssueCounts("github")
	if blocking != 0 {
		t.Fatalf("github blocking issues = %d, want 0 when env fallback is available", blocking)
	}
	if advisory == 0 {
		t.Fatal("github advisory issues should explain runtime fallback")
	}
	if got := v.effectiveGitHubAuthLabel(); got != "env.github_token" {
		t.Fatalf("effective auth = %q, want env.github_token", got)
	}
	if got := v.effectiveGitHubHostLabel(); got != "github.com" {
		t.Fatalf("effective host = %q, want github.com", got)
	}
}

func TestSettingsView_InspectorDataIncludesEffectiveRuntime(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_env_fallback")

	v := NewSettingsView(makeTheme())
	v.setFieldValue("identity.mode", "token", true)
	v.setFieldValue("identity.github_pat", "", true)
	v.setFieldValue("identity.github_app.host", "github.com22321321123", true)

	data := v.InspectorData()
	if data.EffectiveAuth != "env.github_token" {
		t.Fatalf("EffectiveAuth = %q, want env.github_token", data.EffectiveAuth)
	}
	if data.EffectiveHost != "github.com" {
		t.Fatalf("EffectiveHost = %q, want github.com", data.EffectiveHost)
	}
}

func TestSettingsView_ProviderChangeUpdatesDefaults(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.setFieldValue("llm.provider", "deepseek", true)

	if got := v.fieldValue("llm.model"); got != "deepseek-chat" {
		t.Errorf("model = %q, want deepseek-chat", got)
	}
	if got := v.fieldValue("llm.endpoint"); !strings.Contains(got, "deepseek") {
		t.Errorf("endpoint = %q, want deepseek endpoint", got)
	}
}

func TestSettingsView_GetFieldValues(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 40)
	values := v.GetFieldValues()
	if _, ok := values["llm.provider"]; !ok {
		t.Error("GetFieldValues should include llm.provider")
	}
}

func TestSettingsView_Render(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 40)

	output := v.Render()
	plain := stripANSI(output)
	if !strings.Contains(plain, "Settings Control Plane") {
		t.Error("Render should contain title")
	}
	if !strings.Contains(plain, "Configuration Map") {
		t.Error("Render should contain section rail")
	}
}

func TestSettingsView_RenderEmpty(t *testing.T) {
	v := NewSettingsView(makeTheme())
	output := v.Render()
	if output != "" {
		t.Error("Render with zero size should return empty string")
	}
}

func TestSettingsView_EditFlow(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 40)
	v.sectionIdx = 4
	v.fieldIdx = 0

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should begin edit for the selected text field")
	}
	if v.focus != settingsFocusEditor {
		t.Fatal("focus should be editor after Enter")
	}

	v.editor.SetValue("production")
	_, cmd = v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := v.fieldValue("profile"); got != "production" {
		t.Errorf("profile = %q, want production", got)
	}
	if cmd == nil {
		t.Fatal("committing an edit should trigger auto-save")
	}
}

func TestSettingsView_SelectFieldCyclesOptions(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 40)
	v.sectionIdx = 0
	v.fieldIdx = 0

	v.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if got := v.fieldValue("llm.provider"); got != "deepseek" {
		t.Errorf("provider = %q, want deepseek after cycling", got)
	}
}

func TestSettingsView_LeftRightMovesSections(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 40)

	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.currentSection().ID != "git" {
		t.Fatalf("section after right = %q, want git", v.currentSection().ID)
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v.currentSection().ID != "llm" {
		t.Fatalf("section after left = %q, want llm", v.currentSection().ID)
	}
}

func TestSettingsView_EnterStartsEditForOptionField(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 40)
	v.sectionIdx = 0
	v.fieldIdx = 0

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should focus the editor for direct input")
	}
	if v.focus != settingsFocusEditor {
		t.Fatal("focus should be editor after Enter")
	}
	if got := v.editor.Value(); got != "openai" {
		t.Fatalf("editor value = %q, want current field value", got)
	}
}

func TestSettingsView_DirectTypingStartsEdit(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 40)
	v.sectionIdx = 4
	v.fieldIdx = 0

	_, cmd := v.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if cmd == nil {
		t.Fatal("typing should start editing and return a command")
	}
	if v.focus != settingsFocusEditor {
		t.Fatal("focus should be editor after direct typing")
	}
	if got := v.editor.Value(); got != "localx" {
		t.Fatalf("editor value = %q, want localx", got)
	}
}

func TestFieldsToFileConfig(t *testing.T) {
	fields := []ConfigField{
		{Key: "profile", Value: "team"},
		{Key: "llm.provider", Value: "deepseek"},
		{Key: "llm.model", Value: "deepseek-chat"},
		{Key: "llm.api_key", Value: "sk-xxx"},
		{Key: "llm.endpoint", Value: "https://api.deepseek.com/v1"},
		{Key: "identity.mode", Value: "token"},
		{Key: "identity.github_pat", Value: "ghp-xxx"},
		{Key: "identity.github_app.host", Value: "github.com"},
		{Key: "storage.type", Value: "sqlite"},
		{Key: "storage.dsn", Value: "data.db"},
		{Key: "output", Value: "json"},
		{Key: "log_level", Value: "debug"},
		{Key: "daemon.health_address", Value: "0.0.0.0:8888"},
		{Key: "git.user_name", Value: "Jane"},
		{Key: "git.user_email", Value: "jane@example.com"},
		{Key: "git.workspace_roots", Value: "D:\\Code, E:\\Repos"},
	}
	fc := FieldsToFileConfig(fields)
	if fc.Profile != "team" {
		t.Errorf("Profile = %q, want team", fc.Profile)
	}
	if fc.LLM.Provider != "deepseek" {
		t.Errorf("LLM.Provider = %q, want deepseek", fc.LLM.Provider)
	}
	if fc.Identity.GitHubPAT != "ghp-xxx" {
		t.Errorf("Identity.GitHubPAT = %q, want ghp-xxx", fc.Identity.GitHubPAT)
	}
	if fc.Git.UserEmail != "jane@example.com" {
		t.Errorf("Git.UserEmail = %q, want jane@example.com", fc.Git.UserEmail)
	}
	if len(fc.Git.WorkspaceRoots) != 2 {
		t.Fatalf("Git.WorkspaceRoots len = %d, want 2", len(fc.Git.WorkspaceRoots))
	}
	if fc.Git.WorkspaceRoots[0] != "D:\\Code" || fc.Git.WorkspaceRoots[1] != "E:\\Repos" {
		t.Errorf("Git.WorkspaceRoots = %#v, want parsed roots", fc.Git.WorkspaceRoots)
	}
}

func TestFormatConfigSaveMsg(t *testing.T) {
	fields := []ConfigField{
		{Key: "llm.provider", Label: "Provider", Value: "openai"},
		{Key: "llm.api_key", Label: "API Key", Value: "sk-test", Secret: true},
	}
	output := FormatConfigSaveMsg(fields)
	if !strings.Contains(output, "openai") {
		t.Error("should contain provider value")
	}
	if !strings.Contains(output, "***") {
		t.Error("should redact secret values")
	}
	if strings.Contains(output, "sk-test") {
		t.Error("should not expose secret in output")
	}
}

func TestSettingsView_RenderSmallHeightKeepsSelectedFieldVisible(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(90, 18)
	v.sectionIdx = 2
	v.fieldIdx = 4

	plain := stripANSI(v.Render())
	if !strings.Contains(plain, "Private key path") {
		t.Fatalf("small-height render should keep the selected field visible, got:\n%s", plain)
	}
}

func TestSettingsView_SelectedSecretShowsActualValue(t *testing.T) {
	v := NewSettingsView(makeTheme())
	v.SetSize(120, 30)
	v.sectionIdx = 0
	v.fieldIdx = 2
	v.setFieldValue("llm.api_key", "sk-secret-visible", true)

	plain := stripANSI(v.Render())
	if !strings.Contains(plain, "sk-secret-visible") {
		t.Fatalf("selected secret should be visible for verification, got:\n%s", plain)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
