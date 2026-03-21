package views

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type ReflogEntry struct {
	Hash    string
	Action  string
	Message string
	Date    string
}

type ReflogView struct {
	t *theme.Theme

	entries   []ReflogEntry
	cursor    int
	width     int
	height    int
	vp        viewport.Model
	repoPath  string
	statusMsg string

	detail        bool
	detailHash    string
	detailContent string
	detailError   string

	awaitResetMode bool
}

func NewReflogView(t *theme.Theme) *ReflogView {
	return &ReflogView{t: t}
}

func (v *ReflogView) ID() ID        { return ViewReflog }
func (v *ReflogView) Title() string { return "Reflog" }
func (v *ReflogView) Init() tea.Cmd { return nil }

func (v *ReflogView) SetRepoPath(p string) { v.repoPath = p }

func (v *ReflogView) SetSize(w, h int) {
	v.width, v.height = w, h
	v.vp = viewport.New(viewport.WithWidth(w), viewport.WithHeight(max(3, h-10)))
	v.syncViewport()
}

func (v *ReflogView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ReflogListMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
			v.entries = nil
		} else {
			v.entries = msg.Entries
			v.statusMsg = ""
			v.cursor = 0
		}
		v.syncViewport()
	case CommitDetailMsg:
		if msg.Err != nil {
			v.detailError = msg.Err.Error()
			v.detailContent = ""
		} else {
			v.detailHash = msg.Hash
			v.detailContent = msg.Content
			v.detailError = ""
		}
		v.detail = true
		return v, nil
	case ReflogOpResultMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		} else {
			v.statusMsg = msg.Message
		}
		v.awaitResetMode = false
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *ReflogView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.awaitResetMode {
		switch msg.String() {
		case "esc":
			v.awaitResetMode = false
			v.statusMsg = "Reset canceled."
			return v, nil
		case "s":
			return v.dispatchReset(ReflogResetSoft)
		case "m":
			return v.dispatchReset(ReflogResetMixed)
		case "h":
			return v.dispatchReset(ReflogResetHard)
		}
		return v, nil
	}

	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
			v.syncViewport()
		}
	case "down", "j":
		if v.cursor < len(v.entries)-1 {
			v.cursor++
			v.syncViewport()
		}
	case "enter":
		if e := v.selected(); e != nil {
			v.detail = true
			return v, func() tea.Msg { return RequestCommitDetailMsg{Hash: e.Hash} }
		}
	case "esc":
		if v.detail {
			v.detail = false
			return v, nil
		}
	case "r":
		if e := v.selected(); e != nil {
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				return v, nil
			}
			v.awaitResetMode = true
			v.statusMsg = "Reset to " + truncate(e.Hash, 8) + "?  s soft  m mixed  h hard  Esc cancel"
			return v, nil
		}
	default:
		var cmd tea.Cmd
		v.vp, cmd = v.vp.Update(msg)
		v.syncViewport()
		return v, cmd
	}
	return v, nil
}

func (v *ReflogView) dispatchReset(mode ReflogResetMode) (View, tea.Cmd) {
	e := v.selected()
	if e == nil || v.repoPath == "" {
		v.awaitResetMode = false
		return v, nil
	}
	hash := e.Hash
	return v, ReflogResetCmd(v.repoPath, hash, mode)
}

func (v *ReflogView) selected() *ReflogEntry {
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		return &v.entries[v.cursor]
	}
	return nil
}

func (v *ReflogView) Render() string {
	if v.t == nil || v.width < 10 {
		return ""
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("  Reflog")
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).
		Render("  j/k navigate  Enter commit detail  r reset (s/m/h)  PgUp/PgDn scroll")
	if v.statusMsg != "" {
		header += "\n" + lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusMsg)
	}
	header += "\n" + hint + "\n"

	v.syncViewport()
	body := v.vp.View()
	out := header + "\n" + body

	if v.detail && strings.TrimSpace(v.detailContent+v.detailError) != "" {
		detail := v.renderDetail(max(40, v.width-4))
		out += "\n\n" + detail
	}
	return out
}

func (v *ReflogView) renderDetail(w int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Secondary()).Render("Commit")
	if v.detailError != "" {
		return title + "\n" + lipgloss.NewStyle().Foreground(v.t.Warning()).Render(v.detailError)
	}
	return title + "\n" + lipgloss.NewStyle().Width(w).Foreground(v.t.Fg()).Render(v.detailContent)
}

