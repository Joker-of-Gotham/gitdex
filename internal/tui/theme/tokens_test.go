package theme

import (
	"testing"
)

func TestTokenForState_KnownStates(t *testing.T) {
	states := []string{"healthy", "drifting", "blocked", "degraded", "unknown"}
	for _, s := range states {
		token := TokenForState(s)
		if token.Label == "" {
			t.Errorf("TokenForState(%q) returned empty label", s)
		}
		if token.Icon == "" {
			t.Errorf("TokenForState(%q) returned empty icon", s)
		}
		if token.Color == nil {
			t.Errorf("TokenForState(%q) returned nil color", s)
		}
	}
}

func TestTokenForState_Unknown(t *testing.T) {
	token := TokenForState("nonexistent")
	unknown := TokenForState("unknown")
	if token.Label != unknown.Label {
		t.Errorf("expected unknown label %q, got %q", unknown.Label, token.Label)
	}
}

func TestNewTheme_DarkMode(t *testing.T) {
	th := NewTheme(true)
	if !th.IsDark {
		t.Error("expected dark mode")
	}
	if th.Fg() == nil {
		t.Error("Fg() returned nil")
	}
	if th.MutedFg() == nil {
		t.Error("MutedFg() returned nil")
	}
	if th.BorderColor() == nil {
		t.Error("BorderColor() returned nil")
	}
}

func TestNewTheme_LightMode(t *testing.T) {
	th := NewTheme(false)
	if th.IsDark {
		t.Error("expected light mode")
	}
	if th.Fg() == nil {
		t.Error("Fg() returned nil")
	}
}

func TestTheme_AllAccessors(t *testing.T) {
	for _, isDark := range []bool{true, false} {
		th := NewTheme(isDark)
		accessors := map[string]func() interface{}{
			"Fg":              func() interface{} { return th.Fg() },
			"MutedFg":         func() interface{} { return th.MutedFg() },
			"Bg":              func() interface{} { return th.Bg() },
			"SubtleFg":        func() interface{} { return th.SubtleFg() },
			"PrimaryMuted":    func() interface{} { return th.PrimaryMuted() },
			"FocusBg":         func() interface{} { return th.FocusBg() },
			"BorderMuted":     func() interface{} { return th.BorderMuted() },
			"AccentMuted":     func() interface{} { return th.AccentMuted() },
			"OnPrimary":       func() interface{} { return th.OnPrimary() },
			"Primary":         func() interface{} { return th.Primary() },
			"Secondary":       func() interface{} { return th.Secondary() },
			"Success":         func() interface{} { return th.Success() },
			"Warning":         func() interface{} { return th.Warning() },
			"Danger":          func() interface{} { return th.Danger() },
			"Info":            func() interface{} { return th.Info() },
			"Accent":          func() interface{} { return th.Accent() },
			"Surface":         func() interface{} { return th.Surface() },
			"Elevated":        func() interface{} { return th.Elevated() },
			"Divider":         func() interface{} { return th.Divider() },
			"DimText":         func() interface{} { return th.DimText() },
			"CodeBg":          func() interface{} { return th.CodeBg() },
			"Highlight":       func() interface{} { return th.Highlight() },
			"Selection":       func() interface{} { return th.Selection() },
			"LinkText":        func() interface{} { return th.LinkText() },
			"Timestamp":       func() interface{} { return th.Timestamp() },
			"BorderColor":     func() interface{} { return th.BorderColor() },
			"FocusBorderColor": func() interface{} { return th.FocusBorderColor() },
			"GradientStart":   func() interface{} { return th.GradientStart() },
			"GradientMid":     func() interface{} { return th.GradientMid() },
			"GradientEnd":     func() interface{} { return th.GradientEnd() },
		}
		for name, fn := range accessors {
			if fn() == nil {
				t.Errorf("isDark=%v: %s() returned nil", isDark, name)
			}
		}
	}
}

func TestNewTheme_WithCustomPalette(t *testing.T) {
	pal := TokyoNightPalette()
	th := NewTheme(true, pal)
	if th.Primary() == nil {
		t.Error("custom palette Primary() should not be nil")
	}
}

func TestNewStyles(t *testing.T) {
	th := NewTheme(true)
	s := NewStyles(th)

	rendered := RenderStateLabel(s, "healthy")
	if rendered == "" {
		t.Error("RenderStateLabel returned empty string")
	}
}

func TestRenderStateLabel_AllStates(t *testing.T) {
	th := NewTheme(true)
	s := NewStyles(th)

	states := []string{"healthy", "drifting", "blocked", "degraded", "unknown"}
	for _, state := range states {
		result := RenderStateLabel(s, state)
		if result == "" {
			t.Errorf("RenderStateLabel(%q) returned empty", state)
		}
	}
}
