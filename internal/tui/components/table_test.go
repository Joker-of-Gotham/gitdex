package components_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/tui/components"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestNewStyledTable(t *testing.T) {
	tk := theme.NewTheme(true)
	st := components.NewStyledTable(&tk, "Col1", "Col2")
	if st == nil {
		t.Fatal("NewStyledTable() should return non-nil")
	}
}

func TestStyledTable_AddRow(t *testing.T) {
	tk := theme.NewTheme(true)
	st := components.NewStyledTable(&tk, "A", "B")
	st.AddRow("a1", "b1")
	st.AddRow("a2", "b2")
	out := st.Render()
	if out == "" {
		t.Error("AddRow should add rows; Render should be non-empty")
	}
	if !strings.Contains(out, "a1") || !strings.Contains(out, "b2") {
		t.Error("Render should contain added row data")
	}
}

func TestStyledTable_Render(t *testing.T) {
	tk := theme.NewTheme(true)
	st := components.NewStyledTable(&tk, "Name", "Value")
	st.AddRow("foo", "bar")
	out := st.Render()
	if out == "" {
		t.Error("Render with headers and rows should return non-empty string")
	}
	if !strings.Contains(out, "Name") || !strings.Contains(out, "Value") {
		t.Error("Render should contain header names")
	}
	if !strings.Contains(out, "foo") || !strings.Contains(out, "bar") {
		t.Error("Render should contain row data")
	}
}
