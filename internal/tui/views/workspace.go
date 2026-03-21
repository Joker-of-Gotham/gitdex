package views

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type WorkspaceView struct {
	theme     *theme.Theme
	width     int
	height    int
	plans     *PlansView
	tasks     *TasksView
	evidence  *EvidenceView
	cruise    *CruiseStatusView
	approvals *ApprovalQueueView
	active    int
	tabs      []string
}

func NewWorkspaceView(t *theme.Theme) *WorkspaceView {
	return &WorkspaceView{
		theme:     t,
		plans:     NewPlansView(t),
		tasks:     NewTasksView(t),
		evidence:  NewEvidenceView(t),
		cruise:    NewCruiseStatusView(t),
		approvals: NewApprovalQueueView(t),
		tabs:      []string{"Plans", "Tasks", "Evidence", "Cruise", "Approvals"},
	}
}

func (v *WorkspaceView) ID() ID        { return ViewWorkspace }
func (v *WorkspaceView) Title() string { return "Workspace" }
func (v *WorkspaceView) Init() tea.Cmd { return nil }

func (v *WorkspaceView) Plans() *PlansView             { return v.plans }
func (v *WorkspaceView) Tasks() *TasksView             { return v.tasks }
func (v *WorkspaceView) Evidence() *EvidenceView       { return v.evidence }
func (v *WorkspaceView) Cruise() *CruiseStatusView     { return v.cruise }
func (v *WorkspaceView) Approvals() *ApprovalQueueView { return v.approvals }
func (v *WorkspaceView) ActiveTab() int                { return v.active }
func (v *WorkspaceView) ActiveTabName() string {
	if v.active >= 0 && v.active < len(v.tabs) {
		return v.tabs[v.active]
	}
	return ""
}

func (v *WorkspaceView) SetSize(w, h int) {
	v.width = w
	v.height = h
	subH := max(h-3, 5)
	v.plans.SetSize(w, subH)
	v.tasks.SetSize(w, subH)
	v.evidence.SetSize(w, subH)
	v.cruise.SetSize(w, subH)
	v.approvals.SetSize(w, subH)
}

func (v *WorkspaceView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		k := msg.String()
		switch k {
		case "1":
			v.active = 0
			return v, nil
		case "2":
			v.active = 1
			return v, nil
		case "3":
			v.active = 2
			return v, nil
		case "4":
			v.active = 3
			return v, nil
		case "5":
			v.active = 4
			return v, nil
		case "left":
			if v.active > 0 {
				v.active--
			}
			return v, nil
		case "right":
			if v.active < len(v.tabs)-1 {
				v.active++
			}
			return v, nil
		}
		var cmd tea.Cmd
		switch v.active {
		case 0:
			_, cmd = v.plans.Update(msg)
		case 1:
			_, cmd = v.tasks.Update(msg)
		case 2:
			_, cmd = v.evidence.Update(msg)
		case 3:
			_, cmd = v.cruise.Update(msg)
		case 4:
			_, cmd = v.approvals.Update(msg)
		}
		return v, cmd
	default:
		if v.active == 0 {
			_, cmd := v.plans.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func (v *WorkspaceView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderSubTabs(v.tabs, v.active, v.theme, v.width))
	b.WriteString("\n")
	switch v.active {
	case 0:
		b.WriteString(v.plans.Render())
	case 1:
		b.WriteString(v.tasks.Render())
	case 2:
		b.WriteString(v.evidence.Render())
	case 3:
		b.WriteString(v.cruise.Render())
	case 4:
		b.WriteString(v.approvals.Render())
	}
	return b.String()
}
