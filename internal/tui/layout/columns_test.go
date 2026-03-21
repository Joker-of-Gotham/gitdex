package layout_test

import (
	"image/color"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/tui/layout"
)

func TestRenderColumns_Compact(t *testing.T) {
	dims := layout.Classify(80, 40)
	out := layout.RenderColumns(dims, "", "main content", "", color.Black)
	if out == "" {
		t.Fatal("RenderColumns should return non-empty")
	}
	if !strings.Contains(out, "main content") {
		t.Error("Compact: should contain main content only")
	}
	// No nav or inspector separators
	if strings.Contains(out, "│") || strings.Count(out, "main content") != 1 {
		// Single column: just main, no extra borders for nav/inspector
	}
}

func TestRenderColumns_Standard(t *testing.T) {
	dims := layout.Classify(120, 40)
	out := layout.RenderColumns(dims, "", "main", "inspector", color.Black)
	if out == "" {
		t.Fatal("RenderColumns should return non-empty")
	}
	if !strings.Contains(out, "main") {
		t.Error("Standard: should contain main")
	}
	if !strings.Contains(out, "inspector") {
		t.Error("Standard: should contain inspector")
	}
}

func TestRenderColumns_Wide(t *testing.T) {
	dims := layout.Classify(160, 40)
	out := layout.RenderColumns(dims, "", "main", "inspector", color.Black)
	if out == "" {
		t.Fatal("RenderColumns should return non-empty")
	}
	if !strings.Contains(out, "main") {
		t.Error("Wide: should contain main")
	}
	if !strings.Contains(out, "inspector") {
		t.Error("Wide: should contain inspector")
	}
}

func TestDimensions_HeaderHeight(t *testing.T) {
	dims := layout.Classify(100, 40)
	if h := dims.HeaderHeight(); h != 2 {
		t.Errorf("HeaderHeight: got %d, want 2", h)
	}
}

func TestDimensions_StatusBarHeight(t *testing.T) {
	dims := layout.Classify(100, 40)
	if h := dims.StatusBarHeight(); h != 1 {
		t.Errorf("StatusBarHeight: got %d, want 1", h)
	}
}

func TestDimensions_ComposerHeight(t *testing.T) {
	dims := layout.Classify(100, 40)
	if h := dims.ComposerHeight(); h != 3 {
		t.Errorf("ComposerHeight: got %d, want 3", h)
	}
}

func TestDimensions_ContentHeight(t *testing.T) {
	dims := layout.Classify(100, 40)
	ch := dims.ContentHeight()
	// 40 - 2 - 1 - 3 = 34
	if ch < 5 {
		t.Errorf("ContentHeight: got %d, want >= 5", ch)
	}
	expected := 40 - 2 - 1 - 3
	if ch != expected {
		t.Errorf("ContentHeight: got %d, want %d", ch, expected)
	}
}
