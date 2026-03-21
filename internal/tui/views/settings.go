package views

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/llm/adapter"
	"github.com/your-org/gitdex/internal/platform/config"
	"github.com/your-org/gitdex/internal/platform/identity"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type ConfigSaveTarget string

const (
	SaveTargetGlobal ConfigSaveTarget = "global"
	SaveTargetRepo   ConfigSaveTarget = "repo"
)

type ConfigField struct {
	Key         string
	Label       string
	Value       string
	Placeholder string
	Description string
	Help        string
	Secret      bool
	Required    bool
	Options     []string
	Section     string
	Source      config.ValueSource
}

type ConfigSaveMsg struct {
	Fields    []ConfigField
	DirtyKeys []string
	Target    ConfigSaveTarget
}

type SettingsInspectorData struct {
	CurrentSection     string
	Profile            string
	IdentityMode       string
	EffectiveAuth      string
	EffectiveHost      string
	SaveTarget         string
	RecommendedAction  string
	DirtyCount         int
	RepositoryDetected bool
	GlobalConfig       string
	RepoConfig         string
	ActiveFiles        []string
	OverrideFields     []string
	Warnings           []string
}

type SettingsSection struct {
	ID        string
	Title     string
	Summary   string
	Purpose   string
	FieldKeys []string
}

type settingsFocus int

const (
	settingsFocusFields settingsFocus = iota
	settingsFocusEditor
)

type settingsIssue struct {
	SectionID string
	Message   string
	Blocking  bool
}

type SettingsView struct {
	theme      *theme.Theme
	width      int
	height     int
	sections   []SettingsSection
	fields     []ConfigField
	fieldIndex map[string]int
	sectionIdx int
	fieldIdx   int
	focus      settingsFocus
	editor     textinput.Model
	detailVP   viewport.Model
	saveTarget ConfigSaveTarget
	snapshot   config.Snapshot
	dirty      map[string]bool
	statusLine string

	effectiveGitHubAuth string
	effectiveGitHubHost string
	runtimeGitHubReady  bool
}

func NewSettingsView(t *theme.Theme) *SettingsView {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 512

	v := &SettingsView{
		theme:      t,
		sections:   defaultSettingsSections(),
		fields:     defaultSettingsFields(),
		fieldIndex: make(map[string]int),
		editor:     ti,
		detailVP:   viewport.New(viewport.WithWidth(40), viewport.WithHeight(12)),
		saveTarget: SaveTargetGlobal,
		dirty:      make(map[string]bool),
	}
	v.rebuildFieldIndex()
	v.ensureCursorInBounds()
	v.refreshGitHubRuntimeStatus()
	return v
}

func defaultSettingsSections() []SettingsSection {
	return []SettingsSection{
		{
			ID:        "llm",
			Title:     "Model Runtime",
			Summary:   "Provider, model, credentials, and endpoint",
			Purpose:   "Chat, planning, and operator assistance all depend on one coherent runtime. Provider changes must carry model and endpoint defaults with them.",
			FieldKeys: []string{"llm.provider", "llm.model", "llm.api_key", "llm.endpoint"},
		},
		{
			ID:        "git",
			Title:     "Git Identity",
			Summary:   "Commit identity, workspace roots, and SSH execution context",
			Purpose:   "These fields control how Gitdex authors repository writes and where it searches for local clones. Explicit workspace roots are preferred over blind disk scanning.",
			FieldKeys: []string{"git.user_name", "git.user_email", "git.workspace_roots", "git.ssh_key_path"},
		},
		{
			ID:      "github",
			Title:   "GitHub Access",
			Summary: "Authentication mode, host, and live repository access",
			Purpose: "This section decides whether dashboard sync, repository discovery, collaboration objects, and governed remote actions can run at all.",
			FieldKeys: []string{
				"identity.mode",
				"identity.github_app.host",
				"identity.github_pat",
				"identity.github_app.app_id",
				"identity.github_app.installation_id",
				"identity.github_app.private_key_path",
			},
		},
		{
			ID:        "storage",
			Title:     "Storage & State",
			Summary:   "Persistence backend and audit retention",
			Purpose:   "Storage decides whether Gitdex runs statelessly or keeps durable execution state, evidence, and audit history.",
			FieldKeys: []string{"storage.type", "storage.dsn"},
		},
		{
			ID:        "control",
			Title:     "Control Plane",
			Summary:   "Profile, output, logging, and daemon health surface",
			Purpose:   "This section defines how Gitdex presents itself locally, how diagnostics are emitted, and which operator profile is active.",
			FieldKeys: []string{"profile", "output", "log_level", "daemon.health_address"},
		},
	}
}

