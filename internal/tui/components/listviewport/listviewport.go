package listviewport

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Model wraps a scrollable viewport for item-level navigation.
// Aligned with gh-dash's listviewport.Model pattern.
type Model struct {
	ctx            *context.ProgramContext
	width          int
	height         int
	topBoundIdx    int
	bottomBoundIdx int
	currIdx        int
	listItemHeight int
	numItems       int
}

// New creates a listviewport with the given item height.
func New(itemHeight int) Model {
	if itemHeight <= 0 {
		itemHeight = 1
	}
	return Model{
		listItemHeight: itemHeight,
	}
}

// SetDimensions updates the viewport size.
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.recalcBounds()
}

// SetNumItems updates the total number of items.
func (m *Model) SetNumItems(n int) {
	m.numItems = n
	m.recalcBounds()
	if m.currIdx >= n {
		m.currIdx = max(0, n-1)
	}
}

// UpdateProgramContext stores the program context reference.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
}

func (m *Model) recalcBounds() {
	visible := m.visibleCount()
	if m.numItems <= visible {
		m.topBoundIdx = 0
		m.bottomBoundIdx = max(0, m.numItems-1)
		return
	}
	if m.currIdx < m.topBoundIdx {
		m.topBoundIdx = m.currIdx
	}
	if m.currIdx > m.topBoundIdx+visible-1 {
		m.topBoundIdx = m.currIdx - visible + 1
	}
	m.bottomBoundIdx = min(m.topBoundIdx+visible-1, m.numItems-1)
}

func (m *Model) visibleCount() int {
	if m.listItemHeight <= 0 {
		return m.height
	}
	c := m.height / m.listItemHeight
	if c < 1 {
		return 1
	}
	return c
}

// CurrIdx returns the currently selected item index.
func (m *Model) CurrIdx() int { return m.currIdx }

// TopIdx returns the top visible item index.
func (m *Model) TopIdx() int { return m.topBoundIdx }

// BottomIdx returns the bottom visible item index.
func (m *Model) BottomIdx() int { return m.bottomBoundIdx }

// NextItem moves selection down by one.
func (m *Model) NextItem() int {
	if m.currIdx < m.numItems-1 {
		m.currIdx++
		m.recalcBounds()
	}
	return m.currIdx
}

// PrevItem moves selection up by one.
func (m *Model) PrevItem() int {
	if m.currIdx > 0 {
		m.currIdx--
		m.recalcBounds()
	}
	return m.currIdx
}

// FirstItem jumps to the first item.
func (m *Model) FirstItem() int {
	m.currIdx = 0
	m.recalcBounds()
	return m.currIdx
}

// LastItem jumps to the last item.
func (m *Model) LastItem() int {
	if m.numItems > 0 {
		m.currIdx = m.numItems - 1
	}
	m.recalcBounds()
	return m.currIdx
}

// PageDown moves selection down by one page.
func (m *Model) PageDown() int {
	visible := m.visibleCount()
	m.currIdx += visible
	if m.currIdx >= m.numItems {
		m.currIdx = max(0, m.numItems-1)
	}
	m.recalcBounds()
	return m.currIdx
}

// PageUp moves selection up by one page.
func (m *Model) PageUp() int {
	visible := m.visibleCount()
	m.currIdx -= visible
	if m.currIdx < 0 {
		m.currIdx = 0
	}
	m.recalcBounds()
	return m.currIdx
}

// SetCurrIdx sets the current item index directly.
func (m *Model) SetCurrIdx(idx int) {
	if idx < 0 {
		idx = 0
	}
	if idx >= m.numItems {
		idx = max(0, m.numItems-1)
	}
	m.currIdx = idx
	m.recalcBounds()
}

// ScrollPercent returns a string like "42%" indicating scroll position.
func (m *Model) ScrollPercent() string {
	if m.numItems <= 0 {
		return ""
	}
	pct := 0
	if m.numItems > 1 {
		pct = m.currIdx * 100 / (m.numItems - 1)
	}
	return fmt.Sprintf("%d%%", pct)
}

// RenderItems renders the visible slice of items using a render function.
func (m *Model) RenderItems(renderFn func(idx int, selected bool) string) string {
	if m.numItems == 0 {
		return ""
	}
	var rows []string
	end := min(m.bottomBoundIdx+1, m.numItems)
	for i := m.topBoundIdx; i < end; i++ {
		rows = append(rows, renderFn(i, i == m.currIdx))
	}
	content := strings.Join(rows, "\n")

	rendered := lipgloss.NewStyle().
		Width(m.width).
		MaxHeight(m.height).
		Render(content)
	return rendered
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
