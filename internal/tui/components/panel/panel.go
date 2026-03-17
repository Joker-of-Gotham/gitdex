package panel

import (
	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Model provides a reusable bordered panel container with title and content.
type Model struct {
	title   string
	content string
	focused bool
	width   int
	height  int
	ctx     *context.ProgramContext
}

// New creates a panel with the given title.
func New(title string) Model {
	return Model{title: title}
}

// SetDimensions updates the panel size.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused marks the panel as focused (changes border style).
func (m *Model) SetFocused(f bool) {
	m.focused = f
}

// SetContent sets the panel body content.
func (m *Model) SetContent(c string) {
	m.content = c
}

// UpdateProgramContext stores the program context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
}

// View renders the panel with border, title, and content.
func (m Model) View() string {
	var s lipgloss.Style
	if m.ctx != nil {
		if m.focused {
			s = m.ctx.Styles.Panel.FocusedBorderStyle
		} else {
			s = m.ctx.Styles.Panel.BorderStyle
		}
	} else {
		border := lipgloss.RoundedBorder()
		s = lipgloss.NewStyle().Border(border)
	}

	innerW := m.width - 2
	innerH := m.height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	titleStr := ""
	if m.title != "" {
		var titleStyle lipgloss.Style
		if m.ctx != nil {
			titleStyle = m.ctx.Styles.Panel.TitleStyle
		} else {
			titleStyle = lipgloss.NewStyle().Bold(true)
		}
		titleStr = titleStyle.Render(m.title)
		innerH--
	}

	body := lipgloss.NewStyle().
		Width(innerW).
		MaxHeight(innerH).
		Render(m.content)

	if titleStr != "" {
		body = titleStr + "\n" + body
	}

	return s.Width(m.width).Height(m.height).Render(body)
}