func defaultSettingsFields() []ConfigField {
	return []ConfigField{
		{
			Key:         "profile",
			Label:       "Profile",
			Value:       "local",
			Placeholder: "local",
			Description: "Operator profile used to shape Gitdex runtime defaults and context.",
			Help:        "Use a short identifier such as local, team, or production.",
			Section:     "control",
			Source:      config.SourceDefault,
		},
		{
			Key:         "output",
			Label:       "Output format",
			Value:       "text",
			Placeholder: "text",
			Description: "CLI serialization mode for non-TUI commands.",
			Help:        "JSON is better for automation; text is better for operators.",
			Options:     []string{"text", "json"},
			Required:    true,
			Section:     "control",
			Source:      config.SourceDefault,
		},
		{
			Key:         "log_level",
			Label:       "Log level",
			Value:       "info",
			Placeholder: "info",
			Description: "Verbosity for operator diagnostics and background execution.",
			Help:        "Use debug only when investigating runtime behavior.",
			Options:     []string{"debug", "info", "warn", "error"},
			Required:    true,
			Section:     "control",
			Source:      config.SourceDefault,
		},
		{
			Key:         "daemon.health_address",
			Label:       "Health address",
			Value:       "127.0.0.1:7777",
			Placeholder: "127.0.0.1:7777",
			Description: "Bind address for the daemon health surface.",
			Help:        "Expose externally only when you have a concrete monitoring need.",
			Required:    true,
			Section:     "control",
			Source:      config.SourceDefault,
		},
		{
			Key:         "llm.provider",
			Label:       "LLM provider",
			Value:       "openai",
			Placeholder: "openai",
			Description: "Primary LLM backend used by chat and planning surfaces.",
			Help:        "Changing provider can update model and endpoint defaults when those fields are still on provider defaults.",
			Options:     adapter.SupportedProviders,
			Required:    true,
			Section:     "llm",
			Source:      config.SourceDefault,
		},
		{
			Key:         "llm.model",
			Label:       "Model name",
			Value:       adapter.DefaultModelForProvider("openai"),
			Placeholder: adapter.DefaultModelForProvider("openai"),
			Description: "Concrete model name sent to the selected provider.",
			Help:        "Use the provider default unless you need a specific capability or latency profile.",
			Required:    true,
			Section:     "llm",
			Source:      config.SourceDefault,
		},
		{
			Key:         "llm.api_key",
			Label:       "LLM API key",
			Value:       "",
			Placeholder: "sk-...",
			Description: "Credential for hosted providers. Not required for local Ollama by default.",
			Help:        "Secrets are masked in the UI and only written when you edit them.",
			Secret:      true,
			Section:     "llm",
			Source:      config.SourceDefault,
		},
		{
			Key:         "llm.endpoint",
			Label:       "Base URL",
			Value:       adapter.DefaultEndpointForProvider("openai"),
			Placeholder: adapter.DefaultEndpointForProvider("openai"),
			Description: "Base URL for API traffic.",
			Help:        "Useful for self-hosted gateways, proxy routing, and enterprise vendor endpoints.",
			Required:    true,
			Section:     "llm",
			Source:      config.SourceDefault,
		},
		{
			Key:         "git.user_name",
			Label:       "Git author name",
			Value:       "",
			Placeholder: "Jane Operator",
			Description: "Commit author name used for repository write operations.",
			Help:        "Recommended before enabling automated or assisted write flows.",
			Section:     "git",
			Source:      config.SourceDefault,
		},
		{
			Key:         "git.user_email",
			Label:       "Git author email",
			Value:       "",
			Placeholder: "jane@example.com",
			Description: "Commit author email used for repository write operations.",
			Help:        "Keep this aligned with the Git identity expected by your repositories.",
			Section:     "git",
			Source:      config.SourceDefault,
		},
		{
			Key:         "git.ssh_key_path",
			Label:       "SSH key path",
			Value:       "",
			Placeholder: "~/.ssh/id_ed25519",
			Description: "Private key path used when SSH transport is needed.",
			Help:        "Useful for local clone maintenance and write flows outside HTTPS.",
			Section:     "git",
			Source:      config.SourceDefault,
		},
		{
			Key:         "git.workspace_roots",
			Label:       "Workspace roots",
			Value:       "",
			Placeholder: "D:\\Code\\Repos, E:\\ClientWork",
			Description: "Preferred root directories used to discover existing local clones for remote repositories.",
			Help:        "Enter one or more directories separated by commas, semicolons, or new lines. Gitdex will search these before falling back to broad scanning.",
			Section:     "git",
			Source:      config.SourceDefault,
		},
		{
			Key:         "identity.mode",
			Label:       "Auth mode",
			Value:       "github-app",
			Placeholder: "github-app",
			Description: "Select whether Gitdex authenticates with a GitHub App, personal token, or not at all.",
			Help:        "GitHub App is preferred for governed automation; token mode is simpler but broader.",
			Options:     []string{"github-app", "token", "none"},
			Required:    true,
			Section:     "github",
			Source:      config.SourceDefault,
		},
		{
			Key:         "identity.github_app.host",
			Label:       "GitHub host",
			Value:       "github.com",
			Placeholder: "github.com",
			Description: "Host used for GitHub.com or GitHub Enterprise access.",
			Help:        "For enterprise hosts, use the web hostname only; the API base is derived automatically.",
			Section:     "github",
			Source:      config.SourceDefault,
		},
		{
			Key:         "identity.github_pat",
			Label:       "GitHub classic token / PAT",
			Value:       "",
			Placeholder: "ghp_...",
			Description: "Token used when Auth mode is set to token.",
			Help:        "Use the narrowest scope you can get away with.",
			Secret:      true,
			Section:     "github",
			Source:      config.SourceDefault,
		},
		{
			Key:         "identity.github_app.app_id",
			Label:       "App ID",
			Value:       "",
			Placeholder: "12345",
			Description: "GitHub App identifier for app-based authentication.",
			Help:        "Required only in GitHub App mode.",
			Section:     "github",
			Source:      config.SourceDefault,
		},
		{
			Key:         "identity.github_app.installation_id",
			Label:       "Installation ID",
			Value:       "",
			Placeholder: "67890",
			Description: "Installation identifier tying the app to accessible repositories.",
			Help:        "Required only in GitHub App mode.",
			Section:     "github",
			Source:      config.SourceDefault,
		},
		{
			Key:         "identity.github_app.private_key_path",
			Label:       "Private key path",
			Value:       "",
			Placeholder: "/path/to/app.pem",
			Description: "PEM file used to mint GitHub App installation tokens.",
			Help:        "Required only in GitHub App mode.",
			Section:     "github",
			Source:      config.SourceDefault,
		},
		{
			Key:         "storage.type",
			Label:       "Backend",
			Value:       "memory",
			Placeholder: "memory",
			Description: "Persistence mode for execution state and evidence.",
			Help:        "Memory is ephemeral. Use sqlite, bbolt, or postgres when state durability matters.",
			Options:     []string{"memory", "bbolt", "sqlite", "postgres"},
			Required:    true,
			Section:     "storage",
			Source:      config.SourceDefault,
		},
		{
			Key:         "storage.dsn",
			Label:       "Connection string",
			Value:       "",
			Placeholder: "data/gitdex.db or postgres://...",
			Description: "Path or DSN for the chosen backend when persistence is enabled.",
			Help:        "Required for non-memory backends.",
			Secret:      true,
			Section:     "storage",
			Source:      config.SourceDefault,
		},
	}
}

func (v *SettingsView) ID() ID        { return ViewSettings }
func (v *SettingsView) Title() string { return "Settings" }

func (v *SettingsView) SetSize(w, h int) {
	v.width = w
	v.height = h
	if w > 0 && h > 0 {
		v.detailVP.SetWidth(maxInt(20, w-6))
		v.detailVP.SetHeight(maxInt(6, h-10))
	}
}

func (v *SettingsView) Init() tea.Cmd { return nil }

func (v *SettingsView) LoadRuntimeConfig(cfg config.Config) {
	values := map[string]string{
		"profile":                              cfg.Profile,
		"output":                               cfg.Output,
		"log_level":                            cfg.LogLevel,
		"daemon.health_address":                cfg.Daemon.HealthAddress,
		"llm.provider":                         cfg.LLM.Provider,
		"llm.model":                            cfg.LLM.Model,
		"llm.api_key":                          cfg.LLM.APIKey,
		"llm.endpoint":                         cfg.LLM.Endpoint,
		"git.user_name":                        cfg.Git.UserName,
		"git.user_email":                       cfg.Git.UserEmail,
		"git.workspace_roots":                  serializeSettingsPathList(cfg.Git.WorkspaceRoots),
		"git.ssh_key_path":                     cfg.Git.SSHKeyPath,
		"identity.mode":                        cfg.Identity.Mode,
		"identity.github_app.host":             cfg.Identity.GitHubApp.Host,
		"identity.github_pat":                  cfg.Identity.GitHubPAT,
		"identity.github_app.app_id":           cfg.Identity.GitHubApp.AppID,
		"identity.github_app.installation_id":  cfg.Identity.GitHubApp.InstallationID,
		"identity.github_app.private_key_path": cfg.Identity.GitHubApp.PrivateKeyPath,
		"storage.type":                         cfg.Storage.Type,
		"storage.dsn":                          cfg.Storage.DSN,
	}
	v.snapshot = cfg.Snapshot()
	v.applyConfigValues(values, cfg.Sources)
	v.dirty = make(map[string]bool)
	v.statusLine = ""
	v.syncSaveTarget()
	v.refreshGitHubRuntimeStatus()
}

func (v *SettingsView) LoadFromConfig(values map[string]string) {
	v.applyConfigValues(values, nil)
	v.dirty = make(map[string]bool)
	v.statusLine = ""
	v.ensureCursorInBounds()
	v.refreshGitHubRuntimeStatus()
}

