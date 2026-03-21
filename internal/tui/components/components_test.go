package components_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/components"
	"github.com/your-org/gitdex/internal/tui/theme"
	"github.com/your-org/gitdex/internal/tui/views"
)

func makeTheme() *theme.Theme {
	t := theme.NewTheme(true)
	return &t
}

func TestNewHeader(t *testing.T) {
	h := components.NewHeader(makeTheme())
	if h == nil {
		t.Fatal("NewHeader() should return non-nil")
	}
}

func TestHeader_SetWidth(t *testing.T) {
	h := components.NewHeader(makeTheme())
	h.SetWidth(100)
	// Verify by rendering - width affects layout
	out := h.Render()
	if out == "" {
		t.Error("Render after SetWidth should return non-empty")
	}
}

func TestHeader_Render(t *testing.T) {
	h := components.NewHeader(makeTheme())
	h.SetWidth(80)
	out := h.Render()
	if out == "" {
		t.Error("Render() should return non-empty string")
	}
	if !strings.Contains(out, "Gitdex") {
		t.Error("Render() should contain brand name")
	}
}

func TestHeader_SetTabs(t *testing.T) {
	chatView := &mockView{id: views.ViewChat, title: "Chat"}
	statusView := &mockView{id: views.ViewStatus, title: "Status"}
	router := views.NewRouter(views.ViewChat, chatView, statusView)

	h := components.NewHeader(makeTheme())
	h.SetWidth(100)
	h.SetTabs(router)

	out := h.Render()
	if out == "" {
		t.Error("Render() should return non-empty string")
	}
	// Active tab (Chat) should be highlighted/styled
	if !strings.Contains(out, "Chat") {
		t.Error("Render() should show tab titles")
	}
	if !strings.Contains(out, "Status") {
		t.Error("Render() should show Status tab")
	}
}

func TestNewComposer(t *testing.T) {
	c := components.NewComposer(makeTheme())
	if c == nil {
		t.Fatal("NewComposer() should return non-nil")
	}
}

func TestComposer_SetFocused_Focused(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	if !c.Focused() {
		t.Error("Focused() should be true after SetFocused(true)")
	}
	c.SetFocused(false)
	if c.Focused() {
		t.Error("Focused() should be false after SetFocused(false)")
	}
}

func TestComposer_SetWidth(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetWidth(80)
	// Width is used in Render
	out := c.Render()
	if out == "" {
		t.Error("Render after SetWidth should return non-empty")
	}
}

func TestComposer_TypingAppendsToInput(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	// Type 'a', 'b', 'c'
	c.Update(tea.KeyPressMsg{Code: 'a'})
	c.Update(tea.KeyPressMsg{Code: 'b'})
	c.Update(tea.KeyPressMsg{Code: 'c'})

	if c.Value() != "abc" {
		t.Errorf("Value() after typing abc: got %q", c.Value())
	}
}

func TestComposer_Value(t *testing.T) {
	c := components.NewComposer(makeTheme())
	if c.Value() != "" {
		t.Errorf("initial Value() should be empty, got %q", c.Value())
	}
	c.SetFocused(true)
	c.Update(tea.KeyPressMsg{Code: 'x'})
	if c.Value() != "x" {
		t.Errorf("Value() after typing x: got %q", c.Value())
	}
}

func TestComposer_EnterDispatchesSubmitMsg(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	c.Update(tea.KeyPressMsg{Code: 'h'})
	c.Update(tea.KeyPressMsg{Code: 'i'})

	cmd := c.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(Enter) should return a command")
	}
	msg := cmd()
	sm, ok := msg.(components.SubmitMsg)
	if !ok {
		t.Fatalf("command should return SubmitMsg, got %T", msg)
	}
	if sm.Input != "hi" {
		t.Errorf("SubmitMsg.Input: got %q, want hi", sm.Input)
	}
	if sm.IsCommand {
		t.Error("SubmitMsg.IsCommand should be false for regular input")
	}
}

func TestComposer_EnterWithSlashSetsIsCommand(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	c.Update(tea.KeyPressMsg{Code: '/'})
	c.Update(tea.KeyPressMsg{Code: 'h'})
	c.Update(tea.KeyPressMsg{Code: 'e'})
	c.Update(tea.KeyPressMsg{Code: 'l'})
	c.Update(tea.KeyPressMsg{Code: 'p'})

	cmd := c.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(Enter) should return a command")
	}
	msg := cmd()
	sm, ok := msg.(components.SubmitMsg)
	if !ok {
		t.Fatalf("command should return SubmitMsg, got %T", msg)
	}
	if sm.Input != "/help" {
		t.Errorf("SubmitMsg.Input: got %q, want /help", sm.Input)
	}
	if !sm.IsCommand {
		t.Error("SubmitMsg.IsCommand should be true for / prefix")
	}
	if sm.IsIntent {
		t.Error("SubmitMsg.IsIntent should be false for / prefix")
	}
}

