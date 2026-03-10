package theme

import "charm.land/lipgloss/v2"

type Theme struct {
	Name            string
	Header          lipgloss.Style
	ActionBar       lipgloss.Style
	Content         lipgloss.Style
	StatusAdded     lipgloss.Style
	StatusModified  lipgloss.Style
	StatusDeleted   lipgloss.Style
	StatusUntracked lipgloss.Style
	RiskSafe        lipgloss.Style
	RiskCaution     lipgloss.Style
	RiskDanger      lipgloss.Style
}

var Current *Theme

func Init(name string) {
	switch name {
	case "light":
		Current = LightTheme()
	case "high-contrast":
		Current = HighContrastTheme()
	default:
		Current = DarkTheme()
	}
}

// FormatFileStatus returns styled "icon path" for display. kind: "added"/"A", "modified"/"M", "deleted"/"D", or "untracked"/"?"
func FormatFileStatus(kind, path string) string {
	if Current == nil {
		return path
	}
	var style lipgloss.Style
	var icon string
	switch kind {
	case "added", "A":
		style = Current.StatusAdded
		icon = Icons.Added
	case "modified", "M":
		style = Current.StatusModified
		icon = Icons.Modified
	case "deleted", "D":
		style = Current.StatusDeleted
		icon = Icons.Deleted
	default:
		style = Current.StatusUntracked
		icon = Icons.Untracked
	}
	return style.Render(icon + " " + path)
}
