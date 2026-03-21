package views

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type FileEntry struct {
	Path     string
	Name     string
	IsDir    bool
	Size     string
	Mode     string
	Depth    int
	Children []*FileEntry
	Expanded bool
}

type FileTreeDataMsg struct {
	Root *FileEntry
}

type FileContentMsg struct {
	Path    string
	Content string
	// Preview metadata (local files); SizeBytes is full file size on disk.
	SizeBytes int64
	Truncated bool
	IsBinary  bool
	HexDump   string // first 256 bytes as hex when IsBinary
}

type FileDiffMsg struct {
	Diff   string
	Cached bool
}

type FileEditMsg struct {
	Path    string
	Content string
}

type FileSavedMsg struct {
	Path    string
	Content string
	Err     error
}

type filesMode int
type filePromptKind int

const (
	modeTree filesMode = iota
	modeCode
	modeDiff
	modeEdit
)

const (
	filePromptNone filePromptKind = iota
	filePromptCreateFile
	filePromptCreateDir
	filePromptMove
	filePromptDelete
	filePromptBatchRename
	filePromptBatchCopy
	filePromptBatchMove
	filePromptBatchDelete
)

const maxEditorUndoStack = 100

type FilesView struct {
	theme       *theme.Theme
	width       int
	height      int
	root        *FileEntry
	flatList    []*FileEntry
	cursor      int
	scroll      int
	mode        filesMode
	editable    bool
	codeContent string
	codePath    string
	diffContent string
	editor      *EditorView
	editorMode  editorMode
	statusLine  string
	codeVP      viewport.Model
	diffVP      viewport.Model
	repo        RepoListItem
	prompt      textinput.Model
	promptKind  filePromptKind
	promptPath  string
	promptTitle string
	promptHint  string

	// Editor: quit whole editor with unsaved changes (Esc/q), distinct from buffer close prompt
	editorQuitConfirm bool

	undoStack []string
	redoStack []string

	findBarActive    bool
	findCommitted    bool
	findQuery        string
	findMatches      []int
	findMatchOffsets []int
	findIdx          int
	findInput        textinput.Model
	replaceMode      bool
	replaceStr       string
	replaceInput     textinput.Model
	replaceField     int

	gotoLineActive    bool
	gotoInput         textinput.Model
	pendingGotoAfterG bool
	gotoDigitsBuf     string

	diffViewer *DiffViewer
	diffPath   string

	multiSelectMode  bool
	selectedFiles    map[string]bool
	batchPromptPaths []string

	previewSizeBytes     int64
	previewTruncated     bool
	previewIsBinary      bool
	previewHexDump       string
	binaryPreviewShowHex bool
	inspectOnLoad        bool
}

func NewFilesView(t *theme.Theme) *FilesView {
	editor := NewEditorView(t)

	prompt := textinput.New()
	prompt.Prompt = ""
	prompt.CharLimit = 1024

	findIn := textinput.New()
	findIn.Prompt = "/ "
	findIn.CharLimit = 512
	findIn.Placeholder = "search"

	replaceIn := textinput.New()
	replaceIn.Prompt = "→ "
	replaceIn.CharLimit = 512
	replaceIn.Placeholder = "replace"

	gotoIn := textinput.New()
	gotoIn.Prompt = ": "
	gotoIn.CharLimit = 12
	gotoIn.Placeholder = "line"

	return &FilesView{
		theme:        t,
		editor:       editor,
		prompt:       prompt,
		findInput:    findIn,
		replaceInput: replaceIn,
		gotoInput:    gotoIn,
		diffViewer:   NewDiffViewer(t),
	}
}

func (v *FilesView) ID() ID        { return ViewFiles }
func (v *FilesView) Title() string { return "Files" }

func (v *FilesView) SetSize(w, h int) {
	v.width = w
	v.height = h
	vpH := h - 6
	if vpH < 3 {
		vpH = 3
	}
	v.codeVP = viewport.New(viewport.WithWidth(w), viewport.WithHeight(vpH))
	v.diffVP = viewport.New(viewport.WithWidth(w), viewport.WithHeight(vpH))
	v.editor.SetSize(w, h)
	if v.diffViewer != nil {
		v.diffViewer.SetSize(w, h)
	}
	iw := max(20, w-6)
	v.findInput.SetWidth(iw)
	v.replaceInput.SetWidth(iw)
	v.gotoInput.SetWidth(min(40, iw))
}

func (v *FilesView) Init() tea.Cmd { return nil }

func (v *FilesView) SetEditable(editable bool) {
	v.editable = editable
	if v.editor != nil {
		v.editor.SetFilesEditable(editable)
	}
}

func (v *FilesView) Editable() bool {
	return v.editable
}

func (v *FilesView) SetRepository(repo RepoListItem) {
	v.repo = repo
}

// ReloadPathForRefresh returns the path to reload after a tree refresh, and whether to reload a diff instead of file content.
// Paths match those stored on the view (often absolute for local previews). Empty means nothing to reload.
func (v *FilesView) ReloadPathForRefresh() (path string, wantDiff bool) {
	if v.mode == modeEdit {
		return "", false
	}
	if v.mode == modeDiff && v.diffPath != "" {
		return v.diffPath, true
	}
	if v.mode == modeCode && v.codePath != "" {
		return v.codePath, false
	}
	return "", false
}

func (v *FilesView) SetTree(root *FileEntry) {
	v.root = root
	v.rebuildFlatList()
}

