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

type CockpitView struct {
	summary *repo.RepoSummary
	width   int
	height  int
	t       *theme.Theme
	scroll  int
}

func NewCockpitView(t *theme.Theme) *CockpitView {
	return &CockpitView{t: t}
}

func (v *CockpitView) ID() ID        { return ViewCockpit }
func (v *CockpitView) Title() string { return "Cockpit" }
func (v *CockpitView) Init() tea.Cmd { return nil }

func (v *CockpitView) Update(msg tea.Msg) (View, tea.Cmd) {
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

func (v *CockpitView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

func (v *CockpitView) SetSummary(s *repo.RepoSummary) {
	v.summary = s
}

func (v *CockpitView) Render() string {
	if v.summary == nil {
		return v.renderEmptyState()
	}

	content := []string{
		v.renderHeader(),
		v.renderMetricDeck(),
		v.renderHealthBoard(),
		v.renderSignalBoards(),
	}

	lines := strings.Split(strings.Join(content, "\n\n"), "\n")
	if v.scroll > len(lines)-v.height {
		v.scroll = len(lines) - v.height
	}
	if v.scroll < 0 {
		v.scroll = 0
	}
	end := min(len(lines), v.scroll+v.height)
	return strings.Join(lines[v.scroll:end], "\n")
}

func (v *CockpitView) renderEmptyState() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(v.t.Primary()).
		Render(theme.Icons.Dashboard + " Gitdex Cockpit")
	body := strings.Join([]string{
		"Run `gitdex scan` to populate the first repository summary.",
		"Wide layouts expose navigation and inspector panes automatically.",
		"Use Ctrl+P to jump between workspaces and theme variants.",
	}, "\n")
	body = lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(body)
	return render.SurfacePanel(title+"\n\n"+body, max(40, v.width), v.t.Surface(), v.t.BorderColor())
}

func (v *CockpitView) renderHeader() string {
	repoName := fmt.Sprintf("%s/%s", v.summary.Owner, v.summary.Repo)
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(theme.Icons.Branch + " " + repoName)
	subtitle := lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(v.summary.Timestamp.Format("2006-01-02 15:04"))
	status := v.stateBadge(v.summary.OverallLabel)

	branch := v.summary.Local.Branch
	if branch == "" {
		branch = v.summary.Remote.DefaultBranch
	}

	branchChip := lipgloss.NewStyle().
		Foreground(v.t.Secondary()).
		Background(v.t.Surface()).
		Padding(0, 1).
		Render(theme.Icons.Commit + " " + valueOrDash(branch))

	return strings.Join([]string{
		title + "  " + status + "  " + branchChip,
		subtitle,
	}, "\n")
}

func (v *CockpitView) renderMetricDeck() string {
	cols := 1
	switch {
	case v.width >= 132:
		cols = 4
	case v.width >= 92:
		cols = 2
	}

	cardWidth := v.gridCellWidth(cols)
	cards := []string{
		v.metricCard("Pull Requests", fmt.Sprintf("%d", v.summary.Collaboration.OpenPRCount), v.summary.Collaboration.Detail, theme.Icons.PullRequest, cardWidth),
		v.metricCard("Issues", fmt.Sprintf("%d", v.summary.Collaboration.OpenIssueCount), "Open work items", theme.Icons.Issue, cardWidth),
		v.metricCard("Workflow Runs", fmt.Sprintf("%d", len(v.summary.Workflows.Runs)), v.summary.Workflows.Detail, theme.Icons.Task, cardWidth),
		v.metricCard("Deployments", fmt.Sprintf("%d", len(v.summary.Deployments.Deployments)), v.summary.Deployments.Detail, theme.Icons.Rocket, cardWidth),
	}
	return v.renderGrid(cards, cols)
}

func (v *CockpitView) renderHealthBoard() string {
	panelWidth := max(40, v.width)
	rowWidth := panelWidth - 4
	rows := []string{
		v.signalRow(rowWidth, "Local", v.summary.Local.Label, v.summary.Local.Detail),
		v.signalRow(rowWidth, "Remote", v.summary.Remote.Label, v.summary.Remote.Detail),
		v.signalRow(rowWidth, "Collaboration", v.summary.Collaboration.Label, v.summary.Collaboration.Detail),
		v.signalRow(rowWidth, "Workflows", v.summary.Workflows.Label, v.summary.Workflows.Detail),
		v.signalRow(rowWidth, "Deployments", v.summary.Deployments.Label, v.summary.Deployments.Detail),
	}
	return v.card("Health Matrix", rows, panelWidth)
}

func (v *CockpitView) renderSignalBoards() string {
	risks := make([]string, 0, max(1, len(v.summary.Risks)))
	if len(v.summary.Risks) == 0 {
		risks = append(risks, lipgloss.NewStyle().Foreground(v.t.Success()).Render(theme.Icons.Check+" No material risks detected"))
	}
	for _, risk := range v.summary.Risks {
		risks = append(risks, v.riskLine(risk))
	}

	actions := make([]string, 0, max(1, len(v.summary.NextActions)*2))
	if len(v.summary.NextActions) == 0 {
		actions = append(actions, lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render("No follow-up actions queued."))
	}
	for _, action := range v.summary.NextActions {
		actions = append(actions,
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(theme.Icons.ChevronRight+" "+action.Action),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(action.Reason),
		)
	}

	if v.width < 104 {
		return strings.Join([]string{
			v.card("Risk Watchlist", risks, max(40, v.width)),
			v.card("Next Actions", actions, max(40, v.width)),
		}, "\n\n")
	}

	leftWidth := (v.width - 1) / 2
	rightWidth := v.width - leftWidth - 1
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		v.card("Risk Watchlist", risks, leftWidth),
		" ",
		v.card("Next Actions", actions, rightWidth),
	)
}

