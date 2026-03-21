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
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type TagEntry struct {
	Name        string
	IsAnnotated bool
	Message     string
	Commit      string
	Date        string
}

type TagsView struct {
	tags       []TagEntry
	cursor     int
	width      int
	height     int
	vp         viewport.Model
	repoPath   string
	statusMsg  string
	creating   bool
	tagName    string
	tagMsg     string
	t          *theme.Theme
	prompt     textinput.Model
	promptKind tagsPromptKind
	createStep int // 0=name, 1=message
}

type tagsPromptKind int

const (
	tagsPromptNone tagsPromptKind = iota
	tagsPromptCreate
	tagsPromptDelete
	tagsPromptPush
)

func NewTagsView(t *theme.Theme) *TagsView {
	p := textinput.New()
	p.Prompt = ""
	p.CharLimit = 512
	return &TagsView{t: t, prompt: p}
}

func (v *TagsView) ID() ID        { return "tags" }
func (v *TagsView) Title() string { return "Tags" }
func (v *TagsView) Init() tea.Cmd { return nil }

func (v *TagsView) SetRepositoryPath(path string) {
	v.repoPath = path
}

func (v *TagsView) SetSize(w, h int) {
	v.width = w
	v.height = h
	vpH := max(3, h-8)
	v.vp = viewport.New(viewport.WithWidth(w), viewport.WithHeight(vpH))
	v.syncDetailViewport()
}

func (v *TagsView) syncDetailViewport() {
	if v.t == nil || len(v.tags) == 0 {
		return
	}
	tag := v.selected()
	if tag == nil {
		return
	}
	var b strings.Builder
	b.WriteString("Tag: " + tag.Name + "\n")
	b.WriteString("Annotated: " + fmt.Sprintf("%v", tag.IsAnnotated) + "\n")
	b.WriteString("Commit: " + tag.Commit + "\n")
	b.WriteString("Date: " + tag.Date + "\n\n")
	if strings.TrimSpace(tag.Message) != "" {
		b.WriteString(render.Markdown(tag.Message, max(40, v.width-4)))
	} else {
		b.WriteString("(no message)")
	}
	v.vp.SetContent(b.String())
}

func (v *TagsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case TagsListMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
			v.tags = nil
			return v, nil
		}
		v.tags = msg.Tags
		v.cursor = 0
		if len(v.tags) == 0 {
			v.statusMsg = "No tags."
		} else {
			v.statusMsg = ""
		}
		v.syncDetailViewport()
		return v, nil
	case TagOpResultMsg:
		if msg.Err != nil {
			v.statusMsg = fmt.Sprintf("%s failed: %v", msg.Op, msg.Err)
		} else {
			v.statusMsg = msg.Message
		}
		v.creating = false
		v.tagName = ""
		v.tagMsg = ""
		v.promptKind = tagsPromptNone
		v.prompt.Blur()
		if msg.Err == nil {
			return v, LoadTagsCmd(v.repoPath)
		}
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *TagsView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.promptKind != tagsPromptNone || v.creating {
		return v.handlePromptKey(msg)
	}

	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
			v.syncDetailViewport()
		}
	case "down", "j":
		if v.cursor < len(v.tags)-1 {
			v.cursor++
			v.syncDetailViewport()
		}
	case "c":
		if v.repoPath == "" {
			v.statusMsg = "No repository path."
			return v, nil
		}
		v.creating = true
		v.createStep = 0
		v.tagName = ""
		v.tagMsg = ""
		v.prompt.SetValue("")
		v.prompt.Placeholder = "tag name"
		return v, v.prompt.Focus()
	case "d":
		if v.repoPath == "" || v.selected() == nil {
			v.statusMsg = "No tag selected."
			return v, nil
		}
		v.promptKind = tagsPromptDelete
		v.prompt.SetValue("")
		v.prompt.Placeholder = "type delete"
		return v, v.prompt.Focus()
	case "p":
		if v.repoPath == "" || v.selected() == nil {
			v.statusMsg = "No tag selected."
			return v, nil
		}
		v.promptKind = tagsPromptPush
		v.prompt.SetValue("origin")
		v.prompt.Placeholder = "remote (default origin)"
		return v, v.prompt.Focus()
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
		if v.cursor >= len(v.tags) {
			v.cursor = maxInt(0, len(v.tags)-1)
		}
		v.syncDetailViewport()
	default:
		var cmd tea.Cmd
		v.vp, cmd = v.vp.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *TagsView) handlePromptKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.creating = false
		v.tagName = ""
		v.tagMsg = ""
		v.promptKind = tagsPromptNone
		v.prompt.Blur()
		v.statusMsg = "Canceled."
		return v, nil
	case "enter":
		if v.creating {
			switch v.createStep {
			case 0:
				name := strings.TrimSpace(v.prompt.Value())
				if name == "" {
					v.statusMsg = "Tag name required."
					return v, nil
				}
				v.tagName = name
				v.createStep = 1
				v.prompt.SetValue("")
				v.prompt.Placeholder = "message (empty = lightweight tag)"
				return v, nil
			case 1:
				v.tagMsg = strings.TrimSpace(v.prompt.Value())
				v.creating = false
				v.prompt.Blur()
				return v, TagCreateCmd(v.repoPath, v.tagName, v.tagMsg)
			}
		}
		switch v.promptKind {
		case tagsPromptDelete:
			if strings.ToLower(strings.TrimSpace(v.prompt.Value())) != "delete" {
				v.statusMsg = "Type delete to confirm."
				return v, nil
			}
			name := v.selected().Name
			v.promptKind = tagsPromptNone
			v.prompt.Blur()
			return v, TagDeleteCmd(v.repoPath, name)
		case tagsPromptPush:
			remote := strings.TrimSpace(v.prompt.Value())
			if remote == "" {
				remote = "origin"
			}
			name := v.selected().Name
			v.promptKind = tagsPromptNone
			v.prompt.Blur()
			return v, TagPushCmd(v.repoPath, remote, name)
		}
	}
	var cmd tea.Cmd
	v.prompt, cmd = v.prompt.Update(msg)
	return v, cmd
}

