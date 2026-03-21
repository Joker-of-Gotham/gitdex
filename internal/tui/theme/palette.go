package theme

import (
	"image/color"
	"sort"

	"charm.land/lipgloss/v2"
)

// AdaptiveColor holds Light and Dark variants for theme-aware rendering.
type AdaptiveColor struct {
	Light color.Color
	Dark  color.Color
}

// Resolve returns the appropriate color based on dark mode.
func (ac AdaptiveColor) Resolve(isDark bool) color.Color {
	if isDark {
		return ac.Dark
	}
	return ac.Light
}

// Palette holds semantic color tokens for the theme system.
type Palette struct {
	Fg, MutedFg, SubtleFg                     AdaptiveColor
	Bg, SurfaceBg, ElevatedBg                 AdaptiveColor
	Primary, PrimaryMuted, Secondary          AdaptiveColor
	Success, Warning, Danger, Info            AdaptiveColor
	FocusBorder, FocusBg                      AdaptiveColor
	Border, BorderMuted, Divider              AdaptiveColor
	Accent, AccentMuted, Highlight, Selection AdaptiveColor
	DimText, CodeBg, LinkText, Timestamp      AdaptiveColor
	GradientStart, GradientMid, GradientEnd   AdaptiveColor
}

func ac(light, dark string) AdaptiveColor {
	return AdaptiveColor{
		Light: lipgloss.Color(light),
		Dark:  lipgloss.Color(dark),
	}
}

// DefaultPalette is the muted control-console palette used across the TUI.
// It aims for a dark Morandi-like tone: dusty slate base with sage, rose, and
// faded brass accents rather than high-saturation neon.
func DefaultPalette() Palette {
	return Palette{
		Fg:            ac("#2F343A", "#E9E2D8"),
		MutedFg:       ac("#66707A", "#B7AEA2"),
		SubtleFg:      ac("#8F969B", "#938A7F"),
		Bg:            ac("#F3EEE7", "#141619"),
		SurfaceBg:     ac("#E8E2D9", "#20242A"),
		ElevatedBg:    ac("#DDD6CC", "#2A3037"),
		Primary:       ac("#6F8697", "#8FA59D"),
		PrimaryMuted:  ac("#8A9AA6", "#6F827E"),
		Secondary:     ac("#A78D82", "#A99286"),
		Success:       ac("#7D8E6B", "#8CA076"),
		Warning:       ac("#AE966F", "#C0A77A"),
		Danger:        ac("#B17B76", "#BD8981"),
		Info:          ac("#7E8F9D", "#93A3B1"),
		FocusBorder:   ac("#6F8697", "#8FA59D"),
		FocusBg:       ac("#DCE1DF", "#313A41"),
		Border:        ac("#C9C0B6", "#49525B"),
		BorderMuted:   ac("#DDD6CC", "#2A3037"),
		Divider:       ac("#C9C0B6", "#414A53"),
		Accent:        ac("#7E8F9D", "#93A3B1"),
		AccentMuted:   ac("#98A5AF", "#70808B"),
		Highlight:     ac("#B39A72", "#C7AD7E"),
		Selection:     ac("#D7D0C6", "#374047"),
		DimText:       ac("#8F969B", "#7E756C"),
		CodeBg:        ac("#ECE6DD", "#1B1F24"),
		LinkText:      ac("#5F7D8D", "#90A8B3"),
		Timestamp:     ac("#8F969B", "#8E867C"),
		GradientStart: ac("#8CA076", "#8CA076"),
		GradientMid:   ac("#C0A77A", "#C0A77A"),
		GradientEnd:   ac("#BD8981", "#BD8981"),
	}
}