func TestComposer_EnterWithBangSetsIsIntent(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	c.Update(tea.KeyPressMsg{Code: '!'})
	c.Update(tea.KeyPressMsg{Code: 'f'})
	c.Update(tea.KeyPressMsg{Code: 'i'})
	c.Update(tea.KeyPressMsg{Code: 'x'})

	cmd := c.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(Enter) should return a command")
	}
	msg := cmd()
	sm, ok := msg.(components.SubmitMsg)
	if !ok {
		t.Fatalf("command should return SubmitMsg, got %T", msg)
	}
	if sm.Input != "!fix" {
		t.Errorf("SubmitMsg.Input: got %q, want !fix", sm.Input)
	}
	if sm.IsCommand {
		t.Error("SubmitMsg.IsCommand should be false for ! prefix")
	}
	if !sm.IsIntent {
		t.Error("SubmitMsg.IsIntent should be true for ! prefix")
	}
}

func TestComposer_Render_ShowsCursorWhenFocused(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)
	c.Update(tea.KeyPressMsg{Code: 'a'})

	out := c.Render()
	if !strings.Contains(out, "a") {
		t.Error("Render when focused should show typed content")
	}
	if strings.Contains(out, "Type a command or question") {
		t.Error("Render when focused with input should not show placeholder")
	}
}

func TestComposer_Render_PlaceholderWhenNotFocused(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(false)
	c.SetWidth(80)

	out := c.Render()
	if !strings.Contains(out, "Type a command or question") {
		t.Error("Render when not focused and empty should show placeholder")
	}
}

func TestComposer_HistoryNavigation(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	// Submit "first"
	c.Update(tea.KeyPressMsg{Code: 'f'})
	c.Update(tea.KeyPressMsg{Code: 'i'})
	c.Update(tea.KeyPressMsg{Code: 'r'})
	c.Update(tea.KeyPressMsg{Code: 's'})
	c.Update(tea.KeyPressMsg{Code: 't'})
	cmd := c.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		cmd()
	}

	// Submit "second"
	c.Update(tea.KeyPressMsg{Code: 's'})
	c.Update(tea.KeyPressMsg{Code: 'e'})
	c.Update(tea.KeyPressMsg{Code: 'c'})
	c.Update(tea.KeyPressMsg{Code: 'o'})
	c.Update(tea.KeyPressMsg{Code: 'n'})
	c.Update(tea.KeyPressMsg{Code: 'd'})
	cmd = c.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		cmd()
	}

	// Press Up -> should show "second" (most recent)
	c.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if got := c.Value(); got != "second" {
		t.Errorf("after Up: got %q, want second", got)
	}

	// Press Up again -> should show "first" (previous)
	c.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if got := c.Value(); got != "first" {
		t.Errorf("after second Up: got %q, want first", got)
	}

	// Press Down -> should show "second" (next)
	c.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := c.Value(); got != "second" {
		t.Errorf("after Down: got %q, want second", got)
	}

	// Press Down again -> should clear (back to "next" / new input)
	c.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := c.Value(); got != "" {
		t.Errorf("after second Down: got %q, want empty", got)
	}
}

func TestComposer_PasteMultiline(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	multiline := "line1\nline2\nline3"
	c.Update(tea.PasteMsg{Content: multiline})
	if c.Value() != multiline {
		t.Errorf("multi-line paste: got %q, want %q", c.Value(), multiline)
	}
}

func TestComposer_PasteSpecialChars(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	special := "Hello 浣犲ソ 馃實 <script>alert('xss')</script> tab\there"
	c.Update(tea.PasteMsg{Content: special})
	if c.Value() != special {
		t.Errorf("special chars paste: got %q, want %q", c.Value(), special)
	}
}

func TestComposer_PasteMiddleOfText(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	c.Update(tea.KeyPressMsg{Code: 'a'})
	c.Update(tea.KeyPressMsg{Code: 'c'})
	// Move cursor left 1 position to insert between 'a' and 'c'
	c.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	c.Update(tea.PasteMsg{Content: "b"})
	if c.Value() != "abc" {
		t.Errorf("paste in middle: got %q, want 'abc'", c.Value())
	}
}

func TestComposer_PasteEmptyString(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(true)
	c.SetWidth(80)

	c.Update(tea.KeyPressMsg{Code: 'x'})
	c.Update(tea.PasteMsg{Content: ""})
	if c.Value() != "x" {
		t.Errorf("empty paste should not change value: got %q", c.Value())
	}
}

func TestComposer_PasteNotFocused(t *testing.T) {
	c := components.NewComposer(makeTheme())
	c.SetFocused(false)
	c.SetWidth(80)

	c.Update(tea.PasteMsg{Content: "should not paste"})
	if c.Value() != "" {
		t.Errorf("paste while not focused should be ignored, got %q", c.Value())
	}
}

// mockView implements views.View for router tests
type mockView struct {
	id    views.ID
	title string
}

func (m *mockView) Init() tea.Cmd                            { return nil }
func (m *mockView) Update(msg tea.Msg) (views.View, tea.Cmd) { return m, nil }
func (m *mockView) Render() string                           { return m.title + " content" }
func (m *mockView) SetSize(_, _ int)                         {}
func (m *mockView) ID() views.ID                             { return m.id }
func (m *mockView) Title() string                            { return m.title }