func (v *TagsView) selected() *TagEntry {
	if v.cursor >= 0 && v.cursor < len(v.tags) {
		return &v.tags[v.cursor]
	}
	return nil
}

func (v *TagsView) Render() string {
	if v.t == nil {
		return ""
	}
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("  Tags (%d)", len(v.tags))))
	lines = append(lines, lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  ↑/↓  c create  d delete  p push"))
	if v.statusMsg != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusMsg))
	}
	lines = append(lines, "")

	viewH := max(1, v.height-10)
	start := 0
	if v.cursor >= viewH {
		start = v.cursor - viewH + 1
	}
	end := start + viewH
	if end > len(v.tags) {
		end = len(v.tags)
	}

	for i := start; i < end; i++ {
		tg := v.tags[i]
		ann := "light"
		if tg.IsAnnotated {
			ann = "ann"
		}
		line := fmt.Sprintf("  %-28s  %-4s  %-12s  %s", truncate(tg.Name, 28), ann, truncate(tg.Date, 12), truncate(tg.Message, 32))
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

	if v.creating || v.promptKind != tagsPromptNone {
		title := "Create tag"
		hint := "Enter tag name, then message (empty message = lightweight)."
		if v.creating && v.createStep == 1 {
			title = "Tag message"
			hint = "Empty message creates a lightweight tag."
		} else if v.promptKind == tagsPromptDelete {
			title = "Delete tag"
			hint = "Type delete to remove this tag locally."
		} else if v.promptKind == tagsPromptPush {
			title = "Push tag"
			hint = "Enter remote name (git push <remote> tag)."
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

func parseTagsList(stdout string) []TagEntry {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	var out []TagEntry
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 4 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		objType := strings.TrimSpace(parts[1])
		commit := strings.TrimSpace(parts[2])
		subj := strings.TrimSpace(parts[3])
		date := ""
		if len(parts) > 4 {
			date = strings.TrimSpace(parts[4])
		}
		isAnn := objType == "tag"
		out = append(out, TagEntry{
			Name:        name,
			IsAnnotated: isAnn,
			Message:     subj,
			Commit:      commit,
			Date:        date,
		})
	}
	return out
}

// LoadTagsCmd lists tags sorted by creator date (newest first).
func LoadTagsCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		if repoPath == "" {
			return TagsListMsg{Err: fmt.Errorf("no repository path")}
		}
		ex := gitops.NewGitExecutor()
		res, err := ex.Run(context.Background(), repoPath,
			"tag", "-l", "--sort=-creatordate",
			"--format=%(refname:short)%x09%(objecttype)%x09%(object)%x09%(subject)%x09%(creatordate:iso8601)",
		)
		if err != nil {
			return TagsListMsg{Err: err}
		}
		return TagsListMsg{Tags: parseTagsList(res.Stdout)}
	}
}

func tagOpResult(op, msg string, err error) tea.Msg {
	if err != nil {
		return TagOpResultMsg{Op: op, Err: err}
	}
	return TagOpResultMsg{Op: op, Message: msg}
}

// TagCreateCmd creates an annotated tag when message is non-empty.
func TagCreateCmd(repoPath, name, message string) tea.Cmd {
	return func() tea.Msg {
		ex := gitops.NewGitExecutor()
		var err error
		if strings.TrimSpace(message) != "" {
			_, err = ex.Run(context.Background(), repoPath, "tag", "-a", name, "-m", message)
		} else {
			_, err = ex.Run(context.Background(), repoPath, "tag", name)
		}
		return tagOpResult("create", "tag created", err)
	}
}

// TagDeleteCmd runs git tag -d.
func TagDeleteCmd(repoPath, name string) tea.Cmd {
	return func() tea.Msg {
		ex := gitops.NewGitExecutor()
		_, err := ex.Run(context.Background(), repoPath, "tag", "-d", name)
		return tagOpResult("delete", "tag deleted", err)
	}
}

// TagPushCmd runs git push <remote> <tag>.
func TagPushCmd(repoPath, remote, name string) tea.Cmd {
	return func() tea.Msg {
		ex := gitops.NewGitExecutor()
		_, err := ex.Run(context.Background(), repoPath, "push", remote, "refs/tags/"+name)
		if err != nil {
			return tagOpResult("push", "", err)
		}
		return tagOpResult("push", fmt.Sprintf("pushed %s to %s", name, remote), nil)
	}
}
