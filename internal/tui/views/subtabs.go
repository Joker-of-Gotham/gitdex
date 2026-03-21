package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

// explorerMegaGroups partitions Explorer tab indices: GitHub (API) vs local Git (files/commits/branches).
var explorerMegaGroups = [][]int{
	{0, 1, 5, 6, 7},
	{2, 3, 4},
}

var explorerMegaLabels = []string{"GitHub", "Git"}

func explorerMegaForFlat(flat int) int {
	for gi, g := range explorerMegaGroups {
		for _, idx := range g {
			if idx == flat {
				return gi
			}
		}
	}
	return 0
}

func explorerSlotForFlat(flat int) int {
	mega := explorerMegaForFlat(flat)
	g := explorerMegaGroups[mega]
	for i, idx := range g {
		if idx == flat {
			return i
		}
	}
	return 0
}

func renderSubTabs(tabs []string, active int, t *theme.Theme, width int) string {
	return renderSubTabsWithOptions(tabs, active, t, width, nil)
}

// SubTabRenderOptions configures optional mega-tab rendering (Explorer).
type SubTabRenderOptions struct {
	MegaGroups [][]int
	MegaLabels []string
}

func renderSubTabsWithOptions(tabs []string, active int, th *theme.Theme, width int, mega *SubTabRenderOptions) string {
	if width <= 0 {
		return ""
	}

	if mega != nil && len(mega.MegaGroups) > 0 && len(mega.MegaLabels) > 0 {
		return renderMegaSubTabs(tabs, active, th, width, mega.MegaGroups, mega.MegaLabels)
	}

	n := len(tabs)
	jump := "1"
	if n > 1 {
		jump = fmt.Sprintf("1-%d", n)
	}
	hint := lipgloss.NewStyle().
		Foreground(th.DimText()).
		Italic(true).
		Render(jump + " jump  <- -> cycle  Enter drill")

	items := make([]string, 0, n)
	for i, tab := range tabs {
		label := fmt.Sprintf("%d:%s", i+1, tab)
		if i == active {
			items = append(items, lipgloss.NewStyle().
				Bold(true).
				Foreground(th.OnPrimary()).
				Background(th.Primary()).
				Padding(0, 1).
				Render(label))
			continue
		}

		items = append(items, lipgloss.NewStyle().
			Foreground(th.MutedFg()).
			Background(th.Surface()).
			Padding(0, 1).
			Render(label))
	}

	top := strings.Join(items, " ")
	if lipgloss.Width(top)+lipgloss.Width(hint)+2 <= width {
		top += "  " + hint
	}

	divider := lipgloss.NewStyle().
		Foreground(th.Divider()).
		Render(strings.Repeat("─", maxInt(12, width)))

	filterLine := lipgloss.NewStyle().
		Foreground(th.DimText()).
		Italic(true).
		Render("/ filter")

	return top + "\n" + filterLine + "\n" + divider
}

func renderMegaSubTabs(tabs []string, activeFlat int, th *theme.Theme, width int, groups [][]int, megaLabels []string) string {
	mega := explorerMegaForFlat(activeFlat)
	g := groups[mega]

	items := make([]string, 0, len(g))
	for si, ti := range g {
		if ti < 0 || ti >= len(tabs) {
			continue
		}
		label := fmt.Sprintf("%d:%s", si+1, tabs[ti])
		if ti == activeFlat {
			items = append(items, lipgloss.NewStyle().
				Bold(true).
				Foreground(th.OnPrimary()).
				Background(th.Primary()).
				Padding(0, 1).
				Render(label))
			continue
		}
		items = append(items, lipgloss.NewStyle().
			Foreground(th.MutedFg()).
			Background(th.Surface()).
			Padding(0, 1).
			Render(label))
	}

	megaRow := make([]string, 0, len(megaLabels))
	for mi, ml := range megaLabels {
		st := lipgloss.NewStyle().Padding(0, 1)
		if mi == mega {
			megaRow = append(megaRow, st.Bold(true).Foreground(th.OnPrimary()).Background(th.Secondary()).Render(ml))
		} else {
			megaRow = append(megaRow, st.Foreground(th.MutedFg()).Background(th.Surface()).Render(ml))
		}
	}

	subCount := len(g)
	jump := "1"
	if subCount > 1 {
		jump = fmt.Sprintf("1-%d", subCount)
	}
	hint := lipgloss.NewStyle().
		Foreground(th.DimText()).
		Italic(true).
		Render(jump + " jump  <- -> cycle  [ ] group")

	top := strings.Join(megaRow, "  ")
	sub := strings.Join(items, " ")
	if lipgloss.Width(sub)+lipgloss.Width(hint)+2 <= width {
		sub += "  " + hint
	}

	divider := lipgloss.NewStyle().
		Foreground(th.Divider()).
		Render(strings.Repeat("─", maxInt(12, width)))

	filterLine := lipgloss.NewStyle().
		Foreground(th.DimText()).
		Italic(true).
		Render("/ filter")

	return top + "\n" + sub + "\n" + filterLine + "\n" + divider
}
