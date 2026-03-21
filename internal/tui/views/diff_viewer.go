package views

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type DiffMode int

const (
	DiffModeUnified DiffMode = iota
	DiffModeSideBySide
)

type DiffLineType int

const (
	DiffLineContext DiffLineType = iota
	DiffLineAdded
	DiffLineRemoved
)

type DiffLine struct {
	Type    DiffLineType
	Content string
	OldNum  int
	NewNum  int
}

type DiffHunk struct {
	Header    string
	Lines     []DiffLine
	StartLine int
	rawPatch  string
}

// DiffViewer renders an interactive diff with unified/side-by-side modes, hunk navigation,
// staging markers, visual line selection, and optional word-level highlighting.
type DiffViewer struct {
	th *theme.Theme

	width, height int
	mode          DiffMode
	rawDiff       string
	hunks         []DiffHunk
	hunkPatches   []string
	fileHeader    string
	cursorHunk    int
	cursorLine    int
	stagedHunks   map[int]bool
	stagedLines   map[int]bool
	visualMode    bool
	visualStart   int
	visualEnd     int
	wordDiff      bool
	showStaged    bool
	leftVP        viewport.Model
	rightVP       viewport.Model
	unifiedVP     viewport.Model
	repoPath      string
	filePath      string

	bracketWait int // 0 none, 1 after ], 2 after [
	statusLine  string

	flatLineHunk []int // per global line index, hunk index
	flatHunkLine []int // per global line, line index within hunk
	hunkYOff     []int // first rendered line Y for each hunk (unified)
}

func NewDiffViewer(t *theme.Theme) *DiffViewer {
	return &DiffViewer{
		th:          t,
		mode:        DiffModeUnified,
		stagedHunks: make(map[int]bool),
		stagedLines: make(map[int]bool),
	}
}

func (d *DiffViewer) SetTheme(t *theme.Theme) { d.th = t }

// VisualMode reports whether line-range selection is active.
func (d *DiffViewer) VisualMode() bool { return d.visualMode }

func (d *DiffViewer) SetSize(w, h int) {
	d.width = w
	d.height = h
	vpH := max(3, h-8)
	halfW := max(10, (w-3)/2)
	d.unifiedVP = viewport.New(viewport.WithWidth(w), viewport.WithHeight(vpH))
	d.leftVP = viewport.New(viewport.WithWidth(halfW), viewport.WithHeight(vpH))
	d.rightVP = viewport.New(viewport.WithWidth(halfW), viewport.WithHeight(vpH))
}

// Load parses raw git diff output and resets viewer state.
func (d *DiffViewer) Load(repoPath, filePath, raw string) {
	d.repoPath = repoPath
	d.filePath = filePath
	d.rawDiff = raw
	d.hunks, d.hunkPatches, d.fileHeader = parseGitDiff(raw)
	d.cursorHunk = 0
	d.cursorLine = 0
	d.visualMode = false
	d.visualStart = 0
	d.visualEnd = 0
	d.stagedHunks = make(map[int]bool)
	d.stagedLines = make(map[int]bool)
	d.flatLineHunk, d.flatHunkLine, d.hunkYOff = d.buildFlatMaps()
	d.syncScrollToCursor()
}

func (d *DiffViewer) ShowStaged() bool { return d.showStaged }

func (d *DiffViewer) SetShowStaged(v bool) { d.showStaged = v }

func (d *DiffViewer) SetStatus(s string) { d.statusLine = s }

func (d *DiffViewer) buildFlatMaps() (lineToHunk []int, lineToHunkLine []int, hunkY []int) {
	n := 0
	for hi := range d.hunks {
		hunkY = append(hunkY, n)
		for li := range d.hunks[hi].Lines {
			lineToHunk = append(lineToHunk, hi)
			lineToHunkLine = append(lineToHunkLine, li)
			n++
		}
	}
	return lineToHunk, lineToHunkLine, hunkY
}

