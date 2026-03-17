package theme

import "charm.land/lipgloss/v2"

// Theme holds styles for terminal display.
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

	Primary   string
	Secondary string
	Accent    string
	Success   string
	Warning   string
	Danger    string
	Info      string
	Text      string
	TextMuted string
	Border    string
	BorderFoc string
	BgPanel   string
}

// Current is the active theme.
var Current *Theme

// Names returns all available theme names.
func Names() []string {
	return []string{
		"catppuccin",
		"dracula",
		"tokyonight",
		"gruvbox",
		"nord",
		"dark",
		"light",
	}
}

// Init sets the active theme by name.
func Init(name string) {
	switch name {
	case "catppuccin":
		Current = catppuccinTheme()
	case "dracula":
		Current = draculaTheme()
	case "tokyonight":
		Current = tokyonightTheme()
	case "gruvbox":
		Current = gruvboxTheme()
	case "nord":
		Current = nordTheme()
	case "dark":
		Current = darkTheme()
	case "light":
		Current = lightTheme()
	default:
		Current = catppuccinTheme()
	}
}

// InitIcons sets up icon glyphs (no-op in the simplified version).
func InitIcons() {}

func catppuccinTheme() *Theme {
	return &Theme{
		Name:            "catppuccin",
		Header:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA")),
		ActionBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")),
		Content:         lipgloss.NewStyle(),
		StatusAdded:     lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")),
		StatusModified:  lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF")),
		StatusDeleted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")),
		StatusUntracked: lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")),
		RiskSafe:        lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")),
		RiskCaution:     lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF")),
		RiskDanger:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")),
		Primary:   "#89B4FA",
		Secondary: "#B4BEFE",
		Accent:    "#94E2D5",
		Success:   "#A6E3A1",
		Warning:   "#F9E2AF",
		Danger:    "#F38BA8",
		Info:      "#89DCEB",
		Text:      "#CDD6F4",
		TextMuted: "#6C7086",
		Border:    "#45475A",
		BorderFoc: "#89B4FA",
		BgPanel:   "#1E1E2E",
	}
}

func draculaTheme() *Theme {
	return &Theme{
		Name:            "dracula",
		Header:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#BD93F9")),
		ActionBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")),
		Content:         lipgloss.NewStyle(),
		StatusAdded:     lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")),
		StatusModified:  lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C")),
		StatusDeleted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")),
		StatusUntracked: lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")),
		RiskSafe:        lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")),
		RiskCaution:     lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C")),
		RiskDanger:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")),
		Primary:   "#BD93F9",
		Secondary: "#FF79C6",
		Accent:    "#8BE9FD",
		Success:   "#50FA7B",
		Warning:   "#F1FA8C",
		Danger:    "#FF5555",
		Info:      "#8BE9FD",
		Text:      "#F8F8F2",
		TextMuted: "#6272A4",
		Border:    "#44475A",
		BorderFoc: "#BD93F9",
		BgPanel:   "#282A36",
	}
}

func tokyonightTheme() *Theme {
	return &Theme{
		Name:            "tokyonight",
		Header:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7AA2F7")),
		ActionBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")),
		Content:         lipgloss.NewStyle(),
		StatusAdded:     lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")),
		StatusModified:  lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68")),
		StatusDeleted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")),
		StatusUntracked: lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")),
		RiskSafe:        lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")),
		RiskCaution:     lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68")),
		RiskDanger:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")),
		Primary:   "#7AA2F7",
		Secondary: "#BB9AF7",
		Accent:    "#7DCFFF",
		Success:   "#9ECE6A",
		Warning:   "#E0AF68",
		Danger:    "#F7768E",
		Info:      "#7DCFFF",
		Text:      "#C0CAF5",
		TextMuted: "#565F89",
		Border:    "#3B4261",
		BorderFoc: "#7AA2F7",
		BgPanel:   "#1A1B26",
	}
}

