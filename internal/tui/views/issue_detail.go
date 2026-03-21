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

type IssueDetail struct {
	Number    int
	Title     string
	Author    string
	State     string
	Body      string
	Labels    []string
	Assignees []string
	Milestone string
	Comments  []IssueCommentItem
}

type IssueCommentItem struct {
	User    string
	Body    string
	Created string
}

type IssueDetailMsg struct {
	Detail IssueDetail
	Err    error
}

type issueDetailPrompt int

const (
	issuePromptNone issueDetailPrompt = iota
	issuePromptComment
	issuePromptLabel
	issuePromptAssign
	issuePromptClose
	issuePromptReopen
)

type IssueDetailView struct {
	t       *theme.Theme
	detail  *IssueDetail
	width   int
	height  int
	scroll  int
	message string
	prompt  textinput.Model
	mode    issueDetailPrompt
}

func NewIssueDetailView(t *theme.Theme) *IssueDetailView {
	prompt := textinput.New()
	prompt.Prompt = ""
	prompt.CharLimit = 2048
	return &IssueDetailView{t: t, prompt: prompt}
}

func (v *IssueDetailView) ID() ID        { return "issue_detail" }
func (v *IssueDetailView) Title() string { return "Issue Detail" }
func (v *IssueDetailView) Init() tea.Cmd { return nil }

func (v *IssueDetailView) SetDetail(d *IssueDetail) {
	v.detail = d
	v.scroll = 0
}

func (v *IssueDetailView) PromptActive() bool {
	return v.mode != issuePromptNone
}

func (v *IssueDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case IssueDetailMsg:
		if msg.Err != nil {
			v.detail = nil
			v.message = msg.Err.Error()
			v.scroll = 0
			return v, nil
		}
		v.detail = &msg.Detail
		v.message = ""
		v.scroll = 0
	case IssueActionResultMsg:
		if msg.Err != nil {
			v.message = msg.Err.Error()
			return v, nil
		}
		if strings.TrimSpace(msg.Message) != "" {
			v.message = msg.Message
		} else {
			v.message = "Issue action completed."
		}
		v.closePrompt()
	case tea.KeyPressMsg:
		if v.mode != issuePromptNone {
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
			return v.openPrompt(issuePromptComment, "")
		case "l":
			initial := ""
			if v.detail != nil {
				initial = strings.Join(v.detail.Labels, ",")
			}
			return v.openPrompt(issuePromptLabel, initial)
		case "a":
			initial := ""
			if v.detail != nil {
				initial = strings.Join(v.detail.Assignees, ",")
			}
			return v.openPrompt(issuePromptAssign, initial)
		case "x":
			if v.detail != nil && strings.EqualFold(v.detail.State, "open") {
				return v.openPrompt(issuePromptClose, "")
			}
			return v.openPrompt(issuePromptReopen, "")
		}
	}
	return v, nil
}

func (v *IssueDetailView) Render() string {
	if v.detail == nil {
		if v.message != "" {
			return lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  " + v.message)
		}
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  Select an issue to inspect labels, assignees, and discussion.")
	}

	d := v.detail
	var lines []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary())
	lines = append(lines, titleStyle.Render(fmt.Sprintf("  #%d %s", d.Number, d.Title)))

	metaStyle := lipgloss.NewStyle().Foreground(v.t.DimText())
	stateColor := v.t.Success()
	if strings.EqualFold(d.State, "closed") {
		stateColor = v.t.Danger()
	}
	stateBadge := lipgloss.NewStyle().Foreground(stateColor).Bold(true).Render(strings.ToUpper(d.State))
	lines = append(lines, metaStyle.Render(fmt.Sprintf("  %s  by %s", stateBadge, d.Author)))
	lines = append(lines, metaStyle.Render("  Actions: c comment  l labels  a assignees  x close/reopen  Esc back"))
	if v.message != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.message))
	}

	if len(d.Labels) > 0 {
		lines = append(lines, metaStyle.Render("  Labels: "+strings.Join(d.Labels, ", ")))
	}
	if len(d.Assignees) > 0 {
		lines = append(lines, metaStyle.Render("  Assignees: "+strings.Join(d.Assignees, ", ")))
	}
	if d.Milestone != "" {
		lines = append(lines, metaStyle.Render("  Milestone: "+d.Milestone))
	}
	lines = append(lines, "")

	if d.Body != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Fg()).Width(max(20, v.width-4)).Render("  "+d.Body))
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
	if v.mode != issuePromptNone {
		return content + "\n\n" + v.renderPromptPanel()
	}
	return content
}

