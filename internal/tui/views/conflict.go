package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type ConflictView struct {
	filePath  string
	prefix    string
	between   []string
	suffix    string
	hunks     []ConflictHunk
	cursor    int
	width     int
	height    int
	leftVP    viewport.Model
	rightVP   viewport.Model
	repoPath  string
	statusMsg string

	t             *theme.Theme
	hunkConfirmed []bool
}

func NewConflictView(t *theme.Theme) *ConflictView {
	return &ConflictView{t: t}
}

func (v *ConflictView) ID() ID        { return ViewConflict }
func (v *ConflictView) Title() string { return "Conflicts" }
func (v *ConflictView) Init() tea.Cmd { return nil }

func (v *ConflictView) SetRepoPath(p string) { v.repoPath = p }

func (v *ConflictView) SetSize(w, h int) {
	v.width = w
	v.height = h
	halfW := max(10, (w-3)/2)
	vpH := max(3, h-8)
	v.leftVP = viewport.New(viewport.WithWidth(halfW), viewport.WithHeight(vpH))
	v.rightVP = viewport.New(viewport.WithWidth(halfW), viewport.WithHeight(vpH))
}

func (v *ConflictView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ConflictFileMsg:
		if msg.RepoPath != "" && v.repoPath != "" && msg.RepoPath != v.repoPath {
			break
		}
		v.filePath = msg.FilePath
		v.prefix = msg.Prefix
		v.between = msg.Between
		v.suffix = msg.Suffix
		v.hunks = msg.Hunks
		v.hunkConfirmed = make([]bool, len(v.hunks))
		v.cursor = 0
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		} else {
			v.statusMsg = ""
		}
		v.syncVPs()
	case ConflictResolvedMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		} else {
			v.statusMsg = msg.Message
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "pgup", "pgdown", "ctrl+u", "ctrl+d", "up", "down", "k", "j":
			var c0, c1 tea.Cmd
			v.leftVP, c0 = v.leftVP.Update(msg)
			v.rightVP, c1 = v.rightVP.Update(msg)
			if c0 != nil {
				return v, c0
			}
			return v, c1
		case "p", "[":
			if v.cursor > 0 {
				v.cursor--
				v.syncVPs()
			}
		case "n", "]":
			if v.cursor < len(v.hunks)-1 {
				v.cursor++
				v.syncVPs()
			}
		case "o":
			if v.currentHunk() != nil {
				v.currentHunk().Resolution = "ours"
				v.hunkConfirmed[v.cursor] = false
			}
		case "t":
			if v.currentHunk() != nil {
				v.currentHunk().Resolution = "theirs"
				v.hunkConfirmed[v.cursor] = false
			}
		case "b":
			if v.currentHunk() != nil {
				v.currentHunk().Resolution = "both"
				v.hunkConfirmed[v.cursor] = false
			}
		case "enter":
			return v.handleEnter()
		}
	}
	return v, nil
}

func (v *ConflictView) handleEnter() (View, tea.Cmd) {
	if len(v.hunks) == 0 {
		v.statusMsg = "No conflict hunks."
		return v, nil
	}
	h := v.currentHunk()
	if h == nil {
		return v, nil
	}
	if h.Resolution == "" {
		v.statusMsg = "Choose o (ours), t (theirs), or b (both)."
		return v, nil
	}
	v.hunkConfirmed[v.cursor] = true
	if v.allConfirmed() {
		content := v.buildMerged()
		rp := v.repoPath
		fp := v.filePath
		return v, WriteConflictResolvedCmd(rp, fp, content)
	}
	if v.cursor < len(v.hunks)-1 {
		v.cursor++
	}
	v.syncVPs()
	return v, nil
}

func (v *ConflictView) currentHunk() *ConflictHunk {
	if v.cursor >= 0 && v.cursor < len(v.hunks) {
		return &v.hunks[v.cursor]
	}
	return nil
}

func (v *ConflictView) allConfirmed() bool {
	for i := range v.hunks {
		if !v.hunkConfirmed[i] {
			return false
		}
	}
	return len(v.hunks) > 0
}

func (v *ConflictView) buildMerged() string {
	var b strings.Builder
	b.WriteString(v.prefix)
	for i, h := range v.hunks {
		if i > 0 && i-1 < len(v.between) {
			b.WriteString(v.between[i-1])
		}
		b.WriteString(h.resolvedBody())
	}
	b.WriteString(v.suffix)
	return b.String()
}

func (h *ConflictHunk) resolvedBody() string {
	switch h.Resolution {
	case "ours":
		return h.OursContent
	case "theirs":
		return h.TheirsContent
	case "both":
		os := strings.TrimRight(h.OursContent, "\n")
		ts := strings.TrimLeft(h.TheirsContent, "\n")
		return os + "\n" + ts
	default:
		return ""
	}
}

func (v *ConflictView) syncVPs() {
	if v.t == nil {
		t := theme.NewTheme(true)
		v.t = &t
	}
	h := v.currentHunk()
	if h == nil {
		v.leftVP.SetContent("")
		v.rightVP.SetContent("")
		return
	}
	left := lipgloss.NewStyle().Foreground(v.t.Info()).Render("ours") + "\n\n" + h.OursContent
	right := lipgloss.NewStyle().Foreground(v.t.Secondary()).Render("theirs") + "\n\n" + h.TheirsContent
	if strings.TrimSpace(h.BaseContent) != "" {
		left += "\n\n" + lipgloss.NewStyle().Foreground(v.t.DimText()).Render("base:\n"+h.BaseContent)
	}
	v.leftVP.SetContent(left)
	v.rightVP.SetContent(right)
	v.leftVP.SetYOffset(0)
	v.rightVP.SetYOffset(0)
}

