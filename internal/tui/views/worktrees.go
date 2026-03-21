package views

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type WorktreeEntry struct {
	Path     string
	Branch   string
	IsLocked bool
	IsBare   bool
}

type WorktreesView struct {
	entries    []WorktreeEntry
	cursor     int
	width      int
	height     int
	vp         viewport.Model
	repoPath   string
	statusMsg  string
	t          *theme.Theme
	prompt     textinput.Model
	promptKind worktreePromptKind
	createStep int // 0=path, 1=branch
	createPath string
}

type worktreePromptKind int

const (
	worktreePromptNone worktreePromptKind = iota
	worktreePromptCreatePath
	worktreePromptCreateBranch
	worktreePromptRemove
)

func NewWorktreesView(t *theme.Theme) *WorktreesView {
	p := textinput.New()
	p.Prompt = ""
	p.CharLimit = 1024
	return &WorktreesView{t: t, prompt: p}
}

func (v *WorktreesView) ID() ID        { return "worktrees" }
func (v *WorktreesView) Title() string { return "Worktrees" }
func (v *WorktreesView) Init() tea.Cmd { return nil }

func (v *WorktreesView) SetRepositoryPath(path string) {
	v.repoPath = path
}

func (v *WorktreesView) SetSize(w, h int) {
	v.width = w
	v.height = h
	vpH := max(3, h-6)
	v.vp = viewport.New(viewport.WithWidth(w), viewport.WithHeight(vpH))
	v.syncDetailViewport()
}

func (v *WorktreesView) syncDetailViewport() {
	if v.t == nil {
		return
	}
	e := v.selected()
	if e == nil {
		v.vp.SetContent("Select a worktree.")
		return
	}
	var b strings.Builder
	b.WriteString("Path: " + e.Path + "\n")
	b.WriteString("Branch: " + e.Branch + "\n")
	b.WriteString(fmt.Sprintf("Locked: %v\n", e.IsLocked))
	b.WriteString(fmt.Sprintf("Bare: %v\n", e.IsBare))
	v.vp.SetContent(b.String())
}

func (v *WorktreesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case WorktreeListMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
			v.entries = nil
			return v, nil
		}
		v.entries = msg.Entries
		v.cursor = 0
		if len(v.entries) == 0 {
			v.statusMsg = "No worktrees."
		} else {
			v.statusMsg = ""
		}
		v.syncDetailViewport()
		return v, nil
	case WorktreeOpResultMsg:
		if msg.Err != nil {
			v.statusMsg = fmt.Sprintf("%s failed: %v", msg.Op, msg.Err)
		} else {
			v.statusMsg = msg.Message
		}
		v.promptKind = worktreePromptNone
		v.prompt.Blur()
		if msg.Err == nil && msg.Op != "switch" {
			return v, LoadWorktreesCmd(v.repoPath)
		}
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *WorktreesView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.promptKind != worktreePromptNone {
		return v.handlePromptKey(msg)
	}

	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
			v.syncDetailViewport()
		}
	case "down", "j":
		if v.cursor < len(v.entries)-1 {
			v.cursor++
			v.syncDetailViewport()
		}
	case "enter":
		if e := v.selected(); e != nil && e.Path != "" {
			return v, func() tea.Msg {
				return RequestSwitchWorktreeMsg{Path: e.Path}
			}
		}
	case "c":
		if v.repoPath == "" {
			v.statusMsg = "No repository path."
			return v, nil
		}
		v.promptKind = worktreePromptCreatePath
		v.createStep = 0
		v.createPath = ""
		v.prompt.SetValue("")
		v.prompt.Placeholder = filepath.Join(v.repoPath, "wt-branch")
		return v, v.prompt.Focus()
	case "d":
		if v.repoPath == "" || v.selected() == nil {
			v.statusMsg = "No worktree selected."
			return v, nil
		}
		v.promptKind = worktreePromptRemove
		v.prompt.SetValue("")
		v.prompt.Placeholder = "type remove"
		return v, v.prompt.Focus()
	case "l":
		if v.repoPath == "" || v.selected() == nil {
			v.statusMsg = "No worktree selected."
			return v, nil
		}
		e := v.entries[v.cursor]
		return v, WorktreeLockToggleCmd(v.repoPath, e.Path, e.IsLocked)
	case "pgup":
		step := maxInt(1, (v.height-10)/2)
		v.cursor -= step
		if v.cursor < 0 {
			v.cursor = 0
		}
		v.syncDetailViewport()
	case "pgdown":
		step := maxInt(1, (v.height-10)/2)
		v.cursor += step
		if v.cursor >= len(v.entries) {
			v.cursor = maxInt(0, len(v.entries)-1)
		}
		v.syncDetailViewport()
	default:
		var cmd tea.Cmd
		v.vp, cmd = v.vp.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *WorktreesView) handlePromptKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.promptKind = worktreePromptNone
		v.prompt.Blur()
		v.statusMsg = "Canceled."
		return v, nil
	case "enter":
		switch v.promptKind {
		case worktreePromptCreatePath:
			p := strings.TrimSpace(v.prompt.Value())
			if p == "" {
				v.statusMsg = "Path required."
				return v, nil
			}
			v.createPath = p
			v.promptKind = worktreePromptCreateBranch
			v.prompt.SetValue("")
			v.prompt.Placeholder = "branch name or existing branch"
			return v, nil
		case worktreePromptCreateBranch:
			br := strings.TrimSpace(v.prompt.Value())
			if br == "" {
				v.statusMsg = "Branch name required."
				return v, nil
			}
			path := v.createPath
			v.promptKind = worktreePromptNone
			v.prompt.Blur()
			return v, WorktreeAddCmd(v.repoPath, path, br)
		case worktreePromptRemove:
			if strings.ToLower(strings.TrimSpace(v.prompt.Value())) != "remove" {
				v.statusMsg = "Type remove to confirm."
				return v, nil
			}
			path := v.entries[v.cursor].Path
			v.promptKind = worktreePromptNone
			v.prompt.Blur()
			return v, WorktreeRemoveCmd(v.repoPath, path)
		}
	}
	var cmd tea.Cmd
	v.prompt, cmd = v.prompt.Update(msg)
	return v, cmd
}

