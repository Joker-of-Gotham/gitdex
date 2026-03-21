package views

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type BisectState int

const (
	BisectIdle BisectState = iota
	BisectActive
	BisectFound
)

type BisectView struct {
	state       BisectState
	currentHash string
	remaining   int
	goodHash    string
	badHash     string
	width       int
	height      int
	vp          viewport.Model
	repoPath    string
	statusMsg   string
	log         []string

	t *theme.Theme

	promptStart bool
	startInput  textinput.Model
}

func NewBisectView(t *theme.Theme) *BisectView {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "bad_ref good_ref"
	in.CharLimit = 256
	return &BisectView{t: t, startInput: in}
}

func (v *BisectView) ID() ID        { return ViewBisect }
func (v *BisectView) Title() string { return "Bisect" }
func (v *BisectView) Init() tea.Cmd { return nil }

func (v *BisectView) SetRepoPath(p string) { v.repoPath = p }

func (v *BisectView) SetSize(w, h int) {
	v.width, v.height = w, h
	v.vp = viewport.New(viewport.WithWidth(w), viewport.WithHeight(max(3, h-12)))
	iw := max(20, w-6)
	v.startInput.SetWidth(iw)
	v.syncViewport()
}

func (v *BisectView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case BisectResultMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		} else {
			v.statusMsg = msg.Message
		}
		if len(msg.LogLines) > 0 {
			v.log = msg.LogLines
		}
		v.refreshStateFromResult(msg)
		v.promptStart = false
		v.startInput.Blur()
		v.syncViewport()
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *BisectView) refreshStateFromResult(msg BisectResultMsg) {
	if msg.Action != BisectActionLog {
		if msg.CurrentHash != "" {
			v.currentHash = msg.CurrentHash
		}
		if msg.GoodHash != "" {
			v.goodHash = msg.GoodHash
		}
		if msg.BadHash != "" {
			v.badHash = msg.BadHash
		}
		v.remaining = msg.Remaining
	}
	switch msg.Action {
	case BisectActionStart, BisectActionGood, BisectActionBad, BisectActionSkip:
		v.state = BisectActive
	case BisectActionReset:
		v.state = BisectIdle
		v.goodHash, v.badHash = "", ""
		v.remaining = 0
		v.currentHash = ""
	case BisectActionLog:
		// keep refs; log text is in LogLines
	default:
		if v.state == BisectIdle && msg.Action == "" {
			return
		}
	}
	if strings.Contains(strings.ToLower(msg.Message), "first bad commit") {
		v.state = BisectFound
	}
}

func (v *BisectView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.promptStart {
		switch msg.String() {
		case "esc":
			v.promptStart = false
			v.startInput.Blur()
			v.statusMsg = "Bisect start canceled."
			return v, nil
		case "enter":
			line := strings.Fields(strings.TrimSpace(v.startInput.Value()))
			v.promptStart = false
			v.startInput.Blur()
			v.startInput.SetValue("")
			if len(line) < 2 {
				v.statusMsg = "Need two refs: bad_ref good_ref"
				return v, nil
			}
			badRef, goodRef := line[0], line[1]
			return v, BisectStartCmd(v.repoPath, badRef, goodRef)
		}
		var cmd tea.Cmd
		v.startInput, cmd = v.startInput.Update(msg)
		return v, cmd
	}

	switch msg.String() {
	case "s":
		if v.repoPath == "" {
			v.statusMsg = "No repository path."
			return v, nil
		}
		v.promptStart = true
		v.statusMsg = "Enter bad_ref and good_ref (space-separated), Enter confirm, Esc cancel."
		return v, v.startInput.Focus()
	case "g":
		return v, BisectActionCmd(v.repoPath, BisectActionGood)
	case "b":
		return v, BisectActionCmd(v.repoPath, BisectActionBad)
	case "k":
		return v, BisectActionCmd(v.repoPath, BisectActionSkip)
	case "r":
		return v, BisectActionCmd(v.repoPath, BisectActionReset)
	case "l":
		return v, BisectActionCmd(v.repoPath, BisectActionLog)
	default:
		var cmd tea.Cmd
		v.vp, cmd = v.vp.Update(msg)
		v.syncViewport()
		return v, cmd
	}
}

func (v *BisectView) Render() string {
	if v.t == nil || v.width < 10 {
		return ""
	}
	stateLabel := "idle"
	switch v.state {
	case BisectActive:
		stateLabel = "active"
	case BisectFound:
		stateLabel = "found"
	}
	head := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("  Git bisect"),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render(
			fmt.Sprintf("  state=%s  HEAD=%s  good=%s  bad=%s  ~remaining=%d",
				stateLabel, truncate(v.currentHash, 12), truncate(v.goodHash, 10), truncate(v.badHash, 10), v.remaining),
		),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render(
			"  s start  g good  b bad  k skip  r reset  l log  PgUp/PgDn scroll",
		),
	}
	if v.statusMsg != "" {
		head = append(head, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusMsg))
	}
	header := strings.Join(head, "\n") + "\n"
	v.syncViewport()
	body := v.vp.View()
	if v.promptStart {
		panel := lipgloss.NewStyle().Foreground(v.t.Fg()).Render(v.startInput.View())
		return header + "\n" + panel + "\n" + body
	}
	return header + "\n" + body
}

