package views

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

const maxEditorBuffers = 20

// EditorBuffer holds metadata for one open file in the multi-buffer editor.
type EditorBuffer struct {
	FilePath   string
	Content    string
	Original   string
	CursorLine int
	CursorCol  int
	Modified   bool
}

type editorMode int

const (
	editorNormal editorMode = iota
	editorInsert
)

// ErrUnsavedClose is returned when CloseBuffer is aborted due to unsaved changes
// and the caller should prompt the user instead.
var ErrUnsavedClose = errors.New("buffer has unsaved changes")

// ExternalEditorDoneMsg is sent when an external $EDITOR process exits.
type ExternalEditorDoneMsg struct {
	Err      error
	TempPath string
	Content  string
}

// EditorView manages multiple file buffers with a shared textarea for the active buffer.
type EditorView struct {
	theme     *theme.Theme
	buffers   []EditorBuffer
	activeIdx int
	width     int
	height    int
	vp        viewport.Model
	showTabs  bool

	ta textarea.Model

	filesEditable         bool
	mode                  editorMode
	closePrompt           bool
	pendingG              bool
	pendingCloseAfterSave bool
	statusOverride        string
}

// NewEditorView creates a multi-buffer editor shell with a configured textarea.
func NewEditorView(t *theme.Theme) *EditorView {
	ta := textarea.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = true
	ta.SetVirtualCursor(true)
	ta.SetStyles(textarea.DefaultStyles(true))
	ta.Blur()

	return &EditorView{
		theme:    t,
		showTabs: true,
		ta:       ta,
		mode:     editorNormal,
	}
}

// Mode returns vim-style normal vs insert mode for the status line.
func (e *EditorView) Mode() editorMode { return e.mode }

// SetFilesEditable mirrors FilesView editable (local clone vs read-only remote).
func (e *EditorView) SetFilesEditable(editable bool) { e.filesEditable = editable }

// Textarea exposes the underlying textarea for tests and advanced callers.
func (e *EditorView) Textarea() *textarea.Model { return &e.ta }

// IsDirty reports whether the active buffer differs from disk baseline.
func (e *EditorView) IsDirty() bool {
	e.syncActiveToBuffer()
	if len(e.buffers) == 0 {
		return false
	}
	return e.buffers[e.activeIdx].Content != e.buffers[e.activeIdx].Original
}

// CloseAllDiscard clears all buffers (e.g. quit editor and discard).
func (e *EditorView) CloseAllDiscard() {
	e.buffers = nil
	e.activeIdx = 0
	e.mode = editorNormal
	e.closePrompt = false
	e.pendingG = false
	e.pendingCloseAfterSave = false
	e.ta.SetValue("")
	e.ta.Blur()
}

// SetMode switches vim-style normal vs insert mode (blur/focus textarea).
func (e *EditorView) SetMode(m editorMode) tea.Cmd {
	e.mode = m
	if m == editorNormal {
		e.ta.Blur()
		return nil
	}
	return tea.Batch(e.ta.Focus(), textarea.Blink)
}

// Focus focuses the textarea (e.g. after opening find input then returning).
func (e *EditorView) Focus() tea.Cmd {
	return e.ta.Focus()
}

// UpdateTextareaKey forwards a key to the textarea and syncs the active buffer.
func (e *EditorView) UpdateTextareaKey(msg tea.KeyPressMsg) tea.Cmd {
	var cmd tea.Cmd
	e.ta, cmd = e.ta.Update(msg)
	e.syncActiveToBuffer()
	return cmd
}

// SetTextareaStyles updates textarea styling (e.g. find highlight on cursor line).
func (e *EditorView) SetTextareaStyles(s textarea.Styles) {
	e.ta.SetStyles(s)
}

