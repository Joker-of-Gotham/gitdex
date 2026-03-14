package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) updateFileEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	text := msg.Key().Text

	switch {
	case key == "escape" || key == "esc" || msg.Key().Code == tea.KeyEscape:
		m.screen = screenMain
		m.statusMsg = "File editing cancelled"
		return m, nil
	case key == "ctrl+s":
		req := cloneFileWriteInfo(m.fileEditReq)
		if req == nil || strings.TrimSpace(req.Path) == "" {
			m.statusMsg = "No file write request is available to retry"
			return m, nil
		}
		req.Content = m.fileEdit
		if strings.TrimSpace(req.Operation) == "" {
			req.Operation = "update"
		}
		m.screen = screenMain
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Retry edited file write",
			Detail:  req.Path,
		})
		return m, m.executeFileOp(req)
	case key == "backspace":
		m.fileEdit, m.fileCursor = deleteRuneBefore(m.fileEdit, m.fileCursor)
	case key == "delete":
		m.fileEdit, m.fileCursor = deleteRuneAt(m.fileEdit, m.fileCursor)
	case key == "left":
		if m.fileCursor > 0 {
			m.fileCursor--
		}
	case key == "right":
		if m.fileCursor < runeLen(m.fileEdit) {
			m.fileCursor++
		}
	case key == "up":
		m.fileCursor = moveCursorVertical(m.fileEdit, m.fileCursor, -1)
	case key == "down":
		m.fileCursor = moveCursorVertical(m.fileEdit, m.fileCursor, 1)
	case key == "pgup":
		m.fileCursor = moveCursorVertical(m.fileEdit, m.fileCursor, -8)
	case key == "pgdown":
		m.fileCursor = moveCursorVertical(m.fileEdit, m.fileCursor, 8)
	case key == "home" || key == "ctrl+a":
		m.fileCursor = lineStart(m.fileEdit, m.fileCursor)
	case key == "end" || key == "ctrl+e":
		if m.fileEdit == "" {
			m.fileCursor = 0
		} else {
			m.fileCursor = maxInt(0, lineEnd(m.fileEdit, m.fileCursor))
		}
	case key == "enter":
		m.fileEdit, m.fileCursor = insertAtRune(m.fileEdit, m.fileCursor, "\n")
	case key == "tab":
		m.fileEdit, m.fileCursor = insertAtRune(m.fileEdit, m.fileCursor, "  ")
	case key == "ctrl+c":
		return m, tea.Quit
	default:
		if text != "" {
			m.fileEdit, m.fileCursor = insertAtRune(m.fileEdit, m.fileCursor, text)
		}
	}
	return m.syncFileEditor(), nil
}

func (m Model) renderFileEditScreen() string {
	titleStyle := keyStyle().Bold(true)
	hintStyle := mutedStyle()
	panelStyle := panelStyleForStatus("file edit").Padding(0, 1)
	width := maxInt(40, m.width-4)
	height := maxInt(10, m.height-5)
	_, innerHeight := panelInnerSize(panelStyle, width, height)
	before, after := splitAtRune(m.fileEdit, m.fileCursor)
	cursor := "|"
	display := before + cursor + after
	logicalLines := strings.Split(display, "\n")
	viewportHeight := maxInt(4, innerHeight-3)
	if m.fileScroll > len(logicalLines) {
		m.fileScroll = maxInt(0, len(logicalLines)-viewportHeight)
	}
	body := sliceVisibleLines(strings.Join(logicalLines, "\n"), viewportHeight, m.fileScroll)
	content := strings.Join([]string{
		titleStyle.Render("File result editor"),
		hintStyle.Render(fmt.Sprintf("%s  Ctrl+S: rewrite  Esc: cancel  Up/Down: move  PgUp/PgDn: scroll", valueOr(strings.TrimSpace(m.fileTitle), "file write"))),
		body,
		hintStyle.Render(fmt.Sprintf("line %d  scroll %d", cursorLine(m.fileEdit, m.fileCursor)+1, m.fileScroll+1)),
	}, "\n\n")
	return m.padContent(renderBoundedPanel(panelStyle, width, height, content), m.height)
}

func (m Model) handleFileEditPaste(text string) (tea.Model, tea.Cmd) {
	if text == "" {
		return m, nil
	}
	m.fileEdit, m.fileCursor = insertAtRune(m.fileEdit, m.fileCursor, text)
	return m.syncFileEditor(), nil
}
