package panes

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type InputPane struct {
	styles  theme.Styles
	width   int
	focused bool
	input   string
}

func NewInputPane(styles theme.Styles) InputPane {
	return InputPane{styles: styles}
}

func (p InputPane) Init() tea.Cmd {
	return nil
}

func (p InputPane) Update(msg tea.Msg) (InputPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !p.focused {
			return p, nil
		}
		k := msg.String()
		switch {
		case k == "backspace":
			if len(p.input) > 0 {
				p.input = p.input[:len(p.input)-1]
			}
		case k == "enter":
			p.input = ""
		case len(k) == 1:
			p.input += k
		}
	}
	return p, nil
}

func (p InputPane) View() string {
	var b strings.Builder

	prompt := p.styles.Annotation.Render("gitdex> ")
	b.WriteString(prompt)
	b.WriteString(p.input)
	if p.focused {
		b.WriteString("█")
	}

	w := p.width - 2
	if w < 10 {
		w = 10
	}

	style := p.styles.NormalBorder
	if p.focused {
		style = p.styles.FocusedBorder
	}
	return style.Width(w).Render(b.String())
}

func (p *InputPane) SetSize(w, _ int) {
	p.width = w
}

func (p *InputPane) SetFocused(focused bool) {
	p.focused = focused
}

func (p InputPane) Focused() bool {
	return p.focused
}

func (p InputPane) Value() string {
	return p.input
}