// GotoPhysicalLine moves the cursor to the start of a 1-based physical line.
func (e *EditorView) GotoPhysicalLine(line1Based int) {
	lines := strings.Split(e.ta.Value(), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	target := line1Based - 1
	if target < 0 {
		target = 0
	}
	if target >= len(lines) {
		target = len(lines) - 1
	}
	e.ta.MoveToBegin()
	for e.ta.Line() < target {
		e.ta.CursorDown()
	}
	e.ta.CursorStart()
	e.syncActiveToBuffer()
}

// JumpToLineCol moves to 0-based line index and column (for find matches).
func (e *EditorView) JumpToLineCol(line0Based, col int) {
	e.GotoPhysicalLine(line0Based + 1)
	e.ta.SetCursorColumn(col)
	e.syncActiveToBuffer()
}

// SetSize lays out the tab viewport, textarea, and main viewport.
func (e *EditorView) SetSize(w, h int) {
	e.width = w
	e.height = h
	tabH := 1
	if e.showTabs {
		tabH = 2
	}
	contentH := h - tabH - 2 // tabs + status
	if contentH < 3 {
		contentH = 3
	}
	e.ta.SetWidth(max(20, w-4))
	e.ta.SetHeight(max(3, contentH))
	e.vp = viewport.New(viewport.WithWidth(max(20, w-2)), viewport.WithHeight(max(3, contentH)))
}

// OpenBuffer adds a new buffer or switches to an existing path. Returns focus/blink cmds when entering insert is not implied.
func (e *EditorView) OpenBuffer(path, content string) tea.Cmd {
	path = filepath.ToSlash(path)
	for i := range e.buffers {
		if e.buffers[i].FilePath == path {
			e.syncActiveToBuffer()
			e.activeIdx = i
			e.loadBufferIntoTextarea(i)
			e.ta.Blur()
			e.mode = editorNormal
			return nil
		}
	}
	if len(e.buffers) >= maxEditorBuffers {
		e.statusOverride = fmt.Sprintf("Max %d buffers open.", maxEditorBuffers)
		return nil
	}
	e.syncActiveToBuffer()
	c := content
	e.buffers = append(e.buffers, EditorBuffer{
		FilePath: path,
		Content:  c,
		Original: c,
		Modified: false,
	})
	e.activeIdx = len(e.buffers) - 1
	e.ta.SetValue(c)
	e.ta.Blur()
	e.mode = editorNormal
	e.statusOverride = ""
	return tea.Batch(e.ta.Focus(), textarea.Blink)
}

// ActivePath returns the file path of the active buffer.
func (e *EditorView) ActivePath() string {
	if e.activeIdx < 0 || e.activeIdx >= len(e.buffers) {
		return ""
	}
	return e.buffers[e.activeIdx].FilePath
}

// Value returns the current textarea content (active buffer).
func (e *EditorView) Value() string {
	return e.ta.Value()
}

// SetValue replaces the active buffer text and resyncs modification state. Used by tests and post-save refresh.
func (e *EditorView) SetValue(s string) {
	if len(e.buffers) == 0 {
		return
	}
	e.ta.SetValue(s)
	e.buffers[e.activeIdx].Content = s
	e.buffers[e.activeIdx].Modified = e.buffers[e.activeIdx].Content != e.buffers[e.activeIdx].Original
}

// OnFileSaved updates original content after a successful save.
func (e *EditorView) OnFileSaved(path, content string) {
	path = filepath.ToSlash(path)
	closeAfter := e.pendingCloseAfterSave
	e.pendingCloseAfterSave = false
	for i := range e.buffers {
		if e.buffers[i].FilePath == path {
			e.buffers[i].Original = content
			e.buffers[i].Content = content
			e.buffers[i].Modified = false
			if i == e.activeIdx {
				e.ta.SetValue(content)
			}
			if closeAfter {
				e.removeBufferAt(i)
			}
			return
		}
	}
}

// AbortPendingClose clears a pending close-and-save if the save failed.
func (e *EditorView) AbortPendingClose() {
	e.pendingCloseAfterSave = false
}

// HasBuffers reports whether any editor buffers are open.
func (e *EditorView) HasBuffers() bool {
	return len(e.buffers) > 0
}

// CloseBuffer closes a buffer by index. Returns ErrUnsavedClose if modified (caller may use UI prompt first).
func (e *EditorView) CloseBuffer(idx int) error {
	if idx < 0 || idx >= len(e.buffers) {
		return errors.New("invalid buffer index")
	}
	b := e.buffers[idx]
	if b.Content != b.Original {
		return ErrUnsavedClose
	}
	e.removeBufferAt(idx)
	return nil
}

// SaveBuffer returns content to write for the buffer at idx.
func (e *EditorView) SaveBuffer(idx int) (string, error) {
	if idx < 0 || idx >= len(e.buffers) {
		return "", errors.New("invalid buffer index")
	}
	if idx == e.activeIdx {
		e.syncActiveToBuffer()
	}
	return e.buffers[idx].Content, nil
}

func (e *EditorView) syncActiveToBuffer() {
	if e.activeIdx < 0 || e.activeIdx >= len(e.buffers) {
		return
	}
	e.buffers[e.activeIdx].Content = e.ta.Value()
	e.buffers[e.activeIdx].Modified = e.buffers[e.activeIdx].Content != e.buffers[e.activeIdx].Original
	e.buffers[e.activeIdx].CursorLine = e.ta.Line()
	e.buffers[e.activeIdx].CursorCol = e.ta.Column()
}

func (e *EditorView) loadBufferIntoTextarea(idx int) {
	if idx < 0 || idx >= len(e.buffers) {
		return
	}
	e.ta.SetValue(e.buffers[idx].Content)
}

func (e *EditorView) removeBufferAt(idx int) {
	if idx < 0 || idx >= len(e.buffers) {
		return
	}
	e.buffers = append(e.buffers[:idx], e.buffers[idx+1:]...)
	if len(e.buffers) == 0 {
		e.activeIdx = 0
		e.ta.SetValue("")
		return
	}
	if e.activeIdx >= len(e.buffers) {
		e.activeIdx = len(e.buffers) - 1
	} else if idx < e.activeIdx {
		e.activeIdx--
	}
	e.loadBufferIntoTextarea(e.activeIdx)
}

func (e *EditorView) nextBuffer() {
	if len(e.buffers) <= 1 {
		return
	}
	e.syncActiveToBuffer()
	e.activeIdx = (e.activeIdx + 1) % len(e.buffers)
	e.loadBufferIntoTextarea(e.activeIdx)
	e.mode = editorNormal
	e.ta.Blur()
}

func (e *EditorView) prevBuffer() {
	if len(e.buffers) <= 1 {
		return
	}
	e.syncActiveToBuffer()
	e.activeIdx = (e.activeIdx - 1 + len(e.buffers)) % len(e.buffers)
	e.loadBufferIntoTextarea(e.activeIdx)
	e.mode = editorNormal
	e.ta.Blur()
}

func (e *EditorView) closeActiveBuffer() {
	if len(e.buffers) == 0 {
		return
	}
	e.syncActiveToBuffer()
	b := e.buffers[e.activeIdx]
	if b.Content != b.Original {
		e.closePrompt = true
		return
	}
	e.removeBufferAt(e.activeIdx)
}

func (e *EditorView) performCloseDiscard() {
	e.closePrompt = false
	e.removeBufferAt(e.activeIdx)
}

func (e *EditorView) performCloseSave() tea.Cmd {
	e.syncActiveToBuffer()
	path := e.buffers[e.activeIdx].FilePath
	content := e.buffers[e.activeIdx].Content
	e.pendingCloseAfterSave = true
	e.closePrompt = false
	return func() tea.Msg {
		return RequestFileSaveMsg{Path: path, Content: content}
	}
}

// Update handles async editor messages (e.g. external editor finished).
func (e *EditorView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case ExternalEditorDoneMsg:
		if msg.TempPath != "" {
			_ = os.Remove(msg.TempPath)
		}
		if msg.Err != nil {
			e.statusOverride = fmt.Sprintf("External editor: %v", msg.Err)
		} else {
			e.statusOverride = "External editor finished."
		}
		if e.activeIdx >= 0 && e.activeIdx < len(e.buffers) {
			e.ta.SetValue(msg.Content)
			e.syncActiveToBuffer()
		}
		return nil
	}
	return nil
}

