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

type PullsDataMsg struct {
	Items []repo.PullRequestSummary
}

type PullsView struct {
	theme        *theme.Theme
	width        int
	height       int
	items        []repo.PullRequestSummary
	cursor       int
	scroll       int
	detail       bool
	detailNumber int
	detailView   *PRDetailView
}

func NewPullsView(t *theme.Theme) *PullsView {
	return &PullsView{
		theme:      t,
		detailView: NewPRDetailView(t),
	}
}

func (v *PullsView) ID() ID           { return ViewPulls }
func (v *PullsView) Title() string    { return "Pull Requests" }
func (v *PullsView) Init() tea.Cmd    { return nil }
func (v *PullsView) DetailOpen() bool { return v.detail }

func (v *PullsView) SetSize(w, h int) {
	v.width = w
	v.height = h
	listW, detailW := v.splitWidths()
	_ = listW
	v.detailView.SetSize(detailW, max(5, h-4))
}

func (v *PullsView) SetItems(items []repo.PullRequestSummary) {
	v.items = items
	if v.cursor >= len(items) {
		v.cursor = max(0, len(items)-1)
	}
}

func (v *PullsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case PullsDataMsg:
		v.items = msg.Items
		if v.cursor >= len(v.items) {
			v.cursor = max(0, len(v.items)-1)
		}
		return v, nil
	case PRDetailMsg:
		_, cmd := v.detailView.Update(msg)
		return v, cmd
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *PullsView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.detail && !v.detailView.PromptActive() {
		if msg.String() == "esc" {
			v.detail = false
			return v, nil
		}
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
			return v, func() tea.Msg { return RequestPRDetailMsg{Number: item.Number} }
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
			return v, func() tea.Msg { return RequestPRDetailMsg{Number: item.Number} }
		}
	}

	if v.detail {
		_, cmd := v.detailView.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *PullsView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}
	return v.renderList(v.width)
}

func (v *PullsView) renderList(width int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Primary())
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(v.theme.Accent()).Underline(true)
	rowStyle := lipgloss.NewStyle().Foreground(v.theme.Fg())
	selStyle := lipgloss.NewStyle().Foreground(v.theme.Fg()).Background(v.theme.Selection())
	draftStyle := lipgloss.NewStyle().Foreground(v.theme.DimText())
	reviewStyle := lipgloss.NewStyle().Foreground(v.theme.Warning()).Bold(true)
	numStyle := lipgloss.NewStyle().Foreground(v.theme.Info())
	hintStyle := lipgloss.NewStyle().Foreground(v.theme.DimText()).Italic(true)
	dimStyle := lipgloss.NewStyle().Foreground(v.theme.DimText())

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("  Pull Requests (%d)", len(v.items))))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("  Up/Down navigate  Enter inspect  Esc close detail  g/G jump"))
	b.WriteString("\n\n")

	if len(v.items) == 0 {
		b.WriteString("  ")
		b.WriteString(draftStyle.Render("No pull requests found"))
		b.WriteString("\n")
		return b.String()
	}

	colW := max(72, width-6)
	numW := 6
	authorW := 18
	statusW := 14
	updatedW := 8
	titleW := colW - numW - authorW - statusW - updatedW - 4
	if titleW < 18 {
		titleW = 18
	}

	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s",
		numW, "#", titleW, "Title / Labels", authorW, "Author", statusW, "Status", updatedW, "Updated")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	visible := max(3, v.height-6)
	for i, pr := range v.items {
		if i < v.scroll {
			continue
		}
		if i-v.scroll >= visible {
			break
		}

		num := numStyle.Render(fmt.Sprintf("#%-5d", pr.Number))
		titleBase := pr.Title
		if len(pr.Labels) > 0 {
			titleBase += " / " + strings.Join(pr.Labels, ", ")
		}
		title := fmt.Sprintf("%-*s", titleW, truncate(titleBase, titleW))
		author := fmt.Sprintf("%-*s", authorW, truncate(pr.Author, authorW))

		statusText := "OPEN"
		statusStyle := rowStyle
		if pr.IsDraft {
			statusText = "DRAFT"
			statusStyle = draftStyle
		} else if pr.NeedsReview {
			statusText = "REVIEW"
			statusStyle = reviewStyle
		}
		status := statusStyle.Render(fmt.Sprintf("%-*s", statusW, statusText))

		updated := "-"
		if pr.StaleDays > 0 {
			updated = fmt.Sprintf("%dd", pr.StaleDays)
		}
		updated = dimStyle.Render(fmt.Sprintf("%-*s", updatedW, truncate(updated, updatedW)))

		prefix := "  "
		if i == v.cursor {
			prefix = theme.Icons.ChevronRight + " "
		}

		line := prefix + num + " " + title + " " + author + " " + status + " " + updated
		if i == v.cursor {
			line = render.FillBlock(line, max(20, width-2), selStyle)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	if v.detail {
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("  Inspector detail active. Use Ctrl+3 or Ctrl+I to review the selected pull request."))
		b.WriteString("\n")
	}

	return b.String()
}

func (v *PullsView) selected() *repo.PullRequestSummary {
	if v.cursor >= 0 && v.cursor < len(v.items) {
		return &v.items[v.cursor]
	}
	return nil
}

func (v *PullsView) splitWidths() (int, int) {
	listW := max(52, v.width*48/100)
	detailW := max(40, v.width-listW-1)
	return listW, detailW
}

func (v *PullsView) adjustScroll() {
	visible := max(3, v.height-6)
	if v.cursor < v.scroll {
		v.scroll = v.cursor
	}
	if v.cursor >= v.scroll+visible {
		v.scroll = v.cursor - visible + 1
	}
}
