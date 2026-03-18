package context

import (
	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

// Styles centralizes all TUI styles, initialized from ThemeColors.
// Inspired by gh-dash's ui/context/styles.go.
type Styles struct {
	Common   CommonStyles
	Section  SectionStyles
	Table    TableStyles
	Sidebar  SidebarStyles
	Tabs     TabStyles
	Footer   FooterStyles
	Input    InputStyles
	Status   StatusStyles
	Header   HeaderStyles
	Panel    PanelStyles
}

// CommonStyles are base text styles.
type CommonStyles struct {
	MainText lipgloss.Style
	Faint    lipgloss.Style
	Bold     lipgloss.Style
	Error    lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Info     lipgloss.Style
}

// SectionStyles for section containers.
type SectionStyles struct {
	Container       lipgloss.Style
	FocusedBorder   lipgloss.Style
	UnfocusedBorder lipgloss.Style
	Title           lipgloss.Style
	EmptyState      lipgloss.Style
}

// TableStyles for table rendering.
type TableStyles struct {
	Header       lipgloss.Style
	Cell         lipgloss.Style
	SelectedCell lipgloss.Style
	SelectedRow  lipgloss.Style
	Row          lipgloss.Style
	Title        lipgloss.Style
}

// SidebarStyles for sidebar rendering.
type SidebarStyles struct {
	Container lipgloss.Style
	Border    lipgloss.Style
	Pager     lipgloss.Style
	Content   lipgloss.Style
}

// TabStyles for tab rendering.
type TabStyles struct {
	Active    lipgloss.Style
	Inactive  lipgloss.Style
	Separator lipgloss.Style
	Container lipgloss.Style
}

// FooterStyles for footer rendering.
type FooterStyles struct {
	Container   lipgloss.Style
	ViewSwitch  lipgloss.Style
	Help        lipgloss.Style
	KeyBinding  lipgloss.Style
	Description lipgloss.Style
}

// InputStyles for input fields.
type InputStyles struct {
	Prompt lipgloss.Style
	Text   lipgloss.Style
	Cursor lipgloss.Style
}

// StatusStyles for status badges.
type StatusStyles struct {
	OK   lipgloss.Style
	Run  lipgloss.Style
	Err  lipgloss.Style
	Wait lipgloss.Style
	Skip lipgloss.Style
}

// HeaderStyles for the legacy header bar.
type HeaderStyles struct {
	ModeStyle      lipgloss.Style
	FlowStyle      lipgloss.Style
	ContextStyle   lipgloss.Style
	ContainerStyle lipgloss.Style
}

// PanelStyles for the legacy panel component.
type PanelStyles struct {
	BorderStyle        lipgloss.Style
	FocusedBorderStyle lipgloss.Style
	TitleStyle         lipgloss.Style
}

// InitStylesFromLegacy creates Styles from a legacy *theme.Theme.
func InitStylesFromLegacy(t *theme.Theme) *Styles {
	if t == nil {
		return InitStyles(nil)
	}
	return InitStyles(&ThemeColors{
		Primary:       t.Primary,
		Secondary:     t.Secondary,
		Background:    t.BgPanel,
		Foreground:    t.Text,
		Subtle:        t.TextMuted,
		Highlight:     t.Accent,
		SuccessText:   t.Success,
		WarningText:   t.Warning,
		DangerText:    t.Danger,
		InfoText:      t.Info,
		OpenColor:     t.Success,
		ClosedColor:   t.Danger,
		MergedColor:   t.Secondary,
		DraftColor:    t.TextMuted,
		BorderNormal:  t.Border,
		BorderFocused: t.BorderFoc,
		BorderActive:  t.Success,
		TabActive:     t.Primary,
		TabInactive:   t.TextMuted,
		StatusOK:      t.Success,
		StatusRun:     t.Primary,
		StatusErr:     t.Danger,
		StatusWait:    t.TextMuted,
		StatusSkip:    t.Warning,
	})
}

// InitStyles creates Styles from a ThemeColors palette.
func InitStyles(th *ThemeColors) *Styles {
	if th == nil {
		th = DefaultTheme()
	}

	s := &Styles{}

	s.Common = CommonStyles{
		MainText: lipgloss.NewStyle().Foreground(lipgloss.Color(th.Foreground)),
		Faint:    lipgloss.NewStyle().Foreground(lipgloss.Color(th.Subtle)),
		Bold:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Foreground)),
		Error:    lipgloss.NewStyle().Foreground(lipgloss.Color(th.DangerText)),
		Success:  lipgloss.NewStyle().Foreground(lipgloss.Color(th.SuccessText)),
		Warning:  lipgloss.NewStyle().Foreground(lipgloss.Color(th.WarningText)),
		Info:     lipgloss.NewStyle().Foreground(lipgloss.Color(th.InfoText)),
	}

	s.Section = SectionStyles{
		Container:       lipgloss.NewStyle(),
		FocusedBorder:   lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(th.BorderFocused)),
		UnfocusedBorder: lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(th.BorderNormal)),
		Title:           lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Primary)),
		EmptyState:      lipgloss.NewStyle().Foreground(lipgloss.Color(th.Subtle)).Italic(true),
	}

	s.Table = TableStyles{
		Header:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Primary)).Padding(0, 1),
		Cell:         lipgloss.NewStyle().Padding(0, 1),
		SelectedCell: lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color(th.Primary)).Foreground(lipgloss.Color(th.Background)),
		SelectedRow:  lipgloss.NewStyle().Background(lipgloss.Color(th.Primary)).Foreground(lipgloss.Color(th.Background)),
		Row:          lipgloss.NewStyle(),
		Title:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Foreground)),
	}

	s.Sidebar = SidebarStyles{
		Container: lipgloss.NewStyle(),
		Border:    lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(th.BorderNormal)),
		Pager:     lipgloss.NewStyle().Foreground(lipgloss.Color(th.Subtle)),
		Content:   lipgloss.NewStyle().Padding(0, 1),
	}

	s.Tabs = TabStyles{
		Active:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.TabActive)).Underline(true),
		Inactive:  lipgloss.NewStyle().Foreground(lipgloss.Color(th.TabInactive)),
		Separator: lipgloss.NewStyle().Foreground(lipgloss.Color(th.Subtle)),
		Container: lipgloss.NewStyle().Padding(0, 1),
	}

	s.Footer = FooterStyles{
		Container:   lipgloss.NewStyle().Padding(0, 1),
		ViewSwitch:  lipgloss.NewStyle().Foreground(lipgloss.Color(th.Primary)),
		Help:        lipgloss.NewStyle().Foreground(lipgloss.Color(th.Subtle)),
		KeyBinding:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Foreground)),
		Description: lipgloss.NewStyle().Foreground(lipgloss.Color(th.Subtle)),
	}

	s.Input = InputStyles{
		Prompt: lipgloss.NewStyle().Foreground(lipgloss.Color(th.Primary)),
		Text:   lipgloss.NewStyle().Foreground(lipgloss.Color(th.Foreground)),
		Cursor: lipgloss.NewStyle().Foreground(lipgloss.Color(th.Primary)),
	}

	s.Status = StatusStyles{
		OK:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.StatusOK)),
		Run:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.StatusRun)),
		Err:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.StatusErr)),
		Wait: lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusWait)),
		Skip: lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusSkip)),
	}

	s.Header = HeaderStyles{
		ModeStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Background)).Background(lipgloss.Color(th.Primary)),
		FlowStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color(th.Secondary)),
		ContextStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(th.Subtle)),
		ContainerStyle: lipgloss.NewStyle(),
	}

	s.Panel = PanelStyles{
		BorderStyle:        lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(th.BorderNormal)),
		FocusedBorderStyle: lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(th.BorderFocused)),
		TitleStyle:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Primary)),
	}

	return s
}