func gruvboxTheme() *Theme {
	return &Theme{
		Name:            "gruvbox",
		Header:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FABD2F")),
		ActionBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#928374")),
		Content:         lipgloss.NewStyle(),
		StatusAdded:     lipgloss.NewStyle().Foreground(lipgloss.Color("#B8BB26")),
		StatusModified:  lipgloss.NewStyle().Foreground(lipgloss.Color("#FABD2F")),
		StatusDeleted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FB4934")),
		StatusUntracked: lipgloss.NewStyle().Foreground(lipgloss.Color("#928374")),
		RiskSafe:        lipgloss.NewStyle().Foreground(lipgloss.Color("#B8BB26")),
		RiskCaution:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FABD2F")),
		RiskDanger:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FB4934")),
		Primary:   "#FABD2F",
		Secondary: "#D3869B",
		Accent:    "#8EC07C",
		Success:   "#B8BB26",
		Warning:   "#FE8019",
		Danger:    "#FB4934",
		Info:      "#83A598",
		Text:      "#EBDBB2",
		TextMuted: "#928374",
		Border:    "#504945",
		BorderFoc: "#FABD2F",
		BgPanel:   "#282828",
	}
}

func nordTheme() *Theme {
	return &Theme{
		Name:            "nord",
		Header:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")),
		ActionBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")),
		Content:         lipgloss.NewStyle(),
		StatusAdded:     lipgloss.NewStyle().Foreground(lipgloss.Color("#A3BE8C")),
		StatusModified:  lipgloss.NewStyle().Foreground(lipgloss.Color("#EBCB8B")),
		StatusDeleted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#BF616A")),
		StatusUntracked: lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")),
		RiskSafe:        lipgloss.NewStyle().Foreground(lipgloss.Color("#A3BE8C")),
		RiskCaution:     lipgloss.NewStyle().Foreground(lipgloss.Color("#EBCB8B")),
		RiskDanger:      lipgloss.NewStyle().Foreground(lipgloss.Color("#BF616A")),
		Primary:   "#88C0D0",
		Secondary: "#81A1C1",
		Accent:    "#8FBCBB",
		Success:   "#A3BE8C",
		Warning:   "#EBCB8B",
		Danger:    "#BF616A",
		Info:      "#5E81AC",
		Text:      "#ECEFF4",
		TextMuted: "#4C566A",
		Border:    "#3B4252",
		BorderFoc: "#88C0D0",
		BgPanel:   "#2E3440",
	}
}

func darkTheme() *Theme {
	return &Theme{
		Name:            "dark",
		Header:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7BD8FF")),
		ActionBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")),
		Content:         lipgloss.NewStyle(),
		StatusAdded:     lipgloss.NewStyle().Foreground(lipgloss.Color("#7BD389")),
		StatusModified:  lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C572")),
		StatusDeleted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C73")),
		StatusUntracked: lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")),
		RiskSafe:        lipgloss.NewStyle().Foreground(lipgloss.Color("#7BD389")),
		RiskCaution:     lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C572")),
		RiskDanger:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C73")),
		Primary:   "#7BD8FF",
		Secondary: "#6FC3DF",
		Accent:    "#98C6FF",
		Success:   "#7BD389",
		Warning:   "#F2C572",
		Danger:    "#FF8C73",
		Info:      "#A9BBC7",
		Text:      "#DCE7EF",
		TextMuted: "#7A8B99",
		Border:    "#31556F",
		BorderFoc: "#6FC3DF",
		BgPanel:   "#10212B",
	}
}

func lightTheme() *Theme {
	return &Theme{
		Name:            "light",
		Header:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1E66F5")),
		ActionBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA0B0")),
		Content:         lipgloss.NewStyle(),
		StatusAdded:     lipgloss.NewStyle().Foreground(lipgloss.Color("#40A02B")),
		StatusModified:  lipgloss.NewStyle().Foreground(lipgloss.Color("#DF8E1D")),
		StatusDeleted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#D20F39")),
		StatusUntracked: lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA0B0")),
		RiskSafe:        lipgloss.NewStyle().Foreground(lipgloss.Color("#40A02B")),
		RiskCaution:     lipgloss.NewStyle().Foreground(lipgloss.Color("#DF8E1D")),
		RiskDanger:      lipgloss.NewStyle().Foreground(lipgloss.Color("#D20F39")),
		Primary:   "#1E66F5",
		Secondary: "#7287FD",
		Accent:    "#179299",
		Success:   "#40A02B",
		Warning:   "#DF8E1D",
		Danger:    "#D20F39",
		Info:      "#04A5E5",
		Text:      "#4C4F69",
		TextMuted: "#9CA0B0",
		Border:    "#BCC0CC",
		BorderFoc: "#1E66F5",
		BgPanel:   "#EFF1F5",
	}
}
