package components

import (
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type Spinner struct {
	frame   int
	label   string
	theme   *theme.Theme
	visible bool
}

func NewSpinner(t *theme.Theme, label string) *Spinner {
	return &Spinner{
		label:   label,
		theme:   t,
		visible: true,
	}
}

func (s *Spinner) SetLabel(l string)  { s.label = l }
func (s *Spinner) SetVisible(v bool)   { s.visible = v }
func (s *Spinner) Tick()               { s.frame++ }

func (s *Spinner) Render() string {
	if !s.visible {
		return ""
	}
	frames := theme.Icons.Spinner
	if len(frames) == 0 {
		return lipgloss.NewStyle().Foreground(s.theme.MutedFg()).Render(" " + s.label)
	}
	idx := s.frame % len(frames)
	char := frames[idx]
	spinnerStyle := lipgloss.NewStyle().Foreground(s.theme.Primary())
	labelStyle := lipgloss.NewStyle().Foreground(s.theme.MutedFg())
	return spinnerStyle.Render(char) + " " + labelStyle.Render(s.label)
}
