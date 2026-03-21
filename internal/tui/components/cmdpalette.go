package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type PaletteItem struct {
	Category    string
	Label       string
	Description string
	Shortcut    string
	Action      func() tea.Cmd
}

type CmdPalette struct {
	input    []rune
	cursor   int
	items    []PaletteItem
	filtered []int
	selected int
	visible  bool
	theme    *theme.Theme
	width    int
	height   int
}

func NewCmdPalette(t *theme.Theme) *CmdPalette {
	return &CmdPalette{
		theme:   t,
		width:   60,
		height:  20,
		visible: false,
	}
}

func (cp *CmdPalette) AddItem(item PaletteItem) {
	cp.items = append(cp.items, item)
	cp.refilter()
}

func (cp *CmdPalette) Show() {
	cp.visible = true
	cp.input = nil
	cp.cursor = 0
	cp.selected = 0
	cp.refilter()
}

func (cp *CmdPalette) Hide() { cp.visible = false }

func (cp *CmdPalette) IsVisible() bool { return cp.visible }

func (cp *CmdPalette) SetSize(w, h int) {
	cp.width = w
	cp.height = h
}

func (cp *CmdPalette) refilter() {
	query := strings.ToLower(string(cp.input))
	if query == "" {
		cp.filtered = make([]int, len(cp.items))
		for i := range cp.items {
			cp.filtered[i] = i
		}
		cp.selected = 0
		return
	}

	prefix := make([]int, 0, len(cp.items))
	contains := make([]int, 0, len(cp.items))
	for i, item := range cp.items {
		label := strings.ToLower(item.Label)
		desc := strings.ToLower(item.Description)
		category := strings.ToLower(item.Category)
		switch {
		case strings.HasPrefix(label, query), strings.HasPrefix(category, query):
			prefix = append(prefix, i)
		case strings.Contains(label, query), strings.Contains(desc, query), strings.Contains(category, query):
			contains = append(contains, i)
		}
	}

	cp.filtered = append(prefix, contains...)
	cp.selected = 0
}

func (cp *CmdPalette) Update(msg tea.Msg) tea.Cmd {
	if !cp.visible {
		return nil
	}
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}

	switch km.String() {
	case "esc":
		cp.Hide()
		return nil
	case "enter":
		if len(cp.filtered) == 0 {
			return nil
		}
		idx := cp.filtered[cp.selected]
		action := cp.items[idx].Action
		cp.Hide()
		if action != nil {
			return action()
		}
		return nil
	case "up":
		cp.selected--
		if cp.selected < 0 {
			cp.selected = len(cp.filtered) - 1
		}
		if cp.selected < 0 {
			cp.selected = 0
		}
		return nil
	case "down":
		cp.selected++
		if cp.selected >= len(cp.filtered) {
			cp.selected = 0
		}
		return nil
	case "backspace":
		if cp.cursor > 0 {
			cp.input = append(cp.input[:cp.cursor-1], cp.input[cp.cursor:]...)
			cp.cursor--
			cp.refilter()
			cp.clampCursor()
		}
		return nil
	case "delete":
		if cp.cursor < len(cp.input) {
			cp.input = append(cp.input[:cp.cursor], cp.input[cp.cursor+1:]...)
			cp.refilter()
			cp.clampCursor()
		}
		return nil
	case "left":
		if cp.cursor > 0 {
			cp.cursor--
		}
		return nil
	case "right":
		if cp.cursor < len(cp.input) {
			cp.cursor++
		}
		return nil
	default:
		r := []rune(km.String())
		if len(r) == 1 && r[0] >= 32 {
			cp.input = append(cp.input[:cp.cursor], append(r, cp.input[cp.cursor:]...)...)
			cp.cursor++
			cp.refilter()
		}
		return nil
	}
}

func (cp *CmdPalette) clampCursor() {
	if cp.cursor > len(cp.input) {
		cp.cursor = len(cp.input)
	}
	if cp.cursor < 0 {
		cp.cursor = 0
	}
}

func (cp *CmdPalette) Render(totalWidth, totalHeight int) string {
	if !cp.visible {
		return ""
	}

	query := string(cp.input)
	before := query[:cp.cursor]
	after := ""
	if cp.cursor < len(query) {
		after = query[cp.cursor:]
	}

	cursor := lipgloss.NewStyle().
		Foreground(cp.theme.OnPrimary()).
		Background(cp.theme.Accent()).
		Render(" ")
	searchInput := before + cursor + after
	if len(cp.input) == 0 {
		searchInput += lipgloss.NewStyle().
			Foreground(cp.theme.DimText()).
			Italic(true).
			Render(" jump to a view, command, or workflow")
	}

	searchLine := lipgloss.NewStyle().
		Foreground(cp.theme.Fg()).
		Background(cp.theme.Surface()).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cp.theme.BorderColor()).
		Padding(0, 1).
		Width(cp.width - 8).
		Render(searchInput)

	maxItems := 10
	if maxItems > cp.height-8 {
		maxItems = cp.height - 8
	}
	if maxItems < 1 {
		maxItems = 1
	}

	fgStyle := lipgloss.NewStyle().Foreground(cp.theme.Fg())
	dimStyle := lipgloss.NewStyle().Foreground(cp.theme.DimText())
	mutedStyle := lipgloss.NewStyle().Foreground(cp.theme.MutedFg())
	categoryStyle := lipgloss.NewStyle().Foreground(cp.theme.Secondary())
	selectedStyle := lipgloss.NewStyle().
		Foreground(cp.theme.Fg()).
		Background(cp.theme.Selection()).
		Bold(true)

	listLines := make([]string, 0, maxItems)
	if len(cp.filtered) == 0 {
		listLines = append(listLines, dimStyle.Render("No matches. Try dashboard, chat, theme, or settings."))
	}

	displayCount := len(cp.filtered)
	if displayCount > maxItems {
		displayCount = maxItems
	}
	start := cp.selected - displayCount/2
	if start < 0 {
		start = 0
	}
	if start+displayCount > len(cp.filtered) {
		start = len(cp.filtered) - displayCount
	}
	if start < 0 {
		start = 0
	}

	for i := start; i < start+displayCount && i < len(cp.filtered); i++ {
		item := cp.items[cp.filtered[i]]
		category := categoryStyle.Render(strings.ToUpper(item.Category))
		labelStyle := fgStyle
		if i == cp.selected {
			labelStyle = selectedStyle
		}
		label := labelStyle.Render(item.Label)
		desc := dimStyle.Render(item.Description)
		shortcut := mutedStyle.Render(item.Shortcut)
		width := cp.width - 8
		gap := width - lipgloss.Width(category) - lipgloss.Width(label) - lipgloss.Width(desc) - lipgloss.Width(shortcut) - 6
		if gap < 1 {
			gap = 1
		}
		line := category + "  " + label + strings.Repeat(" ", gap) + desc + "  " + shortcut
		listLines = append(listLines, line)
	}

	footer := dimStyle.Render(fmt.Sprintf("%d results  •  Enter run  •  Esc close", len(cp.filtered)))
	content := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(cp.theme.Primary()).Render(theme.Icons.Search + " Command Palette"),
		dimStyle.Render("Type to filter views, actions, and shortcuts"),
		"",
		searchLine,
		"",
		strings.Join(listLines, "\n"),
		"",
		footer,
	}, "\n")

	box := lipgloss.NewStyle().
		Background(cp.theme.Elevated()).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cp.theme.FocusBorderColor()).
		Width(cp.width).
		Height(cp.height).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)
}
