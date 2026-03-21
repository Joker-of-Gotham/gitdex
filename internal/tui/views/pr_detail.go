package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type PRDetail struct {
	Number   int
	Title    string
	Author   string
	State    string
	Body     string
	Labels   []string
	Reviews  []PRReviewItem
	Files    []PRFileItem
	Comments []PRCommentItem
}

type PRReviewItem struct {
	User  string
	State string
}

type PRFileItem struct {
	Filename  string
	Status    string
	Additions int
	Deletions int
}

type PRCommentItem struct {
	User    string
	Body    string
	Created string
}

type PRDetailMsg struct {
	Detail PRDetail
	Err    error
}

type prDetailPrompt int

const (
	prPromptNone prDetailPrompt = iota
	prPromptComment
	prPromptApprove
	prPromptRequestChanges
	prPromptMerge
	prPromptClose
)

type PRDetailView struct {
	t       *theme.Theme
	detail  *PRDetail
	width   int
	height  int
	scroll  int
	message string
	prompt  textinput.Model
	mode    prDetailPrompt
}

func NewPRDetailView(t *theme.Theme) *PRDetailView {
	prompt := textinput.New()
	prompt.Prompt = ""
	prompt.CharLimit = 2048
	return &PRDetailView{t: t, prompt: prompt}
}

func (v *PRDetailView) ID() ID        { return "pr_detail" }
func (v *PRDetailView) Title() string { return "PR Detail" }
func (v *PRDetailView) Init() tea.Cmd { return nil }

func (v *PRDetailView) SetDetail(d *PRDetail) {
	v.detail = d
	v.scroll = 0
}

func (v *PRDetailView) PromptActive() bool {
	return v.mode != prPromptNone
}

func (v *PRDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case PRDetailMsg:
		if msg.Err != nil {
			v.detail = nil
			v.message = msg.Err.Error()
			v.scroll = 0
			return v, nil
		}
		v.detail = &msg.Detail
		v.message = ""
		v.scroll = 0
	case PRActionResultMsg:
		if msg.Err != nil {
			v.message = msg.Err.Error()
			return v, nil
		}
		if strings.TrimSpace(msg.Message) != "" {
			v.message = msg.Message
		} else {
			v.message = "PR action completed."
		}
		v.closePrompt()
	case tea.KeyPressMsg:
		if v.mode != prPromptNone {
			return v.handlePromptKey(msg)
		}
		switch msg.String() {
		case "up", "k":
			if v.scroll > 0 {
				v.scroll--
			}
		case "down", "j":
			v.scroll++
		case "pgup":
			v.scroll -= v.height / 2
			if v.scroll < 0 {
				v.scroll = 0
			}
		case "pgdown":
			v.scroll += v.height / 2
		case "c":
			return v.openPrompt(prPromptComment, "")
		case "a":
			return v.openPrompt(prPromptApprove, "")
		case "r":
			return v.openPrompt(prPromptRequestChanges, "")
		case "m":
			return v.openPrompt(prPromptMerge, "merge")
		case "x":
			return v.openPrompt(prPromptClose, "")
		}
	}
	return v, nil
}

func (v *PRDetailView) Render() string {
	if v.detail == nil {
		if v.message != "" {
			return lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  " + v.message)
		}
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  Select a pull request to inspect reviews, changed files, and discussion.")
	}

	d := v.detail
	var lines []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary())
	lines = append(lines, titleStyle.Render(fmt.Sprintf("  #%d %s", d.Number, d.Title)))

	metaStyle := lipgloss.NewStyle().Foreground(v.t.DimText())
	stateColor := v.t.Success()
	if d.State == "closed" {
		stateColor = v.t.Danger()
	} else if d.State == "merged" {
		stateColor = v.t.Accent()
	}
	stateBadge := lipgloss.NewStyle().Foreground(stateColor).Bold(true).Render(strings.ToUpper(d.State))
	lines = append(lines, metaStyle.Render(fmt.Sprintf("  %s  by %s", stateBadge, d.Author)))
	lines = append(lines, metaStyle.Render("  Actions: c comment  a approve  r request-changes  m merge  x close  Esc back"))
	if v.message != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.message))
	}

	if len(d.Labels) > 0 {
		lines = append(lines, metaStyle.Render("  Labels: "+strings.Join(d.Labels, ", ")))
	}
	lines = append(lines, "")

	if d.Body != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Fg()).Width(max(20, v.width-4)).Render("  "+d.Body))
		lines = append(lines, "")
	}

	if len(d.Reviews) > 0 {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(v.t.Info()).Render("  Reviews"))
		for _, review := range d.Reviews {
			lines = append(lines, fmt.Sprintf("    %s - %s", review.User, review.State))
		}
		lines = append(lines, "")
	}

	if len(d.Files) > 0 {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(v.t.Info()).Render(fmt.Sprintf("  Files Changed (%d)", len(d.Files))))
		for _, file := range d.Files {
			lines = append(lines, fmt.Sprintf("    %-10s %s  +%d -%d", strings.ToUpper(file.Status), file.Filename, file.Additions, file.Deletions))
		}
		lines = append(lines, "")
	}

	if len(d.Comments) > 0 {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(v.t.Info()).Render(fmt.Sprintf("  Comments (%d)", len(d.Comments))))
		for _, comment := range d.Comments {
			lines = append(lines, fmt.Sprintf("    %s (%s)", comment.User, comment.Created))
			lines = append(lines, "      "+comment.Body)
			lines = append(lines, "")
		}
	}

	content := v.applyScroll(lines)
	if v.mode != prPromptNone {
		return content + "\n\n" + v.renderPromptPanel()
	}
	return content
}