func (v *ReflogView) syncViewport() {
	if v.t == nil {
		return
	}
	w := max(20, v.width-2)
	var lines []string
	for i, e := range v.entries {
		badge := reflogBadge(v.t, e.Action)
		line := fmt.Sprintf("  %-10s %s  %s  %s",
			badge,
			truncate(e.Hash, 10),
			truncate(e.Date, 19),
			truncate(e.Message, max(12, w-48)),
		)
		if i == v.cursor {
			line = render.FillBlock(line, max(20, w), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection()))
		} else {
			line = lipgloss.NewStyle().Width(w).Foreground(v.t.Fg()).Render(line)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  No reflog entries. Load with LoadReflogCmd or open a repo.")}
	}
	v.vp.SetContent(strings.Join(lines, "\n"))
	v.vp.SetWidth(v.width)
	vpH := max(3, v.height-10)
	v.vp.SetHeight(vpH)
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		v.vp.EnsureVisible(v.cursor, 0, w)
	}
}

func reflogBadge(t *theme.Theme, action string) string {
	a := strings.ToLower(strings.TrimSpace(action))
	st := lipgloss.NewStyle().Bold(true)
	switch {
	case strings.Contains(a, "commit"):
		st = st.Foreground(t.Success())
	case strings.Contains(a, "checkout"):
		st = st.Foreground(t.Info())
	case strings.Contains(a, "rebase"):
		st = st.Foreground(t.Warning())
	case strings.Contains(a, "merge"):
		st = st.Foreground(t.Accent())
	default:
		st = st.Foreground(t.MutedFg())
	}
	label := truncate(strings.ToUpper(action), 8)
	if label == "" {
		label = "?"
	}
	return st.Render(label)
}

func reflogActionType(gs string) string {
	s := strings.ToLower(strings.TrimSpace(gs))
	switch {
	case strings.Contains(s, "rebase"):
		return "rebase"
	case strings.Contains(s, "checkout"):
		return "checkout"
	case strings.Contains(s, "merge"):
		return "merge"
	case strings.Contains(s, "commit"):
		return "commit"
	default:
		return "other"
	}
}

// LoadReflogCmd loads entries from git reflog (hash, parsed action, subject, date).
func LoadReflogCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(repoPath) == "" {
			return ReflogListMsg{RepoPath: repoPath, Err: fmt.Errorf("repository path is empty")}
		}
		ctx := context.Background()
		ex := gitops.NewGitExecutor()
		res, err := ex.Run(ctx, repoPath, "reflog", "-n", "500", "--format=%H%x09%gs%x09%ci")
		if err != nil {
			return ReflogListMsg{RepoPath: repoPath, Err: err}
		}
		entries := parseReflogOutput(res.Stdout)
		return ReflogListMsg{RepoPath: repoPath, Entries: entries}
	}
}

func parseReflogOutput(out string) []ReflogEntry {
	var entries []ReflogEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		hash := strings.TrimSpace(parts[0])
		gs := strings.TrimSpace(parts[1])
		date := strings.TrimSpace(parts[2])
		entries = append(entries, ReflogEntry{
			Hash:    hash,
			Action:  reflogActionType(gs),
			Message: gs,
			Date:    date,
		})
	}
	return entries
}

// ReflogResetCmd runs git reset --<mode> <hash> in repoPath.
func ReflogResetCmd(repoPath, hash string, mode ReflogResetMode) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		ex := gitops.NewGitExecutor()
		modeArg := string(mode)
		if mode == ReflogResetMixed {
			modeArg = "mixed"
		}
		_, err := ex.Run(ctx, repoPath, "reset", "--"+modeArg, hash)
		if err != nil {
			return ReflogOpResultMsg{
				Kind: ReflogOpReset,
				Hash: hash,
				Mode: mode,
				Err:  err,
			}
		}
		return ReflogOpResultMsg{
			Kind:    ReflogOpReset,
			Hash:    hash,
			Mode:    mode,
			Message: fmt.Sprintf("Reset (%s) to %s", mode, truncate(hash, 12)),
		}
	}
}
