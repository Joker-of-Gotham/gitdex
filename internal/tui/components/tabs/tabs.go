package tabs

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/components/carousel"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Model manages tab-based section switching using a carousel.
// Aligned with gh-dash's tabs.Model.
type Model struct {
	carousel carousel.Model
	ctx      *context.ProgramContext
}

// New creates a tabs model with the given tab labels.
func New(labels []string) Model {
	return Model{
		carousel: carousel.New(labels),
	}
}

// UpdateProgramContext stores the program context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
}

// Next switches to the next tab.
func (m *Model) Next() { m.carousel.Next() }

// Prev switches to the previous tab.
func (m *Model) Prev() { m.carousel.Prev() }

// CurrentIdx returns the currently active tab index.
func (m Model) CurrentIdx() int { return m.carousel.CurrentIdx() }

// SetIdx sets the active tab by index.
func (m *Model) SetIdx(idx int) { m.carousel.SetIdx(idx) }

// View renders the tab bar.
func (m Model) View(width int) string {
	items := m.carousel.Items()
	if len(items) == 0 {
		return ""
	}

	var activeStyle, inactiveStyle, gapStyle lipgloss.Style
	if m.ctx != nil {
		activeStyle = m.ctx.Styles.Tabs.ActiveTab
		inactiveStyle = m.ctx.Styles.Tabs.InactiveTab
		gapStyle = m.ctx.Styles.Tabs.TabGap
	} else {
		activeStyle = lipgloss.NewStyle().Bold(true).Padding(0, 2)
		inactiveStyle = lipgloss.NewStyle().Padding(0, 2)
		gapStyle = lipgloss.NewStyle()
	}

	curr := m.carousel.CurrentIdx()
	var parts []string
	for i, label := range items {
		if i == curr {
			parts = append(parts, activeStyle.Render(label))
		} else {
			parts = append(parts, inactiveStyle.Render(label))
		}
		if i < len(items)-1 {
			parts = append(parts, gapStyle.Render("│"))
		}
	}

	bar := strings.Join(parts, "")
	return lipgloss.NewStyle().Width(width).Render(bar)
}