func (v *IssueDetailView) applyScroll(lines []string) string {
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

func (v *IssueDetailView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *IssueDetailView) openPrompt(mode issueDetailPrompt, initial string) (View, tea.Cmd) {
	v.mode = mode
	v.prompt.SetValue(initial)
	v.prompt.Placeholder = initial
	return v, v.prompt.Focus()
}

func (v *IssueDetailView) closePrompt() {
	v.mode = issuePromptNone
	v.prompt.SetValue("")
	v.prompt.Blur()
}

func (v *IssueDetailView) handlePromptKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.closePrompt()
		v.message = "Issue action canceled."
		return v, nil
	case "enter":
		if v.detail == nil {
			v.closePrompt()
			return v, nil
		}
		body := strings.TrimSpace(v.prompt.Value())
		switch v.mode {
		case issuePromptComment:
			if body == "" {
				v.message = "Comment cannot be empty."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestIssueActionMsg{Number: v.detail.Number, Kind: IssueActionComment, Body: body}
			}
		case issuePromptLabel:
			return v, func() tea.Msg {
				return RequestIssueActionMsg{Number: v.detail.Number, Kind: IssueActionLabel, Values: parseCSVList(body)}
			}
		case issuePromptAssign:
			return v, func() tea.Msg {
				return RequestIssueActionMsg{Number: v.detail.Number, Kind: IssueActionAssign, Values: parseCSVList(body)}
			}
		case issuePromptClose:
			if !strings.EqualFold(body, "close") {
				v.message = "Type close to confirm."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestIssueActionMsg{Number: v.detail.Number, Kind: IssueActionClose}
			}
		case issuePromptReopen:
			if !strings.EqualFold(body, "reopen") {
				v.message = "Type reopen to confirm."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestIssueActionMsg{Number: v.detail.Number, Kind: IssueActionReopen}
			}
		}
	}

	var cmd tea.Cmd
	v.prompt, cmd = v.prompt.Update(msg)
	return v, cmd
}

func (v *IssueDetailView) renderPromptPanel() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(v.promptTitle())
	hint := lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(v.promptHint())
	lines := []string{title, hint, "", v.prompt.View(), "", lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Enter submit  Esc cancel")}
	return render.SurfacePanel(strings.Join(lines, "\n"), max(36, v.width), v.t.Surface(), v.t.BorderColor())
}

func (v *IssueDetailView) promptTitle() string {
	switch v.mode {
	case issuePromptComment:
		return "Comment on Issue"
	case issuePromptLabel:
		return "Set Labels"
	case issuePromptAssign:
		return "Set Assignees"
	case issuePromptClose:
		return "Close Issue"
	case issuePromptReopen:
		return "Reopen Issue"
	default:
		return "Issue Action"
	}
}

func (v *IssueDetailView) promptHint() string {
	switch v.mode {
	case issuePromptComment:
		return "Add a new issue comment."
	case issuePromptLabel:
		return "Comma-separated labels."
	case issuePromptAssign:
		return "Comma-separated GitHub logins."
	case issuePromptClose:
		return "Type close to confirm."
	case issuePromptReopen:
		return "Type reopen to confirm."
	default:
		return ""
	}
}

func parseCSVList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
