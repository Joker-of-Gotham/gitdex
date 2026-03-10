package theme

import "charm.land/lipgloss/v2"

func LightTheme() *Theme {
	return &Theme{
		Name: "light",
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10212B")).
			Background(lipgloss.Color("#CDECF3")).
			Padding(0, 1),
		ActionBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10212B")).
			Background(lipgloss.Color("#E9EEF2")).
			Padding(0, 1),
		Content: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1B2630")).
			Padding(1, 2),
		StatusAdded: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#147A50")),
		StatusModified: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B66A00")),
		StatusDeleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C44536")),
		StatusUntracked: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5D6D7A")),
		RiskSafe: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#147A50")),
		RiskCaution: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B66A00")),
		RiskDanger: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C44536")),
	}
}
