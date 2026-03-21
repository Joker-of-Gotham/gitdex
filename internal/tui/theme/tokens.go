package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type SemanticToken struct {
	Color color.Color
	Label string
	Icon  string
}

// Deprecated: use Palette. Kept for backward compatibility.
var (
	Ink        = lipgloss.Color("#111827")
	Slate      = lipgloss.Color("#334155")
	Cloud      = lipgloss.Color("#F8FAFC")
	Mist       = lipgloss.Color("#E5E7EB")
	SignalBlue = lipgloss.Color("#6F8697")
	FocusCyan  = lipgloss.Color("#7A948E")
	SuccessGrn = lipgloss.Color("#879B75")
	WarningAmb = lipgloss.Color("#B39A72")
	DangerRed  = lipgloss.Color("#B8837D")
)

var StateTokens = func() map[string]SemanticToken {
	p := DefaultPalette()
	return map[string]SemanticToken{
		"healthy":  {Color: p.Success.Dark, Label: "[healthy]", Icon: "OK"},
		"drifting": {Color: p.Warning.Dark, Label: "[drifting]", Icon: "~"},
		"blocked":  {Color: p.Danger.Dark, Label: "[blocked]", Icon: "!"},
		"degraded": {Color: p.Warning.Dark, Label: "[degraded]", Icon: "-"},
		"unknown":  {Color: p.MutedFg.Dark, Label: "[unknown]", Icon: "?"},
	}
}()

func TokenForState(state string) SemanticToken {
	if t, ok := StateTokens[state]; ok {
		return t
	}
	return StateTokens["unknown"]
}

type Theme struct {
	IsDark  bool
	Palette Palette
}

func NewTheme(isDark bool, palettes ...Palette) Theme {
	var pal Palette
	if len(palettes) > 0 {
		pal = palettes[0]
	} else {
		pal = DefaultPalette()
	}
	return Theme{IsDark: isDark, Palette: pal}
}

func (t Theme) Fg() color.Color               { return t.Palette.Fg.Resolve(t.IsDark) }
func (t Theme) MutedFg() color.Color          { return t.Palette.MutedFg.Resolve(t.IsDark) }
func (t Theme) BorderColor() color.Color      { return t.Palette.Border.Resolve(t.IsDark) }
func (t Theme) FocusBorderColor() color.Color { return t.Palette.FocusBorder.Resolve(t.IsDark) }
func (t Theme) Primary() color.Color          { return t.Palette.Primary.Resolve(t.IsDark) }
func (t Theme) Success() color.Color          { return t.Palette.Success.Resolve(t.IsDark) }
func (t Theme) Warning() color.Color          { return t.Palette.Warning.Resolve(t.IsDark) }
func (t Theme) Danger() color.Color           { return t.Palette.Danger.Resolve(t.IsDark) }
func (t Theme) Info() color.Color             { return t.Palette.Info.Resolve(t.IsDark) }
func (t Theme) Accent() color.Color           { return t.Palette.Accent.Resolve(t.IsDark) }
func (t Theme) Surface() color.Color          { return t.Palette.SurfaceBg.Resolve(t.IsDark) }
func (t Theme) Elevated() color.Color         { return t.Palette.ElevatedBg.Resolve(t.IsDark) }
func (t Theme) Divider() color.Color          { return t.Palette.Divider.Resolve(t.IsDark) }
func (t Theme) DimText() color.Color          { return t.Palette.DimText.Resolve(t.IsDark) }
func (t Theme) CodeBg() color.Color           { return t.Palette.CodeBg.Resolve(t.IsDark) }
func (t Theme) Secondary() color.Color        { return t.Palette.Secondary.Resolve(t.IsDark) }
func (t Theme) Highlight() color.Color        { return t.Palette.Highlight.Resolve(t.IsDark) }
func (t Theme) Selection() color.Color        { return t.Palette.Selection.Resolve(t.IsDark) }
func (t Theme) LinkText() color.Color         { return t.Palette.LinkText.Resolve(t.IsDark) }
func (t Theme) Timestamp() color.Color        { return t.Palette.Timestamp.Resolve(t.IsDark) }
func (t Theme) GradientStart() color.Color    { return t.Palette.GradientStart.Resolve(t.IsDark) }
func (t Theme) GradientMid() color.Color      { return t.Palette.GradientMid.Resolve(t.IsDark) }
func (t Theme) GradientEnd() color.Color      { return t.Palette.GradientEnd.Resolve(t.IsDark) }

func (t Theme) Bg() color.Color           { return t.Palette.Bg.Resolve(t.IsDark) }
func (t Theme) SubtleFg() color.Color     { return t.Palette.SubtleFg.Resolve(t.IsDark) }
func (t Theme) PrimaryMuted() color.Color { return t.Palette.PrimaryMuted.Resolve(t.IsDark) }
func (t Theme) FocusBg() color.Color      { return t.Palette.FocusBg.Resolve(t.IsDark) }
func (t Theme) BorderMuted() color.Color  { return t.Palette.BorderMuted.Resolve(t.IsDark) }
func (t Theme) AccentMuted() color.Color  { return t.Palette.AccentMuted.Resolve(t.IsDark) }

func (t Theme) OnPrimary() color.Color {
	return lipgloss.Color("#F7F3EE")
}
