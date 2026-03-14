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
	Action          string
	Reason          string
	Commands        []string
	Notes           []string
	CommandPrefix   string
	Controls        string
	RiskLevel       string // "safe", "caution", "dangerous"
	Expanded        bool   // shows "why" explanation
	Coverage        string
	Adapter         string
	Rollback        string
	Approval        string
	BoundaryReason  string
	RequestIdentity string
}

// NewSuggestionCard creates a new SuggestionCard.
func NewSuggestionCard(action, reason string, commands []string, risk string) SuggestionCard {
	return SuggestionCard{
		Action:        action,
		Reason:        reason,
		Commands:      commands,
		CommandPrefix: "$ ",
		RiskLevel:     risk,
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

	actionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F4F7FA")).
		Padding(0, 1)
	actionBlock := actionStyle.Render(c.Action)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8CA0AF")).
		Padding(0, 1)
	controls := strings.TrimSpace(c.Controls)
	keybinds := ""
	if controls != "" {
		keybinds = keyStyle.Render(controls)
	}

	topParts := []string{bar, actionBlock}
	if keybinds != "" {
		topParts = append(topParts, keybinds)
	}
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topParts...)

	var body strings.Builder
	if len(c.Commands) > 0 {
		cmdStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#93AEC3")).
			PaddingLeft(4)
		prefix := c.CommandPrefix
		for _, cmd := range c.Commands {
			cmd = strings.TrimSpace(cmd)
			if cmd == "" {
				continue
			}
			body.WriteString(cmdStyle.Render(prefix+cmd) + "\n")
		}
	}
	if len(c.Notes) > 0 {
		noteStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C9D9E6")).
			PaddingLeft(4)
		for _, note := range c.Notes {
			note = strings.TrimSpace(note)
			if note == "" {
				continue
			}
			body.WriteString(noteStyle.Render(note) + "\n")
		}
	}

	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9DB3C6")).
		PaddingLeft(4)
	metaLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6FC3DF")).
		Bold(true)

	metaLines := make([]string, 0, 5)
	appendMeta := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		metaLines = append(metaLines, metaStyle.Render(metaLabelStyle.Render(label)+value))
	}
	appendMeta("coverage: ", c.Coverage)
	appendMeta("adapter: ", c.Adapter)
	appendMeta("rollback: ", c.Rollback)
	appendMeta("approval: ", c.Approval)
	appendMeta("request: ", c.RequestIdentity)
	if c.Expanded && len(metaLines) > 0 {
		body.WriteString(strings.Join(metaLines, "\n") + "\n")
	}

	if c.Expanded && c.Reason != "" {
		reasonStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4E1EB")).
			PaddingLeft(4).
			Italic(true)
		body.WriteString(reasonStyle.Render(fmt.Sprintf(i18n.T("suggestions.why_prefix"), c.Reason)) + "\n")
	}
	if c.Expanded && strings.TrimSpace(c.BoundaryReason) != "" {
		boundaryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F2C572")).
			PaddingLeft(4)
		body.WriteString(boundaryStyle.Render("boundary: "+strings.TrimSpace(c.BoundaryReason)) + "\n")
	}

	result := topRow
	if body.Len() > 0 {
		result = lipgloss.JoinVertical(lipgloss.Left, topRow, strings.TrimSuffix(body.String(), "\n"))
	}

	container := lipgloss.NewStyle().
		MaxWidth(width).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#28465B")).
		Padding(0, 1)
	frameWidth, _ := container.GetFrameSize()
	innerWidth := width - frameWidth
	if innerWidth < 1 {
		innerWidth = 1
	}
	container = container.Width(innerWidth)
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
