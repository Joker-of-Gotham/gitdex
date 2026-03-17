package footer

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/context"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/keys"
)

// Model manages the bottom status bar and help display.
// Aligned with gh-dash's footer.Model pattern.
type Model struct {
	ctx     *context.ProgramContext
	ShowAll bool
	width   int
}

// New creates a footer model.
func New() Model {
	return Model{}
}

// SetDimensions sets the footer width.
func (m *Model) SetDimensions(w int) {
	m.width = w
}

// UpdateProgramContext stores the program context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
}

// ToggleHelp toggles the full help display.
func (m *Model) ToggleHelp() {
	m.ShowAll = !m.ShowAll
}

// View renders the footer bar.
func (m Model) View() string {
	var keyStyle, descStyle, containerStyle lipgloss.Style
	if m.ctx != nil {
		keyStyle = m.ctx.Styles.Footer.HelpKeyStyle
		descStyle = m.ctx.Styles.Footer.HelpDescStyle
		containerStyle = m.ctx.Styles.Footer.ContainerStyle
	} else {
		keyStyle = lipgloss.NewStyle().Bold(true)
		descStyle = lipgloss.NewStyle()
		containerStyle = lipgloss.NewStyle()
	}

	viewSwitcher := m.renderViewSwitcher()

	var helpParts []string
	if m.ShowAll {
		for _, group := range keys.Keys.FullHelp() {
			for _, b := range group {
				if b.Enabled() {
					helpParts = append(helpParts,
						keyStyle.Render(b.Help)+" "+descStyle.Render(b.Desc))
				}
			}
		}
	} else {
		helpParts = append(helpParts, keyStyle.Render("?")+" "+descStyle.Render("help"))
		helpParts = append(helpParts, keyStyle.Render("q")+" "+descStyle.Render("quit"))
	}

	helpStr := strings.Join(helpParts, "  ")
	spacer := ""
	usedWidth := lipgloss.Width(viewSwitcher) + lipgloss.Width(helpStr) + 4
	if m.width > usedWidth {
		spacer = strings.Repeat(" ", m.width-usedWidth)
	}

	line := viewSwitcher + spacer + helpStr
	return containerStyle.Width(m.width).Render(line)
}

func (m Model) renderViewSwitcher() string {
	type viewTab struct {
		label string
		key   string
	}
	tabs := []viewTab{
		{"◆ Maintain", "maintain"},
		{"◎ Goal", "goal"},
		{"✦ Creative", "creative"},
	}
	currentView := ""
	if m.ctx != nil {
		currentView = m.ctx.View.String()
	}

	var parts []string
	for _, tab := range tabs {
		if tab.key == currentView {
			style := lipgloss.NewStyle().Bold(true)
			if m.ctx != nil {
				style = style.
					Foreground(m.ctx.Styles.Footer.ViewSwitcherStyle.GetForeground()).
					Background(m.ctx.Styles.Footer.ViewSwitcherStyle.GetForeground()).
					Foreground(lipgloss.Color("#1A1B26")).
					Padding(0, 1)
			}
			parts = append(parts, style.Render(tab.label))
		} else {
			style := lipgloss.NewStyle()
			if m.ctx != nil {
				style = m.ctx.Styles.Footer.HelpDescStyle.Padding(0, 1)
			}
			parts = append(parts, style.Render(tab.label))
		}
	}
	return strings.Join(parts, "")
}