func (v *FilesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case FileTreeDataMsg:
		v.root = msg.Root
		v.selectedFiles = nil
		v.rebuildFlatList()
		return v, nil
	case FileContentMsg:
		v.codePath = msg.Path
		v.codeContent = msg.Content
		v.previewSizeBytes = msg.SizeBytes
		v.previewTruncated = msg.Truncated
		v.previewIsBinary = msg.IsBinary
		v.previewHexDump = msg.HexDump
		v.binaryPreviewShowHex = false
		if v.inspectOnLoad && v.mode == modeTree {
			v.mode = modeTree
			v.statusLine = "Preview loaded in inspector. Press e to edit or d to diff."
		} else {
			v.mode = modeCode
			v.statusLine = v.filePreviewStatusLine(msg)
		}
		v.inspectOnLoad = false
		return v, nil
	case BatchFileOpResultMsg:
		if msg.Err != nil {
			v.statusLine = fmt.Sprintf("Batch op failed: %v", msg.Err)
		} else {
			v.statusLine = msg.Message
		}
		v.multiSelectMode = false
		v.selectedFiles = nil
		v.batchPromptPaths = nil
		return v, nil
	case FileDiffMsg:
		v.diffContent = msg.Diff
		v.mode = modeDiff
		if v.diffPath == "" {
			v.diffPath = v.codePath
		}
		if v.diffViewer != nil {
			v.diffViewer.SetShowStaged(msg.Cached)
			v.diffViewer.Load(v.repo.LocalPath(), v.diffPath, msg.Diff)
		}
		return v, nil
	case FileEditMsg:
		v.codePath = msg.Path
		v.codeContent = msg.Content
		v.resetEditorAuxiliaryState()
		cmd := v.editor.OpenBuffer(msg.Path, msg.Content)
		v.mode = modeEdit
		v.editorMode = v.editor.Mode()
		v.statusLine = v.filetreeEditorStatusLine()
		if cmd != nil {
			return v, cmd
		}
		return v, nil
	case ExternalEditorDoneMsg:
		cmd := v.editor.Update(msg)
		return v, cmd
	case FileSavedMsg:
		if msg.Err != nil {
			v.editor.AbortPendingClose()
			v.statusLine = fmt.Sprintf("Save failed: %v", msg.Err)
			return v, nil
		}
		v.codePath = msg.Path
		v.codeContent = msg.Content
		v.editor.OnFileSaved(msg.Path, msg.Content)
		v.editorMode = v.editor.Mode()
		if !v.editor.HasBuffers() {
			v.mode = modeTree
			v.codePath = ""
			v.codeContent = ""
			v.statusLine = "Saved to disk. Editor closed."
			return v, nil
		}
		v.mode = modeCode
		v.statusLine = "Saved to disk."
		return v, nil
	case FileOpResultMsg:
		if msg.Err != nil {
			v.statusLine = fmt.Sprintf("%s failed: %v", v.fileOpLabel(msg.Kind), msg.Err)
			return v, nil
		}
		switch msg.Kind {
		case FileOpMove:
			if v.codePath == msg.Path {
				v.codePath = msg.Target
			}
		case FileOpDelete:
			if v.codePath == msg.Path {
				v.codePath = ""
				v.codeContent = ""
				v.mode = modeTree
			}
		}
		v.statusLine = v.fileOpSuccessMessage(msg)
		v.promptKind = filePromptNone
		v.promptPath = ""
		v.promptTitle = ""
		v.promptHint = ""
		v.prompt.Blur()
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *FilesView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.promptKind != filePromptNone {
		return v.handlePromptKey(msg)
	}
	switch v.mode {
	case modeCode:
		return v.handlePreviewKey(msg)
	case modeDiff:
		return v.handlePreviewKey(msg)
	case modeEdit:
		return v.handleEditorKey(msg)
	default:
		return v.handleTreeKey(msg)
	}
}

func (v *FilesView) handleTreeKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	s := msg.String()
	switch s {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
			v.adjustScroll()
		}
	case "down", "j":
		if v.cursor < len(v.flatList)-1 {
			v.cursor++
			v.adjustScroll()
		}
	case "enter":
		if entry := v.selectedEntry(); entry != nil {
			if entry.IsDir {
				entry.Expanded = !entry.Expanded
				v.rebuildFlatList()
			} else {
				v.inspectOnLoad = true
				v.statusLine = "Loading preview into inspector..."
				return v, func() tea.Msg { return RequestFileContentMsg{Path: entry.Path} }
			}
		}
	case " ":
		if v.multiSelectMode {
			if entry := v.selectedEntry(); entry != nil {
				if entry.IsDir {
					entry.Expanded = !entry.Expanded
					v.rebuildFlatList()
				} else {
					v.toggleSelection(entry.Path)
				}
			}
			return v, nil
		}
		if entry := v.selectedEntry(); entry != nil {
			if entry.IsDir {
				entry.Expanded = !entry.Expanded
				v.rebuildFlatList()
			} else {
				v.inspectOnLoad = true
				v.statusLine = "Loading preview into inspector..."
				return v, func() tea.Msg { return RequestFileContentMsg{Path: entry.Path} }
			}
		}
	case "m":
		v.multiSelectMode = !v.multiSelectMode
		if !v.multiSelectMode {
			v.selectedFiles = nil
			v.statusLine = "Multi-select off"
		} else {
			v.statusLine = "Multi-select: Space toggle  shift+A all  shift+R rename  shift+C copy  shift+M move  shift+D delete  m exit"
		}
		return v, nil
	case "shift+a", "A":
		if v.multiSelectMode {
			v.selectAllFilesInCurrentDir()
			return v, nil
		}
	case "shift+r":
		if v.multiSelectMode {
			paths := v.selectedPathsForBatch()
			if len(paths) == 0 {
				v.statusLine = "Select files first (Space)."
				return v, nil
			}
			if !v.ensureLocalWriteReady() {
				return v, nil
			}
			v.batchPromptPaths = append([]string(nil), paths...)
			return v.openPrompt(filePromptBatchRename, "", "Batch Rename", "Pattern: *.txt -> *.md (use -> between globs)")
		}
	case "shift+c":
		if v.multiSelectMode {
			paths := v.selectedPathsForBatch()
			if len(paths) == 0 {
				v.statusLine = "Select files first (Space)."
				return v, nil
			}
			if !v.ensureLocalWriteReady() {
				return v, nil
			}
			v.batchPromptPaths = append([]string(nil), paths...)
			return v.openPrompt(filePromptBatchCopy, "", "Batch Copy", "Destination directory (relative to repo root):")
		}
	case "shift+m":
		if v.multiSelectMode {
			paths := v.selectedPathsForBatch()
			if len(paths) == 0 {
				v.statusLine = "Select files first (Space)."
				return v, nil
			}
			if !v.ensureLocalWriteReady() {
				return v, nil
			}
			v.batchPromptPaths = append([]string(nil), paths...)
			return v.openPrompt(filePromptBatchMove, "", "Batch Move", "Destination directory (relative to repo root):")
		}
	case "shift+d":
		if v.multiSelectMode {
			paths := v.selectedPathsForBatch()
			if len(paths) == 0 {
				v.statusLine = "Select files first (Space)."
				return v, nil
			}
			if !v.ensureLocalWriteReady() {
				return v, nil
			}
			v.batchPromptPaths = append([]string(nil), paths...)
			return v.openPrompt(filePromptBatchDelete, "", "Batch Delete", "Type DELETE to remove all selected files.")
		}
	case "d":
		if entry := v.selectedEntry(); entry != nil && !entry.IsDir {
			v.diffPath = entry.Path
			return v, func() tea.Msg { return RequestFileDiffMsg{Path: entry.Path} }
		}
	case "e":
		if entry := v.selectedEntry(); entry != nil && !entry.IsDir {
			if !v.editable {
				v.statusLine = "Read-only remote mode. Clone locally to edit."
				return v, nil
			}
			return v, func() tea.Msg { return RequestFileEditMsg{Path: entry.Path} }
		}
	case "c":
		if v.multiSelectMode {
			v.statusLine = "Use shift+C for batch copy, or press m to exit multi-select and use c to clone."
			return v, nil
		}
		if !v.editable {
			return v.requestCloneLocal()
		}
	case "n":
		if !v.ensureLocalWriteReady() {
			return v, nil
		}
		return v.openPrompt(filePromptCreateFile, v.defaultCreatePath(false), "Create File", "Enter a relative path for the new file. It will open in the editor after creation.")
	case "shift+n":
		if !v.ensureLocalWriteReady() {
			return v, nil
		}
		return v.openPrompt(filePromptCreateDir, v.defaultCreatePath(true), "Create Directory", "Enter a relative path for the new directory.")
	case "r":
		if v.multiSelectMode {
			v.statusLine = "Use shift+R for batch rename, or press m to exit multi-select for single rename (r)."
			return v, nil
		}
		if !v.ensureLocalWriteReady() {
			return v, nil
		}
		if entry := v.selectedEntry(); entry != nil && entry.Path != "" {
			return v.openPrompt(filePromptMove, entry.Path, "Rename / Move", "Enter the new relative path for the selected file or directory.")
		}
	case "x":
		if v.multiSelectMode {
			v.statusLine = "Use shift+D for batch delete, or press m to exit multi-select for single delete (x)."
			return v, nil
		}
		if !v.ensureLocalWriteReady() {
			return v, nil
		}
		if entry := v.selectedEntry(); entry != nil && entry.Path != "" {
			return v.openPrompt(filePromptDelete, "", "Delete Entry", "Type delete and press Enter to remove the selected file or directory recursively.")
		}
	case "g":
		v.cursor = 0
		v.scroll = 0
	case "G":
		v.cursor = max(0, len(v.flatList)-1)
		v.adjustScroll()
	}
	return v, nil
}

