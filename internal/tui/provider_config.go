package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llmfactory"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func providerOptions() []string {
	specs := llm.ProviderSpecs()
	options := make([]string, 0, len(specs))
	for _, spec := range specs {
		options = append(options, spec.ID)
	}
	return options
}

func providerOptionIndex(provider string) int {
	provider = strings.ToLower(strings.TrimSpace(provider))
	options := providerOptions()
	for i, option := range options {
		if option == provider {
			return i
		}
	}
	return 0
}

func (m Model) openProviderConfig(role modelSelectPhase) Model {
	var draft config.ModelConfig
	if role == selectSecondary {
		draft = m.llmConfig.SecondaryRole()
		if strings.TrimSpace(draft.Provider) == "" {
			draft.Provider = m.primaryProvider
			draft.Endpoint = m.llmConfig.PrimaryRole().Endpoint
		}
	} else {
		draft = m.llmConfig.PrimaryRole()
	}
	if strings.TrimSpace(draft.Provider) == "" {
		draft.Provider = "ollama"
	}
	if strings.TrimSpace(draft.Endpoint) == "" {
		draft.Endpoint = defaultProviderEndpoint(draft.Provider)
	}
	if strings.TrimSpace(draft.APIKeyEnv) == "" {
		draft.APIKeyEnv = defaultProviderAPIKeyEnv(draft.Provider)
	}
	if strings.TrimSpace(draft.Model) == "" {
		draft.Model = firstRecommendedModel(draft.Provider)
	}

	m.providerRole = role
	m.providerStoredKey = strings.TrimSpace(draft.APIKey)
	m.providerKeyDirty = false
	if m.providerStoredKey != "" {
		draft.APIKey = ""
	}
	m.providerDraft = draft
	m.providerField = providerFieldProvider
	m.providerCursorAt = 0
	m.screen = screenProviderConfig
	m.statusMsg = i18n.T("provider_config.prompt")
	return m
}

func defaultProviderEndpoint(provider string) string {
	return llm.ProviderSpecFor(provider).DefaultBaseURL
}

func defaultProviderAPIKeyEnv(provider string) string {
	return llm.ProviderSpecFor(provider).APIKeyEnv
}

func firstRecommendedModel(provider string) string {
	spec := llm.ProviderSpecFor(provider)
	if len(spec.RecommendedModels) == 0 {
		return ""
	}
	return spec.RecommendedModels[0]
}

func (m Model) providerFieldCount() int {
	return len(m.providerFields())
}

func (m Model) providerFields() []providerField {
	if strings.EqualFold(strings.TrimSpace(m.providerDraft.Provider), "ollama") {
		return []providerField{providerFieldProvider, providerFieldEndpoint}
	}
	return []providerField{providerFieldProvider, providerFieldModel, providerFieldEndpoint, providerFieldAPIKey}
}

func (m Model) providerFieldIndex(field providerField) int {
	fields := m.providerFields()
	for i, candidate := range fields {
		if candidate == field {
			return i
		}
	}
	return 0
}

func (m Model) providerFieldAt(index int) providerField {
	fields := m.providerFields()
	if len(fields) == 0 {
		return providerFieldProvider
	}
	if index < 0 {
		index = 0
	}
	if index >= len(fields) {
		index = len(fields) - 1
	}
	return fields[index]
}

func (m Model) providerFieldValue(field providerField) string {
	switch field {
	case providerFieldModel:
		return m.providerDraft.Model
	case providerFieldEndpoint:
		return m.providerDraft.Endpoint
	case providerFieldAPIKey:
		return m.providerDraft.APIKey
	default:
		return m.providerDraft.Provider
	}
}

func (m *Model) setProviderFieldValue(field providerField, value string) {
	switch field {
	case providerFieldModel:
		m.providerDraft.Model = value
	case providerFieldEndpoint:
		m.providerDraft.Endpoint = value
	case providerFieldAPIKey:
		m.providerDraft.APIKey = value
		m.providerKeyDirty = true
	default:
		m.providerDraft.Provider = strings.ToLower(strings.TrimSpace(value))
	}
}

