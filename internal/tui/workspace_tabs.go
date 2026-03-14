package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type workspaceTabSegment struct {
	label    string
	rendered string
	tab      workspaceTab
}

func (t workspaceTab) valid() bool {
	return t >= workspaceTabOverview && t <= workspaceTabAnalysis
}

func (t workspaceTab) label() string {
	switch t {
	case workspaceTabSuggestions:
		return localizedText("Suggestions", "建议", "Suggestions")
	case workspaceTabResult:
		return localizedText("Result", "结果", "Result")
	case workspaceTabAnalysis:
		return localizedText("Analysis", "分析", "Analysis")
	default:
		return localizedText("Overview", "总览", "Overview")
	}
}

func workspaceTabs() []workspaceTab {
	return []workspaceTab{
		workspaceTabOverview,
		workspaceTabSuggestions,
		workspaceTabResult,
		workspaceTabAnalysis,
	}
}

func (m Model) renderWorkspaceTabsWithRegions(width int) (string, []clickRegion) {
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F2C572"))
	idleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9FB4C4"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99"))

	segments := make([]workspaceTabSegment, 0, len(workspaceTabs()))
	for _, tab := range workspaceTabs() {
		label := tab.label()
		if tab == m.workspaceTab {
			label = "[" + label + "]"
		}
		style := idleStyle
		if tab == m.workspaceTab {
			style = activeStyle
		}
		segments = append(segments, workspaceTabSegment{
			label:    label,
			rendered: style.Render(label),
			tab:      tab,
		})
	}

	parts := make([]string, 0, len(segments))
	regions := make([]clickRegion, 0, len(segments))
	x := 0
	for i, segment := range segments {
		if i > 0 {
			parts = append(parts, hintStyle.Render(" "))
			x++
		}
		parts = append(parts, segment.rendered)
		widthPart := lipgloss.Width(segment.rendered)
		regions = append(regions, clickRegion{
			action: "workspace_tab",
			index:  int(segment.tab),
			x0:     x,
			y0:     0,
			x1:     x + widthPart,
			y1:     1,
		})
		x += widthPart
	}
	line := strings.Join(parts, "")
	if lipgloss.Width(line) > width {
		line = truncateLine(line, width)
	}
	return line, regions
}

func (m Model) renderWorkspacePrimarySection(width int) (string, []clickRegion) {
	switch m.workspaceTab {
	case workspaceTabSuggestions:
		if len(m.suggestions) == 0 {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(localizedText(
				"No suggestions yet. Run /refresh or set a goal.",
				"当前还没有建议。运行 /refresh 或先设置目标。",
				"No suggestions yet. Run /refresh or set a goal.",
			)), nil
		}
		return m.renderSuggestionCardsCompactWithRegions(width)
	case workspaceTabResult:
		return m.renderWorkspaceResultPanel(width), nil
	case workspaceTabAnalysis:
		if strings.TrimSpace(m.llmAnalysis) == "" {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(localizedText(
				"No analysis yet. Run /refresh or set a goal.",
				"当前还没有分析。运行 /refresh 或先设置目标。",
				"No analysis yet. Run /refresh or set a goal.",
			)), nil
		}
		return m.renderAnalysisPanelCached(width), nil
	default:
		return m.renderWorkspaceOverviewPanel(width), nil
	}
}