func (d *DiffViewer) lineCount() int { return len(d.flatLineHunk) }

func (d *DiffViewer) syncScrollToCursor() {
	if d.lineCount() == 0 {
		return
	}
	if d.cursorLine < 0 {
		d.cursorLine = 0
	}
	if d.cursorLine >= d.lineCount() {
		d.cursorLine = d.lineCount() - 1
	}
	d.cursorHunk = d.flatLineHunk[d.cursorLine]
	d.unifiedVP.SetYOffset(d.cursorLine)
	d.leftVP.SetYOffset(d.cursorLine)
	d.rightVP.SetYOffset(d.cursorLine)
}

func (d *DiffViewer) nextHunk() {
	if len(d.hunks) == 0 {
		return
	}
	if d.cursorHunk < len(d.hunks)-1 {
		d.cursorHunk++
	} else {
		d.cursorHunk = 0
	}
	if d.cursorHunk < len(d.hunkYOff) {
		d.cursorLine = d.hunkYOff[d.cursorHunk]
	}
	d.syncScrollToCursor()
}

func (d *DiffViewer) prevHunk() {
	if len(d.hunks) == 0 {
		return
	}
	if d.cursorHunk > 0 {
		d.cursorHunk--
	} else {
		d.cursorHunk = len(d.hunks) - 1
	}
	if d.cursorHunk < len(d.hunkYOff) {
		d.cursorLine = d.hunkYOff[d.cursorHunk]
	}
	d.syncScrollToCursor()
}

// toggleAllHunksGit stages or unstages the whole file (git add / restore --staged).
func (d *DiffViewer) toggleAllHunksGit() tea.Cmd {
	if d.repoPath == "" || d.filePath == "" {
		d.statusLine = "No repository path for staging."
		return nil
	}
	// Stage entire file when viewing unstaged diff; unstage entire file when viewing staged diff.
	if d.showStaged {
		return func() tea.Msg {
			return RequestGitStageFileMsg{Path: d.filePath, Unstage: true, Cached: true}
		}
	}
	return func() tea.Msg {
		return RequestGitStageFileMsg{Path: d.filePath, Unstage: false, Cached: false}
	}
}

func (d *DiffViewer) toggleHunkStage(hi int) tea.Cmd {
	if hi < 0 || hi >= len(d.hunks) || d.repoPath == "" || d.filePath == "" {
		return nil
	}
	patch := d.fileHeader + d.hunkPatches[hi]
	reverse := d.showStaged
	d.stagedHunks[hi] = !d.stagedHunks[hi]
	return func() tea.Msg {
		return RequestApplyGitPatchMsg{
			RepoPath: d.repoPath,
			FilePath: d.filePath,
			Patch:    patch,
			Reverse:  reverse,
			Cached:   d.showStaged,
		}
	}
}

func (d *DiffViewer) applyVisualLineStage() tea.Cmd {
	if !d.visualMode || d.repoPath == "" || d.filePath == "" {
		return nil
	}
	lo, hi := d.visualStart, d.visualEnd
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo; i <= hi; i++ {
		d.stagedLines[i] = true
	}
	patch, ok := d.buildPatchForLineRange(lo, hi)
	if !ok {
		d.statusLine = "Could not build patch for selection."
		return nil
	}
	reverse := d.showStaged
	return func() tea.Msg {
		return RequestApplyGitPatchMsg{
			RepoPath: d.repoPath,
			FilePath: d.filePath,
			Patch:    patch,
			Reverse:  reverse,
			Cached:   d.showStaged,
		}
	}
}

