package carousel

// Model is a generic carousel for cycling through a fixed set of items.
// Aligned with gh-dash's carousel.Model.
type Model struct {
	items   []string
	current int
}

// New creates a carousel from the given items.
func New(items []string) Model {
	return Model{items: items}
}

// Next cycles to the next item.
func (m *Model) Next() {
	if len(m.items) == 0 {
		return
	}
	m.current = (m.current + 1) % len(m.items)
}

// Prev cycles to the previous item.
func (m *Model) Prev() {
	if len(m.items) == 0 {
		return
	}
	m.current = (m.current - 1 + len(m.items)) % len(m.items)
}

// Current returns the currently selected item.
func (m Model) Current() string {
	if len(m.items) == 0 {
		return ""
	}
	return m.items[m.current]
}

// CurrentIdx returns the current index.
func (m Model) CurrentIdx() int { return m.current }

// SetIdx sets the current index directly.
func (m *Model) SetIdx(idx int) {
	if idx >= 0 && idx < len(m.items) {
		m.current = idx
	}
}

// Items returns all carousel items.
func (m Model) Items() []string { return m.items }

// Len returns the number of items.
func (m Model) Len() int { return len(m.items) }
