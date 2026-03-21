package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type StatusView struct {
	t       *theme.Theme
	summary *repo.RepoSummary
	width   int
	height  int
	scroll  int
}

type StatusDataMsg struct {
	Summary *repo.RepoSummary
}

func NewStatusView(t *theme.Theme) *StatusView { return &StatusView{t: t} }

func (v *StatusView) ID() ID        { return ViewStatus }
func (v *StatusView) Title() string { return "Status" }
func (v *StatusView) Init() tea.Cmd { return nil }

func (v *StatusView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusDataMsg:
		v.summary = msg.Summary
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if v.scroll > 0 {
				v.scroll--
			}
		case "down", "j":
			v.scroll++
		case "pgup":
			v.scroll -= max(1, v.height/2)
			if v.scroll < 0 {
				v.scroll = 0
			}
		case "pgdown":
			v.scroll += max(1, v.height/2)
		}
	}
	return v, nil
}

func (v *StatusView) Render() string {
	if v.summary == nil {
		body := strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Repository Status"),
			"",
			lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No repository data loaded."),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render("Select a repository in Dashboard > Repos to unlock the status matrix."),
		}, "\n")
		return render.SurfacePanel(body, maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
	}

	sections := []string{
		v.renderDimensionBoard(),
		v.renderRiskBoard(),
		v.renderActionBoard(),
	}
	lines := strings.Split(strings.Join(sections, "\n\n"), "\n")
	if v.scroll > len(lines)-v.height {
		v.scroll = len(lines) - v.height
	}
	if v.scroll < 0 {
		v.scroll = 0
	}
	end := minInt(len(lines), v.scroll+v.height)
	return strings.Join(lines[v.scroll:end], "\n")
}

func (v *StatusView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

func (v *StatusView) SetSummary(s *repo.RepoSummary) { v.summary = s }

func (v *StatusView) renderDimensionBoard() string {
	rows := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Repository Status"),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(fmt.Sprintf("%s/%s", v.summary.Owner, v.summary.Repo)),
		"",
	}

	type dimension struct {
		title  string
		label  repo.StateLabel
		detail string
	}
	dims := []dimension{
		{"Local", v.summary.Local.Label, v.summary.Local.Detail},
		{"Remote", v.summary.Remote.Label, v.summary.Remote.Detail},
		{"Collaboration", v.summary.Collaboration.Label, v.summary.Collaboration.Detail},
		{"Workflows", v.summary.Workflows.Label, v.summary.Workflows.Detail},
		{"Deployments", v.summary.Deployments.Label, v.summary.Deployments.Detail},
	}
	for _, dim := range dims {
		rows = append(rows, v.dimensionRow(dim.title, dim.label, dim.detail))
	}
	return render.SurfacePanel(strings.Join(rows, "\n"), maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
}

func (v *StatusView) dimensionRow(title string, label repo.StateLabel, detail string) string {
	state := v.stateBadge(label)
	width := maxInt(24, v.width-8)
	row := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(title) + "  " + state,
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(width).Render(valueOrDash(detail)),
	}
	return strings.Join(row, "\n")
}

func (v *StatusView) renderRiskBoard() string {
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Risk Register")}
	if len(v.summary.Risks) == 0 {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(v.t.Success()).Render(theme.Icons.Check+" No material risks detected"))
		return render.SurfacePanel(strings.Join(lines, "\n"), maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
	}

	lines = append(lines, "")
	for _, risk := range v.summary.Risks {
		lines = append(lines, v.riskRow(risk))
	}
	return render.SurfacePanel(strings.Join(lines, "\n\n"), maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
}

func (v *StatusView) renderActionBoard() string {
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Action Queue")}
	if len(v.summary.NextActions) == 0 {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render("No follow-up actions queued."))
		return render.SurfacePanel(strings.Join(lines, "\n"), maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
	}

	lines = append(lines, "")
	for _, action := range v.summary.NextActions {
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(theme.Icons.ChevronRight+" "+action.Action),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(20, v.width-8)).Render(action.Reason),
			"",
		)
	}
	return render.SurfacePanel(strings.TrimSpace(strings.Join(lines, "\n")), maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
}

func (v *StatusView) stateBadge(label repo.StateLabel) string {
	token := theme.TokenForState(string(label))
	return lipgloss.NewStyle().Bold(true).Foreground(token.Color).Render(token.Icon + " " + strings.ToUpper(string(label)))
}

func (v *StatusView) riskRow(r repo.Risk) string {
	color := v.t.Warning()
	icon := theme.Icons.Warning
	if r.Severity == repo.RiskHigh {
		color = v.t.Danger()
		icon = theme.Icons.Cross
	}
	if r.Severity == repo.RiskLow {
		color = v.t.Info()
		icon = theme.Icons.Info
	}
	return lipgloss.NewStyle().Foreground(color).Render(icon+" "+strings.ToUpper(string(r.Severity))) +
		"\n" + lipgloss.NewStyle().Foreground(v.t.Fg()).Bold(true).Render(r.Description) +
		"\n" + lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(20, v.width-8)).Render(valueOrDash(r.Action))
}