// HandleKey routes keys for edit mode. Returns whether the editor consumed the key, remaining command, and whether caller should exit edit mode (no buffers left).
func (e *EditorView) HandleKey(msg tea.KeyPressMsg) (consumed bool, cmd tea.Cmd, exitEdit bool) {
	if e.closePrompt {
		return e.handleClosePromptKey(msg)
	}

	// Multi-buffer / global shortcuts (before mode dispatch)
	s := msg.String()
	if s == "ctrl+n" {
		e.nextBuffer()
		e.pendingG = false
		return true, nil, len(e.buffers) == 0
	}
	if s == "ctrl+p" {
		e.prevBuffer()
		e.pendingG = false
		return true, nil, len(e.buffers) == 0
	}
	if s == "ctrl+w" {
		e.closeActiveBuffer()
		e.pendingG = false
		return true, nil, len(e.buffers) == 0
	}
	if s == "ctrl+e" {
		cmd = e.launchExternalEditor()
		e.pendingG = false
		return true, cmd, false
	}

	// gt / gT (pending "g" must be handled before a lone "g")
	if e.pendingG {
		e.pendingG = false
		if s == "t" {
			e.nextBuffer()
			return true, nil, len(e.buffers) == 0
		}
		if s == "T" {
			e.prevBuffer()
			return true, nil, len(e.buffers) == 0
		}
	}
	if e.mode == editorNormal && s == "g" {
		e.pendingG = true
		return true, nil, false
	}

	if e.mode == editorInsert {
		if s == "ctrl+s" {
			if !e.filesEditable {
				return true, nil, false
			}
			if len(e.buffers) == 0 {
				return true, nil, false
			}
			e.syncActiveToBuffer()
			idx := e.activeIdx
			return true, func() tea.Msg {
				c, err := e.SaveBuffer(idx)
				if err != nil {
					return RequestFileSaveMsg{Path: e.ActivePath(), Content: ""}
				}
				return RequestFileSaveMsg{Path: e.buffers[idx].FilePath, Content: c}
			}, false
		}
		if s == "esc" {
			e.mode = editorNormal
			e.syncActiveToBuffer()
			e.ta.Blur()
			return true, nil, false
		}
		var c tea.Cmd
		e.ta, c = e.ta.Update(msg)
		e.syncActiveToBuffer()
		return true, c, false
	}

	// NORMAL mode
	switch s {
	case "ctrl+s":
		if !e.filesEditable {
			return true, nil, false
		}
		if len(e.buffers) == 0 {
			return true, nil, false
		}
		e.syncActiveToBuffer()
		idx := e.activeIdx
		return true, func() tea.Msg {
			c, err := e.SaveBuffer(idx)
			if err != nil {
				return RequestFileSaveMsg{Path: e.ActivePath(), Content: ""}
			}
			return RequestFileSaveMsg{Path: e.buffers[idx].FilePath, Content: c}
		}, false
	case "esc", "q":
		e.pendingG = false
		return false, nil, false
	case "i", "a":
		e.mode = editorInsert
		return true, tea.Batch(e.ta.Focus(), textarea.Blink), false
	case "h":
		var c tea.Cmd
		e.ta, c = e.ta.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
		e.syncActiveToBuffer()
		return true, c, false
	case "l":
		var c tea.Cmd
		e.ta, c = e.ta.Update(tea.KeyPressMsg{Code: tea.KeyRight})
		e.syncActiveToBuffer()
		return true, c, false
	case "j":
		var c tea.Cmd
		e.ta, c = e.ta.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		e.syncActiveToBuffer()
		return true, c, false
	case "k":
		var c tea.Cmd
		e.ta, c = e.ta.Update(tea.KeyPressMsg{Code: tea.KeyUp})
		e.syncActiveToBuffer()
		return true, c, false
	case "$":
		var c tea.Cmd
		e.ta, c = e.ta.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
		e.syncActiveToBuffer()
		return true, c, false
	case "0":
		var c tea.Cmd
		e.ta, c = e.ta.Update(tea.KeyPressMsg{Code: tea.KeyHome})
		e.syncActiveToBuffer()
		return true, c, false
	}

	if e.pendingG {
		e.pendingG = false
	}
	return false, nil, false
}