func (v *SettingsView) Update(msg tea.Msg) (View, tea.Cmd) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		if v.focus == settingsFocusEditor {
			var cmd tea.Cmd
			v.editor, cmd = v.editor.Update(msg)
			return v, cmd
		}
		return v, nil
	}

	if v.focus == settingsFocusEditor {
		switch km.String() {
		case "esc":
			v.cancelEdit()
			return v, nil
		case "enter":
			changed := v.commitEdit()
			return v, v.autoSaveCmd(changed)
		case "tab":
			changed := v.commitEdit()
			v.moveField(1)
			return v, v.autoSaveCmd(changed)
		case "shift+tab":
			changed := v.commitEdit()
			v.moveField(-1)
			return v, v.autoSaveCmd(changed)
		}
		var cmd tea.Cmd
		v.editor, cmd = v.editor.Update(msg)
		return v, cmd
	}

	switch km.String() {
	case "up":
		v.moveField(-1)
	case "down":
		v.moveField(1)
	case "left":
		v.moveSection(-1)
	case "right":
		v.moveSection(1)
	case "tab":
		v.moveField(1)
	case "shift+tab":
		v.moveField(-1)
	case "home":
		v.fieldIdx = 0
		v.ensureCursorInBounds()
	case "end":
		visible := v.visibleFieldIndices(v.currentSection())
		if len(visible) > 0 {
			v.fieldIdx = len(visible) - 1
		}
		v.ensureCursorInBounds()
	case "pgup":
		v.moveSection(-1)
	case "pgdown":
		v.moveSection(1)
	case "ctrl+g":
		v.saveTarget = SaveTargetGlobal
		v.statusLine = "Changes will now apply to the global config file."
	case "ctrl+r":
		if v.snapshot.Paths.RepositoryDetected && strings.TrimSpace(v.snapshot.Paths.RepoConfig) != "" {
			v.saveTarget = SaveTargetRepo
			v.statusLine = "Changes will now apply to the repository config file."
		} else {
			v.statusLine = "Repository config is unavailable outside a detected repository."
		}
	case " ", "space":
		if v.selectedFieldHasOptions() {
			changed := v.cycleSelectedOption(1)
			return v, v.autoSaveCmd(changed)
		}
	case "enter":
		return v, v.activateSelectedField()
	default:
		if v.shouldBeginDirectEdit(km) {
			editCmd := v.activateSelectedField()
			var inputCmd tea.Cmd
			v.editor, inputCmd = v.editor.Update(km)
			return v, tea.Batch(editCmd, inputCmd)
		}
	}

	return v, nil
}

func (v *SettingsView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(v.theme.Primary()).
		Render(theme.Icons.Settings + " Settings Control Plane")

	hint := "Left/Right section | Up/Down or Tab field | Type or Enter edit | Space cycle option | Home/End jump | Ctrl+G global | Ctrl+R repo"
	if v.focus == settingsFocusEditor {
		hint = "Editing value | Enter apply | Tab next field | Shift+Tab previous | Esc cancel"
	}
	subtitle := lipgloss.NewStyle().
		Foreground(v.theme.DimText()).
		Italic(true).
		Render(hint)

	parts := []string{title, subtitle, v.renderStatusStrip()}
	header := strings.Join(parts, "\n\n")
	footer := v.renderFooter()
	bodyHeight := maxInt(8, v.height-lipgloss.Height(header)-2)
	if footer != "" {
		bodyHeight = maxInt(8, bodyHeight-lipgloss.Height(footer)-2)
	}

	bodyWidth := v.width
	railWidth := 0
	if v.width >= 88 {
		railWidth = clampInt(v.width*24/100, 24, 30)
	}

	var body string
	if railWidth > 0 && v.width >= 148 && v.focus != settingsFocusEditor {
		mainWidth := maxInt(32, v.width-railWidth-1)
		body = lipgloss.JoinHorizontal(
			lipgloss.Top,
			v.renderSectionDetail(mainWidth, bodyHeight),
			" ",
			v.renderSectionRail(railWidth),
		)
	} else {
		if bodyHeight < 24 || v.focus == settingsFocusEditor {
			body = v.renderSectionDetail(bodyWidth, bodyHeight)
		} else {
			railHeight := minInt(18, maxInt(10, len(v.sections)*4+4))
			if bodyHeight <= railHeight+8 {
				body = v.renderSectionDetail(bodyWidth, bodyHeight)
			} else {
				detailHeight := maxInt(8, bodyHeight-railHeight-1)
				body = strings.Join([]string{
					v.renderSectionDetail(bodyWidth, detailHeight),
					v.renderSectionRail(bodyWidth),
				}, "\n\n")
			}
		}
	}
	parts = append(parts, body)

	if footer != "" {
		parts = append(parts, footer)
	}

	return strings.Join(parts, "\n\n")
}

func (v *SettingsView) GetFields() []ConfigField {
	out := make([]ConfigField, len(v.fields))
	copy(out, v.fields)
	return out
}

func (v *SettingsView) GetFieldValues() map[string]string {
	values := make(map[string]string, len(v.fields))
	for _, field := range v.fields {
		values[field.Key] = field.Value
	}
	return values
}

func (v *SettingsView) InspectorData() SettingsInspectorData {
	overrideFields := make([]string, 0)
	for _, field := range v.fields {
		if field.Source == config.SourceEnv || field.Source == config.SourceFlag {
			overrideFields = append(overrideFields, field.Label)
		}
	}
	warnings := make([]string, 0)
	for _, issue := range v.collectIssues() {
		warnings = append(warnings, issue.Message)
		if len(warnings) == 6 {
			break
		}
	}
	currentSection := ""
	if len(v.sections) > 0 {
		currentSection = v.sections[v.sectionIdx].Title
	}
	return SettingsInspectorData{
		CurrentSection:     currentSection,
		Profile:            v.fieldValue("profile"),
		IdentityMode:       v.identityMode(),
		EffectiveAuth:      v.effectiveGitHubAuthLabel(),
		EffectiveHost:      v.effectiveGitHubHostLabel(),
		SaveTarget:         v.friendlySaveTarget(v.saveTarget),
		RecommendedAction:  v.recommendedAction(),
		DirtyCount:         len(v.dirtyKeys()),
		RepositoryDetected: v.snapshot.Paths.RepositoryDetected,
		GlobalConfig:       v.snapshot.Paths.GlobalConfig,
		RepoConfig:         v.snapshot.Paths.RepoConfig,
		ActiveFiles:        append([]string(nil), v.snapshot.Paths.ActiveFiles...),
		OverrideFields:     overrideFields,
		Warnings:           warnings,
	}
}

func FieldsToFileConfig(fields []ConfigField) config.FileConfig {
	return ApplyFieldsToFileConfig(config.FileConfig{}, fields)
}

func ApplyFieldsToFileConfig(base config.FileConfig, fields []ConfigField) config.FileConfig {
	fc := base
	for _, f := range fields {
		switch f.Key {
		case "profile":
			fc.Profile = f.Value
		case "output":
			fc.Output = f.Value
		case "log_level":
			fc.LogLevel = f.Value
		case "daemon.health_address":
			fc.Daemon.HealthAddress = f.Value
		case "llm.provider":
			fc.LLM.Provider = f.Value
		case "llm.model":
			fc.LLM.Model = f.Value
		case "llm.api_key":
			fc.LLM.APIKey = f.Value
		case "llm.endpoint":
			fc.LLM.Endpoint = f.Value
		case "git.user_name":
			fc.Git.UserName = f.Value
		case "git.user_email":
			fc.Git.UserEmail = f.Value
		case "git.ssh_key_path":
			fc.Git.SSHKeyPath = f.Value
		case "git.workspace_roots":
			fc.Git.WorkspaceRoots = parseSettingsPathList(f.Value)
		case "identity.mode":
			fc.Identity.Mode = f.Value
		case "identity.github_pat":
			fc.Identity.GitHubPAT = f.Value
		case "identity.github_app.host":
			fc.Identity.GitHubApp.Host = f.Value
		case "identity.github_app.app_id":
			fc.Identity.GitHubApp.AppID = f.Value
		case "identity.github_app.installation_id":
			fc.Identity.GitHubApp.InstallationID = f.Value
		case "identity.github_app.private_key_path":
			fc.Identity.GitHubApp.PrivateKeyPath = f.Value
		case "storage.type":
			fc.Storage.Type = f.Value
		case "storage.dsn":
			fc.Storage.DSN = f.Value
		}
	}
	return fc
}

func FilterConfigFields(fields []ConfigField, keys []string) []ConfigField {
	if len(keys) == 0 {
		out := make([]ConfigField, len(fields))
		copy(out, fields)
		return out
	}
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
	}
	out := make([]ConfigField, 0, len(keys))
	for _, field := range fields {
		if allowed[field.Key] {
			out = append(out, field)
		}
	}
	return out
}

