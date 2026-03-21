package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type SubmitMsg struct {
	Input     string
	IsCommand bool
	IsIntent  bool
}

type Composer struct {
	t       *theme.Theme
	input   []rune
	cursor  int
	width   int
	focused bool
	history []string
	histIdx int
}

func NewComposer(t *theme.Theme) *Composer {
	return &Composer{t: t, histIdx: -1}
}

func (c *Composer) SetWidth(w int)    { c.width = w }
func (c *Composer) SetFocused(f bool) { c.focused = f }
func (c *Composer) Focused() bool     { return c.focused }

func (c *Composer) modeMeta() (string, string, lipgloss.Style) {
	if strings.HasPrefix(string(c.input), "/") {
		return "Command", theme.Icons.Search, lipgloss.NewStyle().
			Bold(true).
			Foreground(c.t.OnPrimary()).
			Background(c.t.Warning()).
			Padding(0, 1)
	}
	if strings.HasPrefix(string(c.input), "!") {
		return "Execute", theme.Icons.Rocket, lipgloss.NewStyle().
			Bold(true).
			Foreground(c.t.OnPrimary()).
			Background(c.t.Accent()).
			Padding(0, 1)
	}
	return "Chat", theme.Icons.Chat, lipgloss.NewStyle().
		Bold(true).
		Foreground(c.t.OnPrimary()).
		Background(c.t.Secondary()).
		Padding(0, 1)
}

func (c *Composer) renderInput() string {
	cursorStyle := lipgloss.NewStyle().
		Foreground(c.t.OnPrimary()).
		Background(c.t.Accent()).
		Bold(true)
	placeholder := lipgloss.NewStyle().
		Foreground(c.t.DimText()).
		Italic(true)
	placeholderText := "Type a command or question..."

	if len(c.input) == 0 {
		if c.focused {
			return cursorStyle.Render(" ") + placeholder.Render(" "+placeholderText)
		}
		return placeholder.Render(placeholderText)
	}

	before := string(c.input[:c.cursor])
	current := " "
	after := ""
	if c.cursor < len(c.input) {
		current = string(c.input[c.cursor])
		after = string(c.input[c.cursor+1:])
	}
	if !c.focused {
		return before + current + after
	}
	return before + cursorStyle.Render(current) + after
}

func (c *Composer) Update(msg tea.Msg) tea.Cmd {
	if !c.focused {
		return nil
	}

	if paste, ok := msg.(tea.PasteMsg); ok {
		runes := []rune(paste.Content)
		tail := append([]rune{}, c.input[c.cursor:]...)
		c.input = append(c.input[:c.cursor], append(runes, tail...)...)
		c.cursor += len(runes)
		return nil
	}

	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}

	switch k := km.String(); {
	case k == "enter":
		return c.submit()
	case k == "backspace":
		if c.cursor > 0 {
			c.input = append(c.input[:c.cursor-1], c.input[c.cursor:]...)
			c.cursor--
		}
	case k == "delete":
		if c.cursor < len(c.input) {
			c.input = append(c.input[:c.cursor], c.input[c.cursor+1:]...)
		}
	case k == "left":
		if c.cursor > 0 {
			c.cursor--
		}
	case k == "right":
		if c.cursor < len(c.input) {
			c.cursor++
		}
	case k == "home", k == "ctrl+a":
		c.cursor = 0
	case k == "end", k == "ctrl+e":
		c.cursor = len(c.input)
	case k == "ctrl+u":
		c.input = c.input[c.cursor:]
		c.cursor = 0
	case k == "ctrl+k":
		c.input = c.input[:c.cursor]
	case k == "up":
		c.historyPrev()
	case k == "down":
		c.historyNext()
	default:
		r := []rune(k)
		if len(r) == 1 && r[0] >= 32 {
			c.input = append(c.input[:c.cursor], append(r, c.input[c.cursor:]...)...)
			c.cursor++
		}
	}
	return nil
}

func (c *Composer) Render() string {
	if c.width <= 0 {
		return ""
	}

	mode, icon, modeStyle := c.modeMeta()
	modeChip := modeStyle.Render(icon + " " + strings.ToUpper(mode))
	content := lipgloss.NewStyle().Foreground(c.t.Fg()).Render(c.renderInput())
	hint := lipgloss.NewStyle().
		Foreground(c.t.DimText()).
		Render("Enter send  Up/Down history  Ctrl+P palette")

	borderColor := c.t.BorderColor()
	if c.focused {
		borderColor = c.t.FocusBorderColor()
	}

	innerWidth := c.width - 4
	if innerWidth < 30 {
		innerWidth = 30
	}
	gap := innerWidth - lipgloss.Width(modeChip) - lipgloss.Width(content) - lipgloss.Width(hint)
	if gap < 2 {
		hint = ""
		gap = innerWidth - lipgloss.Width(modeChip) - lipgloss.Width(content)
	}
	if gap < 1 {
		gap = 1
	}

	line := modeChip + strings.Repeat(" ", gap) + content
	if hint != "" {
		line += " " + hint
	}

	return render.SurfacePanel(line, c.width, c.t.Surface(), borderColor)
}

func (c *Composer) Value() string {
	return string(c.input)
}

func (c *Composer) IsEmpty() bool {
	return len(c.input) == 0
}

func (c *Composer) submit() tea.Cmd {
	val := strings.TrimSpace(string(c.input))
	if val == "" {
		return nil
	}

	c.history = append(c.history, val)
	c.histIdx = len(c.history)
	c.input = nil
	c.cursor = 0

	return func() tea.Msg {
		return SubmitMsg{
			Input:     val,
			IsCommand: strings.HasPrefix(val, "/"),
			IsIntent:  strings.HasPrefix(val, "!"),
		}
	}
}

func (c *Composer) historyPrev() {
	if len(c.history) == 0 {
		return
	}
	if c.histIdx > 0 {
		c.histIdx--
	}
	c.input = []rune(c.history[c.histIdx])
	c.cursor = len(c.input)
}

func (c *Composer) historyNext() {
	if c.histIdx >= len(c.history)-1 {
		c.histIdx = len(c.history)
		c.input = nil
		c.cursor = 0
		return
	}
	c.histIdx++
	c.input = []rune(c.history[c.histIdx])
	c.cursor = len(c.input)
}
