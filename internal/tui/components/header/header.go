package header

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Model manages the top status bar.
type Model struct {
	ctx        *context.ProgramContext
	width      int
	mode       string
	flow       string
	tokenUsed  int
	tokenMax   int
	branch     string
	repoClean  bool
}

// New creates a header model.
func New() Model {
	return Model{}
}

// SetDimensions sets the header width.
func (m *Model) SetDimensions(w int) {
	m.width = w
}

// UpdateProgramContext stores the program context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
}

// SetState updates the header display state.
func (m *Model) SetState(mode, flow, branch string, repoClean bool, tokenUsed, tokenMax int) {
	m.mode = mode
	m.flow = flow
	m.branch = branch
	m.repoClean = repoClean
	m.tokenUsed = tokenUsed
	m.tokenMax = tokenMax
}

// View renders the header bar.
func (m Model) View() string {
	var modeStyle, flowStyle, ctxStyle, containerStyle lipgloss.Style
	if m.ctx != nil {
		modeStyle = m.ctx.Styles.Header.ModeStyle
		flowStyle = m.ctx.Styles.Header.FlowStyle
		ctxStyle = m.ctx.Styles.Header.ContextStyle
		containerStyle = m.ctx.Styles.Header.ContainerStyle
	} else {
		modeStyle = lipgloss.NewStyle().Bold(true)
		flowStyle = lipgloss.NewStyle()
		ctxStyle = lipgloss.NewStyle()
		containerStyle = lipgloss.NewStyle()
	}

	modePill := modeStyle.Render(fmt.Sprintf(" %s ", strings.ToUpper(m.mode)))
	flowPill := flowStyle.Render(m.flow)

	branchStr := ctxStyle.Render(fmt.Sprintf("⎇ %s", m.branch))
	cleanStr := ""
	if m.repoClean {
		cleanStr = ctxStyle.Render("✓ clean")
	}

	tokenStr := ""
	if m.tokenMax > 0 {
		tokenStr = ctxStyle.Render(fmt.Sprintf("[%d/%d tok]", m.tokenUsed, m.tokenMax))
	}

	left := modePill + " " + flowPill
	right := strings.Join(filterEmpty(branchStr, cleanStr, tokenStr), "  ")

	spacer := ""
	usedW := lipgloss.Width(left) + lipgloss.Width(right) + 4
	if m.width > usedW {
		spacer = strings.Repeat(" ", m.width-usedW)
	}

	return containerStyle.Width(m.width).Render(left + spacer + right)
}

func filterEmpty(parts ...string) []string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
