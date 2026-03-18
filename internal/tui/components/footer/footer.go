// Package footer provides a dynamic help bar component.
// Inspired by gh-dash's components/footer.
package footer

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Binding describes a single key binding for display.
type Binding struct {
	Key  string
	Desc string
}

// Model is the footer component.
type Model struct {
	ctx      *tuictx.ProgramContext
	bindings []Binding
	width    int
	showAll  bool
}

// New creates a footer model.
func New(ctx *tuictx.ProgramContext) Model {
	return Model{
		ctx: ctx,
	}
}

// SetBindings updates the displayed key bindings.
func (m *Model) SetBindings(bindings []Binding) {
	m.bindings = bindings
}

// SetWidth sets the available width.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// ToggleHelp toggles full help display.
func (m *Model) ToggleHelp() {
	m.showAll = !m.showAll
}

// View renders the footer.
func (m *Model) View() string {
	modeLabel := m.ctx.Styles.Footer.ViewSwitch.Render(m.ctx.Mode)

	ctxLabel := ""
	if m.ctx.ContextMax > 0 {
		ctxLabel = m.ctx.Styles.Footer.Help.Render(
			fmt.Sprintf("ctx[%dk/%dk %d%%]",
				m.ctx.ContextUsed/1000,
				m.ctx.ContextMax/1000,
				m.ctx.ContextPercent(),
			),
		)
	}

	bindings := m.renderBindings()

	spacerWidth := m.width - lipgloss.Width(modeLabel) -
		lipgloss.Width(ctxLabel) - lipgloss.Width(bindings) - 6
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	spacer := strings.Repeat(" ", spacerWidth)

	return m.ctx.Styles.Footer.Container.
		Width(m.width).
		Render(
			lipgloss.JoinHorizontal(lipgloss.Top,
				modeLabel, " ", ctxLabel, spacer, bindings,
			),
		)
}

func (m *Model) renderBindings() string {
	limit := 6
	if m.showAll {
		limit = len(m.bindings)
	}

	var parts []string
	for i, b := range m.bindings {
		if i >= limit {
			parts = append(parts, m.ctx.Styles.Footer.Help.Render("?:more"))
			break
		}
		key := m.ctx.Styles.Footer.KeyBinding.Render(b.Key)
		desc := m.ctx.Styles.Footer.Description.Render(b.Desc)
		parts = append(parts, key+":"+desc)
	}
	return strings.Join(parts, " ")
}

// UpdateProgramContext updates the shared context.
func (m *Model) UpdateProgramContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}
