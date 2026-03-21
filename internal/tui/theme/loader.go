package theme

import (
	"os"
	"path/filepath"
	"runtime"

	"charm.land/lipgloss/v2"
	"go.yaml.in/yaml/v3"
)

// ColorDef holds Light and Dark hex strings for YAML theme files.
type ColorDef struct {
	Light string `yaml:"light"`
	Dark  string `yaml:"dark"`
}

// ThemeFile is the YAML structure for user theme customization.
type ThemeFile struct {
	Name   string              `yaml:"name"`
	Colors map[string]ColorDef `yaml:"colors"`
	Icons  struct {
		NerdFont *bool `yaml:"nerd_font"`
	} `yaml:"icons"`
}

// DefaultThemePath returns ~/.config/gitdex/theme.yaml (or %APPDATA%\gitdex\theme.yaml on Windows).
func DefaultThemePath() string {
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("APPDATA"); dir != "" {
			return filepath.Join(dir, "gitdex", "theme.yaml")
		}
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "gitdex", "theme.yaml")
}

// ThemeSearchPaths returns search paths in order: $GITDEX_THEME, DefaultThemePath(), ./gitdex-theme.yaml.
func ThemeSearchPaths() []string {
	paths := make([]string, 0, 3)
	if p := os.Getenv("GITDEX_THEME"); p != "" {
		paths = append(paths, p)
	}
	if p := DefaultThemePath(); p != "" {
		paths = append(paths, p)
	}
	paths = append(paths, "./gitdex-theme.yaml")
	return paths
}

// LoadUserTheme tries each search path and returns the first found palette, or nil if none.
func LoadUserTheme() (*Palette, error) {
	for _, p := range ThemeSearchPaths() {
		pal, err := LoadThemeFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		return pal, nil
	}
	return nil, nil
}

// LoadThemeFile reads a YAML theme file and merges user customizations onto DefaultPalette.
// Returns error if file does not exist or YAML is invalid.
func LoadThemeFile(path string) (*Palette, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tf ThemeFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, err
	}
	pal := DefaultPalette()
	for name, def := range tf.Colors {
		ac := paletteColorFromDef(def)
		applyColorToPalette(&pal, name, ac)
	}
	return &pal, nil
}

func paletteColorFromDef(def ColorDef) AdaptiveColor {
	light := def.Light
	dark := def.Dark
	if light == "" {
		light = "#111827"
	}
	if dark == "" {
		dark = "#F8FAFC"
	}
	return AdaptiveColor{
		Light: lipgloss.Color(light),
		Dark:  lipgloss.Color(dark),
	}
}

var validColorKeys = map[string]bool{
	"fg": true, "muted_fg": true, "subtle_fg": true, "bg": true,
	"surface_bg": true, "elevated_bg": true, "primary": true, "primary_muted": true,
	"secondary": true, "success": true, "warning": true, "danger": true, "info": true,
	"focus_border": true, "focus_bg": true, "border": true, "border_muted": true,
	"divider": true, "accent": true, "accent_muted": true, "highlight": true,
	"selection": true, "dim_text": true, "code_bg": true, "link_text": true,
	"timestamp": true, "gradient_start": true, "gradient_mid": true, "gradient_end": true,
}

func applyColorToPalette(p *Palette, name string, ac AdaptiveColor) {
	switch name {
	case "fg":
		p.Fg = ac
	case "muted_fg":
		p.MutedFg = ac
	case "subtle_fg":
		p.SubtleFg = ac
	case "bg":
		p.Bg = ac
	case "surface_bg":
		p.SurfaceBg = ac
	case "elevated_bg":
		p.ElevatedBg = ac
	case "primary":
		p.Primary = ac
	case "primary_muted":
		p.PrimaryMuted = ac
	case "secondary":
		p.Secondary = ac
	case "success":
		p.Success = ac
	case "warning":
		p.Warning = ac
	case "danger":
		p.Danger = ac
	case "info":
		p.Info = ac
	case "focus_border":
		p.FocusBorder = ac
	case "focus_bg":
		p.FocusBg = ac
	case "border":
		p.Border = ac
	case "border_muted":
		p.BorderMuted = ac
	case "divider":
		p.Divider = ac
	case "accent":
		p.Accent = ac
	case "accent_muted":
		p.AccentMuted = ac
	case "highlight":
		p.Highlight = ac
	case "selection":
		p.Selection = ac
	case "dim_text":
		p.DimText = ac
	case "code_bg":
		p.CodeBg = ac
	case "link_text":
		p.LinkText = ac
	case "timestamp":
		p.Timestamp = ac
	case "gradient_start":
		p.GradientStart = ac
	case "gradient_mid":
		p.GradientMid = ac
	case "gradient_end":
		p.GradientEnd = ac
	}
}

// IsValidColorKey reports whether a color key name is recognized by the loader.
func IsValidColorKey(name string) bool {
	return validColorKeys[name]
}
