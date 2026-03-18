// Package inputbox provides a text input component for commands and goals.
package inputbox

import (
	"strings"

	"charm.land/lipgloss/v2"
	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Model is the input box component.
type Model struct {
	ctx         *tuictx.ProgramContext
	value       string
	cursorPos   int
	prompt      string
	placeholder string
	focused     bool
	width       int
}

// New creates an input box.
func New(ctx *tuictx.ProgramContext, prompt string) Model {
	return Model{
		ctx:         ctx,
		prompt:      prompt,
		placeholder: "Type a command or goal...",
		width:       40,
	}
}

// Focus gives focus to the input.
func (m *Model) Focus() {
	m.focused = true
}

// Blur removes focus.
func (m *Model) Blur() {
	m.focused = false
}

// IsFocused returns whether the input has focus.
func (m *Model) IsFocused() bool {
	return m.focused
}

// SetValue sets the input value.
func (m *Model) SetValue(s string) {
	m.value = s
	m.cursorPos = len([]rune(s))
}

// Value returns the current input value.
func (m *Model) Value() string {
	return m.value
}

// Reset clears the input.
func (m *Model) Reset() {
	m.value = ""
	m.cursorPos = 0
}

// InsertRune adds a character at the cursor position.
func (m *Model) InsertRune(r rune) {
	runes := []rune(m.value)
	newRunes := make([]rune, 0, len(runes)+1)
	newRunes = append(newRunes, runes[:m.cursorPos]...)
	newRunes = append(newRunes, r)
	newRunes = append(newRunes, runes[m.cursorPos:]...)
	m.value = string(newRunes)
	m.cursorPos++
}

// Backspace deletes the character before the cursor.
func (m *Model) Backspace() {
	if m.cursorPos > 0 {
		runes := []rune(m.value)
		m.value = string(append(runes[:m.cursorPos-1], runes[m.cursorPos:]...))
		m.cursorPos--
	}
}

// Delete deletes the character at the cursor.
func (m *Model) Delete() {
	runes := []rune(m.value)
	if m.cursorPos < len(runes) {
		m.value = string(append(runes[:m.cursorPos], runes[m.cursorPos+1:]...))
	}
}

// CursorLeft moves cursor left.
func (m *Model) CursorLeft() {
	if m.cursorPos > 0 {
		m.cursorPos--
	}
}

// CursorRight moves cursor right.
func (m *Model) CursorRight() {
	if m.cursorPos < len([]rune(m.value)) {
		m.cursorPos++
	}
}

// CursorHome moves cursor to start.
func (m *Model) CursorHome() {
	m.cursorPos = 0
}

// CursorEnd moves cursor to end.
func (m *Model) CursorEnd() {
	m.cursorPos = len([]rune(m.value))
}

// SetWidth sets the input width.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// View renders the input box.
func (m *Model) View() string {
	promptStr := m.ctx.Styles.Input.Prompt.Render(m.prompt)
	promptW := lipgloss.Width(promptStr)

	inputW := m.width - promptW - 2
	if inputW < 1 {
		inputW = 1
	}

	display := m.value
	if display == "" && !m.focused {
		display = m.ctx.Styles.Common.Faint.Render(m.placeholder)
	} else if m.focused {
		runes := []rune(display)
		if m.cursorPos <= len(runes) {
			before := string(runes[:m.cursorPos])
			cursor := m.ctx.Styles.Input.Cursor.Render("_")
			after := ""
			if m.cursorPos < len(runes) {
				after = string(runes[m.cursorPos:])
			}
			display = before + cursor + after
		}
	}

	runes := []rune(display)
	if len(runes) > inputW {
		start := len(runes) - inputW
		if start < 0 {
			start = 0
		}
		display = string(runes[start:])
	}

	inputStr := m.ctx.Styles.Input.Text.Width(inputW).Render(display)
	return promptStr + inputStr
}

// SetPlaceholder sets the placeholder text.
func (m *Model) SetPlaceholder(s string) {
	m.placeholder = s
}

// SetPrompt sets the prompt string.
func (m *Model) SetPrompt(s string) {
	m.prompt = s
}

// UpdateProgramContext updates the shared context.
func (m *Model) UpdateProgramContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// HasSlashCommand checks if the current value starts with '/'.
func (m *Model) HasSlashCommand() bool {
	return strings.HasPrefix(m.value, "/")
}

// SlashCommand returns the slash command name (without '/').
func (m *Model) SlashCommand() string {
	if !m.HasSlashCommand() {
		return ""
	}
	parts := strings.Fields(m.value)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimPrefix(parts[0], "/")
}

// SlashArgs returns the arguments after the slash command.
func (m *Model) SlashArgs() string {
	parts := strings.SplitN(m.value, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
