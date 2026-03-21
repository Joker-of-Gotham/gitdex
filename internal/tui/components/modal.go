package components

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type Modal struct {
	title   string
	content string
	width   int
	height  int
	theme   *theme.Theme
	visible bool
}

func NewModal(t *theme.Theme, title string) *Modal {
	return &Modal{
		title:   title,
		theme:   t,
		width:   60,
		height:  20,
		visible: false,
	}
}

func (m *Modal) Show(content string) {
	m.content = content
	m.visible = true
}

func (m *Modal) Hide() {
	m.visible = false
}

func (m *Modal) IsVisible() bool {
	return m.visible
}

func (m *Modal) Update(msg tea.Msg) tea.Cmd {
	if !m.visible {
		return nil
	}
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "esc" {
		m.Hide()
		return nil
	}
	return nil
}

func (m *Modal) Render(totalWidth, totalHeight int) string {
	if !m.visible {
		return ""
	}
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary()).
		Align(lipgloss.Center)
	contentStyle := lipgloss.NewStyle().Foreground(m.theme.Fg())

	titleBlock := titleStyle.Width(m.width - 2).Render(m.title)
	contentBlock := contentStyle.Width(m.width - 2).Render(m.content)
	inner := titleBlock + "\n\n" + contentBlock

	box := lipgloss.NewStyle().
		Background(m.theme.Elevated()).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.FocusBorderColor()).
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Render(inner)

	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)
}