func TokyoNightPalette() Palette {
	return Palette{
		Fg:            ac("#343B58", "#C0CAF5"),
		MutedFg:       ac("#9699A3", "#565F89"),
		SubtleFg:      ac("#C0CAF5", "#3B4261"),
		Bg:            ac("#D5D6DB", "#1A1B26"),
		SurfaceBg:     ac("#E9E9EC", "#24283B"),
		ElevatedBg:    ac("#FFFFFF", "#414868"),
		Primary:       ac("#34548A", "#7AA2F7"),
		PrimaryMuted:  ac("#5A7BBF", "#3D59A1"),
		Secondary:     ac("#5A3E8E", "#BB9AF7"),
		Success:       ac("#485E30", "#9ECE6A"),
		Warning:       ac("#8F5E15", "#E0AF68"),
		Danger:        ac("#8C4351", "#F7768E"),
		Info:          ac("#166775", "#7DCFFF"),
		FocusBorder:   ac("#34548A", "#7AA2F7"),
		FocusBg:       ac("#D4D7F2", "#292E42"),
		Border:        ac("#C0CAF5", "#3B4261"),
		BorderMuted:   ac("#D5D6DB", "#24283B"),
		Divider:       ac("#C0CAF5", "#3B4261"),
		Accent:        ac("#34548A", "#7AA2F7"),
		AccentMuted:   ac("#5A7BBF", "#3D59A1"),
		Highlight:     ac("#8F5E15", "#E0AF68"),
		Selection:     ac("#D4D7F2", "#283457"),
		DimText:       ac("#9699A3", "#565F89"),
		CodeBg:        ac("#E9E9EC", "#24283B"),
		LinkText:      ac("#166775", "#7DCFFF"),
		Timestamp:     ac("#9699A3", "#565F89"),
		GradientStart: ac("#485E30", "#9ECE6A"),
		GradientMid:   ac("#8F5E15", "#E0AF68"),
		GradientEnd:   ac("#8C4351", "#F7768E"),
	}
}

func CatppuccinPalette() Palette {
	return Palette{
		Fg:            ac("#4C4F69", "#CDD6F4"),
		MutedFg:       ac("#8C8FA1", "#6C7086"),
		SubtleFg:      ac("#9CA0B0", "#585B70"),
		Bg:            ac("#EFF1F5", "#1E1E2E"),
		SurfaceBg:     ac("#E6E9EF", "#313244"),
		ElevatedBg:    ac("#FFFFFF", "#45475A"),
		Primary:       ac("#1E66F5", "#89B4FA"),
		PrimaryMuted:  ac("#209FB5", "#74C7EC"),
		Secondary:     ac("#8839EF", "#CBA6F7"),
		Success:       ac("#40A02B", "#A6E3A1"),
		Warning:       ac("#DF8E1D", "#F9E2AF"),
		Danger:        ac("#D20F39", "#F38BA8"),
		Info:          ac("#04A5E5", "#89DCEB"),
		FocusBorder:   ac("#1E66F5", "#89B4FA"),
		FocusBg:       ac("#DCE0E8", "#313244"),
		Border:        ac("#CCD0DA", "#45475A"),
		BorderMuted:   ac("#E6E9EF", "#313244"),
		Divider:       ac("#CCD0DA", "#45475A"),
		Accent:        ac("#1E66F5", "#89B4FA"),
		AccentMuted:   ac("#209FB5", "#74C7EC"),
		Highlight:     ac("#DF8E1D", "#F9E2AF"),
		Selection:     ac("#BCC0CC", "#45475A"),
		DimText:       ac("#9CA0B0", "#585B70"),
		CodeBg:        ac("#E6E9EF", "#313244"),
		LinkText:      ac("#04A5E5", "#89DCEB"),
		Timestamp:     ac("#8C8FA1", "#6C7086"),
		GradientStart: ac("#40A02B", "#A6E3A1"),
		GradientMid:   ac("#DF8E1D", "#F9E2AF"),
		GradientEnd:   ac("#D20F39", "#F38BA8"),
	}
}

