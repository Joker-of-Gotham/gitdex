package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/tui/theme"
)

// CruiseStatus aliases autonomy cruise state for this view.
type CruiseStatus = autonomy.CruiseState

// CruiseStatusMsg refreshes cruise dashboard data (from daemon/engine or mocks).
type CruiseStatusMsg struct {
	State      autonomy.CruiseState
	Metrics    autonomy.MetricsSummary
	DeadLetter []autonomy.DeadLetterSummary
	History    []string
	Report     *autonomy.CruiseReport
}

// CruiseToggleMsg requests engine control (start/pause/resume/stop).
type CruiseToggleMsg struct {
	Action string
}

// CruiseStatusView shows cruise state, metrics, dead letter, and execution history.
type CruiseStatusView struct {
	status     CruiseStatus
	deadLetter []autonomy.DeadLetterSummary
	metrics    autonomy.MetricsSummary
	history    []string
	report     *autonomy.CruiseReport

	width, height int
	vp            viewport.Model
	deadDetail    bool

	t *theme.Theme
}

func NewCruiseStatusView(t *theme.Theme) *CruiseStatusView {
	return &CruiseStatusView{t: t, status: autonomy.CruiseIdle}
}

func (v *CruiseStatusView) ID() ID        { return ViewCruiseStatus }
func (v *CruiseStatusView) Title() string { return "Cruise" }
func (v *CruiseStatusView) Init() tea.Cmd { return nil }

func (v *CruiseStatusView) SetSize(w, h int) {
	v.width, v.height = w, h
	vpH := max(3, h-8)
	if vpH < 3 {
		vpH = 3
	}
	v.vp = viewport.New(viewport.WithWidth(max(20, w-2)), viewport.WithHeight(vpH))
	v.syncViewport()
}

func (v *CruiseStatusView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case CruiseStatusMsg:
		v.status = msg.State
		v.metrics = msg.Metrics
		v.deadLetter = msg.DeadLetter
		v.history = msg.History
		v.report = msg.Report
		v.syncViewport()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "p":
			return v, func() tea.Msg { return CruiseToggleMsg{Action: "pause"} }
		case "r":
			if v.status == autonomy.CruiseIdle {
				return v, func() tea.Msg { return CruiseToggleMsg{Action: "start"} }
			}
			return v, func() tea.Msg { return CruiseToggleMsg{Action: "resume"} }
		case "s":
			return v, func() tea.Msg { return CruiseToggleMsg{Action: "stop"} }
		case "d":
			v.deadDetail = !v.deadDetail
			v.syncViewport()
		default:
			var cmd tea.Cmd
			v.vp, cmd = v.vp.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func (v *CruiseStatusView) syncViewport() {
	v.vp.SetWidth(max(20, v.width-2))
	v.vp.SetHeight(max(3, v.height-8))
	v.vp.SetContent(v.buildContent())
}

func (v *CruiseStatusView) buildContent() string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary())
	b.WriteString(titleStyle.Render("Cruise status"))
	b.WriteString("\n\n")

	stateIcon := "stopped"
	stateColor := v.t.DimText()
	switch v.status {
	case autonomy.CruiseRunning:
		stateIcon = "running"
		stateColor = v.t.Success()
	case autonomy.CruisePaused:
		stateIcon = "paused"
		stateColor = v.t.Warning()
	case autonomy.CruiseIdle:
		stateIcon = "stopped"
	}
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(stateColor).Render(fmt.Sprintf("State: %s", stateIcon)))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(v.t.DimText()).Render(fmt.Sprintf("Engine cycles: %d", v.metrics.CycleCount)))
	b.WriteString("\n")

	ratePct := v.metrics.SuccessRate * 100
	b.WriteString(fmt.Sprintf("Success rate: %.1f%%  Mean cycle: %d ms\n",
		ratePct, v.metrics.MeanCycleDurationMs))
	b.WriteString(fmt.Sprintf("Executions: %d ok / %d total\n\n",
		v.metrics.SuccessfulExecutions, v.metrics.TotalExecutions))

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(v.t.Info()).Render("Dead letter"))
	b.WriteString("\n")
	if len(v.deadLetter) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, d := range v.deadLetter {
			line := fmt.Sprintf("  [%s] %s — %s", d.Kind, d.Description, d.Reason)
			if v.deadDetail {
				line += fmt.Sprintf("\n     id=%s", d.ID)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(v.t.Info()).Render("Recent execution history"))
	b.WriteString("\n")
	if len(v.history) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  (no runs recorded)"))
		b.WriteString("\n")
	} else {
		for _, line := range v.history {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	if v.report != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(v.t.Warning()).Render("Last cycle report"))
		b.WriteString("\n")
		b.WriteString(autonomy.FormatReport(*v.report))
	}

	b.WriteString("\n")
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render(
		"p pause  r resume/start  s stop  d dead-letter detail  arrows scroll")
	b.WriteString(hint)

	return b.String()
}

func (v *CruiseStatusView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}
	return v.vp.View()
}