func (m Model) providerRoleLabel() string {
	if m.providerRole == selectSecondary {
		return i18n.T("provider_config.role_secondary")
	}
	return i18n.T("provider_config.role_primary")
}

func (m Model) providerFieldLabel(field providerField) string {
	switch field {
	case providerFieldModel:
		return i18n.T("provider_config.field_model")
	case providerFieldEndpoint:
		return i18n.T("provider_config.field_base_url")
	case providerFieldAPIKey:
		return i18n.T("provider_config.field_api_key")
	default:
		return i18n.T("provider_config.field_provider")
	}
}

func (m Model) providerFieldPlaceholder(field providerField) string {
	spec := llm.ProviderSpecFor(m.providerDraft.Provider)
	switch field {
	case providerFieldModel:
		if len(spec.RecommendedModels) > 0 {
			return spec.RecommendedModels[0]
		}
		return i18n.T("provider_config.placeholder_model")
	case providerFieldEndpoint:
		return defaultProviderEndpoint(m.providerDraft.Provider)
	case providerFieldAPIKey:
		if m.providerStoredKey != "" && !m.providerKeyDirty {
			return localizedText(
				"Configured key is hidden. Type to replace it, or leave blank to keep it.",
				"已配置的密钥会被隐藏。直接输入可替换；保持为空则保留原值。",
				"Configured key is hidden. Type to replace it, or leave blank to keep it.",
			)
		}
		envName := defaultProviderAPIKeyEnv(m.providerDraft.Provider)
		if envName == "" {
			return i18n.T("provider_config.placeholder_api_key_optional")
		}
		return fmt.Sprintf(i18n.T("provider_config.placeholder_api_key"), envName)
	default:
		return "ollama / openai / deepseek"
	}
}

