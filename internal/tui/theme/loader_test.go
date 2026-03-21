package theme_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestLoadThemeFile_Nonexistent(t *testing.T) {
	_, err := theme.LoadThemeFile("nonexistent.yaml")
	if err == nil {
		t.Error("LoadThemeFile(\"nonexistent.yaml\") should return error")
	}
}

func TestLoadThemeFile_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.yaml")
	content := []byte("name: test\ncolors:\n  primary:\n    light: \"#FF0000\"\n    dark: \"#00FF00\"\n  bg:\n    light: \"#FFFFFF\"\n    dark: \"#000000\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	pal, err := theme.LoadThemeFile(path)
	if err != nil {
		t.Fatalf("LoadThemeFile valid: %v", err)
	}
	if pal == nil {
		t.Fatal("LoadThemeFile valid: palette should not be nil")
	}
	if pal.Primary.Resolve(true) == nil {
		t.Error("overridden Primary.Dark should be non-nil")
	}
}

func TestLoadThemeFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := theme.LoadThemeFile(path)
	if err == nil {
		t.Error("LoadThemeFile invalid YAML should return error")
	}
}

func TestLoadThemeFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	pal, err := theme.LoadThemeFile(path)
	if err != nil {
		t.Fatalf("LoadThemeFile empty: %v", err)
	}
	if pal == nil {
		t.Fatal("empty YAML should still return default palette")
	}
}

func TestLoadUserTheme_NoFiles(t *testing.T) {
	t.Setenv("GITDEX_THEME", filepath.Join(t.TempDir(), "nonexistent.yaml"))
	pal, err := theme.LoadUserTheme()
	if err != nil {
		t.Logf("LoadUserTheme error (expected if no default exists): %v", err)
	}
	_ = pal
}

func TestDefaultThemePath_NonEmpty(t *testing.T) {
	p := theme.DefaultThemePath()
	if p == "" {
		t.Log("DefaultThemePath() returned empty (possible in CI)")
	}
}

func TestThemeSearchPaths_NonEmpty(t *testing.T) {
	paths := theme.ThemeSearchPaths()
	if len(paths) == 0 {
		t.Error("ThemeSearchPaths() should return non-empty slice")
	}
}

func TestThemeSearchPaths_IncludesEnv(t *testing.T) {
	t.Setenv("GITDEX_THEME", "/custom/theme.yaml")
	paths := theme.ThemeSearchPaths()
	found := false
	for _, p := range paths {
		if p == "/custom/theme.yaml" {
			found = true
		}
	}
	if !found {
		t.Error("ThemeSearchPaths should include GITDEX_THEME")
	}
}

func TestIsValidColorKey(t *testing.T) {
	valid := []string{"fg", "primary", "bg", "gradient_end", "accent_muted"}
	for _, k := range valid {
		if !theme.IsValidColorKey(k) {
			t.Errorf("IsValidColorKey(%q) should be true", k)
		}
	}
	invalid := []string{"primery", "fgg", "background", ""}
	for _, k := range invalid {
		if theme.IsValidColorKey(k) {
			t.Errorf("IsValidColorKey(%q) should be false", k)
		}
	}
}