func FormatConfigSaveMsg(fields []ConfigField) string {
	var b strings.Builder
	for _, f := range fields {
		display := f.Value
		if f.Secret && display != "" {
			display = maskValue(display)
		}
		b.WriteString(fmt.Sprintf("  %s = %s\n", f.Key, display))
	}
	return b.String()
}

func (v *SettingsView) applyConfigValues(values map[string]string, sources map[string]config.ValueSource) {
	for i := range v.fields {
		if value, ok := values[v.fields[i].Key]; ok {
			v.fields[i].Value = value
		}
		if sources != nil {
			if source, ok := sources[v.fields[i].Key]; ok {
				v.fields[i].Source = source
			} else {
				v.fields[i].Source = config.SourceDefault
			}
		}
	}
	v.ensureCursorInBounds()
}

func (v *SettingsView) rebuildFieldIndex() {
	for i, field := range v.fields {
		v.fieldIndex[field.Key] = i
	}
}

func (v *SettingsView) moveSection(delta int) {
	if len(v.sections) == 0 {
		return
	}
	v.sectionIdx = clampInt(v.sectionIdx+delta, 0, len(v.sections)-1)
	v.fieldIdx = 0
	v.ensureCursorInBounds()
}

func (v *SettingsView) moveField(delta int) {
	visible := v.visibleFieldIndices(v.currentSection())
	if len(visible) == 0 {
		v.fieldIdx = 0
		return
	}
	v.fieldIdx = clampInt(v.fieldIdx+delta, 0, len(visible)-1)
}

func (v *SettingsView) selectedFieldHasOptions() bool {
	field := v.selectedField()
	return field != nil && len(field.Options) > 0
}

func (v *SettingsView) activateSelectedField() tea.Cmd {
	field := v.selectedField()
	if field == nil {
		return nil
	}

	v.focus = settingsFocusEditor
	v.editor.SetValue(field.Value)
	v.editor.Placeholder = field.Placeholder
	v.editor.SetWidth(maxInt(12, v.detailValueWidth()))
	v.editor.EchoMode = textinput.EchoNormal
	return v.editor.Focus()
}

func (v *SettingsView) cancelEdit() {
	v.focus = settingsFocusFields
	v.editor.Blur()
	v.statusLine = "Edit cancelled."
}

func (v *SettingsView) commitEdit() bool {
	field := v.selectedField()
	if field == nil {
		return false
	}
	changed := v.setFieldValue(field.Key, v.editor.Value(), true)
	v.focus = settingsFocusFields
	v.editor.Blur()
	if changed {
		v.statusLine = field.Label + " updated. Applying change."
		return true
	}
	v.statusLine = field.Label + " unchanged."
	return false
}

func (v *SettingsView) cycleSelectedOption(delta int) bool {
	field := v.selectedField()
	if field == nil || len(field.Options) == 0 {
		return false
	}
	current := 0
	for i, option := range field.Options {
		if option == field.Value {
			current = i
			break
		}
	}
	next := current + delta
	if next < 0 {
		next = len(field.Options) - 1
	}
	if next >= len(field.Options) {
		next = 0
	}
	return v.setFieldValue(field.Key, field.Options[next], true)
}

func (v *SettingsView) setFieldValue(key, value string, markDirty bool) bool {
	index, ok := v.fieldIndex[key]
	if !ok {
		return false
	}
	previous := v.fields[index].Value
	if previous == value {
		return false
	}
	v.fields[index].Value = value
	if markDirty {
		v.dirty[key] = true
	}

	switch key {
	case "llm.provider":
		v.applyProviderDefaults(previous, value, markDirty)
	case "identity.mode":
		v.statusLine = "Authentication flow updated."
	case "storage.type":
		if value == "memory" && markDirty {
			v.statusLine = "Memory mode selected. Connection string is no longer required."
		}
	}
	if strings.HasPrefix(key, "identity.") {
		v.refreshGitHubRuntimeStatus()
	}

	v.ensureCursorInBounds()
	return true
}

func (v *SettingsView) applyProviderDefaults(previousProvider, nextProvider string, markDirty bool) {
	oldModel := adapter.DefaultModelForProvider(previousProvider)
	newModel := adapter.DefaultModelForProvider(nextProvider)
	if current := v.fieldValue("llm.model"); strings.TrimSpace(current) == "" || current == oldModel || !v.dirty["llm.model"] {
		_ = v.setFieldValue("llm.model", newModel, markDirty)
	}

	oldEndpoint := adapter.DefaultEndpointForProvider(previousProvider)
	newEndpoint := adapter.DefaultEndpointForProvider(nextProvider)
	if current := v.fieldValue("llm.endpoint"); strings.TrimSpace(current) == "" || current == oldEndpoint || !v.dirty["llm.endpoint"] {
		_ = v.setFieldValue("llm.endpoint", newEndpoint, markDirty)
	}

	v.statusLine = "Provider defaults re-evaluated for model and endpoint."
}

func (v *SettingsView) ensureCursorInBounds() {
	if len(v.sections) == 0 {
		v.sectionIdx = 0
		v.fieldIdx = 0
		return
	}
	v.sectionIdx = clampInt(v.sectionIdx, 0, len(v.sections)-1)
	visible := v.visibleFieldIndices(v.currentSection())
	if len(visible) == 0 {
		v.fieldIdx = 0
		return
	}
	v.fieldIdx = clampInt(v.fieldIdx, 0, len(visible)-1)
}

func (v *SettingsView) currentSection() SettingsSection {
	if len(v.sections) == 0 {
		return SettingsSection{}
	}
	return v.sections[v.sectionIdx]
}

func (v *SettingsView) visibleFieldIndices(section SettingsSection) []int {
	out := make([]int, 0, len(section.FieldKeys))
	for _, key := range section.FieldKeys {
		index, ok := v.fieldIndex[key]
		if !ok {
			continue
		}
		if v.isFieldVisible(v.fields[index]) {
			out = append(out, index)
		}
	}
	return out
}

func (v *SettingsView) selectedField() *ConfigField {
	visible := v.visibleFieldIndices(v.currentSection())
	if len(visible) == 0 || v.fieldIdx >= len(visible) {
		return nil
	}
	index := visible[v.fieldIdx]
	return &v.fields[index]
}

func (v *SettingsView) isFieldVisible(field ConfigField) bool {
	switch field.Key {
	case "llm.api_key":
		return v.provider() != "ollama"
	case "identity.github_pat":
		return v.identityMode() == "token"
	case "identity.github_app.host":
		return v.identityMode() != "none"
	case "identity.github_app.app_id", "identity.github_app.installation_id", "identity.github_app.private_key_path":
		return v.identityMode() == "github-app"
	case "storage.dsn":
		return v.storageType() != "memory"
	default:
		return true
	}
}

func (v *SettingsView) provider() string {
	value := strings.TrimSpace(v.fieldValue("llm.provider"))
	if value == "" {
		return "openai"
	}
	return value
}

func (v *SettingsView) identityMode() string {
	value := strings.TrimSpace(v.fieldValue("identity.mode"))
	if value == "" {
		return "github-app"
	}
	return value
}

func (v *SettingsView) storageType() string {
	value := strings.TrimSpace(v.fieldValue("storage.type"))
	if value == "" {
		return "memory"
	}
	return value
}

func (v *SettingsView) fieldValue(key string) string {
	index, ok := v.fieldIndex[key]
	if !ok {
		return ""
	}
	return v.fields[index].Value
}

func (v *SettingsView) syncSaveTarget() {
	if v.snapshot.Paths.RepositoryDetected {
		for _, field := range v.fields {
			if field.Source == config.SourceRepo {
				v.saveTarget = SaveTargetRepo
				return
			}
		}
	}
	v.saveTarget = SaveTargetGlobal
}