func (d *DiffViewer) buildPatchForLineRange(glo, ghi int) (string, bool) {
	if glo < 0 || ghi >= d.lineCount() || glo > ghi {
		return "", false
	}
	h0 := d.flatLineHunk[glo]
	h1 := d.flatLineHunk[ghi]
	if h0 != h1 {
		// Multi-hunk selection: concatenate patches (best-effort).
		var b strings.Builder
		b.WriteString(d.fileHeader)
		for hi := h0; hi <= h1; hi++ {
			b.WriteString(d.hunkPatches[hi])
		}
		return b.String(), true
	}
	// Single hunk: include subset of lines (approximation — full hunk if selection doesn't align).
	hi := h0
	li0 := d.flatHunkLine[glo]
	li1 := d.flatHunkLine[ghi]
	if li0 > li1 {
		li0, li1 = li1, li0
	}
	lines := d.hunks[hi].Lines[li0 : li1+1]
	if len(lines) == 0 {
		return "", false
	}
	hdr := d.hunks[hi].Header
	var body strings.Builder
	body.WriteString(hdr)
	body.WriteString("\n")
	add, del, ctx := 0, 0, 0
	for _, ln := range lines {
		switch ln.Type {
		case DiffLineAdded:
			add++
			body.WriteString("+")
			body.WriteString(ln.Content)
			body.WriteString("\n")
		case DiffLineRemoved:
			del++
			body.WriteString("-")
			body.WriteString(ln.Content)
			body.WriteString("\n")
		default:
			ctx++
			body.WriteString(" ")
			body.WriteString(ln.Content)
			body.WriteString("\n")
		}
	}
	newHdr := adjustHunkHeader(hdr, del, add, ctx)
	bodyStr := body.String()
	// Drop duplicate header line (we wrote hdr twice).
	rest := bodyStr
	if strings.HasPrefix(rest, hdr) {
		rest = rest[len(hdr):]
		rest = strings.TrimPrefix(rest, "\n")
	}
	return d.fileHeader + newHdr + "\n" + rest, true
}

func adjustHunkHeader(hdr string, oldDel, oldAdd, ctx int) string {
	// @@ -oldStart,oldLines +newStart,newLines @@
	parts := strings.SplitN(hdr, "@@", 3)
	if len(parts) < 2 {
		return hdr
	}
	rest := strings.TrimSpace(parts[1])
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return hdr
	}
	oldPart := fields[0] // -1,3
	newPart := fields[1] // +1,4
	oldN := parseNumRange(oldPart)
	newN := parseNumRange(newPart)
	if oldN == nil || newN == nil {
		return hdr
	}
	oldCount := oldDel + ctx
	newCount := oldAdd + ctx
	return fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldN.start, oldCount, newN.start, newCount)
}

type numRange struct {
	start int
}

func parseNumRange(s string) *numRange {
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")
	i := strings.IndexByte(s, ',')
	if i < 0 {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil
		}
		return &numRange{start: n}
	}
	a, err1 := strconv.Atoi(s[:i])
	_, err2 := strconv.Atoi(s[i+1:])
	if err1 != nil || err2 != nil {
		return nil
	}
	return &numRange{start: a}
}

func (d *DiffViewer) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	}
	return nil
}

