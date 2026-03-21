package views

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type CommitEntry struct {
	Hash    string
	Author  string
	Date    string
	Message string
}

type CommitLogDataMsg struct {
	Commits []CommitEntry
}

type CommitDetailMsg struct {
	Hash    string
	Content string
	Err     error
}

type CommitSelectedMsg struct {
	Commit CommitEntry
}

type commitPromptKind int

const (
	commitPromptNone commitPromptKind = iota
	commitPromptCherryPick
	commitPromptRevert
)

type CommitLogView struct {
	t             *theme.Theme
	commits       []CommitEntry
	graphLines    []string
	cursor        int
	width         int
	height        int
	detail        bool
	detailHash    string
	detailContent string
	detailError   string
	statusLine    string
	editable      bool
	prompt        textinput.Model
	promptKind    commitPromptKind
	detailVP      viewport.Model
	detailVPInit  bool
	graphMode     bool
}

var commitHashLineRE = regexp.MustCompile(`\b([0-9a-f]{7,40})\b`)

func NewCommitLogView(t *theme.Theme) *CommitLogView {
	prompt := textinput.New()
	prompt.Prompt = ""
	prompt.CharLimit = 128
	return &CommitLogView{t: t, prompt: prompt}
}

func (v *CommitLogView) ID() ID        { return "commit_log" }
func (v *CommitLogView) Title() string { return "Commits" }
func (v *CommitLogView) Init() tea.Cmd { return nil }

func (v *CommitLogView) SetCommits(c []CommitEntry) {
	v.commits = c
	v.cursor = 0
}

func (v *CommitLogView) SetEditable(editable bool) {
	v.editable = editable
}

func (v *CommitLogView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case CommitLogDataMsg:
		v.commits = msg.Commits
		v.cursor = 0
	case CommitGraphMsg:
		v.graphLines = msg.Lines
		if v.cursor >= len(v.graphLines) && len(v.graphLines) > 0 {
			v.cursor = len(v.graphLines) - 1
		}
		if len(v.graphLines) > 0 {
			v.graphMode = true
		}
	case CommitDetailMsg:
		if msg.Err != nil {
			v.detailError = msg.Err.Error()
			v.detailContent = ""
		} else {
			v.detailHash = msg.Hash
			v.detailContent = msg.Content
			v.detailError = ""
		}
		return v, nil
	case CommitActionResultMsg:
		if msg.Err != nil {
			v.statusLine = msg.Err.Error()
			return v, nil
		}
		v.statusLine = msg.Message
		v.closePrompt()
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *CommitLogView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.promptKind != commitPromptNone {
		return v.handlePromptKey(msg)
	}

	prev := v.cursor
	count := len(v.commits)
	if v.graphMode && len(v.graphLines) > 0 {
		count = len(v.graphLines)
	}
	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case "down", "j":
		if v.cursor < count-1 {
			v.cursor++
		}
	case "g":
		v.cursor = 0
	case "G":
		if count > 0 {
			v.cursor = count - 1
		}
	case "pgup":
		viewH := maxInt(1, v.height-5)
		step := maxInt(1, viewH/2)
		v.cursor -= step
		if v.cursor < 0 {
			v.cursor = 0
		}
	case "pgdown":
		viewH := maxInt(1, v.height-5)
		step := maxInt(1, viewH/2)
		v.cursor += step
		if v.cursor >= count {
			v.cursor = maxInt(0, count-1)
		}
	case "ctrl+g":
		v.graphMode = !v.graphMode
		if v.graphMode {
			if len(v.graphLines) == 0 {
				v.statusLine = "Graph mode enabled. Refresh a local repository to load graph data."
			} else {
				v.cursor = 0
				v.statusLine = "Graph mode ON (Ctrl+G toggles)"
			}
		} else {
			v.cursor = 0
			v.statusLine = "Graph mode OFF (Ctrl+G toggles)"
		}
		return v, nil
	case "enter":
		if v.graphMode && len(v.graphLines) > 0 {
			hash := v.selectedGraphHash()
			if hash == "" {
				v.statusLine = "No commit hash on the selected graph row."
				return v, nil
			}
			v.detail = true
			return v, func() tea.Msg { return RequestCommitDetailMsg{Hash: hash} }
		}
		if commit := v.selected(); commit != nil {
			v.detail = true
			return v, tea.Batch(
				func() tea.Msg { return CommitSelectedMsg{Commit: *commit} },
				func() tea.Msg { return RequestCommitDetailMsg{Hash: commit.Hash} },
			)
		}
	case "esc":
		if v.detail {
			v.detail = false
			return v, nil
		}
	case "p":
		if !v.ensureEditable("Cherry-pick requires a local repository.") {
			return v, nil
		}
		if commit := v.selected(); commit != nil {
			return v.openPrompt(commitPromptCherryPick, commit.Hash)
		}
	case "v":
		if !v.ensureEditable("Revert requires a local repository.") {
			return v, nil
		}
		if commit := v.selected(); commit != nil {
			return v.openPrompt(commitPromptRevert, "")
		}
	}

	if v.cursor != prev {
		if v.graphMode && len(v.graphLines) > 0 {
			if hash := v.selectedGraphHash(); hash != "" && v.detail {
				return v, func() tea.Msg { return RequestCommitDetailMsg{Hash: hash} }
			}
		} else if commit := v.selected(); commit != nil {
			if v.detail {
				return v, func() tea.Msg { return RequestCommitDetailMsg{Hash: commit.Hash} }
			}
			return v, func() tea.Msg { return CommitSelectedMsg{Commit: *commit} }
		}
	}
	if v.detail && v.detailVPInit {
		var cmd tea.Cmd
		v.detailVP, cmd = v.detailVP.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *CommitLogView) Render() string {
	if len(v.commits) == 0 {
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  No commits loaded")
	}
	base := v.renderList(v.width)
	if v.promptKind != commitPromptNone {
		base += "\n\n" + v.renderPromptPanel()
	}
	return base
}

