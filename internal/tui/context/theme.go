package context

// ThemeColors defines the color palette for the TUI.
type ThemeColors struct {
	Primary    string
	Secondary  string
	Background string
	Foreground string
	Subtle     string
	Highlight  string

	SuccessText string
	WarningText string
	DangerText  string
	InfoText    string

	OpenColor   string
	ClosedColor string
	MergedColor string
	DraftColor  string

	BorderNormal  string
	BorderFocused string
	BorderActive  string

	TabActive   string
	TabInactive string

	StatusOK   string
	StatusRun  string
	StatusErr  string
	StatusWait string
	StatusSkip string
}

// DefaultTheme returns the Catppuccin-inspired default theme.
func DefaultTheme() *ThemeColors {
	return &ThemeColors{
		Primary:    "#89b4fa",
		Secondary:  "#a6e3a1",
		Background: "#1e1e2e",
		Foreground: "#cdd6f4",
		Subtle:     "#6c7086",
		Highlight:  "#f5e0dc",

		SuccessText: "#a6e3a1",
		WarningText: "#f9e2af",
		DangerText:  "#f38ba8",
		InfoText:    "#89b4fa",

		OpenColor:   "#a6e3a1",
		ClosedColor: "#f38ba8",
		MergedColor: "#cba6f7",
		DraftColor:  "#6c7086",

		BorderNormal:  "#313244",
		BorderFocused: "#89b4fa",
		BorderActive:  "#a6e3a1",

		TabActive:   "#89b4fa",
		TabInactive: "#6c7086",

		StatusOK:   "#a6e3a1",
		StatusRun:  "#89b4fa",
		StatusErr:  "#f38ba8",
		StatusWait: "#6c7086",
		StatusSkip: "#f9e2af",
	}
}
