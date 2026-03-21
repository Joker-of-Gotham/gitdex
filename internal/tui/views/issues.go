package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type IssuesDataMsg struct {
	Items []repo.IssueSummary
}

type IssuesView struct {
	theme        *theme.Theme
	width        int
	height       int
	items        []repo.IssueSummary
	cursor       int
	scroll       int
	detail       bool
	detailNumber int
	detailView   *IssueDetailView
}

func NewIssuesView(t *theme.Theme) *IssuesView {
	return &IssuesView{
		theme:      t,
		detailView: NewIssueDetailView(t),
	}
}

func (v *IssuesView) ID() ID        { return ViewIssues }
func (v *IssuesView) Title() string { return "Issues" }
func (v *IssuesView) Init() tea.Cmd { return nil }

func (v *IssuesView) SetSize(w, h int) {
	v.width = w
	v.height = h
	listW, detailW := v.splitWidths()
	_ = listW
	v.detailView.SetSize(detailW, max(5, h-4))
}

func (v *IssuesView) SetItems(items []repo.IssueSummary) {
	v.items = items
	if v.cursor >= len(items) {
		v.cursor = max(0, len(items)-1)
	}
}

func (v *IssuesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case IssuesDataMsg:
		v.items = msg.Items
		if v.cursor >= len(v.items) {
			v.cursor = max(0, len(v.items)-1)
		}
		return v, nil
	case IssueDetailMsg:
		_, cmd := v.detailView.Update(msg)
		return v, cmd
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *IssuesView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.detail && !v.detailView.PromptActive() && msg.String() == "esc" {
		v.detail = false
		return v, nil
	}

	if v.detail && v.detailView.PromptActive() {
		_, cmd := v.detailView.Update(msg)
		return v, cmd
	}

	prevCursor := v.cursor
	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
			v.adjustScroll()
		}
	case "down", "j":
		if v.cursor < len(v.items)-1 {
			v.cursor++
			v.adjustScroll()
		}
	case "enter":
		if v.detail {
			_, cmd := v.detailView.Update(msg)
			return v, cmd
		}
		if item := v.selected(); item != nil {
			v.detail = true
			v.detailNumber = item.Number
			return v, func() tea.Msg { return RequestIssueDetailMsg{Number: item.Number} }
		}
	case "g":
		v.cursor = 0
		v.scroll = 0
	case "G":
		v.cursor = max(0, len(v.items)-1)
		v.adjustScroll()
	case "pgup":
		visible := maxInt(3, v.height-6)
		step := maxInt(1, visible/2)
		v.cursor -= step
		if v.cursor < 0 {
			v.cursor = 0
		}
		v.adjustScroll()
	case "pgdown":
		visible := maxInt(3, v.height-6)
		step := maxInt(1, visible/2)
		v.cursor += step
		if v.cursor >= len(v.items) {
			v.cursor = maxInt(0, len(v.items)-1)
		}
		v.adjustScroll()
	}

	if v.detail && v.cursor != prevCursor {
		if item := v.selected(); item != nil {
			v.detailNumber = item.Number
			return v, func() tea.Msg { return RequestIssueDetailMsg{Number: item.Number} }
		}
	}

	if v.detail {
		_, cmd := v.detailView.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *IssuesView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}
	return v.renderList(v.width)
}

func (v *IssuesView) renderList(width int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary())
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Accent()).Underline(true)
	rowStyle := lipgloss.NewStyle().Foreground(v.theme.Fg())
	selStyle := lipgloss.NewStyle().Foreground(v.theme.Fg()).Background(v.theme.Selection())
	dimStyle := lipgloss.NewStyle().Foreground(v.theme.DimText())
	openStyle := lipgloss.NewStyle().Foreground(v.theme.Success())
	closedStyle := lipgloss.NewStyle().Foreground(v.theme.Danger())
	numStyle := lipgloss.NewStyle().Foreground(v.theme.Info())
	hintStyle := lipgloss.NewStyle().Foreground(v.theme.DimText()).Italic(true)

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("  Issues (%d)", len(v.items))))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("  Up/Down navigate  Enter inspect  Esc close detail  g/G jump"))
	b.WriteString("\n\n")

	if len(v.items) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("No issues found"))
		b.WriteString("\n")
		return b.String()
	}

	colW := max(72, width-6)
	numW := 6
	stateW := 10
	commentsW := 8
	authorW := 16
	titleW := colW - numW - stateW - commentsW - authorW - 4
	if titleW < 20 {
		titleW = 20
	}

	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s",
		numW, "#", titleW, "Title", authorW, "Author", stateW, "State", commentsW, "Comments")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	visible := max(3, v.height-6)
	for i, issue := range v.items {
		if i < v.scroll {
			continue
		}
		if i-v.scroll >= visible {
			break
		}

		num := numStyle.Render(fmt.Sprintf("#%-5d", issue.Number))
		title := fmt.Sprintf("%-*s", titleW, truncate(issue.Title, titleW))
		author := fmt.Sprintf("%-*s", authorW, truncate(issue.Author, authorW))

		stateText := strings.ToUpper(issue.State)
		state := openStyle.Render(fmt.Sprintf("%-*s", stateW, stateText))
		if !strings.EqualFold(issue.State, "open") {
			state = closedStyle.Render(fmt.Sprintf("%-*s", stateW, stateText))
		}
		comments := rowStyle.Render(fmt.Sprintf("%*d", commentsW, issue.Comments))

		prefix := "  "
		if i == v.cursor {
			prefix = theme.Icons.ChevronRight + " "
		}

		line := prefix + num + " " + title + " " + author + " " + state + " " + comments
		if i == v.cursor {
			line = render.FillBlock(line, max(20, width-2), selStyle)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	if v.detail {
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("  Inspector detail active. Use Ctrl+3 or Ctrl+I to review the selected issue."))
		b.WriteString("\n")
	}

	return b.String()
}

func (v *IssuesView) selected() *repo.IssueSummary {
	if v.cursor >= 0 && v.cursor < len(v.items) {
		return &v.items[v.cursor]
	}
	return nil
}

func (v *IssuesView) splitWidths() (int, int) {
	listW := max(52, v.width*48/100)
	detailW := max(40, v.width-listW-1)
	return listW, detailW
}

func (v *IssuesView) adjustScroll() {
	visible := max(3, v.height-6)
	if v.cursor < v.scroll {
		v.scroll = v.cursor
	}
	if v.cursor >= v.scroll+visible {
		v.scroll = v.cursor - visible + 1
	}
}
