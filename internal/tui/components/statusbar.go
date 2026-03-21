package components

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type StatusBar struct {
	theme     *theme.Theme
	width     int
	mode      string
	viewName  string
	focusName string
	repoName  string
	branch    string
	repoState string
	themeName string
}

func NewStatusBar(t *theme.Theme) *StatusBar {
	return &StatusBar{
		theme:     t,
		width:     80,
		mode:      "NORMAL",
		themeName: "default",
	}
}

func (sb *StatusBar) SetWidth(w int)        { sb.width = w }
func (sb *StatusBar) SetMode(m string)      { sb.mode = m }
func (sb *StatusBar) SetViewName(v string)  { sb.viewName = v }
func (sb *StatusBar) SetFocusName(f string) { sb.focusName = f }
func (sb *StatusBar) SetRepoName(r string)  { sb.repoName = r }
func (sb *StatusBar) SetBranch(b string)    { sb.branch = b }
func (sb *StatusBar) SetRepoState(s string) { sb.repoState = s }
func (sb *StatusBar) SetThemeName(n string) { sb.themeName = n }

func (sb *StatusBar) badge(icon, label string, bg, fg color.Color) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		Render(icon + " " + label)
}

func (sb *StatusBar) Render() string {
	if sb.width <= 0 {
		return ""
	}

	modeColor := color.Color(lipgloss.Color("#000000"))
	modeIcon := theme.Icons.Dashboard
	switch strings.ToUpper(sb.mode) {
	case "INSERT":
		modeColor = sb.theme.Success()
		modeIcon = theme.Icons.Chat
	case "NORMAL":
		modeColor = sb.theme.Primary()
		modeIcon = theme.Icons.Dashboard
	case "NAV":
		modeColor = sb.theme.Info()
		modeIcon = theme.Icons.Home
	case "INSPECT":
		modeColor = sb.theme.Accent()
		modeIcon = theme.Icons.Search
	case "COMMAND":
		modeColor = sb.theme.Warning()
		modeIcon = theme.Icons.Search
	default:
		modeColor = sb.theme.Primary()
	}

	leftParts := []string{
		sb.badge(modeIcon, sb.mode, modeColor, sb.theme.OnPrimary()),
	}
	if sb.viewName != "" {
		leftParts = append(leftParts, sb.badge(theme.Icons.Dashboard, sb.viewName, sb.theme.Secondary(), sb.theme.OnPrimary()))
	}
	if sb.focusName != "" {
		leftParts = append(leftParts, sb.badge(theme.Icons.ChevronRight, sb.focusName, sb.theme.AccentMuted(), sb.theme.OnPrimary()))
	}
	if sb.repoName != "" {
		leftParts = append(leftParts, lipgloss.NewStyle().Bold(true).Foreground(sb.theme.Fg()).Render(theme.Icons.Branch+" "+sb.repoName))
	}
	if sb.branch != "" {
		leftParts = append(leftParts, lipgloss.NewStyle().Foreground(sb.theme.Secondary()).Render(theme.Icons.Commit+" "+sb.branch))
	}
	if sb.repoState != "" {
		token := theme.TokenForState(strings.ToLower(sb.repoState))
		leftParts = append(leftParts, lipgloss.NewStyle().Foreground(token.Color).Bold(true).Render(token.Icon+" "+strings.ToUpper(sb.repoState)))
	}

	right := lipgloss.NewStyle().
		Foreground(sb.theme.DimText()).
		Render("theme:" + sb.themeName + "  F1-F5 switch  Tab cycle  Ctrl+P palette")

	left := strings.Join(leftParts, " ")
	gap := sb.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return lipgloss.NewStyle().
		Background(sb.theme.Surface()).
		Width(sb.width).
		Render(left + strings.Repeat(" ", gap) + right)
}
