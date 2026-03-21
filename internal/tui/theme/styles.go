package theme

import (
	"charm.land/lipgloss/v2"
)

type Styles struct {
	PageTitle   lipgloss.Style
	PanelTitle  lipgloss.Style
	ObjectTitle lipgloss.Style
	Body        lipgloss.Style
	DenseData   lipgloss.Style
	Annotation  lipgloss.Style

	FocusedBorder lipgloss.Style
	NormalBorder  lipgloss.Style

	StatusHealthy  lipgloss.Style
	StatusDrifting lipgloss.Style
	StatusBlocked  lipgloss.Style
	StatusDegraded lipgloss.Style
	StatusUnknown  lipgloss.Style

	PrimaryAction     lipgloss.Style
	SecondaryAction   lipgloss.Style
	DestructiveAction lipgloss.Style

	KeyHint lipgloss.Style

	Surface       lipgloss.Style
	Elevated      lipgloss.Style
	CodeBlock     lipgloss.Style
	Link          lipgloss.Style
	Timestamp     lipgloss.Style
	Badge         lipgloss.Style
	BadgeSuccess  lipgloss.Style
	BadgeWarning  lipgloss.Style
	BadgeDanger   lipgloss.Style
	Divider       lipgloss.Style
}

func NewStyles(t Theme) Styles {
	fg := t.Fg()
	muted := t.MutedFg()

	return Styles{
		PageTitle:   lipgloss.NewStyle().Bold(true).Foreground(fg),
		PanelTitle:  lipgloss.NewStyle().Bold(true).Foreground(fg).Underline(true),
		ObjectTitle: lipgloss.NewStyle().Bold(true).Foreground(fg),
		Body:        lipgloss.NewStyle().Foreground(fg),
		DenseData:   lipgloss.NewStyle().Foreground(muted),
		Annotation:  lipgloss.NewStyle().Foreground(muted).Italic(true),

		FocusedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.FocusBorderColor()),
		NormalBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderColor()),

		StatusHealthy:  lipgloss.NewStyle().Foreground(t.Success()).Bold(true),
		StatusDrifting: lipgloss.NewStyle().Foreground(t.Warning()).Bold(true),
		StatusBlocked:  lipgloss.NewStyle().Foreground(t.Danger()).Bold(true),
		StatusDegraded: lipgloss.NewStyle().Foreground(t.Danger()),
		StatusUnknown:  lipgloss.NewStyle().Foreground(t.MutedFg()),

		PrimaryAction:     lipgloss.NewStyle().Foreground(t.Primary()).Bold(true),
		SecondaryAction:   lipgloss.NewStyle().Foreground(fg),
		DestructiveAction: lipgloss.NewStyle().Foreground(t.Danger()).Bold(true),

		KeyHint: lipgloss.NewStyle().Foreground(muted),

		Surface:      lipgloss.NewStyle().Background(t.Surface()),
		Elevated:     lipgloss.NewStyle().Background(t.Elevated()),
		CodeBlock:    lipgloss.NewStyle().Background(t.CodeBg()).Padding(0, 1),
		Link:         lipgloss.NewStyle().Foreground(t.LinkText()).Underline(true),
		Timestamp:    lipgloss.NewStyle().Foreground(t.Timestamp()),
		Badge:        lipgloss.NewStyle().Padding(0, 1).Bold(true),
		BadgeSuccess: lipgloss.NewStyle().Padding(0, 1).Bold(true).Background(t.Success()).Foreground(t.OnPrimary()),
		BadgeWarning: lipgloss.NewStyle().Padding(0, 1).Bold(true).Background(t.Warning()).Foreground(t.OnPrimary()),
		BadgeDanger:  lipgloss.NewStyle().Padding(0, 1).Bold(true).Background(t.Danger()).Foreground(t.OnPrimary()),
		Divider:      lipgloss.NewStyle().Foreground(t.Divider()),
	}
}

func (s Styles) StatusStyle(state string) lipgloss.Style {
	switch state {
	case "healthy":
		return s.StatusHealthy
	case "drifting":
		return s.StatusDrifting
	case "blocked":
		return s.StatusBlocked
	case "degraded":
		return s.StatusDegraded
	default:
		return s.StatusUnknown
	}
}

func RenderStateLabel(s Styles, state string) string {
	token := TokenForState(state)
	style := s.StatusStyle(state)
	return style.Render(token.Icon + " " + token.Label)
}
