package views

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type RebaseEntry struct {
	Action  string
	Hash    string
	Message string
}

type InteractiveRebaseView struct {
	t *theme.Theme

	entries    []RebaseEntry
	cursor     int
	width      int
	height     int
	vp         viewport.Model
	repoPath   string
	targetRef  string
	statusMsg  string
	inProgress bool
}

func NewInteractiveRebaseView(t *theme.Theme) *InteractiveRebaseView {
	return &InteractiveRebaseView{t: t}
}

func (v *InteractiveRebaseView) ID() ID        { return ViewInteractiveRebase }
func (v *InteractiveRebaseView) Title() string { return "Interactive rebase" }
func (v *InteractiveRebaseView) Init() tea.Cmd { return nil }

func (v *InteractiveRebaseView) SetRepoPath(p string) { v.repoPath = p }

func (v *InteractiveRebaseView) SetTargetRef(ref string) { v.targetRef = strings.TrimSpace(ref) }

func (v *InteractiveRebaseView) SetSize(w, h int) {
	v.width, v.height = w, h
	v.vp = viewport.New(viewport.WithWidth(w), viewport.WithHeight(max(3, h-8)))
	v.syncViewport()
}

func (v *InteractiveRebaseView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case RebaseCommitsMsg:
		v.inProgress = false
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
			v.entries = nil
		} else {
			v.entries = msg.Entries
			v.targetRef = msg.TargetRef
			v.statusMsg = ""
			v.cursor = 0
		}
		v.syncViewport()
	case RebaseResultMsg:
		v.inProgress = false
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		} else {
			v.statusMsg = msg.Message
		}
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *InteractiveRebaseView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.inProgress {
		return v, nil
	}
	key := msg.String()
	switch key {
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
	case "K":
		if v.cursor > 0 {
			v.entries[v.cursor], v.entries[v.cursor-1] = v.entries[v.cursor-1], v.entries[v.cursor]
			v.cursor--
			v.syncViewport()
		}
	case "J":
		if v.cursor < len(v.entries)-1 {
			v.entries[v.cursor], v.entries[v.cursor+1] = v.entries[v.cursor+1], v.entries[v.cursor]
			v.cursor++
			v.syncViewport()
		}
	case "p", "s", "r", "e", "d", "f":
		if e := v.selected(); e != nil {
			switch key {
			case "p":
				e.Action = "pick"
			case "s":
				e.Action = "squash"
			case "r":
				e.Action = "reword"
			case "e":
				e.Action = "edit"
			case "d":
				e.Action = "drop"
			case "f":
				e.Action = "fixup"
			}
			v.syncViewport()
		}
	case "enter", "x":
		if v.repoPath == "" || v.targetRef == "" {
			v.statusMsg = "Set repo path and target ref first."
			return v, nil
		}
		if len(v.entries) == 0 {
			v.statusMsg = "No commits to rebase."
			return v, nil
		}
		v.inProgress = true
		return v, ExecuteInteractiveRebaseCmd(v.repoPath, v.targetRef, v.entries)
	case "esc":
		v.statusMsg = "Rebase planner canceled (no git command run)."
		return v, nil
	default:
		var cmd tea.Cmd
		v.vp, cmd = v.vp.Update(msg)
		v.syncViewport()
		return v, cmd
	}
	return v, nil
}

func (v *InteractiveRebaseView) selected() *RebaseEntry {
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		return &v.entries[v.cursor]
	}
	return nil
}

func (v *InteractiveRebaseView) Render() string {
	if v.t == nil || v.width < 10 {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("  Interactive rebase")
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render(
		"  p/s/r/e/d/f action  K/J reorder  Enter/x run  Esc cancel  target: " + v.targetRef,
	)
	if v.statusMsg != "" {
		title += "\n" + lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusMsg)
	}
	if v.inProgress {
		title += "\n" + lipgloss.NewStyle().Foreground(v.t.Info()).Render("  Running git rebase …")
	}
	title += "\n" + hint + "\n"
	v.syncViewport()
	return title + "\n" + v.vp.View()
}

