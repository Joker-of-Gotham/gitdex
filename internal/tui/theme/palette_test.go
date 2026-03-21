package theme_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestDefaultPalette(t *testing.T) {
	p := theme.DefaultPalette()
	checkAllPaletteFields(t, "default", p)
}

func TestTokyoNightPalette(t *testing.T) {
	p := theme.TokyoNightPalette()
	checkAllPaletteFields(t, "tokyonight", p)
}

func TestCatppuccinPalette(t *testing.T) {
	p := theme.CatppuccinPalette()
	checkAllPaletteFields(t, "catppuccin", p)
}

func TestDraculaPalette(t *testing.T) {
	p := theme.DraculaPalette()
	checkAllPaletteFields(t, "dracula", p)
}

func TestNordPalette(t *testing.T) {
	p := theme.NordPalette()
	checkAllPaletteFields(t, "nord", p)
}

func checkAllPaletteFields(t *testing.T, name string, p theme.Palette) {
	t.Helper()
	v := reflect.ValueOf(p)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		ac, ok := v.Field(i).Interface().(theme.AdaptiveColor)
		if !ok {
			continue
		}
		for _, isDark := range []bool{true, false} {
			if ac.Resolve(isDark) == nil {
				t.Errorf("palette %s: %s.Resolve(%v) returned nil", name, field.Name, isDark)
			}
		}
	}
}

func TestPaletteNames(t *testing.T) {
	names := theme.PaletteNames()
	if len(names) != 5 {
		t.Errorf("PaletteNames() want 5 names, got %d", len(names))
	}
	if !sort.StringsAreSorted(names) {
		t.Error("PaletteNames() should return sorted names")
	}
}

func TestBuiltinPalettes(t *testing.T) {
	if len(theme.BuiltinPalettes) != 5 {
		t.Errorf("BuiltinPalettes want 5 entries, got %d", len(theme.BuiltinPalettes))
	}
}

func TestAdaptiveColor_ResolveDistinct(t *testing.T) {
	p := theme.DefaultPalette()
	light := p.Fg.Resolve(false)
	dark := p.Fg.Resolve(true)
	if light == nil || dark == nil {
		t.Fatal("Fg.Resolve returned nil")
	}
	lr, lg, lb, _ := light.RGBA()
	dr, dg, db, _ := dark.RGBA()
	if lr == dr && lg == dg && lb == db {
		t.Error("Fg light and dark should be different colors")
	}
}
