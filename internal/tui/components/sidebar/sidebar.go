package sidebar

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Model manages the sidebar preview pane.
// Aligned with gh-dash's sidebar.Model.
type Model struct {
	IsOpen   bool
	content  string
	title    string
	width    int
	height   int
	scrollY  int
	ctx      *context.ProgramContext
}

// New creates a sidebar model.
func New() Model {
	return Model{}
}

// SetDimensions updates the sidebar size.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
}

// SetContent sets the sidebar body.
func (m *Model) SetContent(title, content string) {
	m.title = title
	m.content = content
	m.scrollY = 0
}

// UpdateProgramContext stores the program context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
}

// ScrollDown scrolls the sidebar content down.
func (m *Model) ScrollDown(n int) {
	lines := strings.Count(m.content, "\n") + 1
	maxScroll := lines - m.height + 2
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.scrollY += n
	if m.scrollY > maxScroll {
		m.scrollY = maxScroll
	}
}

// ScrollUp scrolls the sidebar content up.
func (m *Model) ScrollUp(n int) {
	m.scrollY -= n
	if m.scrollY < 0 {
		m.scrollY = 0
	}
}

// View renders the sidebar.
func (m Model) View() string {
	if !m.IsOpen || m.width <= 0 {
		return ""
	}

	var containerStyle lipgloss.Style
	if m.ctx != nil {
		containerStyle = m.ctx.Styles.Sidebar.ContainerStyle
	} else {
		containerStyle = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder())
	}

	innerW := m.width - 2
	if innerW < 1 {
		innerW = 1
	}

	var b strings.Builder
	if m.title != "" {
		var titleStyle lipgloss.Style
		if m.ctx != nil {
			titleStyle = m.ctx.Styles.Sidebar.TitleStyle
		} else {
			titleStyle = lipgloss.NewStyle().Bold(true)
		}
		b.WriteString(titleStyle.Width(innerW).Render(m.title))
		b.WriteString("\n")
	}

	lines := strings.Split(m.content, "\n")
	start := m.scrollY
	if start >= len(lines) {
		start = 0
	}
	end := start + m.height - 2
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]
	body := strings.Join(visible, "\n")

	var contentStyle lipgloss.Style
	if m.ctx != nil {
		contentStyle = m.ctx.Styles.Sidebar.ContentStyle
	} else {
		contentStyle = lipgloss.NewStyle()
	}
	b.WriteString(contentStyle.Width(innerW).Render(body))

	return containerStyle.
		Width(m.width).
		Height(m.height).
		Render(b.String())
}