func (v *InteractiveRebaseView) syncViewport() {
	if v.t == nil {
		return
	}
	w := max(20, v.width-2)
	var lines []string
	for i, e := range v.entries {
		act := e.Action
		if act == "" {
			act = "pick"
		}
		line := fmt.Sprintf("  %-8s %s  %s", act, truncate(e.Hash, 10), truncate(e.Message, max(16, w-30)))
		if i == v.cursor {
			line = lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection()).Width(w).Render(line)
		} else {
			line = lipgloss.NewStyle().Width(w).Foreground(v.t.Fg()).Render(line)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  No commits. LoadRebaseCommitsCmd(repo, targetRef).")}
	}
	v.vp.SetContent(strings.Join(lines, "\n"))
	v.vp.SetWidth(v.width)
	v.vp.SetHeight(max(3, v.height-8))
	if v.cursor >= 0 && v.cursor < len(v.entries) {
		v.vp.EnsureVisible(v.cursor, 0, w)
	}
}

// LoadRebaseCommitsCmd loads commits in `git log --reverse` order for target..HEAD.
func LoadRebaseCommitsCmd(repoPath, targetRef string) tea.Cmd {
	return func() tea.Msg {
		targetRef = strings.TrimSpace(targetRef)
		if targetRef == "" {
			return RebaseCommitsMsg{RepoPath: repoPath, TargetRef: targetRef, Err: fmt.Errorf("target ref is empty")}
		}
		if strings.TrimSpace(repoPath) == "" {
			return RebaseCommitsMsg{RepoPath: repoPath, TargetRef: targetRef, Err: fmt.Errorf("repository path is empty")}
		}
		ctx := context.Background()
		ex := gitops.NewGitExecutor()
		revRange := targetRef + "..HEAD"
		res, err := ex.Run(ctx, repoPath, "log", "--reverse", "--format=%h%x09%s", revRange)
		if err != nil {
			return RebaseCommitsMsg{RepoPath: repoPath, TargetRef: targetRef, Err: err}
		}
		var entries []RebaseEntry
		for _, line := range strings.Split(res.Stdout, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			hash, msg, ok := strings.Cut(line, "\t")
			if !ok {
				continue
			}
			entries = append(entries, RebaseEntry{
				Action:  "pick",
				Hash:    strings.TrimSpace(hash),
				Message: strings.TrimSpace(msg),
			})
		}
		return RebaseCommitsMsg{RepoPath: repoPath, TargetRef: targetRef, Entries: entries}
	}
}

// ExecuteInteractiveRebaseCmd writes a fixed rebase todo and runs `GIT_SEQUENCE_EDITOR=cp <todo> ` git rebase -i <targetRef>.
func ExecuteInteractiveRebaseCmd(repoPath, targetRef string, entries []RebaseEntry) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tmp, err := os.CreateTemp("", "gitdex-rebase-todo-*.txt")
		if err != nil {
			return RebaseResultMsg{RepoPath: repoPath, TargetRef: targetRef, Err: err}
		}
		todoPath := tmp.Name()
		defer func() { _ = os.Remove(todoPath) }()

		for _, e := range entries {
			action := strings.TrimSpace(e.Action)
			if action == "" {
				action = "pick"
			}
			hash := strings.TrimSpace(e.Hash)
			msg := strings.TrimSpace(e.Message)
			msg = strings.ReplaceAll(msg, "\n", " ")
			_, _ = fmt.Fprintf(tmp, "%s %s %s\n", action, hash, msg)
		}
		if err := tmp.Close(); err != nil {
			return RebaseResultMsg{RepoPath: repoPath, TargetRef: targetRef, Err: err}
		}

		env := append(os.Environ(), "GIT_SEQUENCE_EDITOR=cp "+strconv.Quote(todoPath)+" ")
		cmd := exec.CommandContext(ctx, "git", "rebase", "-i", strings.TrimSpace(targetRef))
		cmd.Dir = repoPath
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			return RebaseResultMsg{
				RepoPath:  repoPath,
				TargetRef: targetRef,
				Err:       fmt.Errorf("%w: %s", err, text),
			}
		}
		if text == "" {
			text = "rebase finished"
		}
		return RebaseResultMsg{RepoPath: repoPath, TargetRef: targetRef, Message: text}
	}
}
