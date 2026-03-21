package views

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type StashEntry struct {
	Index   int
	Message string
	Branch  string
	Date    string
}

type StashView struct {
	entries    []StashEntry
	cursor     int
	width      int
	height     int
	vp         viewport.Model
	repoPath   string
	statusMsg  string
	t          *theme.Theme
	showDiff   bool
	diffText   string
	diffErr    string
	prompt     textinput.Model
	promptKind stashPromptKind
}

type stashPromptKind int

const (
	stashPromptNone stashPromptKind = iota
	stashPromptBranchName
	stashPromptDropConfirm
)

func NewStashView(t *theme.Theme) *StashView {
	p := textinput.New()
	p.Prompt = ""
	p.CharLimit = 256
	return &StashView{t: t, prompt: p}
}

func (v *StashView) ID() ID        { return "stash" }
func (v *StashView) Title() string { return "Stash" }
func (v *StashView) Init() tea.Cmd { return nil }

func (v *StashView) SetRepositoryPath(path string) {
	v.repoPath = path
}

func (v *StashView) SetSize(w, h int) {
	v.width = w
	v.height = h
	vpH := max(3, h-6)
	v.vp = viewport.New(viewport.WithWidth(w), viewport.WithHeight(vpH))
	v.syncDiffViewport()
}

func (v *StashView) syncDiffViewport() {
	if !v.showDiff || v.t == nil {
		return
	}
	body := v.diffText
	if v.diffErr != "" {
		body = v.diffErr
	} else if strings.TrimSpace(body) == "" {
		body = "(no diff)"
	} else {
		body = render.Code(body, "diff", max(40, v.width-2), v.t)
	}
	v.vp.SetContent(body)
}

func (v *StashView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case StashListMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
			v.entries = nil
			return v, nil
		}
		v.entries = msg.Entries
		v.cursor = 0
		if len(v.entries) == 0 {
			v.statusMsg = "No stashes."
		} else {
			v.statusMsg = ""
		}
		return v, nil
	case StashDiffMsg:
		if msg.Err != nil {
			v.diffErr = msg.Err.Error()
			v.diffText = ""
		} else {
			v.diffErr = ""
			v.diffText = msg.Diff
		}
		v.syncDiffViewport()
		return v, nil
	case StashOpResultMsg:
		if msg.Err != nil {
			v.statusMsg = fmt.Sprintf("%s failed: %v", msg.Op, msg.Err)
		} else {
			v.statusMsg = msg.Message
		}
		v.promptKind = stashPromptNone
		v.prompt.Blur()
		if msg.Err == nil {
			return v, LoadStashCmd(v.repoPath)
		}
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *StashView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.promptKind != stashPromptNone {
		return v.handlePromptKey(msg)
	}
	if v.showDiff {
		switch msg.String() {
		case "esc", "q":
			v.showDiff = false
			v.diffText = ""
			v.diffErr = ""
			return v, nil
		case "up", "k", "down", "j", "pgup", "pgdown", "ctrl+u", "ctrl+d":
			var cmd tea.Cmd
			v.vp, cmd = v.vp.Update(msg)
			return v, cmd
		}
		return v, nil
	}

	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case "down", "j":
		if v.cursor < len(v.entries)-1 {
			v.cursor++
		}
	case "enter":
		if v.selected() == nil || v.repoPath == "" {
			return v, nil
		}
		v.showDiff = true
		v.diffText = ""
		v.diffErr = ""
		v.syncDiffViewport()
		return v, LoadStashDiffCmd(v.repoPath, v.entries[v.cursor].Index)
	case "a":
		if v.repoPath == "" || v.selected() == nil {
			v.statusMsg = "No repository or stash selected."
			return v, nil
		}
		return v, StashApplyCmd(v.repoPath, v.entries[v.cursor].Index)
	case "p":
		if v.repoPath == "" || v.selected() == nil {
			v.statusMsg = "No repository or stash selected."
			return v, nil
		}
		return v, StashPopCmd(v.repoPath, v.entries[v.cursor].Index)
	case "x":
		if v.repoPath == "" || v.selected() == nil {
			v.statusMsg = "No repository or stash selected."
			return v, nil
		}
		v.promptKind = stashPromptDropConfirm
		v.prompt.SetValue("")
		v.prompt.Placeholder = "type drop"
		return v, v.prompt.Focus()
	case "b":
		if v.repoPath == "" || v.selected() == nil {
			v.statusMsg = "No repository or stash selected."
			return v, nil
		}
		v.promptKind = stashPromptBranchName
		v.prompt.SetValue("")
		v.prompt.Placeholder = "new-branch-name"
		return v, v.prompt.Focus()
	}
	return v, nil
}

func (v *StashView) handlePromptKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.promptKind = stashPromptNone
		v.prompt.Blur()
		v.statusMsg = "Canceled."
		return v, nil
	case "enter":
		switch v.promptKind {
		case stashPromptDropConfirm:
			if strings.ToLower(strings.TrimSpace(v.prompt.Value())) != "drop" {
				v.statusMsg = "Type drop to confirm."
				return v, nil
			}
			idx := v.entries[v.cursor].Index
			v.promptKind = stashPromptNone
			v.prompt.Blur()
			return v, StashDropCmd(v.repoPath, idx)
		case stashPromptBranchName:
			name := strings.TrimSpace(v.prompt.Value())
			if name == "" {
				v.statusMsg = "Branch name required."
				return v, nil
			}
			idx := v.entries[v.cursor].Index
			v.promptKind = stashPromptNone
			v.prompt.Blur()
			return v, StashBranchCmd(v.repoPath, name, idx)
		}
	}
	var cmd tea.Cmd
	v.prompt, cmd = v.prompt.Update(msg)
	return v, cmd
}

