package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type StyledTable struct {
	headers []string
	rows    [][]string
	width   int
	theme   *theme.Theme
}

func NewStyledTable(t *theme.Theme, headers ...string) *StyledTable {
	return &StyledTable{
		headers: headers,
		theme:   t,
		width:   80,
	}
}

func (st *StyledTable) AddRow(cells ...string) {
	st.rows = append(st.rows, cells)
}

func (st *StyledTable) SetRows(rows [][]string) {
	st.rows = rows
}

func (st *StyledTable) SetWidth(w int) { st.width = w }

func (st *StyledTable) Render() string {
	if len(st.headers) == 0 {
		return ""
	}

	colCount := len(st.headers)
	colWidth := (st.width - (colCount - 1)) / colCount
	if colWidth < 2 {
		colWidth = 2
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(st.theme.Primary())
	dividerStyle := lipgloss.NewStyle().Foreground(st.theme.Divider())
	fgStyle := lipgloss.NewStyle().Foreground(st.theme.Fg())
	surfaceStyle := lipgloss.NewStyle().Foreground(st.theme.Fg()).Background(st.theme.Surface())

	pad := func(s string, w int) string {
		cell := lipgloss.NewStyle().Width(w).MaxWidth(w)
		return cell.Render(truncateValue(s, w))
	}

	lines := make([]string, 0, len(st.rows)+2)

	headerCells := make([]string, colCount)
	for i, h := range st.headers {
		headerCells[i] = pad(h, colWidth)
	}
	lines = append(lines, headerStyle.Render(strings.Join(headerCells, " ")))

	parts := make([]string, colCount)
	for i := 0; i < colCount; i++ {
		parts[i] = strings.Repeat("-", colWidth)
	}
	lines = append(lines, dividerStyle.Render(strings.Join(parts, "+")))

	for i, row := range st.rows {
		cells := make([]string, colCount)
		for j := 0; j < colCount; j++ {
			value := ""
			if j < len(row) {
				value = row[j]
			}
			cells[j] = pad(value, colWidth)
		}
		rowLine := strings.Join(cells, " ")
		if i%2 == 1 {
			lines = append(lines, render.FillBlock(rowLine, st.width-4, surfaceStyle))
		} else {
			lines = append(lines, fgStyle.Render(rowLine))
		}
	}

	return render.SurfacePanel(strings.Join(lines, "\n"), st.width, st.theme.Surface(), st.theme.BorderColor())
}

func truncateValue(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	if width <= 3 {
		runes := []rune(value)
		if len(runes) > width {
			runes = runes[:width]
		}
		return string(runes)
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"...") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}