func (v *SettingsView) saveCmd() tea.Cmd {
	target := v.saveTarget
	dirtyKeys := v.dirtyKeys()
	fields := v.GetFields()
	return func() tea.Msg {
		return ConfigSaveMsg{
			Fields:    fields,
			DirtyKeys: dirtyKeys,
			Target:    target,
		}
	}
}

func (v *SettingsView) autoSaveCmd(changed bool) tea.Cmd {
	if !changed {
		return nil
	}
	return v.saveCmd()
}

func (v *SettingsView) dirtyKeys() []string {
	keys := make([]string, 0, len(v.dirty))
	for _, field := range v.fields {
		if v.dirty[field.Key] {
			keys = append(keys, field.Key)
		}
	}
	return keys
}

func (v *SettingsView) renderStatusStrip() string {
	items := []string{
		v.badge("Scope "+v.friendlySaveTarget(v.saveTarget), v.theme.Secondary(), v.theme.OnPrimary()),
		v.badge("Section "+v.currentSection().Title, v.theme.Surface(), v.theme.Fg()),
		lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Files " + fmt.Sprintf("%d", len(v.snapshot.Paths.ActiveFiles))),
		lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Overrides " + fmt.Sprintf("%d", len(v.overrideFieldLabels()))),
	}
	if action := strings.TrimSpace(v.recommendedAction()); action != "" {
		items = append(items, lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render(trimMiddle(action, maxInt(24, v.width/2))))
	}
	return strings.Join(items, "  ")
}

func (v *SettingsView) renderSummaryCards() string {
	cards := []string{
		v.renderSummaryCard(
			"Runtime",
			nonEmptyOr("profile unset", v.fieldValue("profile")),
			[]string{
				"LLM: " + nonEmptyOr("-", v.fieldValue("llm.provider")) + " / " + nonEmptyOr("-", v.fieldValue("llm.model")),
				"GitHub: " + v.effectiveGitHubAuthLabel() + " @ " + v.effectiveGitHubHostLabel(),
				"Storage: " + strings.ToUpper(v.storageType()),
			},
		),
		v.renderSummaryCard(
			"Save Scope",
			v.friendlySaveTarget(v.saveTarget),
			[]string{
				"Target: " + trimMiddle(v.targetPath(v.saveTarget), maxInt(20, v.width/3)),
				fmt.Sprintf("Edited fields: %d", len(v.dirty)),
				fmt.Sprintf("Active layers: %d", len(v.snapshot.Paths.ActiveFiles)),
			},
		),
		v.renderSummaryCard(
			"Next Action",
			v.recommendedAction(),
			[]string{
				"Repository detected: " + boolLabel(v.snapshot.Paths.RepositoryDetected),
				fmt.Sprintf("Workspace roots: %d", len(parseSettingsPathList(v.fieldValue("git.workspace_roots")))),
				"Overrides: " + fmt.Sprintf("%d", len(v.overrideFieldLabels())),
			},
		),
	}

	if v.width >= 108 {
		width := (v.width - 4) / 3
		for i := range cards {
			cards[i] = lipgloss.NewStyle().Width(width).Render(cards[i])
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, cards...)
	}

	return strings.Join(cards, "\n")
}

func (v *SettingsView) renderSummaryCard(title, value string, lines []string) string {
	content := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render(title),
		lipgloss.NewStyle().Bold(true).Foreground(v.theme.Fg()).Render(trimMiddle(value, maxInt(16, v.width/3))),
		lipgloss.NewStyle().Foreground(v.theme.DimText()).Render(strings.Join(lines, "\n")),
	}, "\n")
	return render.SurfacePanel(content, maxInt(24, v.width/3), v.theme.Surface(), v.theme.BorderColor())
}

func (v *SettingsView) renderSectionRail(width int) string {
	panelWidth := maxInt(24, width)
	innerWidth := panelWidth - 4
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render("Configuration Map"),
		lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Each section owns a coherent part of the runtime."),
		"",
	}

	for i, section := range v.sections {
		blocking, advisory := v.sectionIssueCounts(section.ID)
		statusLabel := "READY"
		statusColor := v.theme.Success()
		if blocking > 0 {
			statusLabel = fmt.Sprintf("SETUP %d", blocking)
			statusColor = v.theme.Warning()
		} else if advisory > 0 {
			statusLabel = fmt.Sprintf("REVIEW %d", advisory)
			statusColor = v.theme.Info()
		}

		title := fmt.Sprintf("%d %s", i+1, section.Title)
		block := []string{
			lipgloss.NewStyle().Bold(i == v.sectionIdx).Foreground(v.theme.Fg()).Render(title),
			lipgloss.NewStyle().Foreground(v.theme.DimText()).Render(trimMiddle(section.Summary, maxInt(20, innerWidth))),
			lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusLabel),
		}
		rendered := strings.Join(block, "\n")
		style := lipgloss.NewStyle()
		if i == v.sectionIdx {
			style = style.Background(v.theme.Selection())
		}
		lines = append(lines, render.FillBlock(rendered, innerWidth, style))
		if i < len(v.sections)-1 {
			lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.Divider()).Render(strings.Repeat(".", maxInt(12, innerWidth))))
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Left/Right section | Tab field | Type or Enter edit | Space option | Ctrl+G / Ctrl+R scope"))
	return render.SurfacePanel(strings.Join(lines, "\n"), panelWidth, v.theme.Surface(), v.theme.BorderColor())
}

func (v *SettingsView) renderSectionDetail(width int, height int) string {
	panelWidth := maxInt(32, width)
	innerWidth := panelWidth - 4
	section := v.currentSection()
	compact := height > 0 && height < 20
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render(section.Title),
		lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Width(innerWidth).Render(section.Purpose),
		"",
	}
	anchorLine := 0

	meta := []string{
		v.statusPillForSection(section.ID),
		v.sourceMixLabel(),
		lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Save -> " + v.friendlySaveTarget(v.saveTarget)),
	}
	lines = append(lines, strings.Join(meta, "  "))
	lines = append(lines, "")
	if !compact {
		if spotlight := v.renderSelectedFieldPanel(innerWidth); spotlight != "" {
			lines = append(lines, strings.Split(spotlight, "\n")...)
			lines = append(lines, "")
		}
	}

	visible := v.visibleFieldIndices(section)
	if len(visible) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.DimText()).Italic(true).Render("No fields are active for the current mode."))
	} else {
		for idx, fieldIndex := range visible {
			if idx == v.fieldIdx {
				anchorLine = len(lines)
			}
			lines = append(lines, strings.Split(v.renderFieldRow(innerWidth, idx == v.fieldIdx, v.fields[fieldIndex]), "\n")...)
		}
	}
	if section.ID == "github" {
		lines = append(lines, "")
		lines = append(lines, strings.Split(v.renderGitHubAccessPanel(innerWidth), "\n")...)
	}

	content := strings.Join(lines, "\n")
	innerHeight := maxInt(5, height-2)
	v.detailVP.SetWidth(innerWidth)
	v.detailVP.SetHeight(innerHeight)
	v.detailVP.SetContent(content)
	v.keepSelectedFieldVisible(content, anchorLine)
	return render.SurfacePanel(v.detailVP.View(), panelWidth, v.theme.Surface(), v.theme.BorderColor())
}

