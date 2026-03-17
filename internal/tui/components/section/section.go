package section

import (
	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/context"
)

// Section is the interface for all GitDex TUI panels/sections.
// Aligned with gh-dash's Section interface pattern.
type Section interface {
	ID() string
	Title() string
	Update(tea.Msg) (Section, tea.Cmd)
	View(ctx *context.ProgramContext) string
	SetDimensions(width, height int)
	UpdateProgramContext(ctx *context.ProgramContext)
	GetIsLoading() bool
	GetItemCount() int
}
