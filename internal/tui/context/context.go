package context

import (
	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

// ViewType identifies which main view is active.
type ViewType int

const (
	ViewMaintain ViewType = iota
	ViewGoal
	ViewCreative
	ViewConfig
)

func (v ViewType) String() string {
	switch v {
	case ViewMaintain:
		return "maintain"
	case ViewGoal:
		return "goal"
	case ViewCreative:
		return "creative"
	case ViewConfig:
		return "config"
	default:
		return "unknown"
	}
}

// ProgramContext is the centralized context passed to every TUI component.
// Aligned with gh-dash's context.ProgramContext pattern.
type ProgramContext struct {
	ScreenWidth  int
	ScreenHeight int

	MainContentWidth  int
	MainContentHeight int

	SidebarOpen  bool
	SidebarWidth int

	Config *config.Config
	Theme  *theme.Theme
	Styles Styles
	View   ViewType

	StartTask func(tea.Cmd)
}

// ContentWidth returns the available width for the main content area,
// accounting for sidebar if open.
func (ctx *ProgramContext) ContentWidth() int {
	if ctx.SidebarOpen && ctx.SidebarWidth > 0 {
		w := ctx.MainContentWidth - ctx.SidebarWidth - 1
		if w < 20 {
			return 20
		}
		return w
	}
	return ctx.MainContentWidth
}