func (d *DiffViewer) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	s := msg.String()

	if d.bracketWait == 1 {
		d.bracketWait = 0
		if s == "c" || s == "h" {
			d.nextHunk()
			return nil
		}
	}
	if d.bracketWait == 2 {
		d.bracketWait = 0
		if s == "c" || s == "h" {
			d.prevHunk()
			return nil
		}
	}

	switch s {
	case "]":
		d.bracketWait = 1
		return nil
	case "[":
		d.bracketWait = 2
		return nil
	case "tab":
		if d.mode == DiffModeUnified {
			d.mode = DiffModeSideBySide
		} else {
			d.mode = DiffModeUnified
		}
		return nil
	case "w":
		d.wordDiff = !d.wordDiff
		return nil
	case "s":
		d.showStaged = !d.showStaged
		if d.filePath != "" {
			return func() tea.Msg {
				return RequestFileDiffMsg{Path: d.filePath, Cached: d.showStaged}
			}
		}
		return nil
	case "v":
		d.visualMode = !d.visualMode
		if d.visualMode {
			d.visualStart = d.cursorLine
			d.visualEnd = d.cursorLine
		}
		return nil
	case "a":
		return d.toggleAllHunksGit()
	case "esc":
		if d.visualMode {
			d.visualMode = false
			return nil
		}
	}

	if d.visualMode {
		switch s {
		case "up", "k":
			if d.visualEnd > 0 {
				d.visualEnd--
			}
			return nil
		case "down", "j":
			if d.visualEnd < d.lineCount()-1 {
				d.visualEnd++
			}
			return nil
		case " ":
			return d.applyVisualLineStage()
		}
		var cmd tea.Cmd
		d.unifiedVP, cmd = d.unifiedVP.Update(msg)
		d.leftVP, _ = d.leftVP.Update(msg)
		d.rightVP, _ = d.rightVP.Update(msg)
		return cmd
	}

	switch s {
	case "up", "k":
		if d.cursorLine > 0 {
			d.cursorLine--
			d.syncScrollToCursor()
		}
		return nil
	case "down", "j":
		if d.cursorLine < d.lineCount()-1 {
			d.cursorLine++
			d.syncScrollToCursor()
		}
		return nil
	case " ":
		if d.lineCount() == 0 {
			return nil
		}
		hi := d.flatLineHunk[d.cursorLine]
		return d.toggleHunkStage(hi)
	}

	var cmd tea.Cmd
	d.unifiedVP, cmd = d.unifiedVP.Update(msg)
	d.leftVP, _ = d.leftVP.Update(msg)
	d.rightVP, _ = d.rightVP.Update(msg)
	return cmd
}

// Render builds the diff panel (title + hints should be wrapped by FilesView if needed).
func (d *DiffViewer) Render(viewWidth, viewHeight int) string {
	if d.th == nil {
		t := theme.NewTheme(true)
		d.th = &t
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(d.th.Primary()).Render(theme.Icons.Diff + " Diff " + d.filePath)
	st := "unstaged"
	if d.showStaged {
		st = "staged (cached)"
	}
	hint := lipgloss.NewStyle().Foreground(d.th.DimText()).Render(
		"Tab mode  w word  s " + st + "  ]c/]h next hunk  [c/[h prev  Space hunk stage  a all  v visual  PgUp/Dn scroll  Esc back",
	)
	if d.visualMode {
		hint = lipgloss.NewStyle().Foreground(d.th.Warning()).Render("VISUAL: arrows extend  Space apply  Esc exit visual")
	}
	if d.statusLine != "" {
		hint = hint + "\n" + lipgloss.NewStyle().Foreground(d.th.Warning()).Render(d.statusLine)
	}

	content := d.rawDiff
	if strings.TrimSpace(content) == "" {
		content = "No diff available."
	}

	d.SetSize(viewWidth, viewHeight)
	unifiedStr := d.renderUnifiedString()
	d.unifiedVP.SetWidth(max(20, viewWidth-2))
	d.unifiedVP.SetHeight(max(3, viewHeight-8))
	d.unifiedVP.SetContent(unifiedStr)
	d.unifiedVP.StyleLineFunc = d.styleLineFunc()

	left, right := d.renderSideBySideStrings()
	d.leftVP.SetContent(left)
	d.rightVP.SetContent(right)
	d.leftVP.SetWidth(max(10, (viewWidth-3)/2))
	d.rightVP.SetWidth(max(10, (viewWidth-3)/2))
	d.leftVP.SetHeight(max(3, viewHeight-8))
	d.rightVP.SetHeight(max(3, viewHeight-8))

	var body string
	if d.mode == DiffModeSideBySide {
		leftBox := lipgloss.NewStyle().Width(d.leftVP.Width()).Render(d.leftVP.View())
		rightBox := lipgloss.NewStyle().Width(d.rightVP.Width()).Render(d.rightVP.View())
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(d.th.Divider()).Render(leftBox),
			lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(d.th.Divider()).Render(rightBox),
		)
	} else {
		body = d.unifiedVP.View()
	}
	return strings.Join([]string{title, hint, "", body}, "\n")
}

