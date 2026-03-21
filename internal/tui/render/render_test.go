package render

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

var ansiSeqRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiSeqRE.ReplaceAllString(s, "")
}

func TestDetectLanguage(t *testing.T) {
	tests := map[string]string{
		"Dockerfile":  "docker",
		"Makefile":    "make",
		".gitignore":  "bash",
		"go.mod":      "go",
		"main.go":     "go",
		"app.tsx":     "typescript",
		"script.ps1":  "powershell",
		"query.sql":   "sql",
		"unknown.xyz": "",
	}

	for input, want := range tests {
		if got := DetectLanguage(input); got != want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPlainCodeFallback(t *testing.T) {
	got := plainCodeFallback("first\nsecond", 80)
	want := "  1 | first\n  2 | second\n"
	if got != want {
		t.Fatalf("plainCodeFallback() = %q, want %q", got, want)
	}
}

func TestCodeIncludesLineNumbers(t *testing.T) {
	th := theme.NewTheme(true)
	out := stripANSI(Code("package main\nfunc main() {}", "main.go", 80, &th))

	if !strings.Contains(out, "1 | ") {
		t.Fatalf("Code() output missing first line number: %q", out)
	}
	if !strings.Contains(out, "2 | ") {
		t.Fatalf("Code() output missing second line number: %q", out)
	}
	if !strings.Contains(out, "package main") {
		t.Fatalf("Code() output missing source text: %q", out)
	}
}

func TestFillBlockPadsEachLineToWidth(t *testing.T) {
	out := FillBlock("a\nbb", 5, lipgloss.NewStyle())
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("FillBlock() line count = %d, want 2", len(lines))
	}
	for i, line := range lines {
		if got := lipgloss.Width(stripANSI(line)); got != 5 {
			t.Fatalf("line %d width = %d, want 5", i, got)
		}
	}
}

func TestSurfacePanelRespectsOuterWidth(t *testing.T) {
	panel := SurfacePanel("hello\nworld", 24, lipgloss.Color("#111111"), lipgloss.Color("#333333"))

	if got := lipgloss.Width(stripANSI(panel)); got != 24 {
		t.Fatalf("panel width = %d, want 24", got)
	}
	if !strings.Contains(stripANSI(panel), "hello") {
		t.Fatalf("panel output missing content: %q", stripANSI(panel))
	}
}