func (v *StashView) selected() *StashEntry {
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		return &v.entries[v.cursor]
	}
	return nil
}

func (v *StashView) Render() string {
	if v.t == nil {
		return ""
	}
	if v.showDiff {
		header := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Stash diff (Esc back, arrows scroll)")
		if v.statusMsg != "" {
			header += "\n" + lipgloss.NewStyle().Foreground(v.t.Warning()).Render(v.statusMsg)
		}
		return header + "\n" + v.vp.View()
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("  Stash (%d)", len(v.entries))))
	lines = append(lines, lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  ↑/↓  Enter diff  a apply  p pop  x drop  b branch"))
	if v.statusMsg != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusMsg))
	}
	lines = append(lines, "")

	viewH := max(1, v.height-6)
	start := 0
	if v.cursor >= viewH {
		start = v.cursor - viewH + 1
	}
	end := start + viewH
	if end > len(v.entries) {
		end = len(v.entries)
	}

	for i := start; i < end; i++ {
		e := v.entries[i]
		line := fmt.Sprintf("  stash@{%d}  %s  %s  %s", e.Index, truncate(e.Branch, 20), truncate(e.Message, 40), truncate(e.Date, 16))
		if i == v.cursor {
			lines = append(lines, render.FillBlock(line, max(20, v.width-2), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection())))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Fg()).Render(line))
		}
	}

	out := strings.Join(lines, "\n")
	if v.promptKind != stashPromptNone {
		title := "Confirm drop"
		hint := "Type drop to remove this stash."
		if v.promptKind == stashPromptBranchName {
			title = "Branch from stash"
			hint = "Enter a new branch name (git stash branch)."
		}
		panel := strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(title),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(hint),
			"",
			v.prompt.View(),
		}, "\n")
		out += "\n\n" + panel
	}
	return out
}

var wipOnBranch = regexp.MustCompile(`(?i)\bWIP on ([^:]+):`)

func parseStashList(stdout string) []StashEntry {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	var out []StashEntry
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		ref := strings.TrimSpace(parts[0])
		msg := strings.TrimSpace(parts[1])
		date := ""
		if len(parts) > 2 {
			date = strings.TrimSpace(parts[2])
		}
		idx := parseStashIndex(ref)
		branch := ""
		if m := wipOnBranch.FindStringSubmatch(msg); len(m) > 1 {
			branch = strings.TrimSpace(m[1])
		}
		out = append(out, StashEntry{
			Index:   idx,
			Message: msg,
			Branch:  branch,
			Date:    date,
		})
	}
	return out
}

func parseStashIndex(ref string) int {
	ref = strings.TrimSpace(ref)
	// stash@{N}
	if i := strings.Index(ref, "@{"); i >= 0 {
		j := strings.Index(ref[i+2:], "}")
		if j >= 0 {
			n, err := strconv.Atoi(ref[i+2 : i+2+j])
			if err == nil {
				return n
			}
		}
	}
	return 0
}

// LoadStashCmd runs git stash list and returns StashListMsg.
func LoadStashCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		if repoPath == "" {
			return StashListMsg{Err: fmt.Errorf("no repository path")}
		}
		ex := gitops.NewGitExecutor()
		res, err := ex.Run(context.Background(), repoPath,
			"stash", "list",
			"--format=%gd%x09%gs%x09%cr",
		)
		if err != nil {
			return StashListMsg{Err: err}
		}
		return StashListMsg{Entries: parseStashList(res.Stdout)}
	}
}

// LoadStashDiffCmd loads patch for stash@{index}.
func LoadStashDiffCmd(repoPath string, index int) tea.Cmd {
	ref := fmt.Sprintf("stash@{%d}", index)
	return func() tea.Msg {
		ex := gitops.NewGitExecutor()
		res, err := ex.Run(context.Background(), repoPath, "stash", "show", "-p", ref)
		if err != nil {
			return StashDiffMsg{Err: err}
		}
		return StashDiffMsg{Diff: res.Stdout}
	}
}

func stashOpCmd(repoPath, opName string, args ...string) tea.Cmd {
	return func() tea.Msg {
		ex := gitops.NewGitExecutor()
		_, err := ex.Run(context.Background(), repoPath, args...)
		if err != nil {
			return StashOpResultMsg{Op: opName, Err: err}
		}
		return StashOpResultMsg{Op: opName, Message: opName + " ok"}
	}
}

// StashApplyCmd runs git stash apply stash@{index}.
func StashApplyCmd(repoPath string, index int) tea.Cmd {
	ref := fmt.Sprintf("stash@{%d}", index)
	return stashOpCmd(repoPath, "apply", "stash", "apply", ref)
}

// StashPopCmd runs git stash pop stash@{index}.
func StashPopCmd(repoPath string, index int) tea.Cmd {
	ref := fmt.Sprintf("stash@{%d}", index)
	return stashOpCmd(repoPath, "pop", "stash", "pop", ref)
}

// StashDropCmd runs git stash drop stash@{index}.
func StashDropCmd(repoPath string, index int) tea.Cmd {
	ref := fmt.Sprintf("stash@{%d}", index)
	return stashOpCmd(repoPath, "drop", "stash", "drop", ref)
}

// StashBranchCmd runs git stash branch <name> stash@{index}.
func StashBranchCmd(repoPath, branch string, index int) tea.Cmd {
	ref := fmt.Sprintf("stash@{%d}", index)
	return stashOpCmd(repoPath, "branch", "stash", "branch", branch, ref)
}
