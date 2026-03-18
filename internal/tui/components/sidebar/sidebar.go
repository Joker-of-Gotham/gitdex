// Package sidebar provides a scrollable detail panel with Markdown rendering.
// Inspired by gh-dash's components/sidebar.
package sidebar

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Model is the sidebar component.
type Model struct {
	IsOpen  bool
	ctx     *tuictx.ProgramContext
	content string
	lines   []string
	topLine int
	width   int
	height  int
}

// New creates a sidebar model.
func New(ctx *tuictx.ProgramContext) Model {
	return Model{
		IsOpen: true,
		ctx:    ctx,
		width:  40,
		height: 20,
	}
}

// SetContent sets the sidebar content (plain text or pre-rendered).
func (m *Model) SetContent(text string) {
	m.content = text
	m.lines = strings.Split(text, "\n")
	m.topLine = 0
}

// ScrollDown scrolls down by n lines.
func (m *Model) ScrollDown(n int) {
	maxTop := len(m.lines) - m.height
	if maxTop < 0 {
		maxTop = 0
	}
	m.topLine += n
	if m.topLine > maxTop {
		m.topLine = maxTop
	}
}

// ScrollUp scrolls up by n lines.
func (m *Model) ScrollUp(n int) {
	m.topLine -= n
	if m.topLine < 0 {
		m.topLine = 0
	}
}

// PageDown scrolls down by one page.
func (m *Model) PageDown() {
	m.ScrollDown(m.height)
}

// PageUp scrolls up by one page.
func (m *Model) PageUp() {
	m.ScrollUp(m.height)
}

// View renders the sidebar.
func (m *Model) View() string {
	if !m.IsOpen || m.width <= 0 {
		return ""
	}

	if len(m.lines) == 0 {
		empty := m.ctx.Styles.Section.EmptyState.Render("No content")
		return m.ctx.Styles.Sidebar.Border.
			Width(m.width).
			Height(m.height).
			Render(empty)
	}

	end := m.topLine + m.height
	if end > len(m.lines) {
		end = len(m.lines)
	}

	visible := m.lines[m.topLine:end]
	var sb strings.Builder
	for i, line := range visible {
		if i > 0 {
			sb.WriteString("\n")
		}
		runes := []rune(line)
		maxW := m.width - 4
		if maxW < 1 {
			maxW = 1
		}
		if len(runes) > maxW {
			runes = runes[:maxW]
		}
		sb.WriteString(string(runes))
	}

	body := m.ctx.Styles.Sidebar.Content.Render(sb.String())

	scrollInfo := m.ctx.Styles.Sidebar.Pager.Render(
		fmt.Sprintf(" %d%% ", m.scrollPercent()),
	)

	contentWithScroll := lipgloss.JoinVertical(lipgloss.Left, body, scrollInfo)

	return m.ctx.Styles.Sidebar.Border.
		Width(m.width).
		Height(m.height).
		Render(contentWithScroll)
}

// SetDimensions updates the sidebar size.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
}

// UpdateProgramContext updates the shared context.
func (m *Model) UpdateProgramContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

func (m *Model) scrollPercent() int {
	if len(m.lines) <= m.height {
		return 100
	}
	return m.topLine * 100 / (len(m.lines) - m.height)
}

// Toggle toggles the sidebar open/closed.
func (m *Model) Toggle() {
	m.IsOpen = !m.IsOpen
}

// ContentLength returns the number of content lines.
func (m *Model) ContentLength() int {
	return len(m.lines)
}