func (v *WorktreesView) selected() *WorktreeEntry {
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		return &v.entries[v.cursor]
	}
	return nil
}

func (v *WorktreesView) Render() string {
	if v.t == nil {
		return ""
	}
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("  Worktrees (%d)", len(v.entries))))
	lines = append(lines, lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  ↑/↓  Enter switch  c add  d remove  l lock/unlock"))
	if v.statusMsg != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusMsg))
	}
	lines = append(lines, "")

	viewH := max(1, v.height-8)
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
		lock := ""
		if e.IsLocked {
			lock = "locked"
		}
		bare := ""
		if e.IsBare {
			bare = "bare"
		}
		line := fmt.Sprintf("  %-40s  %-24s  %s %s", truncate(e.Path, 40), truncate(e.Branch, 24), lock, bare)
		if i == v.cursor {
			lines = append(lines, render.FillBlock(line, max(20, v.width-2), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection())))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Fg()).Render(line))
		}
	}

	out := strings.Join(lines, "\n")
	out += "\n\n"
	out += lipgloss.NewStyle().Bold(true).Foreground(v.t.Secondary()).Render("Detail") + "\n"
	out += v.vp.View()

	if v.promptKind != worktreePromptNone {
		title := "Worktree path"
		hint := "Enter directory for new worktree."
		if v.promptKind == worktreePromptCreateBranch {
			title = "Branch"
			hint = "Branch to checkout (git worktree add <path> <branch>)."
		} else if v.promptKind == worktreePromptRemove {
			title = "Remove worktree"
			hint = "Type remove to delete the selected worktree."
		}
		panel := strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(title),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(hint),
			"",
			v.prompt.View(),
		}, "\n")
		out += "\n" + panel
	}
	return out
}

// ParseWorktreePorcelain parses git worktree list --porcelain.
func ParseWorktreePorcelain(stdout string) []WorktreeEntry {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	var out []WorktreeEntry
	var cur *WorktreeEntry
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			if cur != nil {
				out = append(out, *cur)
			}
			p := strings.TrimPrefix(line, "worktree ")
			cur = &WorktreeEntry{Path: p}
			continue
		}
		if cur == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			cur.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			cur.IsBare = true
		case line == "locked":
			cur.IsLocked = true
		case strings.HasPrefix(line, "locked "):
			cur.IsLocked = true
		case strings.HasPrefix(line, "HEAD "):
			if cur.Branch == "" {
				cur.Branch = "(detached)"
			}
		}
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

// LoadWorktreesCmd runs git worktree list --porcelain.
func LoadWorktreesCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		if repoPath == "" {
			return WorktreeListMsg{Err: fmt.Errorf("no repository path")}
		}
		ex := gitops.NewGitExecutor()
		res, err := ex.Run(context.Background(), repoPath, "worktree", "list", "--porcelain")
		if err != nil {
			return WorktreeListMsg{Err: err}
		}
		return WorktreeListMsg{Entries: ParseWorktreePorcelain(res.Stdout)}
	}
}

// WorktreeAddCmd runs git worktree add <path> <branch>.
func WorktreeAddCmd(repoPath, wtPath, branch string) tea.Cmd {
	return func() tea.Msg {
		ex := gitops.NewGitExecutor()
		_, err := ex.Run(context.Background(), repoPath, "worktree", "add", wtPath, branch)
		return worktreeOpResult("create", "worktree added", err)
	}
}

// WorktreeRemoveCmd runs git worktree remove.
func WorktreeRemoveCmd(repoPath, wtPath string) tea.Cmd {
	return func() tea.Msg {
		ex := gitops.NewGitExecutor()
		_, err := ex.Run(context.Background(), repoPath, "worktree", "remove", "--force", wtPath)
		return worktreeOpResult("remove", "worktree removed", err)
	}
}

// WorktreeLockToggleCmd locks or unlocks the worktree at path.
func WorktreeLockToggleCmd(repoPath, wtPath string, isLocked bool) tea.Cmd {
	return func() tea.Msg {
		mgr := gitops.NewWorktreeManager(gitops.NewGitExecutor())
		ctx := context.Background()
		var err error
		op := "unlock"
		msg := "unlocked"
		if isLocked {
			err = mgr.Unlock(ctx, wtPath)
		} else {
			op = "lock"
			msg = "locked"
			err = mgr.Lock(ctx, wtPath, "gitdex")
		}
		if err != nil {
			return WorktreeOpResultMsg{Op: op, Err: err}
		}
		return WorktreeOpResultMsg{Op: op, Message: msg}
	}
}

func worktreeOpResult(op, msg string, err error) tea.Msg {
	if err != nil {
		return WorktreeOpResultMsg{Op: op, Err: err}
	}
	return WorktreeOpResultMsg{Op: op, Message: msg}
}
