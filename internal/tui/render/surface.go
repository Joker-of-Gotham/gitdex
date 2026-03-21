package render

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// FillBlock pads every line in a block to the same width so backgrounds and
// selection fills cover the whole visual surface rather than just the text.
func FillBlock(content string, width int, style lipgloss.Style) string {
	if width < 1 {
		width = 1
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	lineStyle := style.Width(width).MaxWidth(width)
	for i := range lines {
		lines[i] = lineStyle.Render(lines[i])
	}
	return strings.Join(lines, "\n")
}

// SurfacePanel renders a rounded panel with a fully painted background at the
// requested outer width.
func SurfacePanel(content string, outerWidth int, bg, border color.Color) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Background(bg).
		Padding(0, 1)

	innerWidth := outerWidth - style.GetHorizontalFrameSize()
	if innerWidth < 1 {
		innerWidth = 1
	}

	filled := FillBlock(content, innerWidth, lipgloss.NewStyle().Background(bg))
	return style.Width(outerWidth).MaxWidth(outerWidth).Render(filled)
}
