package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m *Model) setCommandResponse(title, body string) {
	m.commandResponseTitle = strings.TrimSpace(title)
	m.commandResponseBody = strings.TrimSpace(body)
	m.leftScroll = 0
	m.workspaceTab = workspaceTabOverview
}

func (m *Model) clearCommandResponse() {
	m.commandResponseTitle = ""
	m.commandResponseBody = ""
}

func (m Model) renderCommandResponsePanel(width int) string {
	if strings.TrimSpace(m.commandResponseBody) == "" {
		return ""
	}
	panelWidth := width
	if panelWidth < 18 {
		panelWidth = 18
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5D7892")).
		Padding(0, 1)
	innerWidth, _ := panelInnerSize(borderStyle, panelWidth, 1)

	title := localizedAssistantTitle()
	if strings.TrimSpace(m.commandResponseTitle) != "" {
		title = m.commandResponseTitle
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9FD6FF")).Render(title),
	}
	lines = append(lines, wrapPlainText(strings.TrimSpace(m.commandResponseBody), innerWidth)...)
	return borderStyle.Width(panelWidth).Render(strings.Join(lines, "\n"))
}

func (m Model) renderAutomationPanel(width int) string {
	panelWidth := width
	if panelWidth < 18 {
		panelWidth = 18
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4F6B82")).
		Padding(0, 1)
	innerWidth, _ := panelInnerSize(borderStyle, panelWidth, 1)
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8FD1A6")).Render(localizedAutomationTitle()),
	}
	lines = append(lines, wrapPlainText(m.automationSummaryText(), innerWidth)...)
	lines = append(lines, wrapPlainText(m.automationGoalRequirementText(), innerWidth)...)
	return borderStyle.Width(panelWidth).Render(strings.Join(lines, "\n"))
}
