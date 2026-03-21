package views

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type DashboardView struct {
	theme   *theme.Theme
	width   int
	height  int
	cockpit *CockpitView
	status  *StatusView
	repos   *ReposView
	active  int
	tabs    []string
}

func NewDashboardView(t *theme.Theme) *DashboardView {
	return &DashboardView{
		theme:   t,
		cockpit: NewCockpitView(t),
		status:  NewStatusView(t),
		repos:   NewReposView(t),
		tabs:    []string{"Overview", "Health", "Repos"},
	}
}

func (v *DashboardView) ID() ID       { return ViewDashboard }
func (v *DashboardView) Title() string { return "Dashboard" }
func (v *DashboardView) Init() tea.Cmd { return nil }

func (v *DashboardView) Cockpit() *CockpitView { return v.cockpit }
func (v *DashboardView) Status() *StatusView    { return v.status }
func (v *DashboardView) ActiveTab() int         { return v.active }

func (v *DashboardView) Repos() *ReposView { return v.repos }

func (v *DashboardView) SetSize(w, h int) {
	v.width = w
	v.height = h
	subH := max(h-3, 5)
	v.cockpit.SetSize(w, subH)
	v.status.SetSize(w, subH)
	v.repos.SetSize(w, subH)
}

func (v *DashboardView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusDataMsg:
		v.cockpit.SetSummary(msg.Summary)
		v.status.SetSummary(msg.Summary)
		return v, nil
	case RepoListMsg:
		v.repos.SetItems(msg.Repos)
		return v, nil
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
			_, cmd = v.cockpit.Update(msg)
		case 1:
			_, cmd = v.status.Update(msg)
		case 2:
			_, cmd = v.repos.Update(msg)
		}
		return v, cmd
	}
	return v, nil
}

func (v *DashboardView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderSubTabs(v.tabs, v.active, v.theme, v.width))
	b.WriteString("\n")
	switch v.active {
	case 0:
		b.WriteString(v.cockpit.Render())
	case 1:
		b.WriteString(v.status.Render())
	case 2:
		b.WriteString(v.repos.Render())
	}
	return b.String()
}

