package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type PlanSummary struct {
	Title          string
	Status         string
	Scope          string
	StepCount      int
	CompletedSteps int
	RiskLevel      string
	// PlanID is set when plans are loaded from persistent stores (for task correlation).
	PlanID string
	// StepLines lists human-readable steps and optional linked tasks for the drill view.
	StepLines []string
}

type PlansView struct {
	plans     []PlanSummary
	selected  int
	width     int
	height    int
	t         *theme.Theme
	drillOpen bool
	drillVP   viewport.Model
	drillInit bool
}

func NewPlansView(t *theme.Theme) *PlansView { return &PlansView{t: t} }

func (v *PlansView) ID() ID        { return ViewPlans }
func (v *PlansView) Title() string { return "Plans" }
func (v *PlansView) Init() tea.Cmd { return nil }

func (v *PlansView) Update(msg tea.Msg) (View, tea.Cmd) {
	if v.drillOpen {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			switch km.String() {
			case "esc", "enter":
				v.drillOpen = false
				return v, nil
			}
		}
		var cmd tea.Cmd
		v.drillVP, cmd = v.drillVP.Update(msg)
		return v, cmd
	}

	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch km.String() {
		case "enter":
			if len(v.plans) > 0 {
				v.drillOpen = !v.drillOpen
				if v.drillOpen {
					v.syncDrillContent()
				}
			}
			return v, nil
		case "up", "k":
			if v.selected > 0 {
				v.selected--
				if v.drillOpen {
					v.syncDrillContent()
				}
			}
		case "down", "j":
			if len(v.plans) > 0 && v.selected < len(v.plans)-1 {
				v.selected++
				if v.drillOpen {
					v.syncDrillContent()
				}
			}
		case "pgup":
			step := maxInt(1, v.listVisibleCount()/2)
			v.selected -= step
			if v.selected < 0 {
				v.selected = 0
			}
			if v.drillOpen {
				v.syncDrillContent()
			}
		case "pgdown":
			step := maxInt(1, v.listVisibleCount()/2)
			v.selected += step
			if v.selected >= len(v.plans) {
				v.selected = maxInt(0, len(v.plans)-1)
			}
			if v.drillOpen {
				v.syncDrillContent()
			}
		}
	}
	return v, nil
}

func (v *PlansView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.width >= 96 {
		detailW := maxInt(30, v.width-maxInt(34, v.width*48/100)-1)
		vpH := maxInt(6, v.height-6)
		v.initDrillVP(detailW, vpH)
	} else {
		vpH := maxInt(6, v.height-8)
		v.initDrillVP(maxInt(24, v.width-4), vpH)
	}
}

func (v *PlansView) initDrillVP(w, h int) {
	if !v.drillInit {
		v.drillVP = viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
		v.drillInit = true
	} else {
		v.drillVP.SetWidth(w)
		v.drillVP.SetHeight(h)
	}
	if v.drillOpen {
		v.syncDrillContent()
	}
}

func (v *PlansView) syncDrillContent() {
	if len(v.plans) == 0 || v.selected >= len(v.plans) {
		v.drillVP.SetContent("")
		return
	}
	lines := v.plans[v.selected].StepLines
	if len(lines) == 0 {
		lines = []string{"(No steps listed yet. Refresh workspace data to populate.)"}
	}
	v.drillVP.SetContent(strings.Join(lines, "\n"))
	v.drillVP.GotoTop()
}

func (v *PlansView) SetPlans(plans []PlanSummary) {
	v.plans = plans
	if v.selected >= len(v.plans) && len(v.plans) > 0 {
		v.selected = len(v.plans) - 1
	}
	if v.drillOpen {
		v.syncDrillContent()
	}
}

func (v *PlansView) Render() string {
	if v.width <= 0 || v.height <= 0 {
		return ""
	}

	if len(v.plans) == 0 {
		body := strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Plan Queue"),
			"",
			lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No active execution plan."),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(24, v.width-8)).
				Render("Select a repository to derive a live stabilization plan from repository signals, risk posture, and pending workflows."),
		}, "\n")
		return render.SurfacePanel(body, maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
	}

	if v.width >= 96 {
		listWidth := maxInt(34, v.width*48/100)
		detailWidth := maxInt(30, v.width-listWidth-1)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			v.renderPlanList(listWidth),
			" ",
			v.renderPlanDetail(detailWidth),
		)
	}

	parts := []string{v.renderPlanList(v.width)}
	if detail := v.renderPlanDetail(v.width); detail != "" {
		parts = append(parts, detail)
	}
	return strings.Join(parts, "\n\n")
}

