package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components"
)

func (m Model) scrollPaneBy(pane scrollPane, delta int) Model {
	m.scrollFocus = pane
	if delta == 0 {
		return m
	}

	metrics := m.computeLayoutMetrics()
	leftWidth, rightWidth, narrow := m.columnWidths()
	switch pane {
	case scrollPaneLog:
		logHeight := metrics.logHeight
		panelStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8A6BFF")).
			Padding(0, 1)
		_, innerHeight := panelInnerSize(panelStyle, leftWidth, logHeight)
		bodyHeight := maxInt(3, innerHeight-3)
		content := strings.Join(m.operationLogLines(maxInt(12, leftWidth-panelStyle.GetHorizontalFrameSize())), "\n")
		m.logScrollOffset = clampOffset(lineCount(content), bodyHeight, m.logScrollOffset-delta)
	case scrollPaneAreas:
		if narrow || rightWidth <= 0 {
			return m
		}
		topHeight := metrics.workspaceHeight
		areasHeight, _ := m.rightPanelHeights(topHeight)
		areasPanelStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6FC3DF")).
			Padding(0, 1)
		_, areasInner := panelInnerSize(areasPanelStyle, rightWidth, areasHeight)
		areasBodyHeight := maxInt(3, areasInner-2)
		content := componentsTreeContent(m.gitState, rightWidth)
		m.areasScroll = clampOffset(lineCount(content), areasBodyHeight, m.areasScroll+delta)
	case scrollPaneObservability:
		if narrow || rightWidth <= 0 {
			return m
		}
		topHeight := metrics.workspaceHeight
		_, obsHeight := m.rightPanelHeights(topHeight)
		obsPanelStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6FC3DF")).
			Padding(0, 1)
		obsInnerWidth, obsInnerHeight := panelInnerSize(obsPanelStyle, rightWidth, obsHeight)
		obsBodyHeight := maxInt(4, obsInnerHeight-6)
		content := m.observabilityBody(obsInnerWidth)
		m.obsScroll = clampOffset(lineCount(content), obsBodyHeight, m.obsScroll+delta)
	default:
		workspaceHeight := metrics.workspaceHeight
		if workspaceHeight < 1 {
			workspaceHeight = 1
		}
		content := m.renderLeftWorkspace(leftWidth)
		m.leftScroll = clampOffset(lineCount(content), workspaceHeight, m.leftScroll+delta)
	}
	return m
}

func (m Model) cycleScrollPane(next bool) Model {
	order := []scrollPane{scrollPaneWorkspace, scrollPaneLog, scrollPaneAreas, scrollPaneObservability}
	idx := 0
	for i, pane := range order {
		if pane == m.scrollFocus {
			idx = i
			break
		}
	}
	if next {
		idx = (idx + 1) % len(order)
	} else {
		idx = (idx - 1 + len(order)) % len(order)
	}
	m.scrollFocus = order[idx]
	return m
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

func componentsTreeContent(state *status.GitState, width int) string {
	return components.NewAreasTree(state).
		SetWidth(maxInt(20, width-4)).
		SetMaxItems(0).
		View()
}
