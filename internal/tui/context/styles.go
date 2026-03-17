package context

import (
	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

// Styles holds all component-level styles, initialized from the active theme.
// Aligned with gh-dash's context.Styles approach: no inline styles in components.
type Styles struct {
	Common  CommonStyles
	Section SectionStyles
	Sidebar SidebarStyles
	Table   TableStyles
	Tabs    TabStyles
	Footer  FooterStyles
	Panel   PanelStyles
	Input   InputStyles
	Header  HeaderStyles
}

type CommonStyles struct {
	MainTextStyle lipgloss.Style
	MutedStyle    lipgloss.Style
	ErrorStyle    lipgloss.Style
	SuccessStyle  lipgloss.Style
	WarningStyle  lipgloss.Style
	InfoStyle     lipgloss.Style
	BoldStyle     lipgloss.Style
}

type SectionStyles struct {
	ContainerStyle       lipgloss.Style
	FocusedContainerStyle lipgloss.Style
	TitleStyle           lipgloss.Style
	EmptyStateStyle      lipgloss.Style
}

type SidebarStyles struct {
	ContainerStyle lipgloss.Style
	TitleStyle     lipgloss.Style
	ContentStyle   lipgloss.Style
}

type TableStyles struct {
	RowStyle         lipgloss.Style
	SelectedRowStyle lipgloss.Style
	HeaderStyle      lipgloss.Style
	CellStyle        lipgloss.Style
}

type TabStyles struct {
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	TabGap      lipgloss.Style
}

type FooterStyles struct {
	ContainerStyle   lipgloss.Style
	ViewSwitcherStyle lipgloss.Style
	HelpKeyStyle     lipgloss.Style
	HelpDescStyle    lipgloss.Style
}

type PanelStyles struct {
	BorderStyle        lipgloss.Style
	FocusedBorderStyle lipgloss.Style
	TitleStyle         lipgloss.Style
	PillStyle          lipgloss.Style
}

type InputStyles struct {
	PromptStyle lipgloss.Style
	TextStyle   lipgloss.Style
	CursorStyle lipgloss.Style
}

type HeaderStyles struct {
	ContainerStyle lipgloss.Style
	ModeStyle      lipgloss.Style
	FlowStyle      lipgloss.Style
	ContextStyle   lipgloss.Style
}

// InitStyles creates the full Styles struct from a theme.
func InitStyles(th *theme.Theme) Styles {
	if th == nil {
		th = &theme.Theme{
			Primary: "#7AA2F7", Secondary: "#BB9AF7", Accent: "#7DCFFF",
			Success: "#9ECE6A", Warning: "#E0AF68", Danger: "#F7768E",
			Info: "#7DCFFF", Text: "#C0CAF5", TextMuted: "#565F89",
			Border: "#3B4261", BorderFoc: "#7AA2F7", BgPanel: "#1A1B26",
		}
	}

	primary := lipgloss.Color(th.Primary)
	text := lipgloss.Color(th.Text)
	muted := lipgloss.Color(th.TextMuted)
	border := lipgloss.Color(th.Border)
	borderFoc := lipgloss.Color(th.BorderFoc)
	success := lipgloss.Color(th.Success)
	warning := lipgloss.Color(th.Warning)
	danger := lipgloss.Color(th.Danger)
	info := lipgloss.Color(th.Info)

	roundedBorder := lipgloss.RoundedBorder()

	return Styles{
		Common: CommonStyles{
			MainTextStyle: lipgloss.NewStyle().Foreground(text),
			MutedStyle:    lipgloss.NewStyle().Foreground(muted),
			ErrorStyle:    lipgloss.NewStyle().Foreground(danger),
			SuccessStyle:  lipgloss.NewStyle().Foreground(success),
			WarningStyle:  lipgloss.NewStyle().Foreground(warning),
			InfoStyle:     lipgloss.NewStyle().Foreground(info),
			BoldStyle:     lipgloss.NewStyle().Bold(true).Foreground(text),
		},
		Section: SectionStyles{
			ContainerStyle: lipgloss.NewStyle().
				Border(roundedBorder).
				BorderForeground(border).
				Padding(0, 1),
			FocusedContainerStyle: lipgloss.NewStyle().
				Border(roundedBorder).
				BorderForeground(borderFoc).
				Padding(0, 1),
			TitleStyle: lipgloss.NewStyle().
				Bold(true).
				Foreground(primary).
				Padding(0, 1),
			EmptyStateStyle: lipgloss.NewStyle().
				Foreground(muted).
				Italic(true).
				Padding(1, 2),
		},
		Sidebar: SidebarStyles{
			ContainerStyle: lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(border).
				Padding(0, 1),
			TitleStyle: lipgloss.NewStyle().Bold(true).Foreground(primary),
			ContentStyle: lipgloss.NewStyle().Foreground(text),
		},
		Table: TableStyles{
			RowStyle:         lipgloss.NewStyle().Foreground(text),
			SelectedRowStyle: lipgloss.NewStyle().Foreground(primary).Bold(true),
			HeaderStyle:      lipgloss.NewStyle().Foreground(muted).Bold(true),
			CellStyle:        lipgloss.NewStyle().Foreground(text),
		},
		Tabs: TabStyles{
			ActiveTab: lipgloss.NewStyle().
				Bold(true).
				Foreground(primary).
				Padding(0, 2),
			InactiveTab: lipgloss.NewStyle().
				Foreground(muted).
				Padding(0, 2),
			TabGap: lipgloss.NewStyle().
				Foreground(muted),
		},
		Footer: FooterStyles{
			ContainerStyle: lipgloss.NewStyle().
				Foreground(muted).
				Padding(0, 1),
			ViewSwitcherStyle: lipgloss.NewStyle().
				Foreground(primary),
			HelpKeyStyle: lipgloss.NewStyle().
				Foreground(primary).
				Bold(true),
			HelpDescStyle: lipgloss.NewStyle().
				Foreground(muted),
		},
		Panel: PanelStyles{
			BorderStyle: lipgloss.NewStyle().
				Border(roundedBorder).
				BorderForeground(border),
			FocusedBorderStyle: lipgloss.NewStyle().
				Border(roundedBorder).
				BorderForeground(borderFoc),
			TitleStyle: lipgloss.NewStyle().
				Bold(true).
				Foreground(primary),
			PillStyle: lipgloss.NewStyle().
				Foreground(text).
				Padding(0, 1),
		},
		Input: InputStyles{
			PromptStyle: lipgloss.NewStyle().Foreground(primary),
			TextStyle:   lipgloss.NewStyle().Foreground(text),
			CursorStyle: lipgloss.NewStyle().Foreground(primary),
		},
		Header: HeaderStyles{
			ContainerStyle: lipgloss.NewStyle().
				Foreground(text).
				Padding(0, 1),
			ModeStyle: lipgloss.NewStyle().
				Bold(true).
				Foreground(primary),
			FlowStyle: lipgloss.NewStyle().
				Foreground(info),
			ContextStyle: lipgloss.NewStyle().
				Foreground(muted),
		},
	}
}
