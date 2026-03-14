package tui

import (
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

func (m *Model) showSelectedSuggestionGuidance() {
	if len(m.suggestions) == 0 || m.suggIdx < 0 || m.suggIdx >= len(m.suggestions) {
		return
	}
	if config.AutomationModeIsAutoLoop(m.automationMode()) {
		return
	}
	suggestion := m.suggestions[m.suggIdx]
	bodyLines := []string{
		localizedText("Next action:", "下一步动作：", "Next action:"),
		"- " + strings.TrimSpace(firstNonEmpty(suggestion.Action, m.suggestionPrimaryActionText(suggestion))),
		"",
		localizedText("How to run it:", "如何执行：", "How to run it:"),
		"- /run accept  (alias: /accept)",
		"- /run all",
		"- /run skip    (alias: /skip)",
		"- /run why     (alias: /why)",
		"- /run refresh (alias: /refresh)",
		"- /quit",
	}

	if preview := strings.TrimSpace(m.suggestionPrimaryActionText(suggestion)); preview != "" {
		bodyLines = append(bodyLines,
			"",
			localizedText("Execution preview:", "执行预览：", "Execution preview:"),
			"- "+preview,
		)
	}
	if suggestion.Interaction == git.PlatformExec && suggestion.PlatformOp != nil {
		if note := strings.TrimSpace(m.previewSuggestionAvailability(suggestion)); note != "" {
			bodyLines = append(bodyLines,
				"",
				localizedText("Before running:", "执行前：", "Before running:"),
				"- "+note,
			)
		}
	}

	m.commandResponseTitle = localizedText("Selected suggestion", "当前建议", "Selected suggestion")
	m.commandResponseBody = strings.Join(bodyLines, "\n")
	m.workspaceTab = workspaceTabSuggestions
}
