package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type TaskItem struct {
	ID           string
	Title        string
	Status       string
	AssignedPlan string
	Priority     int
}

type TasksView struct {
	tasks    []TaskItem
	selected int
	width    int
	height   int
	t        *theme.Theme
}

func NewTasksView(t *theme.Theme) *TasksView { return &TasksView{t: t} }

func (v *TasksView) ID() ID        { return ViewTasks }
func (v *TasksView) Title() string { return "Tasks" }
func (v *TasksView) Init() tea.Cmd { return nil }

func (v *TasksView) visibleCount() int {
	const linesPerCard = 6
	h := v.height - 4
	if h < linesPerCard {
		h = linesPerCard
	}
	return maxInt(1, h/linesPerCard)
}

func (v *TasksView) Update(msg tea.Msg) (View, tea.Cmd) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch km.String() {
		case "up", "k":
			if v.selected > 0 {
				v.selected--
			}
		case "down", "j":
			if len(v.tasks) > 0 && v.selected < len(v.tasks)-1 {
				v.selected++
			}
		case "pgup":
			step := maxInt(1, v.visibleCount()/2)
			v.selected -= step
			if v.selected < 0 {
				v.selected = 0
			}
		case "pgdown":
			step := maxInt(1, v.visibleCount()/2)
			v.selected += step
			if v.selected >= len(v.tasks) {
				v.selected = maxInt(0, len(v.tasks)-1)
			}
		}
	}
	return v, nil
}

func (v *TasksView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

func (v *TasksView) SetTasks(tasks []TaskItem) {
	v.tasks = tasks
	if v.selected >= len(v.tasks) && len(v.tasks) > 0 {
		v.selected = len(v.tasks) - 1
	}
}

func (v *TasksView) Render() string {
	if v.width <= 0 || v.height <= 0 {
		return ""
	}

	if len(v.tasks) == 0 {
		body := strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Task Queue"),
			"",
			lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No tasks are currently queued."),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(24, v.width-8)).
				Render("Live risks and next actions will appear here once repository health, collaboration state, or workflow findings require intervention."),
		}, "\n")
		return render.SurfacePanel(body, maxInt(24, v.width), v.t.Surface(), v.t.BorderColor())
	}

	if v.width >= 96 {
		listWidth := maxInt(34, v.width*52/100)
		detailWidth := maxInt(28, v.width-listWidth-1)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			v.renderTaskList(listWidth),
			" ",
			v.renderTaskDetail(detailWidth),
		)
	}

	return strings.Join([]string{
		v.renderTaskList(v.width),
		v.renderTaskDetail(v.width),
	}, "\n\n")
}

func (v *TasksView) renderTaskList(width int) string {
	rows := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("Task Queue (%d)", len(v.tasks))),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Up/Down select  PgUp/PgDn page"),
		"",
	}

	vis := v.visibleCount()
	start := 0
	if v.selected >= vis {
		start = v.selected - vis + 1
	}
	end := start + vis
	if end > len(v.tasks) {
		end = len(v.tasks)
	}

	for i := start; i < end; i++ {
		task := v.tasks[i]
		card := []string{
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(task.ID + "  " + task.Title),
			lipgloss.NewStyle().Foreground(v.t.DimText()).Render(fmt.Sprintf("Status %s  |  Priority P%d", strings.ToUpper(valueOrDash(task.Status)), task.Priority)),
			lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(18, width-8)).Render("Plan: " + valueOrDash(task.AssignedPlan)),
		}
		block := strings.Join(card, "\n")
		if i == v.selected {
			panelFrame := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).GetHorizontalFrameSize()
			block = render.FillBlock(block, maxInt(12, width-panelFrame), lipgloss.NewStyle().Background(v.t.Selection()))
		}
		rows = append(rows, render.SurfacePanel(block, width, v.t.Surface(), v.t.BorderColor()))
	}
	return strings.Join(rows, "\n")
}

func (v *TasksView) renderTaskDetail(width int) string {
	if len(v.tasks) == 0 || v.selected >= len(v.tasks) {
		return ""
	}

	task := v.tasks[v.selected]
	body := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Selected Task"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(task.Title),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render("ID " + task.ID),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Status"),
		lipgloss.NewStyle().Foreground(v.t.Fg()).Render(strings.ToUpper(valueOrDash(task.Status))),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Assigned Plan"),
		lipgloss.NewStyle().Foreground(v.t.Fg()).Width(maxInt(18, width-8)).Render(valueOrDash(task.AssignedPlan)),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render(fmt.Sprintf("Priority P%d", task.Priority)),
		"",
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(18, width-8)).
			Render("Tasks should stay non-redundant with plans: a task is the smallest operational unit worth acting on, not a restatement of an already visible signal."),
	}
	return render.SurfacePanel(strings.Join(body, "\n"), width, v.t.Surface(), v.t.BorderColor())
}
