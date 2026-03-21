package panes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/keymap"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type RiskPane struct {
	summary  *repo.RepoSummary
	styles   theme.Styles
	listKeys keymap.ListKeys
	width    int
	height   int
	focused  bool
	cursor   int
}

func NewRiskPane(styles theme.Styles) RiskPane {
	return RiskPane{
		styles:   styles,
		listKeys: keymap.DefaultListKeys(),
	}
}

func (p RiskPane) Init() tea.Cmd {
	return nil
}

func (p RiskPane) Update(msg tea.Msg) (RiskPane, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusUpdateMsg:
		p.summary = msg.Summary
		p.cursor = 0
	case tea.KeyPressMsg:
		if !p.focused {
			break
		}
		maxIdx := p.maxItems() - 1
		if maxIdx < 0 {
			maxIdx = 0
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
		}
	}
	return p, nil
}

func (p RiskPane) View() string {
	var b strings.Builder

	title := p.styles.PanelTitle.Render("Risks & Next Actions")
	b.WriteString(title)
	b.WriteString("\n\n")

	if p.summary == nil {
		b.WriteString(p.styles.Annotation.Render("No data"))
		return p.applyBorder(b.String())
	}

	if len(p.summary.Risks) == 0 && len(p.summary.NextActions) == 0 {
		b.WriteString(p.styles.StatusHealthy.Render("✓ No material risks detected"))
		return p.applyBorder(b.String())
	}

	idx := 0
	if len(p.summary.Risks) > 0 {
		b.WriteString(p.styles.ObjectTitle.Render("Risks"))
		b.WriteString("\n")
		for _, r := range p.summary.Risks {
			prefix := "  "
			if p.focused && idx == p.cursor {
				prefix = "> "
			}
			severityStyle := p.styles.StatusStyle(severityToState(string(r.Severity)))
			b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, severityStyle.Render("["+string(r.Severity)+"]"), r.Description))
			idx++
		}
		b.WriteString("\n")
	}

	if len(p.summary.NextActions) > 0 {
		b.WriteString(p.styles.ObjectTitle.Render("Next Actions"))
		b.WriteString("\n")
		for _, a := range p.summary.NextActions {
			prefix := "  "
			if p.focused && idx == p.cursor {
				prefix = "> "
			}
			b.WriteString(fmt.Sprintf("%s→ %s (%s)\n", prefix, a.Action, a.Reason))
			idx++
		}
	}

	return p.applyBorder(b.String())
}

func (p *RiskPane) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *RiskPane) SetFocused(focused bool) {
	p.focused = focused
}

func (p *RiskPane) SetStyles(styles theme.Styles) {
	p.styles = styles
}

func (p RiskPane) Focused() bool {
	return p.focused
}

func (p RiskPane) maxItems() int {
	if p.summary == nil {
		return 0
	}
	return len(p.summary.Risks) + len(p.summary.NextActions)
}

func (p RiskPane) applyBorder(content string) string {
	style := p.styles.NormalBorder
	if p.focused {
		style = p.styles.FocusedBorder
	}
	w := p.width - 2
	if w < 10 {
		w = 10
	}
	return style.Width(w).Render(content)
}

func severityToState(severity string) string {
	switch severity {
	case "high":
		return "blocked"
	case "medium":
		return "drifting"
	case "low":
		return "unknown"
	default:
		return "unknown"
	}
}
