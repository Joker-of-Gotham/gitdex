package theme

import "charm.land/lipgloss/v2"

func DarkTheme() *Theme {
	return &Theme{
		Name: "dark",
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F6F7EB")).
			Background(lipgloss.Color("#124559")).
			Padding(0, 1),
		ActionBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F6F7EB")).
			Background(lipgloss.Color("#1D2D44")).
			Padding(0, 1),
		Content: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDE7EE")).
			Padding(1, 2),
		StatusAdded: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7BD389")),
		StatusModified: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F4B942")),
		StatusDeleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F25F5C")),
		StatusUntracked: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98A6B3")),
		RiskSafe: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7BD389")),
		RiskCaution: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F4B942")),
		RiskDanger: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F25F5C")),
	}
}