func (v *FilesView) toggleSelection(relPath string) {
	if v.selectedFiles == nil {
		v.selectedFiles = make(map[string]bool)
	}
	rel := filepath.ToSlash(relPath)
	v.selectedFiles[rel] = !v.selectedFiles[rel]
}

func (v *FilesView) selectedPathsForBatch() []string {
	if len(v.selectedFiles) == 0 {
		return nil
	}
	var out []string
	for p, on := range v.selectedFiles {
		if on {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func (v *FilesView) selectAllFilesInCurrentDir() {
	ent := v.selectedEntry()
	if ent == nil {
		return
	}
	var dir string
	if ent.IsDir {
		dir = ent.Path
	} else {
		dir = filepath.Dir(ent.Path)
	}
	if dir == "." {
		dir = ""
	}
	dir = filepath.ToSlash(dir)
	if v.selectedFiles == nil {
		v.selectedFiles = make(map[string]bool)
	}
	for _, e := range v.flatList {
		if e.IsDir {
			continue
		}
		parent := filepath.ToSlash(filepath.Dir(e.Path))
		if parent == "." {
			parent = ""
		}
		if parent == dir {
			v.selectedFiles[e.Path] = true
		}
	}
}

func (v *FilesView) filePreviewStatusLine(msg FileContentMsg) string {
	var parts []string
	if msg.Path != "" {
		parts = append(parts, msg.Path)
	}
	if msg.SizeBytes > 0 {
		parts = append(parts, fmt.Sprintf("%d bytes", msg.SizeBytes))
	}
	if msg.Truncated {
		parts = append(parts, "preview truncated to 1 MiB")
	}
	if msg.IsBinary {
		parts = append(parts, "binary preview")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " | ")
}

func openExternally(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func (v *FilesView) resetPreviewMeta() {
	v.previewSizeBytes = 0
	v.previewTruncated = false
	v.previewIsBinary = false
	v.previewHexDump = ""
	v.binaryPreviewShowHex = false
}

func (v *FilesView) handlePreviewKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.mode == modeDiff && v.diffViewer != nil {
		s := msg.String()
		if s == "esc" || s == "q" {
			if v.diffViewer.VisualMode() {
				v.diffViewer.SetTheme(v.theme)
				return v, v.diffViewer.Update(msg)
			}
			v.mode = modeTree
			v.diffPath = ""
			return v, nil
		}
		v.diffViewer.SetTheme(v.theme)
		return v, v.diffViewer.Update(msg)
	}

	if v.mode == modeCode && v.previewIsBinary {
		switch msg.String() {
		case "h":
			v.binaryPreviewShowHex = !v.binaryPreviewShowHex
			return v, nil
		case "o":
			if v.codePath == "" {
				return v, nil
			}
			if err := openExternally(v.codePath); err != nil {
				v.statusLine = fmt.Sprintf("Open failed: %v", err)
			} else {
				v.statusLine = "Opened externally."
			}
			return v, nil
		}
	}

	switch msg.String() {
	case "esc", "q":
		v.resetPreviewMeta()
		v.mode = modeTree
		return v, nil
	case "d":
		if v.mode == modeCode && v.codePath != "" {
			v.diffPath = v.codePath
			return v, func() tea.Msg { return RequestFileDiffMsg{Path: v.codePath} }
		}
	case "e":
		if v.mode == modeCode && v.codePath != "" {
			if !v.editable {
				v.statusLine = "Read-only remote mode. Clone locally to edit."
				return v, nil
			}
			return v, func() tea.Msg { return RequestFileEditMsg{Path: v.codePath} }
		}
	case "c":
		if v.multiSelectMode {
			v.statusLine = "Use shift+C for batch copy, or press m to exit multi-select."
			return v, nil
		}
		if !v.editable {
			return v.requestCloneLocal()
		}
	case "n":
		if !v.ensureLocalWriteReady() {
			return v, nil
		}
		return v.openPrompt(filePromptCreateFile, v.defaultCreatePath(false), "Create File", "Enter a relative path for the new file. It will open in the editor after creation.")
	case "shift+n":
		if !v.ensureLocalWriteReady() {
			return v, nil
		}
		return v.openPrompt(filePromptCreateDir, v.defaultCreatePath(true), "Create Directory", "Enter a relative path for the new directory.")
	case "r":
		if v.multiSelectMode {
			v.statusLine = "Use shift+R for batch rename, or press m to exit multi-select."
			return v, nil
		}
		if !v.ensureLocalWriteReady() {
			return v, nil
		}
		if v.codePath != "" {
			return v.openPrompt(filePromptMove, v.codePath, "Rename / Move", "Enter the new relative path for the selected file or directory.")
		}
	case "x":
		if v.multiSelectMode {
			v.statusLine = "Use shift+D for batch delete, or press m to exit multi-select."
			return v, nil
		}
		if !v.ensureLocalWriteReady() {
			return v, nil
		}
		if v.codePath != "" {
			return v.openPrompt(filePromptDelete, "", "Delete Entry", "Type delete and press Enter to remove the selected file or directory recursively.")
		}
	}

	var cmd tea.Cmd
	v.codeVP, cmd = v.codeVP.Update(msg)
	return v, cmd
}

func (v *FilesView) handleEditorKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	s := msg.String()

	if v.editorQuitConfirm {
		switch s {
		case "y", "Y":
			v.editorQuitConfirm = false
			v.mode = modeTree
			v.statusLine = "Edit canceled."
			v.editor.CloseAllDiscard()
			v.resetEditorAuxiliaryState()
			v.editorMode = v.editor.Mode()
			return v, nil
		case "n", "N", "esc":
			v.editorQuitConfirm = false
			v.statusLine = v.filetreeEditorStatusLine()
			return v, nil
		}
		return v, nil
	}

	if v.findBarActive {
		return v.handleFindBarKey(msg)
	}
	if v.replaceMode {
		return v.handleReplaceBarKey(msg)
	}
	if v.gotoLineActive {
		return v.handleGotoLineInputKey(msg)
	}

	if v.pendingGotoAfterG {
		switch s {
		case "esc":
			v.pendingGotoAfterG = false
			v.gotoDigitsBuf = ""
			v.statusLine = v.filetreeEditorStatusLine()
			return v, nil
		case "enter":
			v.pendingGotoAfterG = false
			if v.gotoDigitsBuf == "" {
				v.statusLine = v.filetreeEditorStatusLine()
				return v, nil
			}
			n, err := strconv.Atoi(v.gotoDigitsBuf)
			v.gotoDigitsBuf = ""
			if err != nil || n < 1 {
				v.statusLine = "Invalid line number."
				return v, nil
			}
			v.editor.GotoPhysicalLine(n)
			v.statusLine = fmt.Sprintf("Line %d", n)
			return v, nil
		case "t":
			v.pendingGotoAfterG = false
			v.gotoDigitsBuf = ""
			v.editor.nextBuffer()
			v.editorMode = v.editor.Mode()
			v.statusLine = v.filetreeEditorStatusLine()
			return v, nil
		case "T":
			v.pendingGotoAfterG = false
			v.gotoDigitsBuf = ""
			v.editor.prevBuffer()
			v.editorMode = v.editor.Mode()
			v.statusLine = v.filetreeEditorStatusLine()
			return v, nil
		}
		if len(s) == 1 {
			c := s[0]
			if c >= '0' && c <= '9' {
				v.gotoDigitsBuf += string(c)
				v.statusLine = "Goto line: " + v.gotoDigitsBuf
				return v, nil
			}
		}
		v.pendingGotoAfterG = false
		v.gotoDigitsBuf = ""
		v.statusLine = v.filetreeEditorStatusLine()
		return v.handleEditorKey(msg)
	}

	if v.findCommitted && len(v.findMatchOffsets) > 0 && v.editor.Mode() == editorNormal {
		switch s {
		case "n":
			v.findAdvance(1)
			return v, nil
		case "N":
			v.findAdvance(-1)
			return v, nil
		}
	}

	switch s {
	case "ctrl+z":
		v.editorUndo()
		v.statusLine = "Undo"
		return v, nil
	case "ctrl+y":
		v.editorRedo()
		v.statusLine = "Redo"
		return v, nil
	}

	if v.editor.Mode() == editorNormal && !v.editor.closePrompt {
		switch s {
		case "/":
			v.findBarActive = true
			v.findCommitted = false
			v.findInput.SetValue(v.findQuery)
			v.statusLine = "Find (Enter search, Esc cancel)"
			return v, v.findInput.Focus()
		case "ctrl+h":
			v.replaceMode = true
			v.replaceField = 0
			v.findInput.SetValue(v.findQuery)
			v.replaceInput.SetValue(v.replaceStr)
			v.statusLine = "Find & replace: Tab fields | r replace | a all | s skip | Esc"
			v.replaceInput.Blur()
			return v, v.findInput.Focus()
		case "ctrl+g":
			v.gotoLineActive = true
			v.gotoInput.SetValue("")
			v.statusLine = "Go to line (Enter)"
			return v, v.gotoInput.Focus()
		case "g":
			v.pendingGotoAfterG = true
			v.gotoDigitsBuf = ""
			v.statusLine = "Goto line: digits + Enter | t/T buffer | Esc"
			return v, nil
		}
	}

	before := v.editor.Value()
	if v.editor.Mode() == editorInsert && !v.isEditorNavigationKey(msg) && s != "ctrl+z" && s != "ctrl+y" && s != "esc" && s != "ctrl+s" {
		v.pushUndoBeforeChange()
	}

	consumed, cmd, exitAll := v.editor.HandleKey(msg)
	v.editorMode = v.editor.Mode()
	if v.editor.Mode() == editorInsert && v.editor.Value() == before {
		v.popUndoIfUnchanged(before)
	}

	if v.findCommitted && len(v.findMatchOffsets) > 0 {
		v.applyEditorFindHighlight()
	} else {
		v.editor.SetTextareaStyles(textarea.DefaultStyles(v.theme.IsDark))
	}

	if msg.String() == "ctrl+s" && v.editable {
		v.statusLine = "Saving..."
	}
	if exitAll {
		v.mode = modeTree
		v.codePath = ""
		v.codeContent = ""
		v.resetEditorAuxiliaryState()
		v.statusLine = "All buffers closed."
		return v, cmd
	}
	if cmd != nil {
		return v, cmd
	}
	if !consumed {
		switch msg.String() {
		case "esc", "q":
			if v.editor.IsDirty() {
				v.editorQuitConfirm = true
				v.statusLine = "Discard unsaved changes? [y/N]"
				return v, nil
			}
			v.mode = modeCode
			v.statusLine = "Edit canceled."
			v.editor.CloseAllDiscard()
			v.resetEditorAuxiliaryState()
			v.editorMode = v.editor.Mode()
			return v, nil
		}
	}
	return v, nil
}

func (v *FilesView) resetEditorAuxiliaryState() {
	v.editorQuitConfirm = false
	v.undoStack = nil
	v.redoStack = nil
	v.findBarActive = false
	v.findCommitted = false
	v.findQuery = ""
	v.findMatches = nil
	v.findMatchOffsets = nil
	v.findIdx = 0
	v.replaceMode = false
	v.replaceStr = ""
	v.replaceField = 0
	v.findInput.Blur()
	v.replaceInput.Blur()
	v.gotoLineActive = false
	v.gotoInput.Blur()
	v.pendingGotoAfterG = false
	v.gotoDigitsBuf = ""
}

func (v *FilesView) filetreeEditorStatusLine() string {
	return "NORMAL: i/a insert  / find  Ctrl+H replace  Ctrl+G goto  g+digits  Ctrl+Z/Y  Ctrl+S save  Ctrl+N/P buffers  gt/gT  Ctrl+W  Ctrl+E  q exit"
}

func (v *FilesView) trimStringStack(st *[]string, max int) {
	if len(*st) <= max {
		return
	}
	*st = (*st)[len(*st)-max:]
}

func (v *FilesView) pushUndoBeforeChange() {
	cur := v.editor.Value()
	if len(v.undoStack) > 0 && v.undoStack[len(v.undoStack)-1] == cur {
		return
	}
	v.undoStack = append(v.undoStack, cur)
	v.trimStringStack(&v.undoStack, maxEditorUndoStack)
	v.redoStack = nil
}

func (v *FilesView) popUndoIfUnchanged(before string) {
	if len(v.undoStack) == 0 {
		return
	}
	if v.undoStack[len(v.undoStack)-1] == before {
		v.undoStack = v.undoStack[:len(v.undoStack)-1]
	}
}

func (v *FilesView) editorUndo() {
	if len(v.undoStack) == 0 {
		return
	}
	cur := v.editor.Value()
	prev := v.undoStack[len(v.undoStack)-1]
	v.undoStack = v.undoStack[:len(v.undoStack)-1]
	v.redoStack = append(v.redoStack, cur)
	v.trimStringStack(&v.redoStack, maxEditorUndoStack)
	v.editor.SetValue(prev)
	v.editorMode = v.editor.Mode()
}

func (v *FilesView) editorRedo() {
	if len(v.redoStack) == 0 {
		return
	}
	cur := v.editor.Value()
	next := v.redoStack[len(v.redoStack)-1]
	v.redoStack = v.redoStack[:len(v.redoStack)-1]
	v.undoStack = append(v.undoStack, cur)
	v.trimStringStack(&v.undoStack, maxEditorUndoStack)
	v.editor.SetValue(next)
	v.editorMode = v.editor.Mode()
}

func (v *FilesView) isEditorNavigationKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "up", "down", "left", "right",
		"pgup", "pgdown", "home", "end",
		"ctrl+home", "ctrl+end",
		"alt+up", "alt+down", "alt+left", "alt+right",
		"tab", "shift+tab":
		return true
	}
	return false
}

