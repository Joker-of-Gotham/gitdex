package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type SubmodulesView struct {
	entries   []SubmoduleEntry
	cursor    int
	width     int
	height    int
	vp        viewport.Model
	repoPath  string
	statusMsg string

	t         *theme.Theme
	adding    bool
	addStep   int // 0 = URL, 1 = path
	urlInput  textinput.Model
	pathInput textinput.Model
	detail    bool
}

func NewSubmodulesView(t *theme.Theme) *SubmodulesView {
	u := textinput.New()
	u.Prompt = ""
	u.Placeholder = "https://github.com/org/repo.git"
	u.CharLimit = 512
	p := textinput.New()
	p.Prompt = ""
	p.Placeholder = "path/to/submodule"
	p.CharLimit = 512
	return &SubmodulesView{t: t, urlInput: u, pathInput: p}
}

func (v *SubmodulesView) ID() ID        { return ViewSubmodules }
func (v *SubmodulesView) Title() string { return "Submodules" }
func (v *SubmodulesView) Init() tea.Cmd { return nil }

func (v *SubmodulesView) SetRepoPath(p string) { v.repoPath = p }

func (v *SubmodulesView) SetSize(w, h int) {
	v.width = w
	v.height = h
	vpH := max(3, h-6)
	v.vp = viewport.New(viewport.WithWidth(max(20, w-2)), viewport.WithHeight(vpH))
}

func (v *SubmodulesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case SubmoduleListMsg:
		if msg.RepoPath != "" && v.repoPath != "" && msg.RepoPath != v.repoPath {
			break
		}
		v.entries = msg.Entries
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		} else {
			v.statusMsg = ""
		}
		v.cursor = 0
		v.syncVP()
	case SubmoduleOpResultMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
			return v, nil
		}
		v.statusMsg = msg.Message
		if v.repoPath != "" {
			return v, LoadSubmodulesCmd(v.repoPath)
		}
	case tea.KeyPressMsg:
		if v.adding {
			return v.handleAddKeys(msg)
		}
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
			if v.cursor < len(v.entries)-1 {
				v.cursor++
				v.syncVP()
			}
		case "g":
			v.cursor = 0
			v.syncVP()
		case "G":
			if len(v.entries) > 0 {
				v.cursor = len(v.entries) - 1
				v.syncVP()
			}
		case "enter":
			v.detail = !v.detail
		case "i":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			if e := v.selected(); e != nil {
				return v, SubmoduleInitCmd(v.repoPath, e.Path)
			}
		case "u":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			path := ""
			if e := v.selected(); e != nil {
				path = e.Path
			}
			return v, SubmoduleUpdateCmd(v.repoPath, path)
		case "s":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			path := ""
			if e := v.selected(); e != nil {
				path = e.Path
			}
			return v, SubmoduleSyncCmd(v.repoPath, path)
		case "a":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			v.adding = true
			v.addStep = 0
			v.urlInput.SetValue("")
			v.pathInput.SetValue("")
			return v, v.urlInput.Focus()
		}
	}
	return v, nil
}

func (v *SubmodulesView) handleAddKeys(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.adding = false
		v.urlInput.Blur()
		v.pathInput.Blur()
		v.statusMsg = "Add submodule canceled."
		return v, nil
	case "enter":
		if v.addStep == 0 {
			if strings.TrimSpace(v.urlInput.Value()) == "" {
				v.statusMsg = "URL required."
				return v, nil
			}
			v.addStep = 1
			return v, v.pathInput.Focus()
		}
		url := strings.TrimSpace(v.urlInput.Value())
		path := strings.TrimSpace(v.pathInput.Value())
		if path == "" {
			v.statusMsg = "Path required."
			return v, nil
		}
		v.adding = false
		v.urlInput.Blur()
		v.pathInput.Blur()
		return v, SubmoduleAddCmd(v.repoPath, url, path)
	}
	var cmd tea.Cmd
	if v.addStep == 0 {
		v.urlInput, cmd = v.urlInput.Update(msg)
	} else {
		v.pathInput, cmd = v.pathInput.Update(msg)
	}
	return v, cmd
}

func (v *SubmodulesView) selected() *SubmoduleEntry {
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		return &v.entries[v.cursor]
	}
	return nil
}

func (v *SubmodulesView) syncVP() {
	v.vp.SetContent(v.listPlain())
	vpH := v.vp.Height()
	if vpH <= 0 {
		vpH = max(3, v.height-8)
	}
	maxOff := max(0, len(v.entries)-vpH)
	v.vp.SetYOffset(clamp(v.cursor-vpH/2, 0, maxOff))
}

