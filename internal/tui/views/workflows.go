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

type WorkflowsView struct {
	t          *theme.Theme
	runs       []WorkflowRunEntry
	cursor     int
	width      int
	height     int
	detail     bool
	statusLine string
	prompt     textinput.Model
	dispatch   bool
}

func NewWorkflowsView(t *theme.Theme) *WorkflowsView {
	prompt := textinput.New()
	prompt.Prompt = ""
	prompt.CharLimit = 256
	return &WorkflowsView{t: t, prompt: prompt}
}

func (v *WorkflowsView) ID() ID        { return "workflows" }
func (v *WorkflowsView) Title() string { return "Workflows" }
func (v *WorkflowsView) Init() tea.Cmd { return nil }

func (v *WorkflowsView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *WorkflowsView) SetRuns(runs []WorkflowRunEntry) {
	v.runs = runs
	if v.cursor >= len(runs) {
		v.cursor = max(0, len(runs)-1)
	}
}

func (v *WorkflowsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case WorkflowRunsDataMsg:
		v.SetRuns(msg.Runs)
		return v, nil
	case WorkflowActionResultMsg:
		if msg.Err != nil {
			v.statusLine = msg.Err.Error()
		} else {
			v.statusLine = msg.Message
		}
		return v, nil
	case WorkflowDispatchResultMsg:
		if msg.Err != nil {
			v.statusLine = msg.Err.Error()
		} else {
			v.statusLine = msg.Message
		}
		v.dispatch = false
		v.prompt.SetValue("")
		v.prompt.Blur()
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *WorkflowsView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.dispatch {
		switch msg.String() {
		case "esc":
			v.dispatch = false
			v.prompt.SetValue("")
			v.prompt.Blur()
			v.statusLine = "Workflow dispatch canceled."
			return v, nil
		case "enter":
			run := v.selected()
			if run == nil {
				v.dispatch = false
				return v, nil
			}
			ref := strings.TrimSpace(v.prompt.Value())
			if ref == "" {
				ref = run.Branch
			}
			if ref == "" {
				v.statusLine = "Ref cannot be empty."
				return v, nil
			}
			if run.WorkflowID == 0 {
				v.statusLine = "Workflow ID unavailable for dispatch."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestWorkflowDispatchMsg{WorkflowID: run.WorkflowID, Ref: ref}
			}
		}
		var cmd tea.Cmd
		v.prompt, cmd = v.prompt.Update(msg)
		return v, cmd
	}

	prev := v.cursor
	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case "down", "j":
		if v.cursor < len(v.runs)-1 {
			v.cursor++
		}
	case "g":
		v.cursor = 0
	case "G":
		if len(v.runs) > 0 {
			v.cursor = len(v.runs) - 1
		}
	case "pgup":
		visible := maxInt(1, v.height-6)
		step := maxInt(1, visible/2)
		v.cursor -= step
		if v.cursor < 0 {
			v.cursor = 0
		}
	case "pgdown":
		visible := maxInt(1, v.height-6)
		step := maxInt(1, visible/2)
		v.cursor += step
		if v.cursor >= len(v.runs) {
			v.cursor = maxInt(0, len(v.runs)-1)
		}
	case "enter":
		run := v.selected()
		v.detail = !v.detail
		if run != nil {
			return v, func() tea.Msg { return WorkflowSelectedMsg{Run: *run} }
		}
	case "R":
		run := v.selected()
		if run == nil {
			return v, nil
		}
		if run.RunID == 0 {
			v.statusLine = "Workflow run ID unavailable."
			return v, nil
		}
		return v, func() tea.Msg {
			return RequestWorkflowActionMsg{RunID: run.RunID, Kind: WorkflowActionRerun}
		}
	case "esc":
		if v.detail {
			v.detail = false
			return v, nil
		}
	case "r":
		run := v.selected()
		if run == nil {
			return v, nil
		}
		if run.WorkflowID == 0 {
			v.statusLine = "Workflow ID unavailable for dispatch."
			return v, nil
		}
		v.dispatch = true
		v.prompt.SetValue(run.Branch)
		v.prompt.Placeholder = run.Branch
		return v, v.prompt.Focus()
	case "x":
		run := v.selected()
		if run == nil {
			return v, nil
		}
		if run.RunID == 0 {
			v.statusLine = "Workflow run ID unavailable."
			return v, nil
		}
		return v, func() tea.Msg {
			return RequestWorkflowActionMsg{RunID: run.RunID, Kind: WorkflowActionCancel}
		}
	}
	if v.cursor != prev && v.detail {
		if run := v.selected(); run != nil {
			return v, func() tea.Msg { return WorkflowSelectedMsg{Run: *run} }
		}
	}
	return v, nil
}

func (v *WorkflowsView) Render() string {
	if len(v.runs) == 0 {
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  No workflow runs loaded")
	}
	out := v.renderList(v.width)
	if v.dispatch {
		out += "\n\n" + v.renderPromptPanel()
	}
	return out
}

func (v *WorkflowsView) renderList(width int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("  Workflows (%d)", len(v.runs)))
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  Up/Down navigate  Enter inspect  r dispatch  R rerun  x cancel  Esc close detail")
	header := lipgloss.NewStyle().Bold(true).Foreground(v.t.Info()).Render("  Run ID   Workflow                  Branch           Status")
	lines := []string{title, hint}
	if v.statusLine != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusLine))
	}
	lines = append(lines, "", header)

	visible := max(1, v.height-6)
	start := 0
	if v.cursor >= visible {
		start = v.cursor - visible + 1
	}
	end := start + visible
	if end > len(v.runs) {
		end = len(v.runs)
	}

	for i := start; i < end; i++ {
		run := v.runs[i]
		state := strings.TrimSpace(run.Conclusion)
		if state == "" {
			state = run.Status
		}
		line := fmt.Sprintf("  %-8d %-24s %-16s %s", run.RunID, truncate(run.Name, 24), truncate(run.Branch, 16), truncate(strings.ToUpper(state), 18))
		if i == v.cursor {
			line = render.FillBlock(line, max(20, v.width-2), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection()))
		}
		lines = append(lines, line)
	}
	if v.detail {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  Inspector detail active. Use Ctrl+3 or Ctrl+I to review the selected workflow run."))
	}
	return strings.Join(lines, "\n")
}

func (v *WorkflowsView) renderPromptPanel() string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Dispatch Workflow"),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render("Enter the Git ref to dispatch. Current branch is prefilled."),
		"",
		v.prompt.View(),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Enter dispatch  Esc cancel"),
	}
	return render.SurfacePanel(strings.Join(lines, "\n"), max(36, v.width), v.t.Surface(), v.t.BorderColor())
}

func (v *WorkflowsView) selected() *WorkflowRunEntry {
	if v.cursor >= 0 && v.cursor < len(v.runs) {
		return &v.runs[v.cursor]
	}
	return nil
}
