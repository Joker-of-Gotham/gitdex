package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func tsStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("#6FC3DF")) }
func keyStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C572")).Bold(true)
}
func valueStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("#DCE7EF")) }
func mutedStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")) }
func successStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#7BD389")).Bold(true)
}
func warnStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#F4B942")).Bold(true)
}
func dangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C73")).Bold(true)
}
func infoStyle() lipgloss.Style    { return lipgloss.NewStyle().Foreground(lipgloss.Color("#A9BBC7")) }
func commandStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("#98C6FF")) }

func statusStyleForText(status string) lipgloss.Style {
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(status, "success"), strings.Contains(status, "done"), strings.Contains(status, "viewed"), strings.Contains(status, "completed"):
		return successStyle()
	case strings.Contains(status, "fail"), strings.Contains(status, "error"), strings.Contains(status, "blocked"):
		return dangerStyle()
	case strings.Contains(status, "run"), strings.Contains(status, "pending"), strings.Contains(status, "wait"):
		return warnStyle()
	default:
		return valueStyle()
	}
}

func eventTypeStyle(kind oplog.EntryType) lipgloss.Style {
	switch kind {
	case oplog.EntryCmdSuccess:
		return successStyle()
	case oplog.EntryCmdFail, oplog.EntryLLMError:
		return dangerStyle()
	case oplog.EntryCmdExec, oplog.EntryLLMStart:
		return warnStyle()
	case oplog.EntryStateRefresh:
		return infoStyle()
	default:
		return valueStyle()
	}
}

func wrapPlainText(text string, width int) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.ReplaceAll(line, "\t", "  ")
		if width <= 0 {
			out = append(out, line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, strings.Split(runewidth.Wrap(line, width), "\n")...)
	}
	return out
}

func renderWrappedField(prefix string, prefixStyle lipgloss.Style, value string, valueStyle lipgloss.Style, width int) []string {
	prefixWidth := runewidth.StringWidth(prefix)
	valueWidth := width - prefixWidth
	if valueWidth < 8 {
		valueWidth = width
		prefixWidth = 0
		prefix = ""
	}
	wrapped := wrapPlainText(value, valueWidth)
	if len(wrapped) == 0 {
		wrapped = []string{""}
	}
	out := make([]string, 0, len(wrapped))
	for i, line := range wrapped {
		lead := prefix
		if i > 0 {
			lead = strings.Repeat(" ", prefixWidth)
		}
		out = append(out, prefixStyle.Render(lead)+valueStyle.Render(line))
	}
	return out
}