func (e *EditorView) handleClosePromptKey(msg tea.KeyPressMsg) (consumed bool, cmd tea.Cmd, exitEdit bool) {
	switch msg.String() {
	case "y", "Y":
		cmd = e.performCloseSave()
		return true, cmd, false
	case "n", "N":
		e.performCloseDiscard()
		return true, nil, len(e.buffers) == 0
	case "c", "C", "esc":
		e.closePrompt = false
		return true, nil, false
	default:
		return true, nil, false
	}
}

func (e *EditorView) launchExternalEditor() tea.Cmd {
	if len(e.buffers) == 0 {
		return nil
	}
	e.syncActiveToBuffer()
	f, err := os.CreateTemp("", "gitdex-edit-*")
	if err != nil {
		e.statusOverride = fmt.Sprintf("Temp file: %v", err)
		return nil
	}
	tmpPath := f.Name()
	if _, err := f.WriteString(e.buffers[e.activeIdx].Content); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		e.statusOverride = fmt.Sprintf("Write temp: %v", err)
		return nil
	}
	_ = f.Close()

	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vim"
		}
	}
	var c *exec.Cmd
	if runtime.GOOS == "windows" && strings.EqualFold(editor, "notepad") {
		c = exec.Command("cmd", "/C", "notepad", tmpPath)
	} else {
		parts := strings.Fields(editor)
		if len(parts) == 0 {
			_ = os.Remove(tmpPath)
			return nil
		}
		args := append(parts[1:], tmpPath)
		c = exec.Command(parts[0], args...)
	}
	fallback := e.buffers[e.activeIdx].Content
	return tea.ExecProcess(c, func(err error) tea.Msg {
		data, rerr := os.ReadFile(tmpPath)
		content := string(data)
		if rerr != nil {
			content = fallback
		}
		return ExternalEditorDoneMsg{Err: err, TempPath: tmpPath, Content: content}
	})
}

