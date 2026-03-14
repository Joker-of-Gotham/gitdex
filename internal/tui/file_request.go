package tui

import (
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

func cloneFileWriteInfo(info *git.FileWriteInfo) *git.FileWriteInfo {
	if info == nil {
		return nil
	}
	return &git.FileWriteInfo{
		Path:      strings.TrimSpace(info.Path),
		Content:   info.Content,
		Operation: strings.TrimSpace(info.Operation),
		Backup:    info.Backup,
		Lines:     append([]string(nil), info.Lines...),
		LineStart: info.LineStart,
		LineEnd:   info.LineEnd,
	}
}

func (m Model) editableFileRequest() *git.FileWriteInfo {
	if m.lastCommand.ResultKind != resultKindFileWrite || strings.TrimSpace(m.lastCommand.FilePath) == "" {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(m.lastCommand.FileOperation)) {
	case "create", "update":
		return &git.FileWriteInfo{
			Path:      strings.TrimSpace(m.lastCommand.FilePath),
			Content:   m.lastCommand.AfterContent,
			Operation: "update",
			Backup:    strings.TrimSpace(m.lastCommand.BeforeContent) != "",
		}
	case "append":
		return &git.FileWriteInfo{
			Path:      strings.TrimSpace(m.lastCommand.FilePath),
			Content:   m.lastCommand.AfterContent,
			Operation: "update",
			Backup:    true,
		}
	default:
		return nil
	}
}

func (m Model) openFileEdit(req *git.FileWriteInfo) Model {
	if req == nil {
		return m
	}
	m.screen = screenFileEdit
	m.fileEditReq = cloneFileWriteInfo(req)
	m.fileEdit = req.Content
	m.fileCursor = 0
	m.fileScroll = 0
	m.fileTitle = fileEditTitle(req)
	m.statusMsg = "Edit file content, then press Ctrl+S to rewrite the file"
	return m.syncFileEditor()
}

func fileEditTitle(req *git.FileWriteInfo) string {
	if req == nil {
		return "file"
	}
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	if op == "" {
		op = "update"
	}
	return fmt.Sprintf("%s %s", op, strings.TrimSpace(req.Path))
}

func (m Model) syncFileEditor() Model {
	line := cursorLine(m.fileEdit, m.fileCursor)
	viewport := maxInt(4, m.height-7)
	if line < m.fileScroll {
		m.fileScroll = line
	}
	if line >= m.fileScroll+viewport {
		m.fileScroll = line - viewport + 1
	}
	if m.fileScroll < 0 {
		m.fileScroll = 0
	}
	return m
}