func (v *CommitLogView) renderList(width int) string {
	var lines []string
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary())
	hintStyle := lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true)
	tableHeader := lipgloss.NewStyle().Bold(true).Foreground(v.t.Info())

	modeLabel := ""
	if v.graphMode {
		modeLabel = "  [GRAPH]"
	}
	lines = append(lines, headerStyle.Render(fmt.Sprintf("  Commits (%d)%s", len(v.commits), modeLabel)))
	lines = append(lines, hintStyle.Render("  Up/Down  Enter inspect  p cherry-pick  v revert  Ctrl+G graph  PgUp/PgDn  g/G jump"))
	if v.statusLine != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusLine))
	}
	lines = append(lines, "")

	if v.graphMode {
		if len(v.graphLines) == 0 {
			lines = append(lines, lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  Commit graph data is not loaded for the current repository mode."))
		} else {
			viewH := maxInt(1, v.height-5)
			start := 0
			if v.cursor >= viewH {
				start = v.cursor - viewH + 1
			}
			end := start + viewH
			if end > len(v.graphLines) {
				end = len(v.graphLines)
			}
			for i := start; i < end; i++ {
				line := colorizeGraphLine(truncate(v.graphLines[i], maxInt(20, width-2)), v.t, false)
				if i == v.cursor {
					line = render.FillBlock(line, maxInt(20, width-2), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection()))
				}
				lines = append(lines, line)
			}
		}
		if v.detail {
			lines = append(lines, "", hintStyle.Render("  Inspector detail active. Use Ctrl+3 or Ctrl+I to review the selected commit."))
		}
		return strings.Join(lines, "\n")
	}

	colW := max(70, width-4)
	hashW := 8
	authorW := 16
	dateW := 12
	msgW := colW - hashW - authorW - dateW - 6
	if msgW < 16 {
		msgW = 16
	}
	lines = append(lines, tableHeader.Render(fmt.Sprintf("  %-8s %-16s %-12s %s", "Hash", "Author", "Date", "Message")))

	viewH := max(1, v.height-5)
	start := 0
	if v.cursor >= viewH {
		start = v.cursor - viewH + 1
	}
	end := start + viewH
	if end > len(v.commits) {
		end = len(v.commits)
	}

	for i := start; i < end; i++ {
		c := v.commits[i]
		hash := truncate(c.Hash, hashW)
		line := fmt.Sprintf("  %-8s %-16s %-12s %s", hash, truncate(c.Author, authorW), truncate(c.Date, dateW), truncate(c.Message, msgW))
		if i == v.cursor {
			lines = append(lines, render.FillBlock(line, maxInt(20, width-2), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection())))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Fg()).Render(line))
		}
	}

	if v.detail {
		lines = append(lines, "", hintStyle.Render("  Inspector detail active. Use Ctrl+3 or Ctrl+I to review the selected commit."))
	}

	return strings.Join(lines, "\n")
}

