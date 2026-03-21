package components

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type ProgressBar struct {
	percent float64
	width   int
	label   string
	theme   *theme.Theme
}

func NewProgressBar(t *theme.Theme) *ProgressBar {
	return &ProgressBar{
		theme:   t,
		width:   40,
		percent: 0,
	}
}

func (p *ProgressBar) SetPercent(v float64) { p.percent = math.Max(0, math.Min(1, v)) }
func (p *ProgressBar) SetWidth(w int)       { p.width = w }
func (p *ProgressBar) SetLabel(l string)    { p.label = l }

func (p *ProgressBar) Render() string {
	barWidth := p.width - 8
	if barWidth < 6 {
		barWidth = 6
	}

	filled := int(math.Round(p.percent * float64(barWidth)))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}

	empty := barWidth - filled
	pct := int(math.Round(p.percent * 100))

	filledStr := lipgloss.NewStyle().
		Background(p.theme.Primary()).
		Render(strings.Repeat(" ", filled))
	emptyStr := lipgloss.NewStyle().
		Background(p.theme.Elevated()).
		Render(strings.Repeat(" ", empty))
	percentStr := lipgloss.NewStyle().
		Foreground(p.theme.MutedFg()).
		Render(fmt.Sprintf(" %d%%", pct))

	bar := lipgloss.NewStyle().
		Foreground(p.theme.BorderColor()).
		Render("[") + filledStr + emptyStr + lipgloss.NewStyle().
		Foreground(p.theme.BorderColor()).
		Render("]") + percentStr

	if p.label == "" {
		return bar
	}

	label := lipgloss.NewStyle().
		Foreground(p.theme.Fg()).
		Render(p.label)
	return label + "\n" + bar
}