func (v *FilesView) runFind(query string) {
	content := v.editor.Value()
	v.findMatches = nil
	v.findMatchOffsets = nil
	v.findIdx = 0
	if query == "" {
		v.statusLine = "Empty search"
		return
	}
	pos := 0
	for {
		idx := strings.Index(content[pos:], query)
		if idx < 0 {
			break
		}
		off := pos + idx
		v.findMatchOffsets = append(v.findMatchOffsets, off)
		v.findMatches = append(v.findMatches, strings.Count(content[:off], "\n")+1)
		pos = off + len(query)
		if len(query) == 0 {
			break
		}
	}
	if len(v.findMatchOffsets) == 0 {
		v.findIdx = -1
		v.statusLine = "No matches for: " + query
		return
	}
	v.findIdx = 0
	v.editorJumpToFindMatch()
	v.statusLine = fmt.Sprintf("Find %d/%d", v.findIdx+1, len(v.findMatchOffsets))
}

func offsetToLineCol(content string, offset int) (line, col int) {
	if offset < 0 || offset > len(content) {
		return 0, 0
	}
	before := content[:offset]
	line = strings.Count(before, "\n")
	lastNL := strings.LastIndex(before, "\n")
	if lastNL < 0 {
		col = offset
	} else {
		col = offset - lastNL - 1
	}
	return line, col
}

