package panes

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/keymap"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type NavItem struct {
	Label string
	Path  string
}

type NavPane struct {
	items      []NavItem
	t          *theme.Theme
	styles     theme.Styles
	listKeys   keymap.ListKeys
	width      int
	height     int
	focused    bool
	cursor     int
	activePath string
}

type NavSelectMsg struct {
	Item NavItem
}

func NewNavPane(t *theme.Theme, styles theme.Styles, items []NavItem) *NavPane {
	if len(items) == 0 {
		items = []NavItem{
			{Label: "Status", Path: "status"},
			{Label: "Risks", Path: "risks"},
		}
	}
	return &NavPane{
		t:          t,
		styles:     styles,
		listKeys:   keymap.DefaultListKeys(),
		items:      items,
		activePath: items[0].Path,
	}
}

func navItemIcon(path string) string {
	switch path {
	case "dashboard":
		return theme.Icons.Dashboard
	case "chat":
		return theme.Icons.Chat
	case "explorer":
		return theme.Icons.Branch
	case "workspace":
		return theme.Icons.Plan
	case "settings":
		return theme.Icons.Settings
	default:
		return theme.Icons.ChevronRight
	}
}

func navItemDescription(path string) string {
	switch path {
	case "dashboard":
		return "Operational health, live summary, and repo intake"
	case "chat":
		return "Conversation stream, slash commands, and AI exchange"
	case "explorer":
		return "PRs, issues, files, and drill-down previews"
	case "workspace":
		return "Plans, tasks, evidence, and execution signals"
	case "settings":
		return "Identity, providers, storage, and output preferences"
	default:
		return ""
	}
}

func (p *NavPane) Init() tea.Cmd { return nil }

func (p *NavPane) Update(msg tea.Msg) (*NavPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !p.focused {
			return p, nil
		}
		maxIdx := len(p.items) - 1
		if maxIdx < 0 {
			return p, nil
		}
		switch {
		case p.listKeys.Up.Matches(msg):
			if p.cursor > 0 {
				p.cursor--
			}
		case p.listKeys.Down.Matches(msg):
			if p.cursor < maxIdx {
				p.cursor++
			}
		case p.listKeys.Select.Matches(msg):
			if p.cursor < len(p.items) {
				p.activePath = p.items[p.cursor].Path
				return p, func() tea.Msg { return NavSelectMsg{Item: p.items[p.cursor]} }
			}
		}
	}
	return p, nil
}

func (p *NavPane) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(p.t.Primary()).Render("Control Plane")
	subtitle := lipgloss.NewStyle().Foreground(p.t.DimText()).Render("Dashboard-first navigation with persistent inspector")
	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.t.Fg()).
		Background(p.t.Selection()).
		Padding(0, 1)
	activeStyle := lipgloss.NewStyle().
		Foreground(p.t.OnPrimary()).
		Background(p.t.Secondary()).
		Padding(0, 1)
	fgStyle := lipgloss.NewStyle().Foreground(p.t.Fg())
	dimStyle := lipgloss.NewStyle().Foreground(p.t.DimText())

	lines := []string{title, subtitle, ""}
	for i, item := range p.items {
		line := navItemIcon(item.Path) + " " + item.Label
		switch {
		case p.focused && i == p.cursor:
			lines = append(lines, selectedStyle.Render(line))
		case item.Path == p.activePath:
			lines = append(lines, activeStyle.Render(line))
		default:
			lines = append(lines, fgStyle.Render(line))
		}
		desc := navItemDescription(item.Path)
		if desc != "" {
			lines = append(lines, dimStyle.Render("  "+desc))
		}
		if i < len(p.items)-1 {
			lines = append(lines, dimStyle.Render(strings.Repeat(".", max(10, p.width-6))))
		}
	}
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Enter open | Ctrl+1 focus nav"))
	return p.applyBorder(strings.Join(lines, "\n"))
}

func (p *NavPane) SelectPath(path string) {
	p.activePath = path
	for i, item := range p.items {
		if item.Path == path {
			p.cursor = i
			break
		}
	}
}

func (p *NavPane) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *NavPane) SetFocused(focused bool) { p.focused = focused }

func (p NavPane) Focused() bool { return p.focused }

func (p NavPane) SelectedItem() NavItem {
	if p.cursor < len(p.items) {
		return p.items[p.cursor]
	}
	return NavItem{}
}

func (p *NavPane) SetStyles(s theme.Styles) { p.styles = s }

func (p *NavPane) SetItems(items []NavItem) {
	p.items = items
	if p.cursor >= len(items) {
		p.cursor = 0
	}
	if len(items) > 0 && p.activePath == "" {
		p.activePath = items[0].Path
	}
}

func (p *NavPane) applyBorder(content string) string {
	border := p.t.BorderColor()
	if p.focused {
		border = p.t.FocusBorderColor()
	}
	w := p.width
	if w < 12 {
		w = 12
	}
	return render.SurfacePanel(content, w, p.t.Surface(), border)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
