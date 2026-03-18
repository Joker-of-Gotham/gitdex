// Package table provides a gh-dash style table component with
// auto-sizing columns, selection, and ListViewport scrolling.
package table

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components/listviewport"
	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Column defines a table column.
type Column struct {
	Title  string
	Width  int
	Hidden bool
	Grow   bool
}

// Row is one table row: a slice of cell strings matching Columns order.
type Row []string

// Model is the table component.
type Model struct {
	Columns      []Column
	rows         []Row
	EmptyMessage string
	IsLoading    bool
	ctx          *tuictx.ProgramContext
	listVP       listviewport.Model
	width        int
	height       int
}

// New creates a table model.
func New(ctx *tuictx.ProgramContext, columns []Column, width, height int) Model {
	m := Model{
		Columns:      columns,
		EmptyMessage: "No items",
		ctx:          ctx,
		width:        width,
		height:       height,
		listVP:       listviewport.New(ctx, 1, height-1),
	}
	return m
}

// SetRows replaces all rows.
func (m *Model) SetRows(rows []Row) {
	m.rows = rows
	m.listVP.SetNumItems(len(rows))
}

// GetRows returns all rows.
func (m *Model) GetRows() []Row {
	return m.rows
}

// GetCurrItem returns the currently selected row.
func (m *Model) GetCurrItem() Row {
	idx := m.listVP.GetCurrItem()
	if idx < 0 || idx >= len(m.rows) {
		return nil
	}
	return m.rows[idx]
}

// CurrIdx returns the current selection index.
func (m *Model) CurrIdx() int {
	return m.listVP.GetCurrItem()
}

// NextItem moves selection down.
func (m *Model) NextItem() { m.listVP.NextItem() }

// PrevItem moves selection up.
func (m *Model) PrevItem() { m.listVP.PrevItem() }

// FirstItem jumps to first row.
func (m *Model) FirstItem() { m.listVP.FirstItem() }

// LastItem jumps to last row.
func (m *Model) LastItem() { m.listVP.LastItem() }

// PageDown moves down one page.
func (m *Model) PageDown() { m.listVP.PageDown() }

// PageUp moves up one page.
func (m *Model) PageUp() { m.listVP.PageUp() }

// SetDimensions updates the table dimensions.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	m.listVP.SetDimensions(h - 1)
}

// UpdateProgramContext updates the shared context.
func (m *Model) UpdateProgramContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
	m.listVP.UpdateProgramContext(ctx)
}

// View renders the table.
func (m *Model) View() string {
	if m.IsLoading {
		return m.ctx.Styles.Section.EmptyState.Render("Loading...")
	}
	if len(m.rows) == 0 {
		return m.ctx.Styles.Section.EmptyState.Render(m.EmptyMessage)
	}

	header := m.renderHeader()
	body := m.listVP.RenderItems(func(idx int, isSelected bool) string {
		return m.renderRow(idx, isSelected)
	})

	return header + "\n" + body
}

func (m *Model) computeWidths() []int {
	widths := make([]int, len(m.Columns))
	totalFixed := 0
	growCount := 0
	for i, col := range m.Columns {
		if col.Hidden {
			widths[i] = 0
			continue
		}
		if col.Grow {
			growCount++
		} else {
			w := col.Width
			if w == 0 {
				w = len(col.Title) + 2
			}
			widths[i] = w
			totalFixed += w
		}
	}

	if growCount > 0 {
		remaining := m.width - totalFixed
		if remaining < growCount {
			remaining = growCount
		}
		perGrow := remaining / growCount
		for i, col := range m.Columns {
			if col.Grow && !col.Hidden {
				widths[i] = perGrow
			}
		}
	}

	return widths
}

func (m *Model) renderHeader() string {
	widths := m.computeWidths()
	var cells []string
	for i, col := range m.Columns {
		if col.Hidden || widths[i] == 0 {
			continue
		}
		cell := m.ctx.Styles.Table.Header.
			Width(widths[i]).
			MaxWidth(widths[i]).
			Render(col.Title)
		cells = append(cells, cell)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func (m *Model) renderRow(idx int, isSelected bool) string {
	if idx < 0 || idx >= len(m.rows) {
		return ""
	}
	row := m.rows[idx]
	widths := m.computeWidths()

	var cells []string
	for i, col := range m.Columns {
		if col.Hidden || widths[i] == 0 {
			continue
		}
		content := ""
		if i < len(row) {
			content = row[i]
		}

		style := m.ctx.Styles.Table.Cell
		if isSelected {
			style = m.ctx.Styles.Table.SelectedCell
		}

		cell := style.Width(widths[i]).MaxWidth(widths[i]).Render(truncate(content, widths[i]-2))
		cells = append(cells, cell)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "~"
}

// TotalCount returns the number of rows.
func (m *Model) TotalCount() int {
	return len(m.rows)
}

// ScrollPercent returns the scroll percentage.
func (m *Model) ScrollPercent() int {
	return m.listVP.ScrollPercent()
}

// StatusLine returns a short status like "5/20 (25%)".
func (m *Model) StatusLine() string {
	if len(m.rows) == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d (%d%%)", m.CurrIdx()+1, len(m.rows), m.ScrollPercent())
}