func (v *PlansView) listVisibleCount() int {
	const linesPerCard = 7
	h := v.height - 4
	if h < linesPerCard {
		h = linesPerCard
	}
	return maxInt(1, h/linesPerCard)
}

func (v *PlansView) renderPlanList(width int) string {
	rows := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("Execution Plans (%d)", len(v.plans))),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Up/Down select  Enter drill  PgUp/PgDn page"),
		"",
	}

	vis := v.listVisibleCount()
	start := 0
	if v.selected >= vis {
		start = v.selected - vis + 1
	}
	end := start + vis
	if end > len(v.plans) {
		end = len(v.plans)
	}

	for i := start; i < end; i++ {
		plan := v.plans[i]
		progress := 0
		if plan.StepCount > 0 {
			progress = plan.CompletedSteps * 100 / plan.StepCount
		}

		cardLines := []string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(plan.Title),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(18, width-8)).Render(valueOrDash(plan.Scope)),
			lipgloss.NewStyle().Foreground(v.t.DimText()).Render(fmt.Sprintf("%s  |  %d/%d steps  |  risk %s", strings.ToUpper(plan.Status), plan.CompletedSteps, plan.StepCount, strings.ToUpper(valueOrDash(plan.RiskLevel)))),
			v.progressBar(maxInt(18, width-12), float64(progress)/100),
		}

		cardBody := strings.Join(cardLines, "\n")
		if i == v.selected {
			panelFrame := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).GetHorizontalFrameSize()
			cardBody = render.FillBlock(cardBody, maxInt(12, width-panelFrame), lipgloss.NewStyle().Background(v.t.Selection()))
		}
		rows = append(rows, render.SurfacePanel(cardBody, width, v.t.Surface(), v.t.BorderColor()))
	}

	return strings.Join(rows, "\n")
}

func (v *PlansView) renderPlanDetail(width int) string {
	if len(v.plans) == 0 || v.selected >= len(v.plans) {
		return ""
	}

	if v.drillOpen {
		header := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Steps & tasks")
		hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Esc back  scroll with arrows / PgUp / PgDn / mouse")
		vp := render.SurfacePanel(v.drillVP.View(), width, v.t.Surface(), v.t.BorderColor())
		return strings.Join([]string{header, hint, vp}, "\n")
	}

	plan := v.plans[v.selected]
	progress := 0
	if plan.StepCount > 0 {
		progress = plan.CompletedSteps * 100 / plan.StepCount
	}

	body := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Selected Plan"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(plan.Title),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(20, width-8)).Render(valueOrDash(plan.Scope)),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Status"),
		lipgloss.NewStyle().Foreground(v.t.Fg()).Render(strings.ToUpper(valueOrDash(plan.Status))),
		"",
		"Completion",
		v.progressBar(maxInt(18, width-8), float64(progress)/100),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render(fmt.Sprintf("Steps: %d total / %d complete", plan.StepCount, plan.CompletedSteps)),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Risk"),
		lipgloss.NewStyle().Foreground(v.t.Fg()).Render(strings.ToUpper(valueOrDash(plan.RiskLevel))),
		"",
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(20, width-8)).
			Render("Plans are execution scaffolds derived from current repository state. They should collapse signal noise into a short actionable sequence, not restate the dashboard."),
	}
	return render.SurfacePanel(strings.Join(body, "\n"), width, v.t.Surface(), v.t.BorderColor())
}

func (v *PlansView) progressBar(width int, percent float64) string {
	if width < 8 {
		width = 8
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	barWidth := width - 7
	if barWidth < 6 {
		barWidth = 6
	}
	filled := int(percent * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled
	return lipgloss.NewStyle().Foreground(v.t.BorderColor()).Render("[") +
		lipgloss.NewStyle().Background(v.t.Primary()).Render(strings.Repeat(" ", filled)) +
		lipgloss.NewStyle().Background(v.t.Elevated()).Render(strings.Repeat(" ", empty)) +
		lipgloss.NewStyle().Foreground(v.t.BorderColor()).Render("]") +
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(fmt.Sprintf(" %d%%", int(percent*100)))
}