// Render draws tabs, textarea, and status line.
func (e *EditorView) Render(theme *theme.Theme, width int) string {
	if width <= 0 {
		return ""
	}
	t := theme
	if t == nil {
		t = e.theme
	}
	var lines []string
	if e.showTabs && len(e.buffers) > 0 {
		lines = append(lines, e.renderTabBar(t, width))
	}
	e.ta.SetWidth(max(20, width-2))
	content := e.ta.View()
	e.vp.SetWidth(max(20, width-2))
	e.vp.SetHeight(max(3, e.ta.Height()))
	e.vp.SetContent(content)
	lines = append(lines, e.vp.View())
	lines = append(lines, e.renderStatusLine(t, width))
	return strings.Join(lines, "\n")
}

func (e *EditorView) renderTabBar(t *theme.Theme, width int) string {
	var parts []string
	for i, b := range e.buffers {
		name := filepath.Base(b.FilePath)
		if name == "" || name == "." {
			name = b.FilePath
		}
		mod := ""
		if b.Content != b.Original {
			mod = "●"
		}
		if b.Content != b.Original {
			mod = "*"
		}
		label := fmt.Sprintf("[%s%s]", mod, name)
		st := lipgloss.NewStyle().Foreground(t.MutedFg())
		if i == e.activeIdx {
			st = st.Bold(true).Foreground(t.Primary()).Background(t.Selection())
		}
		parts = append(parts, st.Render(label))
	}
	line := strings.Join(parts, " ")
	if lipgloss.Width(line) > width {
		line = truncateRunes(line, max(0, width-3)) + "..."
	}
	return line
}

func (e *EditorView) renderStatusLine(t *theme.Theme, width int) string {
	ln, col := 1, 1
	if len(e.buffers) > 0 {
		ln = e.ta.Line() + 1
		col = e.ta.Column() + 1
	}
	mode := "NORMAL"
	if e.mode == editorInsert {
		mode = "INSERT"
	}
	lang := "Plain"
	if p := e.ActivePath(); p != "" {
		lang = languageLabel(p)
	}
	msg := fmt.Sprintf("Ln %d, Col %d | UTF-8 | %s | %s", ln, col, lang, mode)
	if e.closePrompt {
		msg = "Save changes? (y/n/c) — " + msg
	}
	if e.statusOverride != "" {
		msg = e.statusOverride + " — " + msg
	}
	base := fmt.Sprintf("Ln %d, Col %d | UTF-8 | %s | %s", ln, col, lang, mode)
	if e.closePrompt {
		base = "Save changes? (y/n/c) | " + base
	}
	if e.statusOverride != "" {
		base = e.statusOverride + " | " + base
	}
	return lipgloss.NewStyle().Width(width).Foreground(t.DimText()).Render(base)
}

func languageLabel(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "Go"
	case ".rs":
		return "Rust"
	case ".py":
		return "Python"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "JavaScript"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".md", ".mdx":
		return "Markdown"
	case ".json", ".yaml", ".yml":
		return "Data"
	case ".html", ".htm":
		return "HTML"
	case ".css":
		return "CSS"
	default:
		return "Plain"
	}
}

func truncateRunes(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	w := 0
	i := 0
	for i < len(s) {
		r, sz := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && sz == 1 {
			i++
			continue
		}
		rw := lipgloss.Width(string(r))
		if w+rw > maxWidth {
			break
		}
		w += rw
		i += sz
	}
	return s[:i]
}
