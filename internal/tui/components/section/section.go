// Package section defines the Section interface that all TUI view sections
// must implement. Inspired by gh-dash's components/section.
package section

import (
	tea "charm.land/bubbletea/v2"
	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Section is the common interface for all view sections.
// Each tab view contains one or more Sections.
type Section interface {
	ID() string
	Title() string
	ViewType() tuictx.ViewType

	Update(msg tea.Msg) (Section, tea.Cmd)
	View() string

	GetIsLoading() bool
	GetTotalCount() int
	GetCurrItem() any
	GetPagerContent() string

	SetDimensions(w, h int)
	UpdateProgramContext(ctx *tuictx.ProgramContext)
}
