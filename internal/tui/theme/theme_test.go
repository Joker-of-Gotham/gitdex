package theme

import "testing"

func TestInitAndFormatFileStatus(t *testing.T) {
	Init("dark")
	InitIcons()
	if Current == nil {
		t.Fatal("expected active theme")
	}
	formatted := FormatFileStatus("A", "README.md")
	if formatted == "" {
		t.Fatal("expected formatted output")
	}
}