func (v *FilesView) editorJumpToFindMatch() {
	if v.findIdx < 0 || v.findIdx >= len(v.findMatchOffsets) {
		return
	}
	off := v.findMatchOffsets[v.findIdx]
	line, col := offsetToLineCol(v.editor.Value(), off)
	v.editor.JumpToLineCol(line, col)
}

func (v *FilesView) findAdvance(delta int) {
	if len(v.findMatchOffsets) == 0 {
		return
	}
	v.findIdx += delta
	n := len(v.findMatchOffsets)
	for v.findIdx < 0 {
		v.findIdx += n
	}
	v.findIdx %= n
	v.editorJumpToFindMatch()
	v.applyEditorFindHighlight()
	v.statusLine = fmt.Sprintf("Find %d/%d", v.findIdx+1, len(v.findMatchOffsets))
}

func (v *FilesView) applyEditorFindHighlight() {
	st := textarea.DefaultStyles(v.theme.IsDark)
	if v.findCommitted && len(v.findMatchOffsets) > 0 {
		hl := lipgloss.NewStyle().Background(v.theme.Selection())
		st.Focused.CursorLine = st.Focused.CursorLine.Inherit(hl)
		st.Focused.CursorLineNumber = st.Focused.CursorLineNumber.Inherit(hl)
	}
	v.editor.SetTextareaStyles(st)
}

func findMatchLinesSummary(lines []int, max int) string {
	if len(lines) == 0 {
		return ""
	}
	seen := make(map[int]struct{})
	var uniq []int
	for _, ln := range lines {
		if _, ok := seen[ln]; ok {
			continue
		}
		seen[ln] = struct{}{}
		uniq = append(uniq, ln)
		if len(uniq) >= max {
			break
		}
	}
	var b strings.Builder
	for i, ln := range uniq {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Itoa(ln))
	}
	if len(uniq) >= max {
		clean := make([]string, 0, len(uniq))
		for _, ln := range uniq {
			clean = append(clean, strconv.Itoa(ln))
		}
		return strings.Join(clean, ", ") + " ..."
	}
	return b.String()
}

func (v *FilesView) replaceCurrentMatch() {
	q := v.findQuery
	if q == "" || v.findIdx < 0 || v.findIdx >= len(v.findMatchOffsets) {
		return
	}
	v.pushUndoBeforeChange()
	content := v.editor.Value()
	off := v.findMatchOffsets[v.findIdx]
	if off+len(q) > len(content) || content[off:off+len(q)] != q {
		v.runFind(q)
		return
	}
	newContent := content[:off] + v.replaceStr + content[off+len(q):]
	v.editor.SetValue(newContent)
	v.runFind(q)
	if len(v.findMatchOffsets) == 0 {
		v.findIdx = -1
		return
	}
	if v.findIdx >= len(v.findMatchOffsets) {
		v.findIdx = len(v.findMatchOffsets) - 1
	}
}

func (v *FilesView) replaceAllMatches() {
	q := v.findQuery
	if q == "" {
		return
	}
	v.pushUndoBeforeChange()
	newContent := strings.ReplaceAll(v.editor.Value(), q, v.replaceStr)
	v.editor.SetValue(newContent)
	v.runFind(q)
}

func (v *FilesView) handleFindBarKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.findBarActive = false
		v.findCommitted = false
		v.findMatches = nil
		v.findMatchOffsets = nil
		v.statusLine = v.filetreeEditorStatusLine()
		return v, v.editor.Focus()
	case "enter":
		v.findQuery = strings.TrimSpace(v.findInput.Value())
		v.findBarActive = false
		v.findCommitted = true
		v.runFind(v.findQuery)
		return v, v.editor.Focus()
	}
	var cmd tea.Cmd
	v.findInput, cmd = v.findInput.Update(msg)
	return v, cmd
}

