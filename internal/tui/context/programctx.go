package context

import (
	"github.com/Joker-of-Gotham/gitdex/internal/config"
)

// ViewType identifies the active top-level view.
type ViewType int

const (
	AgentView     ViewType = 0
	GitView       ViewType = 1
	WorkspaceView ViewType = 2
	GitHubView    ViewType = 3
	ConfigView    ViewType = 4
)

// ViewName returns a human-readable name for the view.
func (v ViewType) ViewName() string {
	switch v {
	case AgentView:
		return "Agent"
	case GitView:
		return "Git"
	case WorkspaceView:
		return "Workspace"
	case GitHubView:
		return "GitHub"
	case ConfigView:
		return "Config"
	default:
		return "Unknown"
	}
}

// ProgramContext is the shared global context passed to all components.
// Inspired by gh-dash's ui/context/context.go.
type ProgramContext struct {
	ScreenWidth  int
	ScreenHeight int

	MainContentWidth  int
	MainContentHeight int

	SidebarOpen  bool
	SidebarWidth int

	Config  *config.Config
	Theme   *ThemeColors
	Styles  *Styles
	View    ViewType
	Version string

	RepoPath  string
	RepoName  string
	User      string
	Error     error

	ContextUsed int
	ContextMax  int

	Mode string
}

// New creates a ProgramContext with defaults.
func New() *ProgramContext {
	theme := DefaultTheme()
	styles := InitStyles(theme)
	return &ProgramContext{
		ScreenWidth:  80,
		ScreenHeight: 24,
		SidebarOpen:  true,
		SidebarWidth: 40,
		View:         AgentView,
		Theme:        theme,
		Styles:       styles,
		Mode:         "manual",
	}
}

// UpdateDimensions recalculates layout dimensions.
func (ctx *ProgramContext) UpdateDimensions(w, h int) {
	ctx.ScreenWidth = w
	ctx.ScreenHeight = h

	tabsHeight := 1
	footerHeight := 1
	borderPadding := 2

	ctx.MainContentHeight = h - tabsHeight - footerHeight - borderPadding
	if ctx.MainContentHeight < 1 {
		ctx.MainContentHeight = 1
	}

	if ctx.SidebarOpen && w > 60 {
		sidebarW := w * ctx.SidebarWidth / 100
		if sidebarW < 20 {
			sidebarW = 20
		}
		if sidebarW > w/2 {
			sidebarW = w / 2
		}
		ctx.MainContentWidth = w - sidebarW - borderPadding
	} else {
		ctx.MainContentWidth = w - borderPadding
	}
	if ctx.MainContentWidth < 1 {
		ctx.MainContentWidth = 1
	}
}

// GetSidebarWidth returns the calculated sidebar width in characters.
func (ctx *ProgramContext) GetSidebarWidth() int {
	if !ctx.SidebarOpen {
		return 0
	}
	sidebarW := ctx.ScreenWidth * ctx.SidebarWidth / 100
	if sidebarW < 20 {
		sidebarW = 20
	}
	if sidebarW > ctx.ScreenWidth/2 {
		sidebarW = ctx.ScreenWidth / 2
	}
	return sidebarW
}

// ContextPercent returns the context usage percentage.
func (ctx *ProgramContext) ContextPercent() int {
	if ctx.ContextMax == 0 {
		return 0
	}
	return ctx.ContextUsed * 100 / ctx.ContextMax
}
