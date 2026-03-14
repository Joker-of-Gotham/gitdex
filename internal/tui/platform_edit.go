package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) updatePlatformEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	text := msg.Key().Text

	switch {
	case key == "escape" || key == "esc" || msg.Key().Code == tea.KeyEscape:
		m.screen = screenMain
		m.statusMsg = "Platform request editing cancelled"
		return m, nil
	case key == "ctrl+s":
		op, err := parsePlatformRequest(m.platformEdit)
		if err != nil {
			m.statusMsg = "Invalid platform request JSON: " + err.Error()
			return m, nil
		}
		revision := 1
		if m.lastPlatform != nil && m.lastPlatform.RequestRevision > 0 {
			revision = m.lastPlatform.RequestRevision + 1
		}
		req := platformExecRequest{Op: op, Revision: revision}
		flow := strings.ToLower(strings.TrimSpace(op.Flow))
		if (flow == "validate" || flow == "rollback") && m.lastPlatform != nil && m.lastPlatform.Mutation != nil {
			req.Mutation = cloneMutation(m.lastPlatform.Mutation)
		}
		m.screen = screenMain
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Retry edited platform request",
			Detail:  git.PlatformExecIdentity(op),
		})
		return m.beginPlatformExecution(req, "Executing edited platform action: "+platformActionTitle(op))
	case key == "backspace":
		m.platformEdit, m.platformCursor = deleteRuneBefore(m.platformEdit, m.platformCursor)
	case key == "delete":
		m.platformEdit, m.platformCursor = deleteRuneAt(m.platformEdit, m.platformCursor)
	case key == "left":
		if m.platformCursor > 0 {
			m.platformCursor--
		}
	case key == "right":
		if m.platformCursor < runeLen(m.platformEdit) {
			m.platformCursor++
		}
	case key == "up":
		m.platformCursor = moveCursorVertical(m.platformEdit, m.platformCursor, -1)
	case key == "down":
		m.platformCursor = moveCursorVertical(m.platformEdit, m.platformCursor, 1)
	case key == "pgup":
		m.platformCursor = moveCursorVertical(m.platformEdit, m.platformCursor, -8)
	case key == "pgdown":
		m.platformCursor = moveCursorVertical(m.platformEdit, m.platformCursor, 8)
	case key == "home" || key == "ctrl+a":
		m.platformCursor = lineStart(m.platformEdit, m.platformCursor)
	case key == "end" || key == "ctrl+e":
		if m.platformEdit == "" {
			m.platformCursor = 0
		} else {
			m.platformCursor = maxInt(0, lineEnd(m.platformEdit, m.platformCursor))
		}
	case key == "enter":
		m.platformEdit, m.platformCursor = insertAtRune(m.platformEdit, m.platformCursor, "\n")
	case key == "tab":
		m.platformEdit, m.platformCursor = insertAtRune(m.platformEdit, m.platformCursor, "  ")
	case key == "ctrl+c":
		return m, tea.Quit
	default:
		if text != "" {
			m.platformEdit, m.platformCursor = insertAtRune(m.platformEdit, m.platformCursor, text)
		}
	}
	return m.syncPlatformEditor(), nil
}

func (m Model) renderPlatformEditScreen() string {
	titleStyle := keyStyle().Bold(true)
	hintStyle := mutedStyle()
	panelStyle := panelStyleForStatus("platform edit").Padding(0, 1)
	width := maxInt(40, m.width-4)
	height := maxInt(10, m.height-5)
	_, innerHeight := panelInnerSize(panelStyle, width, height)
	before, after := splitAtRune(m.platformEdit, m.platformCursor)
	cursor := "|"
	display := before + cursor + after
	logicalLines := strings.Split(display, "\n")
	viewportHeight := maxInt(4, innerHeight-3)
	if m.platformScroll > len(logicalLines) {
		m.platformScroll = maxInt(0, len(logicalLines)-viewportHeight)
	}
	body := sliceVisibleLines(strings.Join(logicalLines, "\n"), viewportHeight, m.platformScroll)
	content := strings.Join([]string{
		titleStyle.Render("Platform request editor"),
		hintStyle.Render(fmt.Sprintf("%s  Ctrl+S: run  Esc: cancel  Up/Down: move  PgUp/PgDn: scroll", valueOr(strings.TrimSpace(m.platformTitle), "platform action"))),
		body,
		hintStyle.Render(fmt.Sprintf("line %d  scroll %d", cursorLine(m.platformEdit, m.platformCursor)+1, m.platformScroll+1)),
	}, "\n\n")
	return m.padContent(renderBoundedPanel(panelStyle, width, height, content), m.height)
}

func (m Model) handlePlatformEditPaste(text string) (tea.Model, tea.Cmd) {
	if text == "" {
		return m, nil
	}
	m.platformEdit, m.platformCursor = insertAtRune(m.platformEdit, m.platformCursor, text)
	return m.syncPlatformEditor(), nil
}