func (m Model) renderProviderConfigScreen() string {
	spec := llm.ProviderSpecFor(m.providerDraft.Provider)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7BD8FF"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#97A9B8"))
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F4C46B"))
	boxActive := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6FC3DF")).
		Padding(0, 1).
		Width(maxInt(24, minInt(72, m.width-8)))
	boxIdle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#31556F")).
		Padding(0, 1).
		Width(maxInt(24, minInt(72, m.width-8)))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99"))
	choiceOn := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10212B")).
		Background(lipgloss.Color("#F2C572")).
		Bold(true).
		Padding(0, 1)
	choiceOff := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C7D5E0")).
		Background(lipgloss.Color("#233645")).
		Padding(0, 1)

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf(i18n.T("provider_config.title"), m.providerRoleLabel())))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(i18n.T("provider_config.hint")))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render(fmt.Sprintf(i18n.T("provider_config.provider_summary"), spec.Label, spec.DefaultBaseURL)))
	if spec.APIKeyEnv != "" {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf(i18n.T("provider_config.provider_api_env"), spec.APIKeyEnv)))
	}
	if len(spec.RecommendedModels) > 0 {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf(i18n.T("provider_config.provider_models"), strings.Join(spec.RecommendedModels, ", "))))
	}
	if spec.DocsURL != "" {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf(i18n.T("provider_config.provider_docs"), spec.DocsURL)))
	}
	b.WriteString("\n\n")

	for _, field := range m.providerFields() {
		b.WriteString(labelStyle.Render(m.providerFieldLabel(field)))
		b.WriteString("\n")
		if field == providerFieldProvider {
			options := make([]string, 0, len(providerOptions()))
			for _, option := range providerOptions() {
				style := choiceOff
				if option == strings.ToLower(strings.TrimSpace(m.providerDraft.Provider)) {
					style = choiceOn
				}
				options = append(options, style.Render(option))
			}
			body := strings.Join(options, " ")
			style := boxIdle
			if m.providerField == field {
				style = boxActive
			}
			b.WriteString(style.Render(body))
		} else {
			value := m.providerFieldValue(field)
			display := value
			if display == "" && m.providerField != field {
				display = lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")).Render(m.providerFieldPlaceholder(field))
			}
			if m.providerField == field {
				before, after := splitAtRune(value, m.providerCursorAt)
				cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C572")).Bold(true).Render("|")
				if value == "" {
					display = cursor + lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")).Render(m.providerFieldPlaceholder(field))
				} else {
					display = before + cursor + after
				}
				b.WriteString(boxActive.Render(display))
			} else {
				b.WriteString(boxIdle.Render(display))
			}
		}
		b.WriteString("\n")
		if field == providerFieldAPIKey {
			envName := strings.TrimSpace(m.providerDraft.APIKeyEnv)
			if envName != "" {
				b.WriteString(helpStyle.Render(fmt.Sprintf(i18n.T("provider_config.api_key_env_hint"), envName)))
				b.WriteString("\n")
			}
			if m.providerStoredKey != "" && !m.providerKeyDirty {
				b.WriteString(helpStyle.Render(localizedText(
					"A saved key already exists and is currently hidden.",
					"已存在一个已保存的密钥，当前已隐藏显示。",
					"A saved key already exists and is currently hidden.",
				)))
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render(i18n.T("provider_config.footer")))
	return m.padContent(b.String(), m.height)
}

func (m Model) updateProviderConfig(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	text := msg.Key().Text

	switch {
	case key == "q" || key == "ctrl+c":
		return m, tea.Quit
	case key == "escape" || key == "esc" || msg.Key().Code == tea.KeyEscape:
		m = m.openModelSetup(m.providerRole)
		m.statusMsg = i18n.T("provider_config.cancelled")
		return m, nil
	case key == "up" || key == "shift+tab":
		if index := m.providerFieldIndex(m.providerField); index > 0 {
			m.providerField = m.providerFieldAt(index - 1)
			m.providerCursorAt = runeLen(m.providerFieldValue(m.providerField))
		}
		return m, nil
	case key == "down" || key == "tab":
		if index := m.providerFieldIndex(m.providerField); index < m.providerFieldCount()-1 {
			m.providerField = m.providerFieldAt(index + 1)
			m.providerCursorAt = runeLen(m.providerFieldValue(m.providerField))
		}
		return m, nil
	case key == "left":
		if m.providerField == providerFieldProvider {
			m = m.shiftProviderOption(-1)
			return m, nil
		}
		if m.providerCursorAt > 0 {
			m.providerCursorAt--
		}
		return m, nil
	case key == "right":
		if m.providerField == providerFieldProvider {
			m = m.shiftProviderOption(1)
			return m, nil
		}
		if m.providerCursorAt < runeLen(m.providerFieldValue(m.providerField)) {
			m.providerCursorAt++
		}
		return m, nil
	case key == "home" || key == "ctrl+a":
		if m.providerField != providerFieldProvider {
			m.providerCursorAt = 0
		}
		return m, nil
	case key == "end" || key == "ctrl+e":
		if m.providerField != providerFieldProvider {
			m.providerCursorAt = runeLen(m.providerFieldValue(m.providerField))
		}
		return m, nil
	case key == "backspace":
		if m.providerField == providerFieldProvider {
			return m, nil
		}
		value, nextCursor := deleteRuneBefore(m.providerFieldValue(m.providerField), m.providerCursorAt)
		m.setProviderFieldValue(m.providerField, value)
		m.providerCursorAt = nextCursor
		return m, nil
	case key == "delete":
		if m.providerField == providerFieldProvider {
			return m, nil
		}
		value, nextCursor := deleteRuneAt(m.providerFieldValue(m.providerField), m.providerCursorAt)
		m.setProviderFieldValue(m.providerField, value)
		m.providerCursorAt = nextCursor
		return m, nil
	case key == "enter":
		return m.persistProviderConfig()
	default:
		if m.providerField == providerFieldProvider {
			prev := strings.ToLower(strings.TrimSpace(m.providerDraft.Provider))
			switch strings.ToLower(strings.TrimSpace(text)) {
			case "o":
				m.providerDraft.Provider = "ollama"
			case "p":
				m.providerDraft.Provider = "openai"
			case "d":
				m.providerDraft.Provider = "deepseek"
			}
			if prev != m.providerDraft.Provider {
				m.providerDraft.Endpoint = defaultProviderEndpoint(m.providerDraft.Provider)
				m.providerDraft.APIKeyEnv = defaultProviderAPIKeyEnv(m.providerDraft.Provider)
				m.providerDraft.APIKey = ""
				m.providerStoredKey = ""
				m.providerKeyDirty = false
				if recommended := firstRecommendedModel(m.providerDraft.Provider); recommended != "" {
					m.providerDraft.Model = recommended
				}
			}
			if strings.EqualFold(m.providerDraft.Provider, "ollama") {
				m.providerField = providerFieldEndpoint
				m.providerCursorAt = runeLen(m.providerDraft.Endpoint)
			} else {
				m.providerField = providerFieldModel
				m.providerCursorAt = runeLen(m.providerDraft.Model)
			}
			return m, nil
		}
		if text != "" {
			return m.handleProviderPaste(text)
		}
	}
	return m, nil
}

func (m Model) handleProviderPaste(text string) (tea.Model, tea.Cmd) {
	if text == "" || m.providerField == providerFieldProvider {
		return m, nil
	}
	value, nextCursor := insertAtRune(m.providerFieldValue(m.providerField), m.providerCursorAt, text)
	m.setProviderFieldValue(m.providerField, value)
	m.providerCursorAt = nextCursor
	return m, nil
}

func (m Model) shiftProviderOption(delta int) Model {
	options := providerOptions()
	if len(options) == 0 {
		return m
	}
	oldProvider := m.providerDraft.Provider
	idx := providerOptionIndex(oldProvider)
	idx = (idx + delta + len(options)) % len(options)
	nextProvider := options[idx]
	nextSpec := llm.ProviderSpecFor(nextProvider)
	prevEndpoint := strings.TrimSpace(m.providerDraft.Endpoint)
	prevAPIEnv := strings.TrimSpace(m.providerDraft.APIKeyEnv)
	m.providerDraft.Provider = nextProvider
	if prevEndpoint == "" || prevEndpoint == defaultProviderEndpoint(oldProvider) {
		m.providerDraft.Endpoint = defaultProviderEndpoint(nextProvider)
	} else {
		m.providerDraft.Endpoint = prevEndpoint
	}
	m.providerDraft.APIKey = ""
	m.providerStoredKey = ""
	m.providerKeyDirty = false
	if prevAPIEnv == "" || prevAPIEnv == defaultProviderAPIKeyEnv(oldProvider) {
		m.providerDraft.APIKeyEnv = defaultProviderAPIKeyEnv(nextProvider)
	}
	if len(nextSpec.RecommendedModels) > 0 {
		prevModel := strings.TrimSpace(m.providerDraft.Model)
		if prevModel == "" || prevModel == firstRecommendedModel(oldProvider) {
			m.providerDraft.Model = nextSpec.RecommendedModels[0]
		}
	}
	if strings.EqualFold(nextProvider, "ollama") && m.providerField == providerFieldAPIKey {
		m.providerField = providerFieldEndpoint
		m.providerCursorAt = runeLen(m.providerDraft.Endpoint)
	}
	if !strings.EqualFold(nextProvider, "ollama") && m.providerField == providerFieldEndpoint && strings.TrimSpace(m.providerDraft.Model) == "" {
		m.providerField = providerFieldModel
		m.providerCursorAt = runeLen(m.providerDraft.Model)
	}
	return m
}

func (m Model) persistProviderConfig() (tea.Model, tea.Cmd) {
	next := config.Get()
	if next == nil {
		next = config.DefaultConfig()
	}
	cfg := *next

	draft := m.providerDraft
	draft.Provider = strings.ToLower(strings.TrimSpace(draft.Provider))
	draft.Model = strings.TrimSpace(draft.Model)
	draft.Endpoint = strings.TrimSpace(draft.Endpoint)
	draft.APIKey = strings.TrimSpace(draft.APIKey)
	if !m.providerKeyDirty && draft.APIKey == "" {
		draft.APIKey = strings.TrimSpace(m.providerStoredKey)
	}
	if draft.Provider == "ollama" && draft.Model == "" {
		if current := strings.TrimSpace(m.roleModel(m.providerRole)); current != "" {
			draft.Model = current
		} else {
			spec := llm.ProviderSpecFor(draft.Provider)
			if len(spec.RecommendedModels) > 0 {
				draft.Model = spec.RecommendedModels[0]
			}
		}
	}
	if draft.Model == "" {
		m.statusMsg = i18n.T("provider_config.model_required")
		m.providerField = providerFieldModel
		m.providerCursorAt = 0
		return m, nil
	}
	if strings.TrimSpace(draft.Endpoint) == "" {
		draft.Endpoint = defaultProviderEndpoint(draft.Provider)
	}
	if strings.TrimSpace(draft.APIKeyEnv) == "" {
		draft.APIKeyEnv = defaultProviderAPIKeyEnv(draft.Provider)
	}

	if m.providerRole == selectSecondary {
		draft.Enabled = strings.TrimSpace(draft.Model) != ""
		cfg.LLM.Secondary = draft
	} else {
		draft.Enabled = true
		cfg.LLM.Provider = draft.Provider
		cfg.LLM.Model = draft.Model
		cfg.LLM.Endpoint = draft.Endpoint
		cfg.LLM.APIKey = draft.APIKey
		cfg.LLM.APIKeyEnv = draft.APIKeyEnv
		cfg.LLM.Primary = draft
	}

	if err := config.SaveGlobal(&cfg); err != nil {
		m.statusMsg = fmt.Sprintf(i18n.T("provider_config.failed"), err)
		return m, nil
	}

	config.Set(&cfg)
	m = m.applyLLMConfigRuntime(cfg.LLM)
	m.statusMsg = fmt.Sprintf(i18n.T("provider_config.saved"), draft.Provider)
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: fmt.Sprintf("Configured %s provider: %s", m.providerRoleLabel(), draft.Provider),
		Detail:  fmt.Sprintf("model=%s endpoint=%s", draft.Model, draft.Endpoint),
	})
	if draft.Provider == "ollama" {
		m = m.openLocalModelSelection(m.providerRole, draft.Provider)
		return m, func() tea.Msg {
			return m.fetchSetupProviderModels(draft.Provider)
		}
	}
	if m.providerRole == selectPrimary {
		m = m.openModelSetup(selectSecondary)
		return m, nil
	}
	m.screen = screenMain
	return m, m.refreshGitState
}

func (m Model) applyLLMConfigRuntime(cfg config.LLMConfig) Model {
	provider, effective := llmfactory.Build(cfg)
	m = m.SetLLMConfig(effective)
	m.llmProvider = provider
	role := effective.PrimaryRole()
	switch config.RoleProvider(role) {
	case "openai", "deepseek":
		if config.ResolveRoleAPIKey(role) == "" {
			m.startupInfo.AIStatus = config.RoleProvider(role) + ": missing API key"
		} else {
			m.startupInfo.AIStatus = config.RoleProvider(role) + ": configured"
		}
	default:
		m.startupInfo.AIStatus = config.RoleProvider(role)
	}
	if m.pipeline != nil {
		m.pipeline.SetLLMProvider(provider, effective)
	}
	if provider == nil {
		m.availModels = nil
	}
	return m
}
