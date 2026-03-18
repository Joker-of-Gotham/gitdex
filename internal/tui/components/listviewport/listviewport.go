// Package listviewport provides a scrollable viewport for list items.
// Inspired by gh-dash's components/listviewport.
package listviewport

import (
	"strings"

	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Model manages the visible window over a list of items.
type Model struct {
	ctx            *tuictx.ProgramContext
	topBoundIdx    int
	bottomBoundIdx int
	currIdx        int
	numItems       int
	listItemHeight int
	totalHeight    int
}

// New creates a ListViewport model.
func New(ctx *tuictx.ProgramContext, itemHeight, totalHeight int) Model {
	m := Model{
		ctx:            ctx,
		listItemHeight: itemHeight,
		totalHeight:    totalHeight,
	}
	m.recalcBounds()
	return m
}

// SetDimensions updates the viewport height.
func (m *Model) SetDimensions(height int) {
	m.totalHeight = height
	m.recalcBounds()
}

// SetNumItems updates the total number of items.
func (m *Model) SetNumItems(n int) {
	m.numItems = n
	if m.currIdx >= n && n > 0 {
		m.currIdx = n - 1
	}
	if n == 0 {
		m.currIdx = 0
	}
	m.recalcBounds()
}

// NextItem moves selection down by one.
func (m *Model) NextItem() int {
	if m.currIdx < m.numItems-1 {
		m.currIdx++
		if m.currIdx > m.bottomBoundIdx {
			m.topBoundIdx++
			m.bottomBoundIdx++
		}
	}
	return m.currIdx
}

// PrevItem moves selection up by one.
func (m *Model) PrevItem() int {
	if m.currIdx > 0 {
		m.currIdx--
		if m.currIdx < m.topBoundIdx {
			m.topBoundIdx--
			m.bottomBoundIdx--
		}
	}
	return m.currIdx
}

// FirstItem jumps to the first item.
func (m *Model) FirstItem() int {
	m.currIdx = 0
	m.topBoundIdx = 0
	m.recalcBounds()
	return m.currIdx
}

// LastItem jumps to the last item.
func (m *Model) LastItem() int {
	if m.numItems > 0 {
		m.currIdx = m.numItems - 1
	}
	m.recalcBounds()
	if m.numItems > 0 {
		visible := m.visibleCount()
		if m.numItems > visible {
			m.topBoundIdx = m.numItems - visible
			m.bottomBoundIdx = m.numItems - 1
		}
	}
	return m.currIdx
}

// PageDown moves down by one page.
func (m *Model) PageDown() int {
	pageSize := m.visibleCount()
	for i := 0; i < pageSize; i++ {
		if m.currIdx >= m.numItems-1 {
			break
		}
		m.NextItem()
	}
	return m.currIdx
}

// PageUp moves up by one page.
func (m *Model) PageUp() int {
	pageSize := m.visibleCount()
	for i := 0; i < pageSize; i++ {
		if m.currIdx <= 0 {
			break
		}
		m.PrevItem()
	}
	return m.currIdx
}

// GetCurrItem returns the current selection index.
func (m *Model) GetCurrItem() int {
	return m.currIdx
}

// SetCurrItem sets the current selection index.
func (m *Model) SetCurrItem(idx int) {
	if idx < 0 {
		idx = 0
	}
	if idx >= m.numItems {
		idx = m.numItems - 1
	}
	if idx < 0 {
		idx = 0
	}
	m.currIdx = idx
	m.recalcBounds()
}

// RenderItems renders only the visible items using the provided render function.
func (m *Model) RenderItems(renderFn func(idx int, isSelected bool) string) string {
	if m.numItems == 0 {
		return ""
	}

	var sb strings.Builder
	for i := m.topBoundIdx; i <= m.bottomBoundIdx && i < m.numItems; i++ {
		if i > m.topBoundIdx {
			sb.WriteString("\n")
		}
		sb.WriteString(renderFn(i, i == m.currIdx))
	}
	return sb.String()
}

// TopIdx returns the top visible index.
func (m *Model) TopIdx() int {
	return m.topBoundIdx
}

// BottomIdx returns the bottom visible index.
func (m *Model) BottomIdx() int {
	return m.bottomBoundIdx
}

// ScrollPercent returns the scroll position as a percentage.
func (m *Model) ScrollPercent() int {
	if m.numItems <= 1 {
		return 100
	}
	return m.currIdx * 100 / (m.numItems - 1)
}

func (m *Model) visibleCount() int {
	if m.listItemHeight == 0 {
		return m.totalHeight
	}
	return m.totalHeight / m.listItemHeight
}

func (m *Model) recalcBounds() {
	visible := m.visibleCount()
	if visible <= 0 {
		visible = 1
	}

	if m.currIdx < m.topBoundIdx {
		m.topBoundIdx = m.currIdx
	}
	m.bottomBoundIdx = m.topBoundIdx + visible - 1
	if m.bottomBoundIdx >= m.numItems {
		m.bottomBoundIdx = m.numItems - 1
		m.topBoundIdx = m.bottomBoundIdx - visible + 1
		if m.topBoundIdx < 0 {
			m.topBoundIdx = 0
		}
	}
	if m.currIdx > m.bottomBoundIdx {
		m.bottomBoundIdx = m.currIdx
		m.topBoundIdx = m.bottomBoundIdx - visible + 1
		if m.topBoundIdx < 0 {
			m.topBoundIdx = 0
		}
	}
}

// UpdateProgramContext updates the shared context.
func (m *Model) UpdateProgramContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}