func (d *DiffViewer) styleLineFunc() func(int) lipgloss.Style {
	return func(i int) lipgloss.Style {
		base := lipgloss.NewStyle()
		if d.visualMode && i >= min(d.visualStart, d.visualEnd) && i <= max(d.visualStart, d.visualEnd) {
			return base.Background(d.th.Selection())
		}
		if i == d.cursorLine && !d.visualMode {
			return base.Background(d.th.Selection())
		}
		return base
	}
}

func (d *DiffViewer) renderUnifiedString() string {
	if len(d.hunks) == 0 {
		return render.Diff(d.rawDiff, d.th)
	}
	var b strings.Builder
	addSt := lipgloss.NewStyle().Foreground(d.th.Success())
	delSt := lipgloss.NewStyle().Foreground(d.th.Danger())
	ctxSt := lipgloss.NewStyle().Foreground(d.th.Fg())
	hunkSt := lipgloss.NewStyle().Foreground(d.th.Info())
	markSt := lipgloss.NewStyle().Foreground(d.th.Warning())

	gi := 0
	for hi, h := range d.hunks {
		prefix := "  "
		if d.stagedHunks[hi] {
			prefix = markSt.Render("●")
		}
		b.WriteString(prefix)
		b.WriteString(hunkSt.Render(h.Header))
		b.WriteString("\n")

		for li := 0; li < len(h.Lines); li++ {
			ln := h.Lines[li]
			if ln.Type == DiffLineRemoved && d.wordDiff && li+1 < len(h.Lines) && h.Lines[li+1].Type == DiffLineAdded {
				p := "  "
				if d.stagedLines[gi] {
					p = markSt.Render("│")
				}
				oldT, newT := wordDiffInline(ln.Content, h.Lines[li+1].Content, d.th)
				b.WriteString(p)
				b.WriteString(delSt.Render("-"))
				b.WriteString(oldT)
				b.WriteString("\n")
				gi++
				b.WriteString(p)
				b.WriteString(addSt.Render("+"))
				b.WriteString(newT)
				b.WriteString("\n")
				gi++
				li++
				continue
			}
			p := "  "
			if d.stagedLines[gi] {
				p = markSt.Render("│")
			}
			switch ln.Type {
			case DiffLineAdded:
				if d.wordDiff && li > 0 && h.Lines[li-1].Type == DiffLineRemoved {
					continue
				}
				b.WriteString(p)
				b.WriteString(addSt.Render("+" + ln.Content))
				b.WriteString("\n")
				gi++
			case DiffLineRemoved:
				b.WriteString(p)
				b.WriteString(delSt.Render("-" + ln.Content))
				b.WriteString("\n")
				gi++
			default:
				b.WriteString(p)
				b.WriteString(ctxSt.Render(" " + ln.Content))
				b.WriteString("\n")
				gi++
			}
		}
	}
	return b.String()
}

func wordDiffInline(oldPlain, newPlain string, t *theme.Theme) (string, string) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldPlain, newPlain, false)
	var oldB, newB strings.Builder
	delSt := lipgloss.NewStyle().Foreground(t.Danger()).Bold(true)
	addSt := lipgloss.NewStyle().Foreground(t.Success()).Bold(true)
	eqSt := lipgloss.NewStyle().Foreground(t.Fg())
	for _, df := range diffs {
		switch df.Type {
		case diffmatchpatch.DiffEqual:
			oldB.WriteString(eqSt.Render(df.Text))
			newB.WriteString(eqSt.Render(df.Text))
		case diffmatchpatch.DiffDelete:
			oldB.WriteString(delSt.Render(df.Text))
		case diffmatchpatch.DiffInsert:
			newB.WriteString(addSt.Render(df.Text))
		}
	}
	return oldB.String(), newB.String()
}