func (v *SubmodulesView) listPlain() string {
	var b strings.Builder
	for i, e := range v.entries {
		prefix := "  "
		if i == v.cursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-10s %-12s %s", prefix, truncate(e.Hash, 10), e.Status, e.Path)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (v *SubmodulesView) Render() string {
	if v.t == nil {
		t := theme.NewTheme(true)
		v.t = &t
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Submodules")
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Render(
		"↑/↓  i init  u update  s sync  a add  Enter details  PgUp/Dn scroll  Esc (add)",
	)
	var status string
	if v.statusMsg != "" {
		status = lipgloss.NewStyle().Foreground(v.t.Warning()).Render(v.statusMsg)
	}
	if len(v.entries) == 0 {
		body := strings.Join([]string{title, hint, status, "", lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No submodules loaded.")}, "\n")
		if v.adding {
			return body + "\n\n" + v.renderAddPanel()
		}
		return body
	}

	v.vp.SetWidth(max(20, v.width-2))
	v.vp.SetHeight(max(3, v.height-8))
	v.syncVP()
	listBox := v.vp.View()

	var detail string
	if v.detail && v.selected() != nil {
		e := v.selected()
		detail = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(v.t.Divider()).
			PaddingLeft(1).
			Width(max(30, v.width/3)).
			Render(strings.Join([]string{
				lipgloss.NewStyle().Bold(true).Foreground(v.t.Secondary()).Render(e.Path),
				fmt.Sprintf("Hash: %s", e.Hash),
				fmt.Sprintf("Status: %s", e.Status),
				fmt.Sprintf("URL: %s", e.URL),
			}, "\n"))
	}

	top := strings.Join([]string{title, hint, status, ""}, "\n")
	if v.width >= 100 && detail != "" {
		row := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(v.width-detailWidth(detail)-2).Render(listBox), detail)
		if v.adding {
			return row + "\n\n" + v.renderAddPanel()
		}
		return row
	}
	out := top + listBox
	if detail != "" {
		out += "\n" + detail
	}
	if v.adding {
		out += "\n\n" + v.renderAddPanel()
	}
	return out
}

func detailWidth(s string) int {
	lines := strings.Split(s, "\n")
	w := 0
	for _, ln := range lines {
		if len(ln) > w {
			w = len(ln)
		}
	}
	return w
}

func (v *SubmodulesView) renderAddPanel() string {
	step := "URL"
	inp := v.urlInput.View()
	if v.addStep == 1 {
		step = "path"
		inp = v.pathInput.View()
	}
	body := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Add submodule"),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render("Step: " + step),
		"",
		inp,
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Enter next / submit  Esc cancel"),
	}, "\n")
	return render.SurfacePanel(body, max(36, v.width), v.t.Surface(), v.t.BorderColor())
}

// LoadSubmodulesCmd loads submodule rows from `git submodule status`.
func LoadSubmodulesCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		entries, err := loadSubmoduleEntries(repoPath)
		return SubmoduleListMsg{RepoPath: repoPath, Entries: entries, Err: err}
	}
}

func loadSubmoduleEntries(repoPath string) ([]SubmoduleEntry, error) {
	out, err := runGit(repoPath, "submodule", "status")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, trimGitOut(out))
	}
	var entries []SubmoduleEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		e, ok := parseSubmoduleStatusLine(line)
		if !ok {
			continue
		}
		e.URL = submoduleURLFromConfig(repoPath, e.Path)
		entries = append(entries, e)
	}
	return entries, nil
}

func parseSubmoduleStatusLine(line string) (SubmoduleEntry, bool) {
	if len(line) < 3 {
		return SubmoduleEntry{}, false
	}
	statusChar := line[0]
	rest := strings.TrimSpace(line[1:])
	sp := strings.IndexByte(rest, ' ')
	if sp <= 0 {
		return SubmoduleEntry{}, false
	}
	sha := rest[:sp]
	tail := strings.TrimSpace(rest[sp+1:])
	path := tail
	if i := strings.Index(tail, " ("); i >= 0 {
		path = strings.TrimSpace(tail[:i])
	}
	st := "initialized"
	switch statusChar {
	case '-':
		st = "uninitialized"
	case '+':
		st = "modified"
	case 'U':
		st = "merge conflict"
	}
	return SubmoduleEntry{Name: path, Path: path, Hash: sha, Status: st}, true
}

func submoduleURLFromConfig(repoPath, subPath string) string {
	key := "submodule." + strings.ReplaceAll(subPath, `\`, `/`) + ".url"
	out, err := runGit(repoPath, "config", "-f", ".gitmodules", "--get", key)
	if err != nil {
		out, err = runGit(repoPath, "config", "--get", key)
	}
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func SubmoduleInitCmd(repoPath, subPath string) tea.Cmd {
	return func() tea.Msg {
		out, err := runGit(repoPath, "submodule", "init", subPath)
		if err != nil {
			return SubmoduleOpResultMsg{Op: "init", Path: subPath, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return SubmoduleOpResultMsg{Op: "init", Path: subPath, Message: trimGitOut(out)}
	}
}

func SubmoduleUpdateCmd(repoPath, subPath string) tea.Cmd {
	return func() tea.Msg {
		args := []string{"submodule", "update", "--recursive"}
		if subPath != "" {
			args = append(args, subPath)
		}
		out, err := runGit(repoPath, args...)
		if err != nil {
			return SubmoduleOpResultMsg{Op: "update", Path: subPath, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return SubmoduleOpResultMsg{Op: "update", Path: subPath, Message: trimGitOut(out)}
	}
}

func SubmoduleSyncCmd(repoPath, subPath string) tea.Cmd {
	return func() tea.Msg {
		args := []string{"submodule", "sync"}
		if subPath != "" {
			args = append(args, subPath)
		}
		out, err := runGit(repoPath, args...)
		if err != nil {
			return SubmoduleOpResultMsg{Op: "sync", Path: subPath, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return SubmoduleOpResultMsg{Op: "sync", Path: subPath, Message: trimGitOut(out)}
	}
}

func SubmoduleAddCmd(repoPath, url, subPath string) tea.Cmd {
	return func() tea.Msg {
		out, err := runGit(repoPath, "submodule", "add", url, subPath)
		if err != nil {
			return SubmoduleOpResultMsg{Op: "add", Path: subPath, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return SubmoduleOpResultMsg{Op: "add", Path: subPath, Message: trimGitOut(out)}
	}
}