func (v *PRDetailView) applyScroll(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	if v.scroll >= len(lines) {
		v.scroll = len(lines) - 1
	}
	if v.scroll < 0 {
		v.scroll = 0
	}
	end := v.scroll + max(1, v.height)
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[v.scroll:end], "\n")
}

func (v *PRDetailView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *PRDetailView) openPrompt(mode prDetailPrompt, initial string) (View, tea.Cmd) {
	v.mode = mode
	v.prompt.SetValue(initial)
	v.prompt.Placeholder = initial
	return v, v.prompt.Focus()
}

func (v *PRDetailView) closePrompt() {
	v.mode = prPromptNone
	v.prompt.SetValue("")
	v.prompt.Blur()
}

func (v *PRDetailView) handlePromptKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.closePrompt()
		v.message = "PR action canceled."
		return v, nil
	case "enter":
		if v.detail == nil {
			v.closePrompt()
			return v, nil
		}
		body := strings.TrimSpace(v.prompt.Value())
		switch v.mode {
		case prPromptComment:
			if body == "" {
				v.message = "Comment cannot be empty."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestPRActionMsg{Number: v.detail.Number, Kind: PRActionComment, Body: body}
			}
		case prPromptApprove:
			return v, func() tea.Msg {
				return RequestPRActionMsg{Number: v.detail.Number, Kind: PRActionApprove, Body: body}
			}
		case prPromptRequestChanges:
			if body == "" {
				v.message = "Request changes requires a review note."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestPRActionMsg{Number: v.detail.Number, Kind: PRActionRequestChanges, Body: body}
			}
		case prPromptMerge:
			method, commitMsg, err := parseMergePrompt(body)
			if err != nil {
				v.message = err.Error()
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestPRActionMsg{Number: v.detail.Number, Kind: PRActionMerge, MergeMethod: method, Body: commitMsg}
			}
		case prPromptClose:
			if !strings.EqualFold(body, "close") {
				v.message = "Type close to confirm."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestPRActionMsg{Number: v.detail.Number, Kind: PRActionClose}
			}
		}
	}

	var cmd tea.Cmd
	v.prompt, cmd = v.prompt.Update(msg)
	return v, cmd
}

func (v *PRDetailView) renderPromptPanel() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(v.promptTitle())
	hint := lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(v.promptHint())
	lines := []string{title, hint, "", v.prompt.View(), "", lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Enter submit  Esc cancel")}
	return render.SurfacePanel(strings.Join(lines, "\n"), max(36, v.width), v.t.Surface(), v.t.BorderColor())
}

func (v *PRDetailView) promptTitle() string {
	switch v.mode {
	case prPromptComment:
		return "Comment on PR"
	case prPromptApprove:
		return "Approve PR"
	case prPromptRequestChanges:
		return "Request Changes"
	case prPromptMerge:
		return "Merge PR"
	case prPromptClose:
		return "Close PR"
	default:
		return "PR Action"
	}
}

func (v *PRDetailView) promptHint() string {
	switch v.mode {
	case prPromptComment:
		return "Add a discussion comment to the selected pull request."
	case prPromptApprove:
		return "Submit an approval review. Review text is optional."
	case prPromptRequestChanges:
		return "Submit a request-changes review with feedback."
	case prPromptMerge:
		return "Use merge, squash, or rebase. Add `-- commit message` if needed."
	case prPromptClose:
		return "Type close to confirm closing the selected pull request."
	default:
		return ""
	}
}

func parseMergePrompt(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "merge", "", nil
	}
	method := raw
	body := ""
	if idx := strings.Index(raw, " -- "); idx >= 0 {
		method = strings.TrimSpace(raw[:idx])
		body = strings.TrimSpace(raw[idx+4:])
	}
	if method == "" {
		method = "merge"
	}
	switch method {
	case "merge", "squash", "rebase":
		return method, body, nil
	default:
		return "", "", fmt.Errorf("merge strategy must be merge, squash, or rebase")
	}
}