func (v *SettingsView) renderFieldRow(width int, selected bool, field ConfigField) string {
	labelWidth := clampInt(width/3, 16, 24)
	metaWidth := 12
	valueWidth := maxInt(12, width-labelWidth-metaWidth-4)

	indicator := "  "
	if selected {
		indicator = "> "
	}

	labelStyle := lipgloss.NewStyle().Width(labelWidth).Foreground(v.theme.Fg())
	if selected {
		labelStyle = labelStyle.Bold(true)
	}

	metaTokens := []string{v.sourceBadge(field.Source)}
	if v.dirty[field.Key] {
		metaTokens = append(metaTokens, v.badge("EDITED", v.theme.Highlight(), v.theme.OnPrimary()))
	}
	if v.isFieldRequired(field) && strings.TrimSpace(field.Value) == "" {
		metaTokens = append(metaTokens, v.badge("REQUIRED", v.theme.Warning(), v.theme.OnPrimary()))
	}
	if valMsg, valOk := v.validateField(field); valMsg != "" {
		if valOk {
			metaTokens = append(metaTokens, lipgloss.NewStyle().Foreground(v.theme.Success()).Render("✓"))
		} else {
			metaTokens = append(metaTokens, lipgloss.NewStyle().Foreground(v.theme.Danger()).Render("✗ "+valMsg))
		}
	}
	meta := lipgloss.NewStyle().Width(metaWidth).Align(lipgloss.Right).Render(strings.Join(metaTokens, " "))

	value := v.renderFieldValue(field, selected, valueWidth)
	valueStyle := lipgloss.NewStyle().Width(valueWidth).Foreground(v.theme.Fg())
	if strings.TrimSpace(field.Value) == "" && v.focus != settingsFocusEditor {
		valueStyle = valueStyle.Foreground(v.theme.DimText()).Italic(true)
	}

	line1 := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Foreground(v.theme.Primary()).Render(indicator),
		labelStyle.Render(field.Label),
		meta,
		valueStyle.Render(value),
	)

	desc := lipgloss.NewStyle().
		Foreground(v.theme.DimText()).
		PaddingLeft(2).
		Width(maxInt(20, width)).
		Render(field.Description)

	help := field.Help
	if selected {
		if len(field.Options) > 0 {
			help = "Type or Enter edits this value directly. Space cycles the recommended options. Left/Right moves between sections. " + field.Help
		} else {
			help = "Type or Enter edits this field. Enter again applies the value immediately. Left/Right moves between sections. " + field.Help
		}
	}
	helpLine := lipgloss.NewStyle().
		Foreground(v.theme.MutedFg()).
		Italic(true).
		PaddingLeft(2).
		Width(maxInt(20, width)).
		Render(help)

	block := strings.Join([]string{line1, desc, helpLine}, "\n")
	if selected {
		block = render.FillBlock(block, width, lipgloss.NewStyle().Background(v.theme.Selection()))
	}
	return block
}

func (v *SettingsView) renderSelectedFieldPanel(width int) string {
	field := v.selectedField()
	if field == nil {
		return ""
	}

	value := strings.TrimSpace(field.Value)
	switch {
	case value == "":
		value = nonEmptyOr("(not set)", field.Placeholder)
	case field.Secret:
		value = field.Value
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render("Selected Field"),
		lipgloss.NewStyle().Bold(true).Foreground(v.theme.Fg()).Render(field.Label),
		lipgloss.NewStyle().Foreground(v.theme.DimText()).Render(field.Description),
		"",
		lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Config key"),
		lipgloss.NewStyle().Foreground(v.theme.Fg()).Render(field.Key),
		"",
		lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Current value"),
		lipgloss.NewStyle().Foreground(v.theme.Fg()).Render(value),
	}
	if field.Secret && strings.TrimSpace(field.Value) != "" {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(v.theme.Warning()).Render("Selected secret values are shown in full here so they can be verified before applying changes."),
		)
	}
	lines = append(lines,
		"",
		lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Placeholder"),
		lipgloss.NewStyle().Foreground(v.theme.DimText()).Render(nonEmptyOr("(none)", field.Placeholder)),
		"",
		lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Edit flow"),
		lipgloss.NewStyle().Foreground(v.theme.Fg()).Render(v.selectedFieldEditHint(*field)),
	)
	if len(field.Options) > 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Recommended options"))
		lines = append(lines, v.renderOptionChoices(*field))
	}
	return render.SurfacePanel(strings.Join(lines, "\n"), width, v.theme.Surface(), v.theme.BorderColor())
}

func (v *SettingsView) renderGitHubAccessPanel(width int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render("Effective GitHub Runtime"),
		lipgloss.NewStyle().Foreground(v.theme.Fg()).Render("Configured mode: " + v.identityMode()),
		lipgloss.NewStyle().Foreground(v.theme.Fg()).Render("Configured host: " + v.configuredGitHubHost()),
		lipgloss.NewStyle().Foreground(v.theme.Fg()).Render("Effective auth: " + v.effectiveGitHubAuthLabel()),
		lipgloss.NewStyle().Foreground(v.theme.Fg()).Render("Effective host: " + v.effectiveGitHubHostLabel()),
	}
	if v.runtimeGitHubReady && strings.HasPrefix(v.effectiveGitHubAuth, "env.") {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.Warning()).Render("Runtime fallback is active from environment credentials. Persist fields only if you want deterministic config."))
	}
	if v.runtimeGitHubReady && strings.HasPrefix(v.effectiveGitHubAuth, "gh.") {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.Warning()).Render("Runtime fallback is active from gh auth. Keep the CLI logged in or persist credentials explicitly."))
	}
	return render.SurfacePanel(strings.Join(lines, "\n"), width, v.theme.Surface(), v.theme.BorderColor())
}

func (v *SettingsView) renderFieldValue(field ConfigField, selected bool, width int) string {
	if selected && v.focus == settingsFocusEditor && v.selectedField() != nil && v.selectedField().Key == field.Key {
		return v.editor.View()
	}

	value := strings.TrimSpace(field.Value)
	if value == "" {
		value = nonEmptyOr("(not set)", field.Placeholder)
	}
	if field.Secret && strings.TrimSpace(field.Value) != "" && !selected {
		value = maskValue(field.Value)
	}
	if selected && len(field.Options) > 0 {
		value = value + "  [Space cycle | Enter edit]"
	}
	return value
}

func (v *SettingsView) selectedFieldEditHint(field ConfigField) string {
	if len(field.Options) > 0 {
		return "Type or Enter edits directly. Space cycles the recommended options. Enter applies immediately."
	}
	return "Type or Enter edits directly. Enter applies immediately. Esc cancels the current edit."
}

func (v *SettingsView) renderOptionChoices(field ConfigField) string {
	options := make([]string, 0, len(field.Options))
	for _, option := range field.Options {
		if option == field.Value {
			options = append(options, v.badge(option, v.theme.Primary(), v.theme.OnPrimary()))
			continue
		}
		options = append(options, lipgloss.NewStyle().Foreground(v.theme.DimText()).Render(option))
	}
	return strings.Join(options, "  ")
}

func (v *SettingsView) shouldBeginDirectEdit(km tea.KeyPressMsg) bool {
	if v.focus == settingsFocusEditor || km.Mod != 0 {
		return false
	}
	text := km.Text
	if text == "" {
		text = km.String()
	}
	if utf8.RuneCountInString(text) != 1 {
		return false
	}
	switch text {
	case " ", "space":
		return false
	}
	return true
}

