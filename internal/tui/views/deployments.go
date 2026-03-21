package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type DeploymentsView struct {
	t           *theme.Theme
	deployments []DeploymentEntry
	cursor      int
	width       int
	height      int
	detail      bool
	statusLine  string
}

func NewDeploymentsView(t *theme.Theme) *DeploymentsView {
	return &DeploymentsView{t: t}
}

func (v *DeploymentsView) ID() ID        { return "deployments" }
func (v *DeploymentsView) Title() string { return "Deployments" }
func (v *DeploymentsView) Init() tea.Cmd { return nil }

func (v *DeploymentsView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *DeploymentsView) SetDeployments(items []DeploymentEntry) {
	v.deployments = items
	if v.cursor >= len(items) {
		v.cursor = max(0, len(items)-1)
	}
}

func (v *DeploymentsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case DeploymentDataMsg:
		v.SetDeployments(msg.Deployments)
		return v, nil
	case tea.KeyPressMsg:
		prev := v.cursor
		switch msg.String() {
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			if v.cursor < len(v.deployments)-1 {
				v.cursor++
			}
		case "g":
			v.cursor = 0
		case "G":
			if len(v.deployments) > 0 {
				v.cursor = len(v.deployments) - 1
			}
		case "enter":
			item := v.selected()
			v.detail = !v.detail
			if item != nil {
				return v, func() tea.Msg { return DeploymentSelectedMsg{Deployment: *item} }
			}
		case "esc":
			if v.detail {
				v.detail = false
				return v, nil
			}
		case "pgup":
			visible := maxInt(1, v.height-6)
			step := maxInt(1, visible/2)
			v.cursor -= step
			if v.cursor < 0 {
				v.cursor = 0
			}
		case "pgdown":
			visible := maxInt(1, v.height-6)
			step := maxInt(1, visible/2)
			v.cursor += step
			if v.cursor >= len(v.deployments) {
				v.cursor = maxInt(0, len(v.deployments)-1)
			}
		}
		if v.cursor != prev && v.detail {
			if item := v.selected(); item != nil {
				return v, func() tea.Msg { return DeploymentSelectedMsg{Deployment: *item} }
			}
		}
	}
	return v, nil
}

func (v *DeploymentsView) Render() string {
	if len(v.deployments) == 0 {
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  No deployments loaded")
	}
	return v.renderList(v.width)
}

func (v *DeploymentsView) renderList(width int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("  Deployments (%d)", len(v.deployments)))
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  Up/Down navigate  Enter inspect  Esc close detail")
	header := lipgloss.NewStyle().Bold(true).Foreground(v.t.Info()).Render("  ID       Environment              State           Ref")
	lines := []string{title, hint}
	if v.statusLine != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusLine))
	}
	lines = append(lines, "", header)

	visible := max(1, v.height-6)
	start := 0
	if v.cursor >= visible {
		start = v.cursor - visible + 1
	}
	end := start + visible
	if end > len(v.deployments) {
		end = len(v.deployments)
	}

	for i := start; i < end; i++ {
		item := v.deployments[i]
		line := fmt.Sprintf("  %-8d %-24s %-14s %s", item.ID, truncate(item.Environment, 24), truncate(strings.ToUpper(item.State), 14), truncate(item.Ref, 18))
		if i == v.cursor {
			line = render.FillBlock(line, max(20, v.width-2), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection()))
		}
		lines = append(lines, line)
	}
	if v.detail {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  Inspector detail active. Use Ctrl+3 or Ctrl+I to review the selected deployment."))
	}
	return strings.Join(lines, "\n")
}

func (v *DeploymentsView) selected() *DeploymentEntry {
	if v.cursor >= 0 && v.cursor < len(v.deployments) {
		return &v.deployments[v.cursor]
	}
	return nil
}
