package views

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type CommitGraphView struct {
	graphText   string
	lines       []string
	cursor      int
	width       int
	height      int
	vp          viewport.Model
	repoPath    string
	compareMode bool
	commitA     string
	commitB     string

	t         *theme.Theme
	statusMsg string
}

var commitHashRE = regexp.MustCompile(`\b([0-9a-f]{7,40})\b`)

func NewCommitGraphView(t *theme.Theme) *CommitGraphView {
	return &CommitGraphView{t: t}
}

func (v *CommitGraphView) ID() ID        { return ViewCommitGraph }
func (v *CommitGraphView) Title() string { return "Commit graph" }
func (v *CommitGraphView) Init() tea.Cmd { return nil }

func (v *CommitGraphView) SetRepoPath(p string) { v.repoPath = p }

func (v *CommitGraphView) SetSize(w, h int) {
	v.width = w
	v.height = h
	vpH := max(3, h-6)
	v.vp = viewport.New(viewport.WithWidth(max(20, w-2)), viewport.WithHeight(vpH))
}

func (v *CommitGraphView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case CommitGraphMsg:
		if msg.RepoPath != "" && v.repoPath != "" && msg.RepoPath != v.repoPath {
			break
		}
		v.lines = msg.Lines
		v.graphText = strings.Join(msg.Lines, "\n")
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		} else {
			v.statusMsg = ""
		}
		v.cursor = 0
		v.syncVP()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "pgup", "pgdown", "ctrl+u", "ctrl+d":
			var cmd tea.Cmd
			v.vp, cmd = v.vp.Update(msg)
			return v, cmd
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
				v.syncVP()
			}
		case "down", "j":
			if v.cursor < len(v.lines)-1 {
				v.cursor++
				v.syncVP()
			}
		case "g":
			v.cursor = 0
			v.syncVP()
		case "G":
			if len(v.lines) > 0 {
				v.cursor = len(v.lines) - 1
				v.syncVP()
			}
		case "c":
			v.compareMode = !v.compareMode
			if !v.compareMode {
				v.commitA, v.commitB = "", ""
			}
			v.statusMsg = ""
			if v.compareMode {
				v.statusMsg = "Compare mode: Space pick A/B  d diff  c exit"
			}
		case " ":
			if !v.compareMode {
				break
			}
			h := v.lineHash(v.cursor)
			if h == "" {
				break
			}
			if v.commitA == "" {
				v.commitA = h
				v.statusMsg = "Compare: A=" + h + " (Space for B)"
			} else if v.commitB == "" && h != v.commitA {
				v.commitB = h
				v.statusMsg = "Compare: A=" + v.commitA + " B=" + v.commitB + "  d diff"
			} else {
				v.commitA = h
				v.commitB = ""
				v.statusMsg = "Compare: A=" + h + " (Space for B)"
			}
		case "enter":
			h := v.lineHash(v.cursor)
			if h == "" {
				v.statusMsg = "No commit hash on this line."
				break
			}
			return v, func() tea.Msg { return RequestCommitDetailMsg{Hash: h} }
		case "d":
			if v.commitA == "" || v.commitB == "" {
				v.statusMsg = "Select two commits (compare mode + Space) first."
				break
			}
			rp := v.repoPath
			a, b := v.commitA, v.commitB
			return v, func() tea.Msg { return RequestCommitGraphDiffMsg{RepoPath: rp, A: a, B: b} }
		}
	}
	return v, nil
}