func (v *SettingsView) renderFooter() string {
	parts := []string{}
	if len(v.overrideFieldLabels()) > 0 {
		parts = append(parts, "Overrides active: "+strings.Join(v.overrideFieldLabels(), ", "))
	}
	if v.statusLine != "" {
		parts = append(parts, v.statusLine)
	}
	if len(v.dirty) > 0 {
		parts = append(parts, fmt.Sprintf("%d field(s) edited in this session", len(v.dirty)))
	}
	if len(parts) == 0 {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(v.theme.DimText()).
		Render(strings.Join(parts, " | "))
}

func (v *SettingsView) collectIssues() []settingsIssue {
	issues := make([]settingsIssue, 0)
	githubFallback := v.runtimeGitHubReady && (strings.HasPrefix(v.effectiveGitHubAuth, "env.") || strings.HasPrefix(v.effectiveGitHubAuth, "gh."))
	configuredHost := v.configuredGitHubHost()
	if v.runtimeGitHubReady && configuredHost != "" && !strings.EqualFold(configuredHost, v.effectiveGitHubHostLabel()) {
		issues = append(issues, settingsIssue{
			SectionID: "github",
			Message:   fmt.Sprintf("Configured GitHub host %q is not the active runtime host. Gitdex is currently operating against %s via %s.", configuredHost, v.effectiveGitHubHostLabel(), v.effectiveGitHubAuthLabel()),
			Blocking:  false,
		})
	}

	if strings.TrimSpace(v.fieldValue("profile")) == "" {
		issues = append(issues, settingsIssue{SectionID: "control", Message: "Profile is empty. Name the operator profile so logs and behavior can be reasoned about.", Blocking: false})
	}
	if strings.TrimSpace(v.fieldValue("daemon.health_address")) == "" {
		issues = append(issues, settingsIssue{SectionID: "control", Message: "Daemon health address is empty.", Blocking: true})
	}

	if strings.TrimSpace(v.fieldValue("llm.provider")) == "" {
		issues = append(issues, settingsIssue{SectionID: "llm", Message: "Select an LLM provider.", Blocking: true})
	}
	if strings.TrimSpace(v.fieldValue("llm.model")) == "" {
		issues = append(issues, settingsIssue{SectionID: "llm", Message: "Model name is missing.", Blocking: true})
	}
	if v.provider() != "ollama" && strings.TrimSpace(v.fieldValue("llm.api_key")) == "" {
		issues = append(issues, settingsIssue{SectionID: "llm", Message: "Hosted provider selected but API key is missing.", Blocking: true})
	}
	if strings.TrimSpace(v.fieldValue("llm.endpoint")) == "" {
		issues = append(issues, settingsIssue{SectionID: "llm", Message: "Endpoint is empty.", Blocking: true})
	}

	if strings.TrimSpace(v.fieldValue("git.user_name")) == "" {
		issues = append(issues, settingsIssue{SectionID: "git", Message: "Git author name is empty. Read flows work, but write flows will be weaker.", Blocking: false})
	}
	if strings.TrimSpace(v.fieldValue("git.user_email")) == "" {
		issues = append(issues, settingsIssue{SectionID: "git", Message: "Git author email is empty. Read flows work, but write flows will be weaker.", Blocking: false})
	}
	if len(parseSettingsPathList(v.fieldValue("git.workspace_roots"))) == 0 {
		issues = append(issues, settingsIssue{SectionID: "git", Message: "Workspace roots are empty. Local clone discovery will fall back to broader scanning and slower matching.", Blocking: false})
	}

	switch v.identityMode() {
	case "none":
		if v.runtimeGitHubReady {
			issues = append(issues, settingsIssue{SectionID: "github", Message: fmt.Sprintf("Config mode is none, but runtime fallback is active through %s at %s.", v.effectiveGitHubAuthLabel(), v.effectiveGitHubHostLabel()), Blocking: false})
		} else {
			issues = append(issues, settingsIssue{SectionID: "github", Message: "GitHub access is disabled. Explorer and dashboard sync will stay local-only.", Blocking: false})
		}
	case "token":
		if strings.TrimSpace(v.fieldValue("identity.github_pat")) == "" && !v.runtimeGitHubReady {
			issues = append(issues, settingsIssue{SectionID: "github", Message: "Token mode is selected but no personal token is configured.", Blocking: true})
		}
		if strings.TrimSpace(v.fieldValue("identity.github_pat")) == "" && githubFallback {
			issues = append(issues, settingsIssue{SectionID: "github", Message: fmt.Sprintf("No PAT is stored in config, but runtime fallback is active through %s.", v.effectiveGitHubAuthLabel()), Blocking: false})
		}
	case "github-app":
		missing := make([]string, 0, 3)
		if strings.TrimSpace(v.fieldValue("identity.github_app.app_id")) == "" {
			missing = append(missing, "GitHub App ID")
		}
		if strings.TrimSpace(v.fieldValue("identity.github_app.installation_id")) == "" {
			missing = append(missing, "GitHub App installation ID")
		}
		if strings.TrimSpace(v.fieldValue("identity.github_app.private_key_path")) == "" {
			missing = append(missing, "GitHub App private key path")
		}
		if len(missing) > 0 {
			if githubFallback {
				issues = append(issues, settingsIssue{SectionID: "github", Message: fmt.Sprintf("GitHub App fields are incomplete (%s), but runtime fallback is currently active through %s.", strings.Join(missing, ", "), v.effectiveGitHubAuthLabel()), Blocking: false})
			} else {
				for _, label := range missing {
					issues = append(issues, settingsIssue{SectionID: "github", Message: label + " is missing.", Blocking: true})
				}
			}
		}
	default:
		if githubFallback {
			issues = append(issues, settingsIssue{SectionID: "github", Message: fmt.Sprintf("Authentication mode %q is invalid, but runtime fallback is currently active through %s.", v.identityMode(), v.effectiveGitHubAuthLabel()), Blocking: false})
		} else {
			issues = append(issues, settingsIssue{SectionID: "github", Message: "Authentication mode is invalid or empty.", Blocking: true})
		}
	}

	if strings.TrimSpace(v.fieldValue("storage.type")) == "" {
		issues = append(issues, settingsIssue{SectionID: "storage", Message: "Storage backend is missing.", Blocking: true})
	}
	if v.storageType() != "memory" && strings.TrimSpace(v.fieldValue("storage.dsn")) == "" {
		issues = append(issues, settingsIssue{SectionID: "storage", Message: "Persistent backend selected but connection string is missing.", Blocking: true})
	}

	return issues
}

func (v *SettingsView) sectionIssueCounts(sectionID string) (blocking int, advisory int) {
	for _, issue := range v.collectIssues() {
		if issue.SectionID != sectionID {
			continue
		}
		if issue.Blocking {
			blocking++
		} else {
			advisory++
		}
	}
	return blocking, advisory
}

func (v *SettingsView) recommendedAction() string {
	for _, issue := range v.collectIssues() {
		if issue.Blocking {
			return issue.Message
		}
	}
	for _, issue := range v.collectIssues() {
		if !issue.Blocking {
			return issue.Message
		}
	}
	return "Configuration is coherent. Changes apply immediately."
}

func (v *SettingsView) statusPillForSection(sectionID string) string {
	blocking, advisory := v.sectionIssueCounts(sectionID)
	switch {
	case blocking > 0:
		return v.badge(fmt.Sprintf("SETUP %d", blocking), v.theme.Warning(), v.theme.OnPrimary())
	case advisory > 0:
		return v.badge(fmt.Sprintf("REVIEW %d", advisory), v.theme.Info(), v.theme.OnPrimary())
	default:
		return v.badge("READY", v.theme.Success(), v.theme.OnPrimary())
	}
}

func (v *SettingsView) sourceMixLabel() string {
	counts := map[config.ValueSource]int{}
	for _, field := range v.fields {
		counts[field.Source]++
	}
	parts := make([]string, 0, 4)
	for _, source := range []config.ValueSource{config.SourceEnv, config.SourceRepo, config.SourceGlobal, config.SourceDefault} {
		if counts[source] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", strings.ToUpper(string(source)), counts[source]))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return lipgloss.NewStyle().Foreground(v.theme.DimText()).Render(strings.Join(parts, " | "))
}

func (v *SettingsView) refreshGitHubRuntimeStatus() {
	v.runtimeGitHubReady = false
	v.effectiveGitHubAuth = ""
	v.effectiveGitHubHost = normalizeGitHubHost(v.fieldValue("identity.github_app.host"))
	if !v.shouldProbeGitHubRuntime() {
		if v.effectiveGitHubHost == "" {
			v.effectiveGitHubHost = "github.com"
		}
		return
	}
	if resolved, err := identity.ResolveTransport(v.identityConfig(), nil); err == nil {
		v.runtimeGitHubReady = true
		v.effectiveGitHubAuth = strings.TrimSpace(resolved.Source)
		if strings.TrimSpace(resolved.Host) != "" {
			v.effectiveGitHubHost = strings.TrimSpace(resolved.Host)
		}
	}
	if v.effectiveGitHubHost == "" {
		v.effectiveGitHubHost = "github.com"
	}
}

func (v *SettingsView) identityConfig() config.IdentityConfig {
	return config.IdentityConfig{
		Mode:      v.identityMode(),
		GitHubPAT: v.fieldValue("identity.github_pat"),
		GitHubApp: config.GitHubAppConfig{
			Host:           v.fieldValue("identity.github_app.host"),
			AppID:          v.fieldValue("identity.github_app.app_id"),
			InstallationID: v.fieldValue("identity.github_app.installation_id"),
			PrivateKeyPath: v.fieldValue("identity.github_app.private_key_path"),
		},
	}
}

func (v *SettingsView) shouldProbeGitHubRuntime() bool {
	if strings.TrimSpace(v.fieldValue("identity.github_pat")) != "" {
		return true
	}
	if strings.TrimSpace(v.fieldValue("identity.github_app.app_id")) != "" ||
		strings.TrimSpace(v.fieldValue("identity.github_app.installation_id")) != "" ||
		strings.TrimSpace(v.fieldValue("identity.github_app.private_key_path")) != "" {
		return true
	}
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"} {
		if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return v.snapshot.Paths.WorkingDir != "" || len(v.snapshot.Paths.ActiveFiles) > 0 || v.snapshot.Paths.RepositoryDetected
}

func (v *SettingsView) effectiveGitHubAuthLabel() string {
	if strings.TrimSpace(v.effectiveGitHubAuth) == "" {
		return "unavailable"
	}
	return v.effectiveGitHubAuth
}

func (v *SettingsView) effectiveGitHubHostLabel() string {
	return nonEmptyOr("github.com", v.effectiveGitHubHost)
}

func (v *SettingsView) configuredGitHubHost() string {
	return normalizeGitHubHost(v.fieldValue("identity.github_app.host"))
}

func (v *SettingsView) overrideFieldLabels() []string {
	labels := make([]string, 0)
	for _, field := range v.fields {
		if field.Source == config.SourceEnv || field.Source == config.SourceFlag {
			labels = append(labels, field.Label)
		}
	}
	return labels
}

func (v *SettingsView) validateField(field ConfigField) (string, bool) {
	val := strings.TrimSpace(field.Value)
	if val == "" {
		if v.isFieldRequired(field) {
			return "Required", false
		}
		return "", true
	}
	switch field.Key {
	case "llm.api_key":
		if !strings.HasPrefix(val, "sk-") && v.provider() == "openai" {
			return "OpenAI keys start with sk-", false
		}
		return "Valid", true
	case "llm.endpoint":
		if !strings.HasPrefix(val, "http://") && !strings.HasPrefix(val, "https://") {
			return "Must start with http:// or https://", false
		}
		return "Valid", true
	case "identity.github_app.host":
		if strings.ContainsAny(val, " \t\n") || (!strings.Contains(val, ".") && val != "localhost") {
			return "Invalid hostname", false
		}
		return "Valid", true
	case "identity.github_pat":
		if len(val) < 10 {
			return "Token too short", false
		}
		return "Valid", true
	case "git.user_email":
		if !strings.Contains(val, "@") {
			return "Must contain @", false
		}
		return "Valid", true
	case "identity.github_app.private_key_path", "git.ssh_key_path":
		expanded := os.ExpandEnv(val)
		if _, err := os.Stat(expanded); err != nil {
			return "File not found", false
		}
		return "Valid", true
	}
	return "", true
}

func (v *SettingsView) isFieldRequired(field ConfigField) bool {
	switch field.Key {
	case "llm.api_key":
		return v.provider() != "ollama"
	case "identity.github_pat":
		return v.identityMode() == "token"
	case "identity.github_app.host":
		return v.identityMode() != "none"
	case "identity.github_app.app_id", "identity.github_app.installation_id", "identity.github_app.private_key_path":
		return v.identityMode() == "github-app"
	case "storage.dsn":
		return v.storageType() != "memory"
	default:
		return field.Required
	}
}

func (v *SettingsView) sourceBadge(source config.ValueSource) string {
	label := strings.ToUpper(string(source))
	bg := v.theme.Surface()
	fg := v.theme.DimText()
	switch source {
	case config.SourceGlobal:
		bg = v.theme.Primary()
		fg = v.theme.OnPrimary()
	case config.SourceRepo:
		bg = v.theme.Secondary()
		fg = v.theme.OnPrimary()
	case config.SourceEnv:
		bg = v.theme.Warning()
		fg = v.theme.OnPrimary()
	case config.SourceFlag:
		bg = v.theme.Danger()
		fg = v.theme.OnPrimary()
	case config.SourceExplicitConfig:
		bg = v.theme.Info()
		fg = v.theme.OnPrimary()
	}
	return v.badge(label, bg, fg)
}

func (v *SettingsView) badge(label string, bg, fg color.Color) string {
	return lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Background(bg).
		Foreground(fg).
		Render(label)
}

func (v *SettingsView) friendlySaveTarget(target ConfigSaveTarget) string {
	switch target {
	case SaveTargetRepo:
		return "Repository"
	default:
		return "Global"
	}
}

func (v *SettingsView) targetPath(target ConfigSaveTarget) string {
	switch target {
	case SaveTargetRepo:
		return nonEmptyOr("(repo config unavailable)", v.snapshot.Paths.RepoConfig)
	default:
		return nonEmptyOr("(global config unavailable)", v.snapshot.Paths.GlobalConfig)
	}
}

func (v *SettingsView) detailValueWidth() int {
	labelWidth := clampInt(v.width/3, 16, 24)
	metaWidth := 12
	return maxInt(12, v.width-labelWidth-metaWidth-12)
}

func (v *SettingsView) keepSelectedFieldVisible(content string, anchorLine int) {
	lineCount := len(strings.Split(content, "\n"))
	if lineCount <= v.detailVP.Height() {
		v.detailVP.GotoTop()
		return
	}
	if anchorLine < v.detailVP.YOffset() {
		v.detailVP.SetYOffset(anchorLine)
		return
	}
	bottom := v.detailVP.YOffset() + v.detailVP.Height()
	if anchorLine >= bottom {
		v.detailVP.SetYOffset(anchorLine - v.detailVP.Height() + 3)
	}
}

func maskValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if len(value) <= 4 {
		return "***"
	}
	return strings.Repeat("*", minInt(8, len(value)-2))
}

func normalizeGitHubHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if host == "" {
		return "github.com"
	}
	return host
}

func serializeSettingsPathList(values []string) string {
	return strings.Join(values, ", ")
}

func parseSettingsPathList(raw string) []string {
	splitter := func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	}
	parts := strings.FieldsFunc(raw, splitter)
	seen := make(map[string]bool, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, part)
	}
	return out
}

func nonEmptyOr(fallback, value string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func trimMiddle(value string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxWidth {
		return value
	}
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	head := (maxWidth - 3) / 2
	tail := maxWidth - head - 3
	if head < 1 {
		head = 1
	}
	if tail < 1 {
		tail = 1
	}
	return string(runes[:head]) + "..." + string(runes[len(runes)-tail:])
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