func (v *BisectView) syncViewport() {
	if v.t == nil {
		return
	}
	content := strings.Join(v.log, "\n")
	if strings.TrimSpace(content) == "" {
		content = lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  (bisect log empty — press l after starting, or start with s)")
	}
	v.vp.SetContent(content)
	v.vp.SetWidth(v.width)
	v.vp.SetHeight(max(3, v.height-12))
}

// BisectStartCmd runs `git bisect start <badRef> <goodRef>`.
func BisectStartCmd(repoPath, badRef, goodRef string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(repoPath) == "" {
			return BisectResultMsg{Action: BisectActionStart, Err: fmt.Errorf("repository path is empty")}
		}
		badRef, goodRef = strings.TrimSpace(badRef), strings.TrimSpace(goodRef)
		if badRef == "" || goodRef == "" {
			return BisectResultMsg{Action: BisectActionStart, Err: fmt.Errorf("bad and good refs are required")}
		}
		ctx := context.Background()
		ex := gitops.NewGitExecutor()
		res, err := ex.Run(ctx, repoPath, "bisect", "start", badRef, goodRef)
		if err != nil {
			return BisectResultMsg{Action: BisectActionStart, Err: err}
		}
		msg := strings.TrimSpace(res.Stdout)
		if msg == "" {
			msg = strings.TrimSpace(res.Stderr)
		}
		if msg == "" {
			msg = "bisect started"
		}
		out := BisectResultMsg{Action: BisectActionStart, Message: msg}
		enrichBisectSnapshot(ctx, ex, repoPath, &out)
		return out
	}
}

// BisectActionCmd runs a bisect subcommand (good, bad, skip, reset, log).
func BisectActionCmd(repoPath string, action BisectActionKind) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(repoPath) == "" {
			return BisectResultMsg{Action: action, Err: fmt.Errorf("repository path is empty")}
		}
		ctx := context.Background()
		ex := gitops.NewGitExecutor()
		switch action {
		case BisectActionGood:
			res, err := ex.Run(ctx, repoPath, "bisect", "good")
			return bisectFinishWithSnapshot(ctx, ex, repoPath, action, res, err)
		case BisectActionBad:
			res, err := ex.Run(ctx, repoPath, "bisect", "bad")
			return bisectFinishWithSnapshot(ctx, ex, repoPath, action, res, err)
		case BisectActionSkip:
			res, err := ex.Run(ctx, repoPath, "bisect", "skip")
			return bisectFinishWithSnapshot(ctx, ex, repoPath, action, res, err)
		case BisectActionReset:
			res, err := ex.Run(ctx, repoPath, "bisect", "reset")
			return bisectFinish(action, res, err)
		case BisectActionLog:
			res, err := ex.Run(ctx, repoPath, "bisect", "log")
			if err != nil {
				return BisectResultMsg{Action: action, Err: err}
			}
			lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
			return BisectResultMsg{Action: action, Message: "bisect log", LogLines: lines}
		default:
			return BisectResultMsg{Action: action, Err: fmt.Errorf("unsupported bisect action")}
		}
	}
}

func bisectFinish(action BisectActionKind, res *gitops.GitResult, err error) tea.Msg {
	if err != nil {
		return BisectResultMsg{Action: action, Err: err}
	}
	msg := strings.TrimSpace(res.Stdout)
	if msg == "" {
		msg = strings.TrimSpace(res.Stderr)
	}
	if msg == "" {
		msg = "ok"
	}
	return BisectResultMsg{Action: action, Message: msg}
}

func bisectFinishWithSnapshot(ctx context.Context, ex *gitops.GitExecutor, repoPath string, action BisectActionKind, res *gitops.GitResult, err error) tea.Msg {
	msg := bisectFinish(action, res, err)
	b, ok := msg.(BisectResultMsg)
	if !ok || b.Err != nil {
		return msg
	}
	enrichBisectSnapshot(ctx, ex, repoPath, &b)
	return b
}

func enrichBisectSnapshot(ctx context.Context, ex *gitops.GitExecutor, repoPath string, out *BisectResultMsg) {
	head, err := ex.Run(ctx, repoPath, "rev-parse", "HEAD")
	if err == nil {
		out.CurrentHash = strings.TrimSpace(head.Stdout)
	}
	if g, err := ex.Run(ctx, repoPath, "rev-parse", "--verify", "refs/bisect/good"); err == nil {
		out.GoodHash = strings.TrimSpace(g.Stdout)
	}
	if b, err := ex.Run(ctx, repoPath, "rev-parse", "--verify", "refs/bisect/bad"); err == nil {
		out.BadHash = strings.TrimSpace(b.Stdout)
	}
	if log, err := ex.Run(ctx, repoPath, "bisect", "log"); err == nil {
		out.Remaining = strings.Count(strings.TrimSpace(log.Stdout), "\n")
		if out.Remaining == 0 && strings.TrimSpace(log.Stdout) != "" {
			out.Remaining = 1
		}
	}
}