func (v *CommitLogView) renderDetail(width int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Secondary()).Render("Detail")
	if v.detailError != "" {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(v.t.Warning()).Render(v.detailError)
	}
	if strings.TrimSpace(v.detailContent) == "" {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Select a commit to inspect metadata and patch content.")
	}
	body := render.Code(v.detailContent, "commit.diff", max(40, width-2), v.t)
	v.detailVP.SetContent(body)
	mode := "remote read-only"
	if v.editable {
		mode = "local writable"
	}
	return strings.Join([]string{
		title,
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Raw show output / patch  (PgUp/PgDn scroll)"),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Repository mode: " + mode),
		"",
		v.detailVP.View(),
	}, "\n")
}

func (v *CommitLogView) selected() *CommitEntry {
	if v.cursor >= 0 && v.cursor < len(v.commits) {
		return &v.commits[v.cursor]
	}
	return nil
}

func (v *CommitLogView) selectedGraphHash() string {
	if v.cursor < 0 || v.cursor >= len(v.graphLines) {
		return ""
	}
	match := commitHashLineRE.FindStringSubmatch(v.graphLines[v.cursor])
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func (v *CommitLogView) splitWidths() (int, int) {
	listW := max(48, v.width*44/100)
	detailW := max(50, v.width-listW-1)
	return listW, detailW
}

func (v *CommitLogView) SetSize(w, h int) {
	v.width = w
	v.height = h
	vpH := maxInt(3, h-8)
	_, detailW := v.splitWidths()
	vpW := maxInt(20, detailW-4)
	if !v.detailVPInit {
		v.detailVP = viewport.New(viewport.WithWidth(vpW), viewport.WithHeight(vpH))
		v.detailVPInit = true
	} else {
		v.detailVP.SetWidth(vpW)
		v.detailVP.SetHeight(vpH)
	}
}

func (v *CommitLogView) ensureEditable(message string) bool {
	if v.editable {
		return true
	}
	v.statusLine = message
	return false
}

func (v *CommitLogView) openPrompt(kind commitPromptKind, initial string) (View, tea.Cmd) {
	v.promptKind = kind
	v.prompt.SetValue(initial)
	v.prompt.Placeholder = initial
	return v, v.prompt.Focus()
}

func (v *CommitLogView) closePrompt() {
	v.promptKind = commitPromptNone
	v.prompt.SetValue("")
	v.prompt.Blur()
}

func (v *CommitLogView) handlePromptKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.closePrompt()
		v.statusLine = "Commit action canceled."
		return v, nil
	case "enter":
		commit := v.selected()
		if commit == nil {
			v.closePrompt()
			return v, nil
		}
		switch v.promptKind {
		case commitPromptCherryPick:
			hash := strings.TrimSpace(v.prompt.Value())
			if hash == "" {
				hash = commit.Hash
			}
			return v, func() tea.Msg {
				return RequestCommitActionMsg{Hash: hash, Kind: CommitActionCherryPick}
			}
		case commitPromptRevert:
			if !strings.EqualFold(strings.TrimSpace(v.prompt.Value()), "revert") {
				v.statusLine = "Type revert to confirm."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestCommitActionMsg{Hash: commit.Hash, Kind: CommitActionRevert}
			}
		}
	}

	var cmd tea.Cmd
	v.prompt, cmd = v.prompt.Update(msg)
	return v, cmd
}

func (v *CommitLogView) renderPromptPanel() string {
	title := "Commit Action"
	hint := ""
	switch v.promptKind {
	case commitPromptCherryPick:
		title = "Cherry-pick Commit"
		hint = "Enter a commit hash to cherry-pick. Current selection is prefilled."
	case commitPromptRevert:
		title = "Revert Commit"
		hint = "Type revert to confirm creating a revert commit."
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(title),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(hint),
		"",
		v.prompt.View(),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Enter submit  Esc cancel"),
	}
	return render.SurfacePanel(strings.Join(lines, "\n"), max(36, v.width), v.t.Surface(), v.t.BorderColor())
}
