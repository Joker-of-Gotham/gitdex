package theme

import (
	"testing"
)

func TestInit_Dark(t *testing.T) {
	Init("dark")
	if Current == nil {
		t.Fatal("Current must not be nil after Init")
	}
	if Current.Name != "dark" {
		t.Errorf("Init(\"dark\") should set dark theme, got Name=%q", Current.Name)
	}
}

func TestInit_Light(t *testing.T) {
	Init("light")
	if Current == nil {
		t.Fatal("Current must not be nil after Init")
	}
	if Current.Name != "light" {
		t.Errorf("Init(\"light\") should set light theme, got Name=%q", Current.Name)
	}
}

func TestInit_Default(t *testing.T) {
	Init("")
	if Current == nil {
		t.Fatal("Current must not be nil after Init")
	}
	if Current.Name != "catppuccin" {
		t.Errorf("Init(\"\") should default to catppuccin theme, got Name=%q", Current.Name)
	}
}

func TestInitIcons(t *testing.T) {
	InitIcons()
}

func TestTheme_Name(t *testing.T) {
	Init("light")
	if Current.Name != "light" {
		t.Errorf("Theme.Name want light, got %q", Current.Name)
	}
	Init("dark")
	if Current.Name != "dark" {
		t.Errorf("Theme.Name want dark, got %q", Current.Name)
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) < 5 {
		t.Errorf("expected at least 5 themes, got %d", len(names))
	}
}

func TestAllThemes_HaveColors(t *testing.T) {
	for _, name := range Names() {
		Init(name)
		if Current.Primary == "" {
			t.Errorf("theme %q has empty Primary", name)
		}
		if Current.Success == "" {
			t.Errorf("theme %q has empty Success", name)
		}
		if Current.Danger == "" {
			t.Errorf("theme %q has empty Danger", name)
		}
		if Current.Text == "" {
			t.Errorf("theme %q has empty Text", name)
		}
		if Current.Border == "" {
			t.Errorf("theme %q has empty Border", name)
		}
	}
}

func TestInit_AllKnownThemes(t *testing.T) {
	for _, name := range Names() {
		Init(name)
		if Current == nil {
			t.Fatalf("Current is nil after Init(%q)", name)
		}
		if Current.Name != name {
			t.Errorf("Init(%q): Name=%q", name, Current.Name)
		}
	}
}
