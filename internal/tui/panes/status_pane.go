package panes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/keymap"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type StatusPane struct {
	summary  *repo.RepoSummary
	styles   theme.Styles
	listKeys keymap.ListKeys
	width    int
	height   int
	focused  bool
	cursor   int
}

type StatusUpdateMsg struct {
	Summary *repo.RepoSummary
}

func NewStatusPane(styles theme.Styles) StatusPane {
	return StatusPane{
		styles:   styles,
		listKeys: keymap.DefaultListKeys(),
	}
}

func (p StatusPane) Init() tea.Cmd {
	return nil
}

func (p StatusPane) Update(msg tea.Msg) (StatusPane, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusUpdateMsg:
		p.summary = msg.Summary
	case tea.KeyPressMsg:
		if !p.focused {
			break
		}
		switch {
		case p.listKeys.Up.Matches(msg):
			if p.cursor > 0 {
				p.cursor--
			}
		case p.listKeys.Down.Matches(msg):
			if p.cursor < 4 {
				p.cursor++
			}
		}
	}
	return p, nil
}

func (p StatusPane) View() string {
	var b strings.Builder

	title := p.styles.PanelTitle.Render("Repository Status")
	b.WriteString(title)
	b.WriteString("\n\n")

	if p.summary == nil {
		b.WriteString(p.styles.Annotation.Render("No repository data loaded"))
		return p.applyBorder(b.String())
	}

	overall := theme.RenderStateLabel(p.styles, string(p.summary.OverallLabel))
	b.WriteString(fmt.Sprintf("%s/%s  %s\n\n", p.summary.Owner, p.summary.Repo, overall))

	dimensions := []struct {
		name  string
		label repo.StateLabel
	}{
		{"Local", p.summary.Local.Label},
		{"Remote", p.summary.Remote.Label},
		{"Collaboration", p.summary.Collaboration.Label},
		{"Workflows", p.summary.Workflows.Label},
		{"Deployments", p.summary.Deployments.Label},
	}

	for i, d := range dimensions {
		prefix := "  "
		if p.focused && i == p.cursor {
			prefix = "> "
		}
		rendered := theme.RenderStateLabel(p.styles, string(d.label))
		line := fmt.Sprintf("%s%-15s %s", prefix, d.name, rendered)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return p.applyBorder(b.String())
}

func (p *StatusPane) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *StatusPane) SetFocused(focused bool) {
	p.focused = focused
}

func (p StatusPane) Focused() bool {
	return p.focused
}

func (p StatusPane) applyBorder(content string) string {
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

func (p StatusPane) Summary() *repo.RepoSummary {
	return p.summary
}

func RenderDimensionTable(s *repo.RepoSummary, styles theme.Styles) string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	rows := []struct {
		name   string
		label  string
		detail string
	}{
		{"Local", string(s.Local.Label), s.Local.Detail},
		{"Remote", string(s.Remote.Label), s.Remote.Detail},
		{"Collaboration", string(s.Collaboration.Label), s.Collaboration.Detail},
		{"Workflows", string(s.Workflows.Label), s.Workflows.Detail},
		{"Deployments", string(s.Deployments.Label), s.Deployments.Detail},
	}
	header := fmt.Sprintf("  %-15s %-12s %s\n", "Dimension", "Status", "Detail")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	b.WriteString(strings.Repeat("─", 60) + "\n")
	for _, r := range rows {
		rendered := theme.RenderStateLabel(styles, r.label)
		detail := r.detail
		if detail == "" {
			detail = "-"
		}
		b.WriteString(fmt.Sprintf("  %-15s %s  %s\n", r.name, rendered, detail))
	}
	return b.String()
}
