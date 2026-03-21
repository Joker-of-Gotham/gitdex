package keymap

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestBinding_Matches(t *testing.T) {
	b := Binding{Keys: []string{"q", "ctrl+c"}}

	msg := tea.KeyPressMsg{Code: 'q'}
	if !b.Matches(msg) {
		t.Error("expected 'q' to match")
	}
}

func TestBinding_NoMatch(t *testing.T) {
	b := Binding{Keys: []string{"q"}}

	msg := tea.KeyPressMsg{Code: 'x'}
	if b.Matches(msg) {
		t.Error("expected 'x' not to match")
	}
}

func TestDefaultGlobalKeys(t *testing.T) {
	gk := DefaultGlobalKeys()
	if len(gk.Quit.Keys) == 0 {
		t.Error("Quit should have keys")
	}
	if gk.Quit.Help == "" {
		t.Error("Quit should have help text")
	}
	if len(gk.Back.Keys) == 0 {
		t.Error("Back should have keys")
	}
	if len(gk.Refresh.Keys) == 0 {
		t.Error("Refresh should have keys")
	}
}

func TestDefaultListKeys(t *testing.T) {
	lk := DefaultListKeys()
	if len(lk.Up.Keys) == 0 {
		t.Error("Up should have keys")
	}
	if len(lk.Down.Keys) == 0 {
		t.Error("Down should have keys")
	}
	if len(lk.Select.Keys) == 0 {
		t.Error("Select should have keys")
	}
}

func TestGlobalHelpItems(t *testing.T) {
	items := GlobalHelpItems()
	if len(items) == 0 {
		t.Error("expected non-empty help items")
	}
	for _, item := range items {
		if item.Key == "" {
			t.Error("help item has empty key")
		}
		if item.Desc == "" {
			t.Error("help item has empty description")
		}
	}
}

func TestKeyBindings_NoDuplicateGlobalKeys(t *testing.T) {
	gk := DefaultGlobalKeys()
	seen := make(map[string]string)
	bindings := []struct {
		name    string
		binding Binding
	}{
		{"Quit", gk.Quit},
		{"Help", gk.Help},
		{"Back", gk.Back},
		{"Refresh", gk.Refresh},
	}

	for _, b := range bindings {
		for _, k := range b.binding.Keys {
			if existing, ok := seen[k]; ok {
				t.Errorf("key %q used by both %s and %s", k, existing, b.name)
			}
			seen[k] = b.name
		}
	}
}
