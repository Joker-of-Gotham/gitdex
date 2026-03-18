// Package tabs provides a gh-dash style tab bar component.
package tabs

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// ViewTab defines a single tab entry.
type ViewTab struct {
	Title    string
	Icon     string
	ViewType tuictx.ViewType
}

// Model is the tabs component.
type Model struct {
	views   []ViewTab
	current int
	ctx     *tuictx.ProgramContext
	counts  map[tuictx.ViewType]int
	loading map[tuictx.ViewType]bool
	width   int
}

// New creates a tabs model with the standard GitDex views.
func New(ctx *tuictx.ProgramContext) Model {
	return Model{
		views: []ViewTab{
			{Title: "Agent", Icon: ">>", ViewType: tuictx.AgentView},
			{Title: "Git", Icon: "*", ViewType: tuictx.GitView},
			{Title: "Workspace", Icon: "#", ViewType: tuictx.WorkspaceView},
			{Title: "GitHub", Icon: "@", ViewType: tuictx.GitHubView},
			{Title: "Config", Icon: "~", ViewType: tuictx.ConfigView},
		},
		current: 0,
		ctx:     ctx,
		counts:  make(map[tuictx.ViewType]int),
		loading: make(map[tuictx.ViewType]bool),
	}
}

// Next switches to the next tab.
func (m *Model) Next() {
	m.current = (m.current + 1) % len(m.views)
}

// Prev switches to the previous tab.
func (m *Model) Prev() {
	m.current--
	if m.current < 0 {
		m.current = len(m.views) - 1
	}
}

// SetCurrent sets the active tab by ViewType.
func (m *Model) SetCurrent(vt tuictx.ViewType) {
	for i, v := range m.views {
		if v.ViewType == vt {
			m.current = i
			return
		}
	}
}

// CurrentView returns the active ViewType.
func (m *Model) CurrentView() tuictx.ViewType {
	if m.current < 0 || m.current >= len(m.views) {
		return tuictx.AgentView
	}
	return m.views[m.current].ViewType
}

// SetCount updates the badge count for a view.
func (m *Model) SetCount(vt tuictx.ViewType, count int) {
	m.counts[vt] = count
}

// SetLoading sets the loading state for a view.
func (m *Model) SetLoading(vt tuictx.ViewType, loading bool) {
	m.loading[vt] = loading
}

// SetWidth sets the available width.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// View renders the tab bar.
func (m *Model) View() string {
	logo := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.ctx.Theme.Primary)).
		Render("GitDex")

	var tabs []string
	sep := m.ctx.Styles.Tabs.Separator.Render(" | ")

	for i, v := range m.views {
		label := v.Title
		if count, ok := m.counts[v.ViewType]; ok && count > 0 {
			label = fmt.Sprintf("%s(%d)", label, count)
		}
		if m.loading[v.ViewType] {
			label = label + ".."
		}

		var style lipgloss.Style
		if i == m.current {
			style = m.ctx.Styles.Tabs.Active
		} else {
			style = m.ctx.Styles.Tabs.Inactive
		}
		tabs = append(tabs, style.Render(label))
	}

	tabsLine := strings.Join(tabs, sep)
	spacer := ""
	tabWidth := lipgloss.Width(tabsLine) + lipgloss.Width(logo) + 4
	if m.width > tabWidth {
		spacer = strings.Repeat(" ", m.width-tabWidth)
	}

	return m.ctx.Styles.Tabs.Container.Render(
		lipgloss.JoinHorizontal(lipgloss.Top, tabsLine, spacer, logo),
	)
}

// UpdateProgramContext updates the shared context.
func (m *Model) UpdateProgramContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}