func (v *FilesView) handleReplaceBarKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.replaceMode = false
		v.statusLine = v.filetreeEditorStatusLine()
		return v, v.editor.Focus()
	case "tab":
		if v.replaceField == 0 {
			v.replaceField = 1
			v.findInput.Blur()
			return v, v.replaceInput.Focus()
		}
		v.replaceField = 0
		v.replaceInput.Blur()
		return v, v.findInput.Focus()
	case "r":
		v.findQuery = v.findInput.Value()
		v.replaceStr = v.replaceInput.Value()
		v.replaceCurrentMatch()
		v.statusLine = "Replaced match"
		return v, v.editor.Focus()
	case "a":
		v.findQuery = v.findInput.Value()
		v.replaceStr = v.replaceInput.Value()
		v.replaceAllMatches()
		v.statusLine = "Replace all done"
		return v, v.editor.Focus()
	case "s":
		v.findQuery = v.findInput.Value()
		v.replaceStr = v.replaceInput.Value()
		if len(v.findMatchOffsets) > 0 {
			v.findAdvance(1)
		} else {
			v.runFind(v.findQuery)
		}
		v.statusLine = "Skipped"
		return v, v.editor.Focus()
	}
	if v.replaceField == 0 {
		var cmd tea.Cmd
		v.findInput, cmd = v.findInput.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.replaceInput, cmd = v.replaceInput.Update(msg)
	return v, cmd
}

func (v *FilesView) handleGotoLineInputKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.gotoLineActive = false
		v.statusLine = v.filetreeEditorStatusLine()
		return v, v.editor.Focus()
	case "enter":
		v.gotoLineActive = false
		s := strings.TrimSpace(v.gotoInput.Value())
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			v.statusLine = "Invalid line number."
			return v, v.editor.Focus()
		}
		v.editor.GotoPhysicalLine(n)
		v.statusLine = fmt.Sprintf("Line %d", n)
		return v, v.editor.Focus()
	}
	var cmd tea.Cmd
	v.gotoInput, cmd = v.gotoInput.Update(msg)
	return v, cmd
}

func (v *FilesView) handlePromptKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.promptKind = filePromptNone
		v.promptPath = ""
		v.promptTitle = ""
		v.promptHint = ""
		v.batchPromptPaths = nil
		v.prompt.Blur()
		v.statusLine = "File operation canceled."
		return v, nil
	case "enter":
		target := strings.TrimSpace(v.prompt.Value())
		switch v.promptKind {
		case filePromptCreateFile, filePromptCreateDir, filePromptMove:
			if target == "" {
				v.statusLine = "Path cannot be empty."
				return v, nil
			}
			msg := RequestFileOpMsg{Kind: v.promptOpKind(), Path: v.promptPath, Target: filepath.ToSlash(target)}
			return v, func() tea.Msg { return msg }
		case filePromptDelete:
			if !strings.EqualFold(target, "delete") {
				v.statusLine = "Type delete to confirm removal."
				return v, nil
			}
			msg := RequestFileOpMsg{Kind: FileOpDelete, Path: v.promptPath}
			return v, func() tea.Msg { return msg }
		case filePromptBatchRename:
			if target == "" {
				v.statusLine = "Pattern cannot be empty."
				return v, nil
			}
			paths := append([]string(nil), v.batchPromptPaths...)
			v.batchPromptPaths = nil
			return v, func() tea.Msg {
				return RequestBatchFileOpMsg{Kind: "rename", Paths: paths, Pattern: target}
			}
		case filePromptBatchCopy, filePromptBatchMove:
			if target == "" {
				v.statusLine = "Destination cannot be empty."
				return v, nil
			}
			kind := "copy"
			if v.promptKind == filePromptBatchMove {
				kind = "move"
			}
			paths := append([]string(nil), v.batchPromptPaths...)
			v.batchPromptPaths = nil
			return v, func() tea.Msg {
				return RequestBatchFileOpMsg{Kind: kind, Paths: paths, TargetDir: filepath.ToSlash(target)}
			}
		case filePromptBatchDelete:
			if !strings.EqualFold(target, "DELETE") {
				v.statusLine = "Type DELETE to confirm batch delete."
				return v, nil
			}
			paths := append([]string(nil), v.batchPromptPaths...)
			v.batchPromptPaths = nil
			return v, func() tea.Msg {
				return RequestBatchFileOpMsg{Kind: "delete", Paths: paths}
			}
		}
	}

	var cmd tea.Cmd
	v.prompt, cmd = v.prompt.Update(msg)
	return v, cmd
}

func (v *FilesView) openPrompt(kind filePromptKind, initialValue, title, hint string) (View, tea.Cmd) {
	v.promptKind = kind
	v.promptPath = filepath.ToSlash(initialValue)
	if kind == filePromptDelete {
		if entry := v.selectedEntry(); entry != nil && entry.Path != "" && v.mode == modeTree {
			v.promptPath = entry.Path
		} else if v.codePath != "" {
			v.promptPath = v.codePath
		}
	}
	v.promptTitle = title
	v.promptHint = hint
	v.prompt.SetValue(initialValue)
	switch kind {
	case filePromptDelete:
		v.prompt.SetValue("")
		v.prompt.Placeholder = "delete"
	case filePromptBatchDelete:
		v.prompt.SetValue("")
		v.prompt.Placeholder = "DELETE"
	case filePromptBatchRename:
		v.prompt.Placeholder = "*.txt -> *.md"
	case filePromptBatchCopy, filePromptBatchMove:
		v.prompt.Placeholder = "dest/subdir"
	default:
		v.prompt.Placeholder = initialValue
	}
	v.statusLine = title
	return v, v.prompt.Focus()
}

func (v *FilesView) ensureLocalWriteReady() bool {
	if v.editable {
		return true
	}
	v.statusLine = "Local clone required for write operations. Press c to clone locally."
	return false
}

func (v *FilesView) requestCloneLocal() (View, tea.Cmd) {
	if v.repo.FullName == "" {
		v.statusLine = "No repository context is active."
		return v, nil
	}
	return v, func() tea.Msg {
		return CloneRepoRequestMsg{Repo: v.repo}
	}
}

func (v *FilesView) defaultCreatePath(directory bool) string {
	base := ""
	if entry := v.selectedEntry(); entry != nil {
		if entry.IsDir {
			base = entry.Path
		} else {
			base = filepath.Dir(entry.Path)
			if base == "." {
				base = ""
			}
		}
	} else if v.codePath != "" {
		base = filepath.Dir(v.codePath)
		if base == "." {
			base = ""
		}
	}
	base = filepath.ToSlash(strings.TrimSpace(base))
	if base == "" || base == "." {
		return ""
	}
	if directory || strings.HasSuffix(base, "/") {
		return base + "/"
	}
	return base + "/"
}

