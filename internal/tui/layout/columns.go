package layout

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// ColumnLayout holds the content strings for the three-column layout.
type ColumnLayout struct {
	Nav       string
	Main      string
	Inspector string
}

// RenderColumns renders the nav, main, and inspector content according to
// dimensions. Panes render their own borders; this layout only constrains width
// and inserts a single gutter so parent chrome does not cut through child
// panels.
func RenderColumns(dims Dimensions, nav, main, inspector string, borderColor color.Color) string {
	h := dims.ContentHeight()
	hasNav := dims.ShowNav() && strings.TrimSpace(nav) != ""
	hasInspector := dims.ShowInspector() && strings.TrimSpace(inspector) != ""

	baseStyle := lipgloss.NewStyle().
		Height(h).
		MaxHeight(h)

	gutterStyle := lipgloss.NewStyle().
		Height(h).
		MaxHeight(h).
		Width(1).
		Background(borderColor)
	gutter := gutterStyle.Render(" ")

	switch {
	case !hasNav && !hasInspector:
		return baseStyle.Width(dims.Width).Render(main)

	case hasInspector && !hasNav:
		mainW := dims.MainWidth()
		inspW := dims.InspectorWidth()
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			baseStyle.Width(mainW).Render(main),
			gutter,
			baseStyle.Width(inspW).Render(inspector),
		)

	default:
		navW := dims.NavWidth()
		mainW := dims.MainWidth()
		inspW := dims.InspectorWidth()

		parts := []string{
			baseStyle.Width(navW).Render(nav),
			gutter,
			baseStyle.Width(mainW).Render(main),
		}
		if hasInspector {
			parts = append(parts, gutter, baseStyle.Width(inspW).Render(inspector))
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	}
}
