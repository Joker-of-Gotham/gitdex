package theme_test

import (
	"reflect"
	"testing"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestNerdFontIcons_AllFieldsNonEmpty(t *testing.T) {
	checkAllIconFields(t, "NerdFont", theme.NerdFontIcons)
}

func TestUnicodeIcons_AllFieldsNonEmpty(t *testing.T) {
	checkAllIconFields(t, "Unicode", theme.UnicodeIcons)
}

func checkAllIconFields(t *testing.T, name string, icons theme.IconSet) {
	t.Helper()
	v := reflect.ValueOf(icons)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		switch field.Type.Kind() {
		case reflect.String:
			if v.Field(i).String() == "" {
				t.Errorf("%s.%s is empty", name, field.Name)
			}
		case reflect.Slice:
			if v.Field(i).Len() < 2 {
				t.Errorf("%s.%s has fewer than 2 frames (%d)", name, field.Name, v.Field(i).Len())
			}
		}
	}
}

func TestSetNerdFont_True(t *testing.T) {
	theme.SetNerdFont(true)
	defer theme.SetNerdFont(false)
	if theme.Icons.Healthy != theme.NerdFontIcons.Healthy {
		t.Error("SetNerdFont(true) should set Icons = NerdFontIcons")
	}
}

func TestSetNerdFont_False(t *testing.T) {
	theme.SetNerdFont(false)
	defer theme.SetNerdFont(false)
	if theme.Icons.Healthy != theme.UnicodeIcons.Healthy {
		t.Error("SetNerdFont(false) should set Icons = UnicodeIcons")
	}
}

func TestIcons_SpinnerHasAtLeast4Frames(t *testing.T) {
	if len(theme.Icons.Spinner) < 4 {
		t.Errorf("Icons.Spinner want at least 4 frames, got %d", len(theme.Icons.Spinner))
	}
}

func TestDetectNerdFont_EnvVar(t *testing.T) {
	t.Setenv("GITDEX_NERD_FONT", "1")
	if !theme.DetectNerdFont() {
		t.Error("DetectNerdFont should return true when GITDEX_NERD_FONT=1")
	}
	t.Setenv("GITDEX_NERD_FONT", "0")
	if theme.DetectNerdFont() {
		t.Error("DetectNerdFont should return false when GITDEX_NERD_FONT=0")
	}
}

func TestDetectNerdFont_TrueValues(t *testing.T) {
	for _, val := range []string{"true", "yes", "1"} {
		t.Setenv("GITDEX_NERD_FONT", val)
		if !theme.DetectNerdFont() {
			t.Errorf("DetectNerdFont should return true for %q", val)
		}
	}
}