func (v *CommitGraphView) lineHash(lineIdx int) string {
	if lineIdx < 0 || lineIdx >= len(v.lines) {
		return ""
	}
	m := commitHashRE.FindStringSubmatch(v.lines[lineIdx])
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func (v *CommitGraphView) syncVP() {
	v.vp.SetContent(v.coloredGraph())
	vpH := v.vp.Height()
	if vpH <= 0 {
		vpH = max(3, v.height-8)
	}
	maxOff := max(0, len(v.lines)-vpH)
	v.vp.SetYOffset(clamp(v.cursor-vpH/2, 0, maxOff))
}

func (v *CommitGraphView) coloredGraph() string {
	var b strings.Builder
	lineWidth := max(20, v.width-2)
	for i, line := range v.lines {
		s := colorizeGraphLine(truncate(line, lineWidth), v.t, i == v.cursor)
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

var branchPalette = []string{
	"#22D3EE", "#A78BFA", "#34D399", "#FB923C",
	"#F472B6", "#60A5FA", "#FBBF24", "#F87171",
}

func colorizeGraphLine(line string, t *theme.Theme, selected bool) string {
	if t == nil {
		tt := theme.NewTheme(true)
		t = &tt
	}
	if selected {
		return lipgloss.NewStyle().Bold(true).Background(t.Selection()).Foreground(t.Fg()).Render(line)
	}
	i := 0
	var graphBuf strings.Builder
	col := 0
	for i < len(line) {
		r, w := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && w == 1 {
			break
		}
		if r == ' ' || r == '*' || r == '|' || r == '/' || r == '\\' || r == '-' || r == '─' {
			c := branchPalette[col%len(branchPalette)]
			if r == ' ' {
				graphBuf.WriteRune(r)
			} else {
				graphBuf.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(string(r)))
			}
			if r != ' ' {
				col++
			}
			i += w
			continue
		}
		break
	}
	rest := line[i:]
	return graphBuf.String() + styleGraphRest(rest, t)
}

func styleGraphRest(rest string, t *theme.Theme) string {
	if rest == "" {
		return ""
	}
	open := strings.Index(rest, "(")
	close := strings.LastIndex(rest, ")")
	if open < 0 || close <= open {
		return lipgloss.NewStyle().Foreground(t.Fg()).Render(rest)
	}
	before := lipgloss.NewStyle().Foreground(t.Fg()).Render(rest[:open])
	inside := rest[open+1 : close]
	after := lipgloss.NewStyle().Foreground(t.Fg()).Render(rest[close+1:])
	var b strings.Builder
	b.WriteString(before)
	b.WriteString(lipgloss.NewStyle().Foreground(t.Fg()).Render("("))
	parts := strings.Split(inside, ", ")
	for i, p := range parts {
		if i > 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(t.DimText()).Render(", "))
		}
		p = strings.TrimSpace(p)
		switch {
		case strings.HasPrefix(p, "HEAD"):
			b.WriteString(lipgloss.NewStyle().Foreground(t.Primary()).Bold(true).Render(p))
		case strings.HasPrefix(p, "tag:"):
			b.WriteString(lipgloss.NewStyle().Foreground(t.Secondary()).Render(p))
		case strings.Contains(p, "origin/") || strings.Contains(p, "remotes/"):
			b.WriteString(lipgloss.NewStyle().Foreground(t.Info()).Render(p))
		default:
			b.WriteString(lipgloss.NewStyle().Foreground(t.Fg()).Render(p))
		}
	}
	b.WriteString(lipgloss.NewStyle().Foreground(t.Fg()).Render(")"))
	b.WriteString(after)
	return b.String()
}

func (v *CommitGraphView) Render() string {
	if v.t == nil {
		t := theme.NewTheme(true)
		v.t = &t
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Commit graph")
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Render(
		"Up/Down  Enter details  c compare  Space pick A/B  d diff  PgUp/PgDn scroll",
	)
	st := ""
	if v.statusMsg != "" {
		st = lipgloss.NewStyle().Foreground(v.t.Warning()).Render(v.statusMsg)
	}
	if len(v.lines) == 0 {
		return strings.Join([]string{title, hint, st, "", lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No graph loaded.")}, "\n")
	}
	v.vp.SetWidth(max(20, v.width-2))
	v.vp.SetHeight(max(3, v.height-8))
	v.syncVP()
	return strings.Join([]string{title, hint, st, "", v.vp.View()}, "\n")
}

// LoadCommitGraphCmd runs `git log --graph --oneline --all --decorate` (no --color).
func LoadCommitGraphCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		out, err := runGit(repoPath, "log", "--graph", "--oneline", "--all", "--decorate")
		if err != nil {
			return CommitGraphMsg{RepoPath: repoPath, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		raw := strings.TrimSuffix(out, "\n")
		var lines []string
		if raw != "" {
			lines = strings.Split(raw, "\n")
		}
		return CommitGraphMsg{RepoPath: repoPath, Lines: lines}
	}
}
