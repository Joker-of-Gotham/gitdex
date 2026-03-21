package views

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type RemotesView struct {
	entries   []RemoteEntry
	cursor    int
	width     int
	height    int
	vp        viewport.Model
	repoPath  string
	statusMsg string
	adding     bool
	remoteName string
	remoteURL  string

	t          *theme.Theme
	nameInput  textinput.Model
	urlInput   textinput.Model
	setURLOnly bool // true when u: only URL
	addStep    int  // 0 name, 1 url (add flow)
}

func NewRemotesView(t *theme.Theme) *RemotesView {
	n := textinput.New()
	n.Prompt = ""
	n.Placeholder = "remote name"
	n.CharLimit = 256
	u := textinput.New()
	u.Prompt = ""
	u.Placeholder = "https://github.com/org/repo.git"
	u.CharLimit = 512
	return &RemotesView{t: t, nameInput: n, urlInput: u}
}

func (v *RemotesView) ID() ID        { return ViewRemotes }
func (v *RemotesView) Title() string { return "Remotes" }
func (v *RemotesView) Init() tea.Cmd { return nil }

func (v *RemotesView) SetRepoPath(p string) { v.repoPath = p }

func (v *RemotesView) SetSize(w, h int) {
	v.width = w
	v.height = h
	vpH := max(3, h-6)
	v.vp = viewport.New(viewport.WithWidth(max(20, w-2)), viewport.WithHeight(vpH))
}

func (v *RemotesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case RemoteListMsg:
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
	case RemoteOpResultMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
			return v, nil
		}
		v.statusMsg = msg.Message
		if v.repoPath != "" {
			return v, LoadRemotesCmd(v.repoPath)
		}
	case tea.KeyPressMsg:
		if v.adding {
			return v.handlePromptKeys(msg)
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
		case "a":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			v.adding = true
			v.setURLOnly = false
			v.addStep = 0
			v.nameInput.SetValue("")
			v.urlInput.SetValue("")
			return v, v.nameInput.Focus()
		case "d":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			if e := v.selected(); e != nil {
				return v, RemoteRemoveCmd(v.repoPath, e.Name)
			}
		case "u":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			if e := v.selected(); e != nil {
				v.adding = true
				v.setURLOnly = true
				v.remoteName = e.Name
				v.urlInput.SetValue(e.FetchURL)
				return v, v.urlInput.Focus()
			}
		case "f":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			name := ""
			if e := v.selected(); e != nil {
				name = e.Name
			}
			return v, RemoteFetchCmd(v.repoPath, name)
		case "p":
			if v.repoPath == "" {
				v.statusMsg = "No repository path."
				break
			}
			name := ""
			if e := v.selected(); e != nil {
				name = e.Name
			}
			return v, RemotePruneCmd(v.repoPath, name)
		}
	}
	return v, nil
}

func (v *RemotesView) handlePromptKeys(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.adding = false
		v.setURLOnly = false
		v.addStep = 0
		v.nameInput.Blur()
		v.urlInput.Blur()
		v.statusMsg = "Canceled."
		return v, nil
	case "enter":
		if v.setURLOnly {
			url := strings.TrimSpace(v.urlInput.Value())
			if url == "" {
				v.statusMsg = "URL required."
				return v, nil
			}
			n := v.remoteName
			v.adding = false
			v.setURLOnly = false
			v.urlInput.Blur()
			return v, RemoteSetURLCmd(v.repoPath, n, url)
		}
		if v.addStep == 0 {
			if strings.TrimSpace(v.nameInput.Value()) == "" {
				v.statusMsg = "Name required."
				return v, nil
			}
			v.addStep = 1
			return v, v.urlInput.Focus()
		}
		if strings.TrimSpace(v.urlInput.Value()) == "" {
			v.statusMsg = "URL required."
			return v, nil
		}
		n := strings.TrimSpace(v.nameInput.Value())
		u := strings.TrimSpace(v.urlInput.Value())
		v.adding = false
		v.addStep = 0
		v.nameInput.Blur()
		v.urlInput.Blur()
		return v, RemoteAddCmd(v.repoPath, n, u)
	}
	var cmd tea.Cmd
	if v.setURLOnly {
		v.urlInput, cmd = v.urlInput.Update(msg)
	} else if v.addStep == 0 {
		v.nameInput, cmd = v.nameInput.Update(msg)
	} else {
		v.urlInput, cmd = v.urlInput.Update(msg)
	}
	return v, cmd
}

func (v *RemotesView) selected() *RemoteEntry {
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		return &v.entries[v.cursor]
	}
	return nil
}

func (v *RemotesView) syncVP() {
	v.vp.SetContent(v.listPlain())
	vpH := v.vp.Height()
	if vpH <= 0 {
		vpH = max(3, v.height-8)
	}
	maxOff := max(0, len(v.entries)-vpH)
	v.vp.SetYOffset(clamp(v.cursor-vpH/2, 0, maxOff))
}

