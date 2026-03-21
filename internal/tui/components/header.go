package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/your-org/gitdex/internal/tui/theme"
	"github.com/your-org/gitdex/internal/tui/views"
)

var tabIcons = map[views.ID]string{
	views.ViewDashboard: theme.Icons.Dashboard,
	views.ViewChat:      theme.Icons.Chat,
	views.ViewExplorer:  theme.Icons.Branch,
	views.ViewWorkspace: theme.Icons.Plan,
	views.ViewSettings:  theme.Icons.Settings,
	views.ViewReflog:    theme.Icons.Commit,
}

var tabAbbrev = map[views.ID]string{
	views.ViewDashboard: "Dash",
	views.ViewChat:      "Chat",
	views.ViewExplorer:  "Expl",
	views.ViewWorkspace: "Work",
	views.ViewSettings:  "Cfg",
	views.ViewReflog:    "Ref",
}

type Header struct {
	t        *theme.Theme
	width    int
	tabs     []views.ID
	active   views.ID
	titles   map[views.ID]string
	repoName string
}

func NewHeader(t *theme.Theme) *Header {
	return &Header{
		t:      t,
		titles: make(map[views.ID]string),
	}
}

func (h *Header) SetTabs(router *views.Router) {
	h.tabs = router.Order()
	h.active = router.ActiveID()
	h.titles = make(map[views.ID]string, len(h.tabs))
	for _, id := range h.tabs {
		h.titles[id] = router.ViewTitle(id)
	}
}

func (h *Header) SetWidth(w int)      { h.width = w }
func (h *Header) SetRepo(name string) { h.repoName = name }

func (h *Header) buildTabs(labels map[views.ID]string) string {
	var tabs []string
	for i, id := range h.tabs {
		icon := tabIcons[id]
		if icon == "" {
			icon = theme.Icons.Dot
		}
		name := labels[id]
		if name == "" {
			name = h.titles[id]
		}
		label := icon + " " + name
		if id == h.active {
			tabs = append(tabs, lipgloss.NewStyle().
				Bold(true).
				Foreground(h.t.OnPrimary()).
				Background(h.t.Secondary()).
				Padding(0, 1).
				Render(label))
			continue
		}
		shortcut := lipgloss.NewStyle().
			Foreground(h.t.DimText()).
			Render(fmt.Sprintf("F%d", i+1))
		tab := lipgloss.NewStyle().
			Foreground(h.t.MutedFg()).
			Background(h.t.Surface()).
			Padding(0, 1).
			Render(label)
		tabs = append(tabs, shortcut+" "+tab)
	}
	return strings.Join(tabs, " ")
}

func (h *Header) Render() string {
	if h.width <= 0 {
		return ""
	}

	brand := lipgloss.NewStyle().
		Bold(true).
		Foreground(h.t.OnPrimary()).
		Background(h.t.Primary()).
		Padding(0, 1).
		Render(theme.Icons.Dashboard + " Gitdex")

	repoChip := ""
	if h.repoName != "" {
		repoChip = lipgloss.NewStyle().
			Bold(true).
			Foreground(h.t.Fg()).
			Background(h.t.Surface()).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(h.t.BorderColor()).
			Padding(0, 1).
			Render(theme.Icons.Branch + " " + runewidth.Truncate(h.repoName, 28, ".."))
	}

	rightFull := lipgloss.NewStyle().
		Foreground(h.t.DimText()).
		Render("Ctrl+P Palette  Ctrl+T Theme  Ctrl+I Inspector  ? Help")
	rightShort := lipgloss.NewStyle().
		Foreground(h.t.DimText()).
		Render("Ctrl+P  ?")

	assemble := func(includeRepo bool, tabStr, rightStr string) string {
		parts := []string{brand}
		if includeRepo && repoChip != "" {
			parts = append(parts, repoChip)
		}
		if strings.TrimSpace(tabStr) != "" {
			parts = append(parts, tabStr)
		}
		left := strings.Join(parts, " ")
		gap := h.width - lipgloss.Width(left) - lipgloss.Width(rightStr)
		if gap < 0 {
			return ""
		}
		if gap == 0 && rightStr == "" {
			return left
		}
		if gap < 1 {
			gap = 1
		}
		return left + strings.Repeat(" ", gap) + rightStr
	}

	tabsFull := h.buildTabs(h.titles)
	tabsAbbr := h.buildTabs(tabAbbrev)
	activeOnly := h.buildTabs(map[views.ID]string{h.active: h.titles[h.active]})

	candidates := []string{
		assemble(true, tabsFull, rightFull),
		assemble(true, tabsFull, rightShort),
		assemble(true, tabsAbbr, rightShort),
		assemble(false, tabsFull, rightShort),
		assemble(false, tabsAbbr, rightShort),
		assemble(false, tabsAbbr, ""),
		assemble(false, activeOnly, ""),
	}
	for _, line := range candidates {
		if line == "" {
			continue
		}
		if lipgloss.Width(line) <= h.width {
			return h.renderLine(line)
		}
	}

	line := runewidth.Truncate(brand, h.width, "")
	return h.renderLine(line)
}

func (h *Header) renderLine(line string) string {
	return lipgloss.NewStyle().
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(h.t.Divider()).
		Width(h.width).
		Render(line)
}
