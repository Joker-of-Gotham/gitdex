package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

// SuggestionCard renders a suggestion with action, risk color, and keybindings.
type SuggestionCard struct {
	Action    string
	Reason    string
	Commands  []string
	RiskLevel string // "safe", "caution", "dangerous"
	Expanded  bool   // shows "why" explanation
}

// NewSuggestionCard creates a new SuggestionCard.
func NewSuggestionCard(action, reason string, commands []string, risk string) SuggestionCard {
	return SuggestionCard{
		Action:    action,
		Reason:    reason,
		Commands:  commands,
		RiskLevel: risk,
	}
}

// View renders the card with lipgloss.
// Layout: risk color bar | action text | [y]accept [n]skip [w]why
func (c SuggestionCard) View(width int) string {
	if width <= 0 {
		width = 60
	}
	riskStyle := c.riskStyle()

	barStyle := riskStyle.Width(2).Align(lipgloss.Center)
	bar := barStyle.Render("  ")

	actionStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	actionBlock := actionStyle.Render(c.Action)

	keyStyle := lipgloss.NewStyle().Faint(true).Padding(0, 1)
	keybinds := keyStyle.Render(i18n.T("suggestions.keybinds"))

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, bar, actionBlock, keybinds)

	var body strings.Builder
	if len(c.Commands) > 0 {
		cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(4)
		for _, cmd := range c.Commands {
			body.WriteString(cmdStyle.Render("$ "+cmd) + "\n")
		}
	}

	if c.Expanded && c.Reason != "" {
		reasonStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			PaddingLeft(4).
			Italic(true)
		body.WriteString(reasonStyle.Render(fmt.Sprintf(i18n.T("suggestions.why_prefix"), c.Reason)) + "\n")
	}

	result := topRow
	if body.Len() > 0 {
		result = lipgloss.JoinVertical(lipgloss.Left, topRow, strings.TrimSuffix(body.String(), "\n"))
	}

	container := lipgloss.NewStyle().Width(width).MaxWidth(width)
	return container.Render(result)
}

func (c SuggestionCard) riskStyle() lipgloss.Style {
	if theme.Current == nil {
		return lipgloss.NewStyle()
	}
	switch c.RiskLevel {
	case "safe":
		return theme.Current.RiskSafe
	case "caution":
		return theme.Current.RiskCaution
	case "dangerous":
		return theme.Current.RiskDanger
	default:
		return theme.Current.RiskSafe
	}
}
