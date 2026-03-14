package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

type languageOption struct {
	Code  string
	Label string
}

func languageOptions() []languageOption {
	return []languageOption{
		{Code: "auto", Label: "Auto / System"},
		{Code: "en", Label: "English"},
		{Code: "zh", Label: "Chinese (Simplified)"},
		{Code: "ja", Label: "Japanese"},
	}
}

func configuredLanguage() string {
	cfg := config.Get()
	if cfg == nil {
		return "auto"
	}
	lang := strings.TrimSpace(cfg.I18n.Language)
	if lang == "" {
		return "auto"
	}
	return lang
}

func (m Model) currentLanguageOptionLabel() string {
	code := configuredLanguage()
	for _, option := range languageOptions() {
		if option.Code == code {
			return option.Label
		}
	}
	return code
}

func (m Model) currentResolvedLanguage() string {
	lang := strings.TrimSpace(i18n.Lang())
	if lang == "" {
		return "en"
	}
	return lang
}

func (m Model) shouldShowFirstRunLanguageSelection() bool {
	return m.startupInfo.FirstRun && !m.languageConfigured
}

func (m Model) openLanguageSelection(returnTo screenMode) Model {
	m.languageReturnTo = returnTo
	m.languageCursor = m.languageCursorFor(configuredLanguage())
	m.screen = screenLanguageSelect
	m.statusMsg = i18n.T("language_select.prompt")
	return m
}

func (m Model) languageCursorFor(code string) int {
	for i, option := range languageOptions() {
		if option.Code == code {
			return i
		}
	}
	return 0
}

func languageOptionByCode(code string) (languageOption, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, option := range languageOptions() {
		if option.Code == code {
			return option, true
		}
	}
	return languageOption{}, false
}

func (m Model) syncSessionLanguagePreference() Model {
	if m.session.Preferences == nil {
		m.session.Preferences = make(map[string]string)
	}
	lang := m.currentResolvedLanguage()
	m.session.Preferences["language"] = lang
	m.rememberPreference("language", lang)
	return m
}

func localizedLanguageSwitchBody(label string) string {
	return fmt.Sprintf(
		localizedText(
			"Language switched to %s.\nRun /refresh to regenerate analysis and suggestions in this language.",
			"语言已切换为 %s。\n运行 /refresh 以用该语言重新生成分析和建议。",
			"Language switched to %s.\nRun /refresh to regenerate analysis and suggestions in this language.",
		),
		label,
	)
}

func (m Model) applyLanguagePreference(code string) (Model, error) {
	option, ok := languageOptionByCode(code)
	if !ok {
		return m, fmt.Errorf("unsupported language %q", code)
	}
	if err := i18n.Init(option.Code); err != nil {
		return m, err
	}
	if err := config.UpdateLanguagePreference(option.Code); err != nil {
		return m, err
	}
	m.languageConfigured = true
	m = m.syncSessionLanguagePreference()
	m.renderCache = newRenderCache()

	m.llmAnalysis = ""
	m.llmThinking = ""
	m.llmReason = ""
	m.llmPlanOverview = ""
	m.llmGoalStatus = ""
	m.suggestions = nil
	m.suggExecState = nil
	m.suggExecMsg = nil
	m.suggIdx = 0
	m.expanded = false

	m.setCommandResponse(localizedAssistantTitle(), localizedLanguageSwitchBody(option.Label))
	m.statusMsg = fmt.Sprintf(
		localizedText("Language switched: %s", "语言已切换：%s", "Language switched: %s"),
		option.Label,
	)
	return m, nil
}

func (m Model) resumeAfterLanguageSelection() (tea.Model, tea.Cmd) {
	m.languageConfigured = true
	returnTo := m.languageReturnTo
	m.languageReturnTo = screenMain

	switch returnTo {
	case screenMain:
		m.screen = screenMain
		return m, nil
	case screenModelSelect:
		m.screen = screenModelSelect
		return m, nil
	default:
		if len(m.availModels) > 0 {
			m.screen = screenModelSelect
			return m, nil
		}
		m.screen = screenMain
		return m, nil
	}
}

func (m Model) updateLanguageSelect(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	options := languageOptions()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.languageCursor > 0 {
			m.languageCursor--
		}
		return m, nil
	case "down", "j":
		if m.languageCursor < len(options)-1 {
			m.languageCursor++
		}
		return m, nil
	case "escape":
		if m.languageReturnTo == screenLoading {
			return m, nil
		}
		m.screen = m.languageReturnTo
		m.statusMsg = i18n.T("language_select.cancelled")
		return m, nil
	case "enter":
		choice := options[m.languageCursor]
		next, err := m.applyLanguagePreference(choice.Code)
		if err != nil {
			m.statusMsg = i18n.T("language_select.failed")
			return m, nil
		}
		next = next.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Language selected: " + choice.Code,
			Detail:  choice.Label,
		})
		return next.resumeAfterLanguageSelection()
	}
	return m, nil
}