func (v *FilesView) promptOpKind() FileOpKind {
	switch v.promptKind {
	case filePromptCreateFile:
		return FileOpCreateFile
	case filePromptCreateDir:
		return FileOpCreateDir
	case filePromptMove:
		return FileOpMove
	default:
		return ""
	}
}

func (v *FilesView) fileOpLabel(kind FileOpKind) string {
	switch kind {
	case FileOpCreateFile:
		return "Create file"
	case FileOpCreateDir:
		return "Create directory"
	case FileOpMove:
		return "Move"
	case FileOpDelete:
		return "Delete"
	default:
		return "File operation"
	}
}

func (v *FilesView) fileOpSuccessMessage(msg FileOpResultMsg) string {
	switch msg.Kind {
	case FileOpCreateFile:
		return "Created file " + msg.Target
	case FileOpCreateDir:
		return "Created directory " + msg.Target
	case FileOpMove:
		return "Moved " + msg.Path + " -> " + msg.Target
	case FileOpDelete:
		return "Deleted " + msg.Path
	default:
		return "File operation completed."
	}
}

func (v *FilesView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}
	var body string
	if v.mode == modeTree {
		body = v.renderTreeOnly(v.width)
	} else if v.width >= 100 {
		body = v.renderWorkbench()
	} else {
		switch v.mode {
		case modeCode:
			body = v.renderCodePanel(v.width)
		case modeDiff:
			body = v.renderDiffPanel(v.width)
		case modeEdit:
			body = v.renderEditorPanel(v.width)
		default:
			body = v.renderTreeOnly(v.width)
		}
	}
	if v.promptKind != filePromptNone {
		return body + "\n\n" + v.renderPromptPanel(v.width)
	}
	return body
}

func (v *FilesView) renderWorkbench() string {
	listWidth := max(32, v.width*42/100)
	previewWidth := max(42, v.width-listWidth-1)
	left := lipgloss.NewStyle().
		Width(listWidth).
		PaddingRight(1).
		Render(v.renderTreeOnly(listWidth))
	right := lipgloss.NewStyle().
		Width(previewWidth).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(v.theme.Divider()).
		PaddingLeft(1).
		Render(v.renderPreview(previewWidth))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (v *FilesView) renderTreeOnly(width int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render(theme.Icons.Folder + " File Explorer")
	hint := lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Up/Down  Enter inspect  Space select  m multi  shift+n dir  n file  shift+R/C/M/D batch  d diff  e edit  r move  x delete  c clone")
	lines := []string{title, hint, ""}

	if len(v.flatList) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("No files loaded. Repository tree will appear here once a repo is selected."))
		return strings.Join(lines, "\n")
	}

	visible := max(4, v.height-6)
	for i := v.scroll; i < len(v.flatList) && i < v.scroll+visible; i++ {
		entry := v.flatList[i]
		prefix := "  "
		if i == v.cursor {
			prefix = theme.Icons.ChevronRight + " "
		}
		icon := fileIconForName(entry.Name)
		if entry.IsDir {
			if entry.Expanded {
				icon = theme.Icons.FolderOpen
			} else {
				icon = theme.Icons.Folder
			}
		}
		indent := strings.Repeat("  ", entry.Depth)
		name := entry.Name
		if entry.IsDir {
			name += "/"
		}
		selMark := "  "
		if v.multiSelectMode && !entry.IsDir && v.selectedFiles[filepath.ToSlash(entry.Path)] {
			selMark = "* "
		}
		line := prefix + indent + selMark + icon + " " + name
		if entry.Size != "" && !entry.IsDir {
			line += "  " + entry.Size
		}
		if i == v.cursor {
			lines = append(lines, render.FillBlock(line, max(20, width-2), lipgloss.NewStyle().
				Bold(true).
				Foreground(v.theme.Fg()).
				Background(v.theme.Selection())))
			continue
		}
		color := v.theme.Fg()
		if entry.IsDir {
			color = v.theme.Secondary()
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(color).Render(line))
	}
	return strings.Join(lines, "\n")
}

func (v *FilesView) renderPreview(width int) string {
	switch v.mode {
	case modeCode:
		return v.renderCodePanel(width)
	case modeDiff:
		return v.renderDiffPanel(width)
	case modeEdit:
		return v.renderEditorPanel(width)
	default:
		return v.renderSelectionPanel(width)
	}
}

func (v *FilesView) renderSelectionPanel(width int) string {
	entry := v.selectedEntry()
	title := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Secondary()).Render(theme.Icons.Eye + " Preview")
	if entry == nil {
		return title + "\n" + lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Select a file to inspect metadata. Press Enter to load preview into the inspector.")
	}

	lines := []string{
		title,
		"",
		lipgloss.NewStyle().Bold(true).Foreground(v.theme.Fg()).Render(entry.Path),
		lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Type: " + ternary(entry.IsDir, "directory", "file")),
	}
	if !entry.IsDir {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Inspect: Enter"),
			lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Diff: d"),
			lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Edit: "+ternary(v.editable, "e", "c to clone locally")),
			lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Ops: n file  m dir  r move  x delete"),
			lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Detail surface: inspector"),
		)
	} else {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Toggle expansion with Enter"),
			lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Ops: n file  m dir  r move  x delete"),
		)
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Repository mode: "+ternary(v.editable, "local writable", "remote read-only")))
	if len(v.repo.LocalPaths) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Local path: "+strings.Join(v.repo.LocalPaths, ", ")))
	}
	if v.statusLine != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(v.theme.Warning()).Render(v.statusLine))
	}
	return strings.Join(lines, "\n")
}

func (v *FilesView) renderCodePanel(width int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render(theme.Icons.FileCode + " " + v.codePath)
	hint := lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("PgUp/PgDn scroll  d diff  e edit  n file  shift+n dir  r move  x delete  Esc back")
	if v.previewIsBinary {
		hint = lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("h hex preview  o open externally  Esc back")
	}

	var content string
	if v.previewIsBinary {
		if v.binaryPreviewShowHex && v.previewHexDump != "" {
			content = v.previewHexDump
		} else {
			content = lipgloss.NewStyle().Foreground(v.theme.Fg()).Render(v.codeContent) + "\n\n" +
				lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render("Open externally (o) or hex preview (h).")
		}
	} else {
		ext := strings.ToLower(filepath.Ext(v.codePath))
		base := strings.ToLower(filepath.Base(v.codePath))
		raw := v.codeContent
		if ext == ".md" || ext == ".markdown" || base == "readme" {
			content = render.Markdown(raw, max(40, width-6))
		} else {
			vpH := max(3, v.height-8)
			content = render.CodeChunked(raw, v.codePath, max(40, width-6), v.theme, v.codeVP.ScrollPercent(), vpH)
		}
	}

	v.codeVP.SetWidth(max(20, width-2))
	v.codeVP.SetHeight(max(3, v.height-8))
	v.codeVP.SetContent(content)
	lines := []string{title, hint}
	if v.statusLine != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.Success()).Render(v.statusLine))
	}
	lines = append(lines, "", v.codeVP.View())
	return strings.Join(lines, "\n")
}