func (v *CockpitView) metricCard(title, value, subtitle, icon string, width int) string {
	lines := []string{
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(icon + " " + title),
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(value),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Width(max(18, width-4)).Render(valueOrDash(subtitle)),
	}
	return render.SurfacePanel(strings.Join(lines, "\n"), max(24, width), v.t.Surface(), v.t.BorderColor())
}

func (v *CockpitView) signalRow(width int, title string, label repo.StateLabel, detail string) string {
	labelWidth := clamp(width/5, 12, 16)
	stateWidth := 12
	barWidth := clamp(width/4, 12, 24)
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(labelWidth).Bold(true).Foreground(v.t.Fg()).Render(title),
		" ",
		lipgloss.NewStyle().Width(stateWidth).Render(v.stateBadge(label)),
		" ",
		lipgloss.NewStyle().Width(barWidth).Render(v.progressBar(label, barWidth)),
	)
	detailLine := lipgloss.NewStyle().
		Foreground(v.t.MutedFg()).
		PaddingLeft(2).
		Width(max(16, width-2)).
		Render(valueOrDash(detail))
	return header + "\n" + detailLine
}

func (v *CockpitView) riskLine(r repo.Risk) string {
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

	return strings.Join([]string{
		lipgloss.NewStyle().Foreground(color).Render(icon + " " + strings.ToUpper(string(r.Severity))),
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(r.Description),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(valueOrDash(r.Action)),
	}, "\n")
}

func (v *CockpitView) card(title string, lines []string, width int) string {
	body := append([]string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(title),
	}, lines...)
	return render.SurfacePanel(strings.Join(body, "\n"), max(24, width), v.t.Surface(), v.t.BorderColor())
}

func (v *CockpitView) stateBadge(label repo.StateLabel) string {
	token := theme.TokenForState(string(label))
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(token.Color).
		Render(token.Icon + " " + strings.ToUpper(string(label)))
}

func (v *CockpitView) progressBar(label repo.StateLabel, width int) string {
	ratio := map[repo.StateLabel]float64{
		repo.Healthy:  1,
		repo.Unknown:  0.45,
		repo.Drifting: 0.6,
		repo.Degraded: 0.3,
		repo.Blocked:  0.1,
	}[label]

	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	full := lipgloss.NewStyle().Background(v.t.Secondary()).Render(strings.Repeat(" ", filled))
	rest := lipgloss.NewStyle().Background(v.t.Elevated()).Render(strings.Repeat(" ", max(0, width-filled)))
	return full + rest
}

func (v *CockpitView) renderGrid(cards []string, cols int) string {
	if cols <= 1 || len(cards) <= 1 {
		return strings.Join(cards, "\n")
	}

	rows := make([]string, 0, (len(cards)+cols-1)/cols)
	for start := 0; start < len(cards); start += cols {
		end := min(len(cards), start+cols)
		chunk := cards[start:end]
		parts := make([]string, 0, len(chunk)*2)
		for i, card := range chunk {
			parts = append(parts, card)
			if i < len(chunk)-1 {
				parts = append(parts, " ")
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}
	return strings.Join(rows, "\n")
}

func (v *CockpitView) gridCellWidth(cols int) int {
	if cols <= 1 {
		return max(24, v.width)
	}
	return max(18, (v.width-(cols-1))/cols)
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