func (d *DiffViewer) renderSideBySideStrings() (string, string) {
	if len(d.hunks) == 0 {
		u := render.Diff(d.rawDiff, d.th)
		return u, ""
	}
	var leftB, rightB strings.Builder
	lctx := lipgloss.NewStyle().Foreground(d.th.Fg())
	for _, h := range d.hunks {
		leftB.WriteString(lipgloss.NewStyle().Foreground(d.th.Info()).Render(h.Header) + "\n")
		rightB.WriteString(lipgloss.NewStyle().Foreground(d.th.Info()).Render(h.Header) + "\n")
		for _, ln := range h.Lines {
			switch ln.Type {
			case DiffLineRemoved:
				leftB.WriteString(lipgloss.NewStyle().Foreground(d.th.Danger()).Render(ln.Content) + "\n")
				rightB.WriteString("\n")
			case DiffLineAdded:
				leftB.WriteString("\n")
				rightB.WriteString(lipgloss.NewStyle().Foreground(d.th.Success()).Render(ln.Content) + "\n")
			default:
				s := lctx.Render(ln.Content)
				leftB.WriteString(s + "\n")
				rightB.WriteString(s + "\n")
			}
		}
	}
	return leftB.String(), rightB.String()
}

func parseGitDiff(raw string) (hunks []DiffHunk, patches []string, fileHeader string) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	var headerEnd int
	for i, line := range lines {
		if strings.HasPrefix(line, "@@") {
			headerEnd = i
			break
		}
	}
	if headerEnd == 0 {
		headerEnd = len(lines)
	}
	fileHeader = strings.Join(lines[:headerEnd], "\n")
	if fileHeader != "" {
		fileHeader += "\n"
	}
	i := headerEnd
	for i < len(lines) {
		line := lines[i]
		if !strings.HasPrefix(line, "@@") {
			i++
			continue
		}
		h := DiffHunk{Header: line}
		oldN, newN := parseHunkNumbers(line)
		var hb strings.Builder
		hb.WriteString(line)
		hb.WriteString("\n")
		i++
		for i < len(lines) && !strings.HasPrefix(lines[i], "@@") {
			l := lines[i]
			if len(l) == 0 {
				i++
				continue
			}
			dl := DiffLine{Content: trimFirst(l)}
			switch l[0] {
			case '-':
				dl.Type = DiffLineRemoved
				dl.OldNum = oldN
				oldN++
			case '+':
				dl.Type = DiffLineAdded
				dl.NewNum = newN
				newN++
			default:
				dl.Type = DiffLineContext
				dl.OldNum = oldN
				dl.NewNum = newN
				oldN++
				newN++
			}
			h.Lines = append(h.Lines, dl)
			hb.WriteString(l)
			hb.WriteString("\n")
			i++
		}
		h.StartLine = oldN
		h.rawPatch = hb.String()
		hunks = append(hunks, h)
		patches = append(patches, h.rawPatch)
	}
	return hunks, patches, fileHeader
}

func parseHunkNumbers(header string) (oldStart, newStart int) {
	// @@ -1,7 +1,9 @@
	p := strings.Split(header, "@@")
	if len(p) < 2 {
		return 1, 1
	}
	fields := strings.Fields(strings.TrimSpace(p[1]))
	if len(fields) < 2 {
		return 1, 1
	}
	oldStart = parseOneRange(fields[0])
	newStart = parseOneRange(fields[1])
	return oldStart, newStart
}

func parseOneRange(s string) int {
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")
	j := strings.IndexByte(s, ',')
	if j < 0 {
		n, _ := strconv.Atoi(s)
		if n < 1 {
			return 1
		}
		return n
	}
	n, _ := strconv.Atoi(s[:j])
	if n < 1 {
		return 1
	}
	return n
}

func trimFirst(s string) string {
	if len(s) <= 1 {
		return ""
	}
	return s[1:]
}
