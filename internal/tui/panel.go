package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func renderBoundedPanel(style lipgloss.Style, totalWidth, totalHeight int, content string) string {
	innerWidth, innerHeight := panelInnerSize(style, totalWidth, totalHeight)
	return style.Width(totalWidth).Render(fitPanelContent(content, innerWidth, innerHeight))
}

func panelInnerSize(style lipgloss.Style, totalWidth, totalHeight int) (int, int) {
	frameWidth, frameHeight := style.GetFrameSize()
	innerWidth := totalWidth - frameWidth
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := totalHeight - frameHeight
	if innerHeight < 1 {
		innerHeight = 1
	}
	return innerWidth, innerHeight
}

func fitPanelContent(content string, width, height int) string {
	if height < 1 {
		height = 1
	}
	lines := make([]string, 0, height)
	for _, raw := range strings.Split(content, "\n") {
		if width <= 0 {
			lines = append(lines, raw)
			continue
		}
		wrapped := ansi.Wrap(raw, width, " ")
		lines = append(lines, strings.Split(wrapped, "\n")...)
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