func (v *RemotesView) listPlain() string {
	var b strings.Builder
	for i, e := range v.entries {
		prefix := "  "
		if i == v.cursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-12s  fetch: %s  push: %s", prefix, e.Name, truncate(e.FetchURL, 40), truncate(e.PushURL, 40))
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (v *RemotesView) Render() string {
	if v.t == nil {
		t := theme.NewTheme(true)
		v.t = &t
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Remotes")
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Render(
		"a add  d remove  u set-url  f fetch  p prune  Tab (add)  Esc",
	)
	st := ""
	if v.statusMsg != "" {
		st = lipgloss.NewStyle().Foreground(v.t.Warning()).Render(v.statusMsg)
	}
	if len(v.entries) == 0 && !v.adding {
		return strings.Join([]string{title, hint, st, "", lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No remotes loaded.")}, "\n")
	}
	v.vp.SetWidth(max(20, v.width-2))
	v.vp.SetHeight(max(3, v.height-8))
	v.syncVP()
	body := v.vp.View()
	if v.adding {
		panel := v.renderPromptPanel()
		return strings.Join([]string{title, hint, st, "", body, "", panel}, "\n")
	}
	return strings.Join([]string{title, hint, st, "", body}, "\n")
}

func (v *RemotesView) renderPromptPanel() string {
	if v.setURLOnly {
		body := strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Set remote URL"),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render("Remote: " + v.remoteName),
			"",
			v.urlInput.View(),
			"",
			lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Enter save  Esc cancel"),
		}, "\n")
		return render.SurfacePanel(body, max(36, v.width), v.t.Surface(), v.t.BorderColor())
	}
	body := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Add remote"),
		v.nameInput.View(),
		v.urlInput.View(),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Tab between fields  Enter submit  Esc cancel"),
	}, "\n")
	return render.SurfacePanel(body, max(36, v.width), v.t.Surface(), v.t.BorderColor())
}

// LoadRemotesCmd parses `git remote -v`.
func LoadRemotesCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		entries, err := loadRemotes(repoPath)
		return RemoteListMsg{RepoPath: repoPath, Entries: entries, Err: err}
	}
}

func loadRemotes(repoPath string) ([]RemoteEntry, error) {
	out, err := runGit(repoPath, "remote", "-v")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, trimGitOut(out))
	}
	m := map[string]*RemoteEntry{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		name, kind, url, ok := parseRemoteVerboseLine(line)
		if !ok {
			continue
		}
		e, ok := m[name]
		if !ok {
			e = &RemoteEntry{Name: name}
			m[name] = e
		}
		switch kind {
		case "fetch":
			e.FetchURL = url
		case "push":
			e.PushURL = url
		}
	}
	var entries []RemoteEntry
	for _, e := range m {
		entries = append(entries, *e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

func parseRemoteVerboseLine(line string) (name, kind, url string, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasSuffix(line, ")") {
		return "", "", "", false
	}
	// ... url (fetch)
	open := strings.LastIndex(line, "(")
	if open < 0 {
		return "", "", "", false
	}
	suffix := strings.TrimSpace(line[open+1 : len(line)-1])
	if suffix == "fetch" {
		kind = "fetch"
	} else if suffix == "push" {
		kind = "push"
	} else {
		return "", "", "", false
	}
	before := strings.TrimSpace(line[:open])
	// name and url: first field name, rest url (may contain spaces? URLs usually no spaces)
	parts := strings.Fields(before)
	if len(parts) < 2 {
		return "", "", "", false
	}
	name = parts[0]
	url = strings.Join(parts[1:], " ")
	return name, kind, url, true
}

func RemoteAddCmd(repoPath, name, url string) tea.Cmd {
	return func() tea.Msg {
		out, err := runGit(repoPath, "remote", "add", name, url)
		if err != nil {
			return RemoteOpResultMsg{Op: "add", Name: name, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return RemoteOpResultMsg{Op: "add", Name: name, Message: trimGitOut(out)}
	}
}

func RemoteRemoveCmd(repoPath, name string) tea.Cmd {
	return func() tea.Msg {
		out, err := runGit(repoPath, "remote", "remove", name)
		if err != nil {
			return RemoteOpResultMsg{Op: "remove", Name: name, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return RemoteOpResultMsg{Op: "remove", Name: name, Message: trimGitOut(out)}
	}
}

func RemoteSetURLCmd(repoPath, name, url string) tea.Cmd {
	return func() tea.Msg {
		out, err := runGit(repoPath, "remote", "set-url", name, url)
		if err != nil {
			return RemoteOpResultMsg{Op: "set-url", Name: name, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return RemoteOpResultMsg{Op: "set-url", Name: name, Message: trimGitOut(out)}
	}
}

func RemoteFetchCmd(repoPath, remote string) tea.Cmd {
	return func() tea.Msg {
		args := []string{"fetch"}
		if remote != "" {
			args = append(args, remote)
		}
		out, err := runGit(repoPath, args...)
		if err != nil {
			return RemoteOpResultMsg{Op: "fetch", Name: remote, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return RemoteOpResultMsg{Op: "fetch", Name: remote, Message: trimGitOut(out)}
	}
}

func RemotePruneCmd(repoPath, remote string) tea.Cmd {
	return func() tea.Msg {
		args := []string{"remote", "prune"}
		if remote != "" {
			args = append(args, remote)
		} else {
			args = append(args, "origin")
		}
		out, err := runGit(repoPath, args...)
		if err != nil {
			return RemoteOpResultMsg{Op: "prune", Name: remote, Err: fmt.Errorf("%w: %s", err, trimGitOut(out))}
		}
		return RemoteOpResultMsg{Op: "prune", Name: remote, Message: trimGitOut(out)}
	}
}
