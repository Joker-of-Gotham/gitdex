package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/tui/theme"
)

// ApprovalQueueMsg replaces the pending approvals list.
type ApprovalQueueMsg struct {
	Pending []ApprovalItem
}

// ApprovalActionMsg is emitted when the user approves or rejects an item.
type ApprovalActionMsg struct {
	PlanID   string
	Approved bool
}

// ApprovalItem is one pending approval row in the queue UI.
type ApprovalItem struct {
	ID          string
	Action      string
	RiskLevel   string
	Requester   string
	Timestamp   time.Time
	Description string
}

// ApprovalItemFromPlan maps an autonomy plan to a row.
func ApprovalItemFromPlan(p autonomy.ActionPlan) ApprovalItem {
	action := ""
	if len(p.Steps) > 0 {
		action = p.Steps[0].Action
	}
	risk := p.RiskLevel.String()
	if p.RiskLevel == 0 && p.RiskLevelStr != "" {
		risk = p.RiskLevelStr
	}
	ts := p.CreatedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return ApprovalItem{
		ID:          p.ID,
		Action:      action,
		RiskLevel:   risk,
		Requester:   "cruise",
		Timestamp:   ts,
		Description: p.Description,
	}
}

// ApprovalQueueView lists pending approvals with approve/reject/detail keys.
type ApprovalQueueView struct {
	t       *theme.Theme
	pending []ApprovalItem
	cursor  int

	width, height int
	vp            viewport.Model
	detail        bool
}

func NewApprovalQueueView(t *theme.Theme) *ApprovalQueueView {
	return &ApprovalQueueView{t: t}
}

func (v *ApprovalQueueView) ID() ID        { return ViewApprovalQueue }
func (v *ApprovalQueueView) Title() string { return "Approvals" }
func (v *ApprovalQueueView) Init() tea.Cmd { return nil }

func (v *ApprovalQueueView) SetPendingItems(items []ApprovalItem) {
	v.pending = items
	v.cursor = 0
	v.syncViewport()
}

func (v *ApprovalQueueView) SetPendingPlans(plans []autonomy.ActionPlan) {
	items := make([]ApprovalItem, 0, len(plans))
	for _, p := range plans {
		items = append(items, ApprovalItemFromPlan(p))
	}
	v.SetPendingItems(items)
}

func (v *ApprovalQueueView) SetSize(w, h int) {
	v.width, v.height = w, h
	vpH := max(3, h-6)
	v.vp = viewport.New(viewport.WithWidth(max(20, w-2)), viewport.WithHeight(vpH))
	v.syncViewport()
}

func (v *ApprovalQueueView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ApprovalQueueMsg:
		v.pending = msg.Pending
		v.cursor = 0
		v.detail = false
		v.syncViewport()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
				v.syncViewport()
			}
		case "down", "j":
			if v.cursor < len(v.pending)-1 {
				v.cursor++
				v.syncViewport()
			}
		case "enter":
			v.detail = !v.detail
			v.syncViewport()
		case "a":
			if v.cursor < len(v.pending) {
				id := v.pending[v.cursor].ID
				return v, func() tea.Msg {
					return ApprovalActionMsg{PlanID: id, Approved: true}
				}
			}
		case "r":
			if v.cursor < len(v.pending) {
				id := v.pending[v.cursor].ID
				return v, func() tea.Msg {
					return ApprovalActionMsg{PlanID: id, Approved: false}
				}
			}
		default:
			var cmd tea.Cmd
			v.vp, cmd = v.vp.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func (v *ApprovalQueueView) syncViewport() {
	v.vp.SetWidth(max(20, v.width-2))
	v.vp.SetHeight(max(3, v.height-6))
	v.vp.SetContent(v.buildContent())
}

func (v *ApprovalQueueView) buildContent() string {
	if len(v.pending) == 0 {
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No pending approvals.")
	}

	var b strings.Builder
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(v.t.Warning())
	b.WriteString(headerStyle.Render(fmt.Sprintf("Pending approvals (%d)", len(v.pending))))
	b.WriteString("\n\n")

	for i, item := range v.pending {
		line := fmt.Sprintf("  %s  risk=%s  req=%s  %s",
			item.Action, item.RiskLevel, item.Requester, item.Timestamp.Format("15:04:05"))
		if item.Description != "" {
			line += "\n    " + item.Description
		}
		if i == v.cursor {
			line = lipgloss.NewStyle().
				Bold(true).
				Foreground(v.t.Bg()).
				Background(v.t.Primary()).
				Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")

		if i == v.cursor && v.detail {
			b.WriteString(lipgloss.NewStyle().Foreground(v.t.DimText()).Render(
				fmt.Sprintf("    id=%s", item.ID)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render(
		"a approve  r reject  Enter details  arrows move"))
	return b.String()
}

func (v *ApprovalQueueView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}
	return v.vp.View()
}
