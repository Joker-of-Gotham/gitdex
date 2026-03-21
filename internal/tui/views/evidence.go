package views

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type EvidenceEntry struct {
	Timestamp time.Time
	Action    string
	Result    string
	Detail    string
	Success   bool
}

type EvidenceView struct {
	entries  []EvidenceEntry
	selected int
	width    int
	height   int
	t        *theme.Theme
}

func NewEvidenceView(t *theme.Theme) *EvidenceView { return &EvidenceView{t: t} }

func (v *EvidenceView) ID() ID        { return ViewEvidence }
func (v *EvidenceView) Title() string { return "Evidence" }
func (v *EvidenceView) Init() tea.Cmd { return nil }

func (v *EvidenceView) visibleCount() int {
	const linesPerCard = 6
	h := v.height - 4
	if h < linesPerCard {
		h = linesPerCard
	}
	return maxInt(1, h/linesPerCard)
}

func (v *EvidenceView) Update(msg tea.Msg) (View, tea.Cmd) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch km.String() {
		case "up", "k":
			if v.selected > 0 {
				v.selected--
			}
		case "down", "j":
			if len(v.entries) > 0 && v.selected < len(v.entries)-1 {
				v.selected++
			}
		case "pgup":
			step := maxInt(1, v.visibleCount()/2)
			v.selected -= step
			if v.selected < 0 {
				v.selected = 0
			}
		case "pgdown":
			step := maxInt(1, v.visibleCount()/2)
			v.selected += step
			if v.selected >= len(v.entries) {
				v.selected = maxInt(0, len(v.entries)-1)
			}
		}
	}
	return v, nil
}

func (v *EvidenceView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

func (v *EvidenceView) SetEntries(entries []EvidenceEntry) {
	v.entries = entries
	if v.selected >= len(v.entries) && len(v.entries) > 0 {
		v.selected = len(v.entries) - 1
	}
}

func (v *EvidenceView) Render() string {
	if v.width <= 0 || v.height <= 0 {
		return ""
	}

	if len(v.entries) == 0 {
		body := strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Evidence Stream"),
			"",
			lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No execution evidence yet."),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(24, v.width-8)).
				Render("Command outcomes, workflow checks, file mutations, and remote actions will stream into this ledger once the active repository starts emitting signals."),
		}, "\n")
		return render.SurfacePanel(body, maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
	}

	if v.width >= 96 {
		listWidth := maxInt(34, v.width*54/100)
		detailWidth := maxInt(28, v.width-listWidth-1)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			v.renderEvidenceList(listWidth),
			" ",
			v.renderEvidenceDetail(detailWidth),
		)
	}

	return strings.Join([]string{
		v.renderEvidenceList(v.width),
		v.renderEvidenceDetail(v.width),
	}, "\n\n")
}

func (v *EvidenceView) renderEvidenceList(width int) string {
	rows := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("Evidence Stream (%d)", len(v.entries))),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Up/Down select  PgUp/PgDn page"),
		"",
	}

	vis := v.visibleCount()
	start := 0
	if v.selected >= vis {
		start = v.selected - vis + 1
	}
	end := start + vis
	if end > len(v.entries) {
		end = len(v.entries)
	}

	for i := start; i < end; i++ {
		entry := v.entries[i]
		color := v.t.Success()
		status := "SUCCESS"
		if !entry.Success {
			color = v.t.Warning()
			status = "REVIEW"
		}

		card := []string{
			lipgloss.NewStyle().Foreground(v.t.Timestamp()).Render(entry.Timestamp.Format("15:04:05")) + "  " +
				lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(entry.Action),
			lipgloss.NewStyle().Foreground(color).Bold(true).Render(status + "  " + strings.ToUpper(valueOrDash(entry.Result))),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(18, width-8)).Render(valueOrDash(entry.Detail)),
		}
		block := strings.Join(card, "\n")
		if i == v.selected {
			panelFrame := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).GetHorizontalFrameSize()
			block = render.FillBlock(block, maxInt(12, width-panelFrame), lipgloss.NewStyle().Background(v.t.Selection()))
		}
		rows = append(rows, render.SurfacePanel(block, width, v.t.Surface(), v.t.BorderColor()))
	}
	return strings.Join(rows, "\n")
}

func (v *EvidenceView) renderEvidenceDetail(width int) string {
	if len(v.entries) == 0 || v.selected >= len(v.entries) {
		return ""
	}

	entry := v.entries[v.selected]
	statusColor := v.t.Success()
	status := "Success"
	if !entry.Success {
		statusColor = v.t.Warning()
		status = "Needs Review"
	}

	body := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Selected Evidence"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(entry.Action),
		lipgloss.NewStyle().Foreground(v.t.Timestamp()).Render(entry.Timestamp.Format(time.RFC3339)),
		"",
		lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(status + "  " + strings.ToUpper(valueOrDash(entry.Result))),
		"",
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(18, width-8)).Render(valueOrDash(entry.Detail)),
	}
	return render.SurfacePanel(strings.Join(body, "\n"), width, v.t.Surface(), v.t.BorderColor())
}
