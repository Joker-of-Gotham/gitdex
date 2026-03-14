package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderAutomationConfigScreenV2() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7BD8FF"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#97A9B8"))
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F4C46B"))
	boxActive := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6FC3DF")).
		Padding(0, 1).
		Width(maxInt(28, minInt(72, m.width-8)))
	boxIdle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#31556F")).
		Padding(0, 1).
		Width(maxInt(28, minInt(72, m.width-8)))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99"))

	var b strings.Builder
	b.WriteString(titleStyle.Render(localizedText("Automation settings", "自动化设置", "Automation settings")))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(localizedText(
		"Choose one clear operating mode, then tune cadence and trust.",
		"先选定一个清晰的运行模式，再调整巡检节奏与信任策略。",
		"Choose one clear operating mode, then tune cadence and trust.",
	)))
	b.WriteString("\n\n")

	for _, field := range automationFields() {
		b.WriteString(labelStyle.Render(m.automationFieldLabel(field)))
		b.WriteString("\n")

		style := boxIdle
		if m.automationField == field {
			style = boxActive
		}

		value := m.automationFieldValue(field)
		if field == automationFieldMode {
			value = localizedAutomationModeLabel(m.automationDraftMode())
		}
		b.WriteString(style.Render(value))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(m.automationFieldHelp(field)))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render(m.automationGoalRequirementText()))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(localizedAutomationModeDescription(m.automationDraftMode())))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(localizedText(
		"Left/Right or Space: adjust  Up/Down/Tab: move  Enter: save  Esc: cancel",
		"左右或空格：调整  上下或 Tab：移动  Enter：保存  Esc：取消",
		"Left/Right or Space: adjust  Up/Down/Tab: move  Enter: save  Esc: cancel",
	)))

	return m.padContent(b.String(), m.height)
}