func (v *ConflictView) Render() string {
	if v.t == nil {
		t := theme.NewTheme(true)
		v.t = &t
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Merge conflict")
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Render(
		"o ours  t theirs  b both  Enter confirm  p/[ prev hunk  n/] next  PgUp/Dn scroll",
	)
	st := ""
	if v.statusMsg != "" {
		st = lipgloss.NewStyle().Foreground(v.t.Warning()).Render(v.statusMsg)
	}
	if len(v.hunks) == 0 {
		return strings.Join([]string{title, hint, st, "", lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No conflict data.")}, "\n")
	}
	h := v.currentHunk()
	meta := ""
	if h != nil {
		meta = lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(
			fmt.Sprintf("File: %s  Hunk %d/%d  resolution=%s", v.filePath, v.cursor+1, len(v.hunks), h.Resolution),
		)
	}
	v.SetSize(v.width, v.height)
	v.leftVP.SetWidth(max(10, (v.width-3)/2))
	v.rightVP.SetWidth(max(10, (v.width-3)/2))
	vpH := max(3, v.height-10)
	v.leftVP.SetHeight(vpH)
	v.rightVP.SetHeight(vpH)
	v.syncVPs()
	leftBox := lipgloss.NewStyle().Width(v.leftVP.Width()).Render(v.leftVP.View())
	rightBox := lipgloss.NewStyle().Width(v.rightVP.Width()).Render(v.rightVP.View())
	row := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, lipgloss.NewStyle().Foreground(v.t.Divider()).Render(" │ "), rightBox)
	return strings.Join([]string{title, hint, st, meta, "", row}, "\n")
}

// LoadConflictFileCmd reads a conflicted file from disk and parses conflict markers.
func LoadConflictFileCmd(repoPath, filePath string) tea.Cmd {
	return func() tea.Msg {
		full := filepath.Join(repoPath, filePath)
		b, err := os.ReadFile(full)
		if err != nil {
			return ConflictFileMsg{RepoPath: repoPath, FilePath: filePath, Err: err}
		}
		prefix, betweens, suffix, hunks, err := parseConflictMarkers(string(b))
		if err != nil {
			return ConflictFileMsg{RepoPath: repoPath, FilePath: filePath, Err: err}
		}
		if len(hunks) == 0 {
			return ConflictFileMsg{RepoPath: repoPath, FilePath: filePath, Err: fmt.Errorf("no conflict markers in file")}
		}
		return ConflictFileMsg{
			RepoPath: repoPath,
			FilePath: filePath,
			Prefix:   prefix,
			Between:  betweens,
			Suffix:   suffix,
			Hunks:    hunks,
		}
	}
}

// WriteConflictResolvedCmd writes merged content and runs `git add` on the file.
func WriteConflictResolvedCmd(repoPath, filePath, content string) tea.Cmd {
	return func() tea.Msg {
		full := filepath.Join(repoPath, filePath)
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return ConflictResolvedMsg{RepoPath: repoPath, FilePath: filePath, Err: err}
		}
		out, err := runGit(repoPath, "add", filePath)
		if err != nil {
			return ConflictResolvedMsg{RepoPath: repoPath, FilePath: filePath, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		_ = out
		return ConflictResolvedMsg{RepoPath: repoPath, FilePath: filePath, Message: "Resolved and staged."}
	}
}

func parseConflictMarkers(content string) (prefix string, betweens []string, suffix string, hunks []ConflictHunk, err error) {
	rest := content
	for {
		idx := strings.Index(rest, "<<<<<<<")
		if idx < 0 {
			suffix = rest
			break
		}
		if len(hunks) == 0 {
			prefix = rest[:idx]
		} else {
			betweens = append(betweens, rest[:idx])
		}
		rest = rest[idx+len("<<<<<<<"):]
		nl := strings.Index(rest, "\n")
		if nl < 0 {
			return "", nil, "", nil, fmt.Errorf("malformed conflict start")
		}
		rest = rest[nl+1:]
		end := strings.Index(rest, ">>>>>>>")
		if end < 0 {
			return "", nil, "", nil, fmt.Errorf("missing closing conflict marker")
		}
		block := rest[:end]
		rest = rest[end:]
		if nl2 := strings.Index(rest, "\n"); nl2 >= 0 {
			rest = rest[nl2+1:]
		} else {
			rest = ""
		}
		hunks = append(hunks, parseConflictBlock(block))
	}
	return prefix, betweens, suffix, hunks, nil
}

func parseConflictBlock(block string) ConflictHunk {
	h := ConflictHunk{}
	parts := strings.SplitN(block, "=======", 2)
	if len(parts) < 2 {
		h.OursContent = strings.TrimSpace(block)
		return h
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	if strings.Contains(left, "|||||||") {
		p := strings.SplitN(left, "|||||||", 2)
		h.OursContent = strings.TrimSpace(p[0])
		if len(p) > 1 {
			h.BaseContent = strings.TrimSpace(p[1])
		}
	} else {
		h.OursContent = left
	}
	h.TheirsContent = right
	return h
}