func (v *FilesView) renderDiffPanel(width int) string {
	if v.diffViewer != nil {
		v.diffViewer.SetTheme(v.theme)
		return v.diffViewer.Render(width, v.height)
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render(theme.Icons.Diff + " Diff")
	hint := lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("PgUp/PgDn scroll  n file  m dir  r move  x delete  Esc back")
	content := v.diffContent
	if strings.TrimSpace(content) == "" {
		content = "No diff available."
	} else {
		content = render.Diff(content, v.theme)
	}
	v.diffVP.SetWidth(max(20, width-2))
	v.diffVP.SetHeight(max(3, v.height-8))
	v.diffVP.SetContent(content)
	return strings.Join([]string{title, hint, "", v.diffVP.View()}, "\n")
}

func (v *FilesView) renderEditorPanel(width int) string {
	path := v.codePath
	if ap := v.editor.ActivePath(); ap != "" {
		path = ap
	}
	mod := ""
	if v.editor.IsDirty() {
		mod = lipgloss.NewStyle().Foreground(v.theme.Warning()).Render(" ● [modified] ")
	}
	if v.editor.IsDirty() {
		mod = lipgloss.NewStyle().Foreground(v.theme.Warning()).Render(" [modified]")
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render(theme.Icons.FileCode+" Edit "+path) + mod
	hint := lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("i/a insert  / find  Ctrl+H replace  Ctrl+G goto  Ctrl+Z/Y  Ctrl+S save  buffers  Ctrl+W  Ctrl+E  q exit")
	lines := []string{title, hint}
	if v.statusLine != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.Warning()).Render(v.statusLine))
	}
	if v.editorQuitConfirm {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(v.theme.Danger()).Render("Unsaved changes — y discard  n stay"))
	}
	if v.editorQuitConfirm && len(lines) > 0 {
		lines[len(lines)-1] = lipgloss.NewStyle().Bold(true).Foreground(v.theme.Danger()).Render("Unsaved changes | y discard | n stay")
	}
	if v.findCommitted && len(v.findMatches) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.Info()).Render("Match lines: "+findMatchLinesSummary(v.findMatches, 12)))
	}
	lines = append(lines, "", v.editor.Render(v.theme, width))
	if v.findBarActive {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Find:"), v.findInput.View())
	}
	if v.replaceMode {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Find & replace (Tab)  r  a  s skip  Esc"),
			v.findInput.View(),
			v.replaceInput.View(),
		)
	}
	if v.gotoLineActive {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Go to line:"), v.gotoInput.View())
	}
	return strings.Join(lines, "\n")
}

func (v *FilesView) renderPromptPanel(width int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary()).Render(v.promptTitle)
	lines := []string{title}
	if v.promptPath != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Target: "+v.promptPath))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.MutedFg()).Render(v.promptHint))
	lines = append(lines, "")
	lines = append(lines, v.prompt.View())
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(v.theme.DimText()).Render("Enter confirm  Esc cancel"))
	return render.SurfacePanel(strings.Join(lines, "\n"), max(32, width), v.theme.Surface(), v.theme.BorderColor())
}

func (v *FilesView) selectedEntry() *FileEntry {
	if v.cursor >= 0 && v.cursor < len(v.flatList) {
		return v.flatList[v.cursor]
	}
	return nil
}

func (v *FilesView) rebuildFlatList() {
	v.flatList = nil
	if v.root == nil {
		return
	}
	v.flattenEntry(v.root, 0)
}

func (v *FilesView) flattenEntry(entry *FileEntry, depth int) {
	entry.Depth = depth
	v.flatList = append(v.flatList, entry)
	if entry.IsDir && entry.Expanded {
		sorted := make([]*FileEntry, len(entry.Children))
		copy(sorted, entry.Children)
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].IsDir != sorted[j].IsDir {
				return sorted[i].IsDir
			}
			return sorted[i].Name < sorted[j].Name
		})
		for _, child := range sorted {
			v.flattenEntry(child, depth+1)
		}
	}
}

func (v *FilesView) adjustScroll() {
	visible := max(4, v.height-6)
	if v.cursor < v.scroll {
		v.scroll = v.cursor
	}
	if v.cursor >= v.scroll+visible {
		v.scroll = v.cursor - visible + 1
	}
}

type RequestFileContentMsg struct {
	Path string
}

type RequestFileEditMsg struct {
	Path string
}

type RequestFileDiffMsg struct {
	Path   string
	Cached bool // when true, load git diff --cached
}

// RequestApplyGitPatchMsg applies a unified diff hunk to the index (stage/unstage).
type RequestApplyGitPatchMsg struct {
	RepoPath string
	FilePath string
	Patch    string
	Reverse  bool
	Cached   bool // reload diff with same staged/unstaged mode after apply
}

// RequestGitStageFileMsg stages or unstages the entire file (git add / git restore --staged).
type RequestGitStageFileMsg struct {
	Path    string
	Unstage bool
	Cached  bool // reload diff with git diff --cached when true
}

type RequestFileSaveMsg struct {
	Path    string
	Content string
}

func fileIconForName(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "_test.go") {
		return theme.Icons.FileTest
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".rb", ".java", ".c", ".h", ".cpp", ".hpp", ".cs":
		return theme.Icons.FileCode
	case ".md", ".mdx":
		return theme.Icons.FileDoc
	case ".yaml", ".yml", ".json", ".toml", ".mod", ".sum", ".env", ".ini", ".cfg":
		return theme.Icons.FileConfig
	default:
		return theme.Icons.FileCode
	}
}

func BuildFileTree(entries []string) *FileEntry {
	root := &FileEntry{Name: ".", IsDir: true, Expanded: true}
	dirs := map[string]*FileEntry{".": root}

	for _, path := range entries {
		parts := strings.Split(filepath.ToSlash(path), "/")
		current := root
		for i, part := range parts {
			isLast := i == len(parts)-1
			fullPath := strings.Join(parts[:i+1], "/")
			if !isLast {
				if existing, ok := dirs[fullPath]; ok {
					current = existing
					continue
				}
				dir := &FileEntry{Path: fullPath, Name: part, IsDir: true}
				current.Children = append(current.Children, dir)
				dirs[fullPath] = dir
				current = dir
				continue
			}
			current.Children = append(current.Children, &FileEntry{
				Path:  path,
				Name:  part,
				IsDir: false,
			})
		}
	}
	return root
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