func DraculaPalette() Palette {
	return Palette{
		Fg:            ac("#282A36", "#F8F8F2"),
		MutedFg:       ac("#6272A4", "#6272A4"),
		SubtleFg:      ac("#D4D4D4", "#44475A"),
		Bg:            ac("#F8F8F2", "#282A36"),
		SurfaceBg:     ac("#EAEAEA", "#44475A"),
		ElevatedBg:    ac("#FFFFFF", "#6272A4"),
		Primary:       ac("#7C3AED", "#BD93F9"),
		PrimaryMuted:  ac("#8B5CF6", "#6272A4"),
		Secondary:     ac("#DB2777", "#FF79C6"),
		Success:       ac("#16A34A", "#50FA7B"),
		Warning:       ac("#CA8A04", "#F1FA8C"),
		Danger:        ac("#DC2626", "#FF5555"),
		Info:          ac("#0891B2", "#8BE9FD"),
		FocusBorder:   ac("#7C3AED", "#BD93F9"),
		FocusBg:       ac("#EDE9FE", "#44475A"),
		Border:        ac("#D4D4D4", "#6272A4"),
		BorderMuted:   ac("#EAEAEA", "#44475A"),
		Divider:       ac("#D4D4D4", "#6272A4"),
		Accent:        ac("#7C3AED", "#BD93F9"),
		AccentMuted:   ac("#8B5CF6", "#6272A4"),
		Highlight:     ac("#CA8A04", "#F1FA8C"),
		Selection:     ac("#EDE9FE", "#44475A"),
		DimText:       ac("#A1A1AA", "#6272A4"),
		CodeBg:        ac("#EAEAEA", "#44475A"),
		LinkText:      ac("#0891B2", "#8BE9FD"),
		Timestamp:     ac("#A1A1AA", "#6272A4"),
		GradientStart: ac("#16A34A", "#50FA7B"),
		GradientMid:   ac("#CA8A04", "#F1FA8C"),
		GradientEnd:   ac("#DC2626", "#FF5555"),
	}
}

func NordPalette() Palette {
	return Palette{
		Fg:            ac("#2E3440", "#ECEFF4"),
		MutedFg:       ac("#4C566A", "#4C566A"),
		SubtleFg:      ac("#D8DEE9", "#434C5E"),
		Bg:            ac("#ECEFF4", "#2E3440"),
		SurfaceBg:     ac("#E5E9F0", "#3B4252"),
		ElevatedBg:    ac("#FFFFFF", "#434C5E"),
		Primary:       ac("#5E81AC", "#88C0D0"),
		PrimaryMuted:  ac("#81A1C1", "#81A1C1"),
		Secondary:     ac("#B48EAD", "#B48EAD"),
		Success:       ac("#A3BE8C", "#A3BE8C"),
		Warning:       ac("#EBCB8B", "#EBCB8B"),
		Danger:        ac("#BF616A", "#BF616A"),
		Info:          ac("#5E81AC", "#88C0D0"),
		FocusBorder:   ac("#5E81AC", "#88C0D0"),
		FocusBg:       ac("#E5E9F0", "#3B4252"),
		Border:        ac("#D8DEE9", "#434C5E"),
		BorderMuted:   ac("#E5E9F0", "#3B4252"),
		Divider:       ac("#D8DEE9", "#434C5E"),
		Accent:        ac("#5E81AC", "#88C0D0"),
		AccentMuted:   ac("#81A1C1", "#81A1C1"),
		Highlight:     ac("#EBCB8B", "#EBCB8B"),
		Selection:     ac("#D8DEE9", "#434C5E"),
		DimText:       ac("#4C566A", "#4C566A"),
		CodeBg:        ac("#E5E9F0", "#3B4252"),
		LinkText:      ac("#5E81AC", "#88C0D0"),
		Timestamp:     ac("#4C566A", "#4C566A"),
		GradientStart: ac("#A3BE8C", "#A3BE8C"),
		GradientMid:   ac("#EBCB8B", "#EBCB8B"),
		GradientEnd:   ac("#BF616A", "#BF616A"),
	}
}

// BuiltinPalettes maps palette names to constructor functions.
var BuiltinPalettes = map[string]func() Palette{
	"default":     DefaultPalette,
	"tokyo-night": TokyoNightPalette,
	"catppuccin":  CatppuccinPalette,
	"dracula":     DraculaPalette,
	"nord":        NordPalette,
}

// PaletteNames returns sorted names of built-in palettes.
func PaletteNames() []string {
	names := make([]string, 0, len(BuiltinPalettes))
	for k := range BuiltinPalettes {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
