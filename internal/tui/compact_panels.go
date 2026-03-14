package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) renderCommandResponsePanelCached(width int) string {
	cacheKey := joinHashParts(strconv.Itoa(width), m.commandResponseTitle, m.commandResponseBody)
	if content, ok := m.renderCache.textValue("command_response", cacheKey); ok {
		return content
	}
	content := m.renderCommandResponsePanel(width)
	m.renderCache.setTextValue("command_response", cacheKey, content)
	return content
}

func (m Model) renderAutomationPanelCached(width int) string {
	cacheKey := joinHashParts(
		strconv.Itoa(width),
		strconv.Itoa(m.automation.MonitorInterval),
		m.automation.Mode,
		strconv.FormatBool(m.automation.Enabled),
		strconv.FormatBool(m.automation.TrustedMode),
		strconv.Itoa(m.automation.MaxAutoSteps),
	)
	if content, ok := m.renderCache.textValue("automation_panel", cacheKey); ok {
		return content
	}
	content := m.renderAutomationPanel(width)
	m.renderCache.setTextValue("automation_panel", cacheKey, content)
	return content
}

func (m Model) renderAnalysisPanelCached(width int) string {
	cacheKey := joinHashParts(
		strconv.Itoa(width),
		m.session.ActiveGoal,
		m.llmGoalStatus,
		m.llmAnalysis,
	)
	if content, ok := m.renderCache.textValue("analysis_panel", cacheKey); ok {
		return content
	}
	content := m.renderAnalysisPanel(width)
	m.renderCache.setTextValue("analysis_panel", cacheKey, content)
	return content
}

func (m Model) renderLatestResultPanel(width int) string {
	panelWidth := width
	if panelWidth < 18 {
		panelWidth = 18
	}
	cacheKey := joinHashParts(
		strconv.Itoa(panelWidth),
		m.lastCommand.Title,
		m.lastCommand.Status,
		m.lastCommand.Output,
		m.lastCommand.ResultKind,
		m.lastCommand.PlatformCapability,
		m.lastCommand.PlatformFlow,
		m.lastCommand.PlatformOperation,
		m.lastCommand.PlatformResourceID,
		m.lastCommand.FilePath,
		m.lastCommand.FileOperation,
	)
	if content, ok := m.renderCache.textValue("latest_result_compact", cacheKey); ok {
		return content
	}
	if m.lastCommand.Title == "" {
		return ""
	}

	borderStyle := panelStyleForStatus(m.lastCommand.Status).Padding(0, 1)
	innerWidth, _ := panelInnerSize(borderStyle, panelWidth, 1)
	lines := []string{
		keyStyle().Render(localizedLatestResultTitle()),
	}
	lines = append(lines, renderWrappedField(localizedText("Status: ", "状态：", "Status: "), keyStyle(), localizedStatusText(m.lastCommand.Status), statusStyleForText(m.lastCommand.Status), innerWidth)...)
	lines = append(lines, renderWrappedField(localizedText("Target: ", "目标：", "Target: "), keyStyle(), m.latestResultTarget(), commandStyle(), innerWidth)...)
	if summary := firstNonEmptyLine(m.lastCommand.Output); summary != "" {
		lines = append(lines, renderWrappedField(localizedText("Summary: ", "摘要：", "Summary: "), keyStyle(), summary, infoStyle(), innerWidth)...)
	}

	content := borderStyle.Width(panelWidth).Render(strings.Join(lines, "\n"))
	m.renderCache.setTextValue("latest_result_compact", cacheKey, content)
	return content
}

func (m Model) renderOperationLogPanelCached(width, height int) string {
	entries := m.opLogEntries()
	lastKey := ""
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		lastKey = strings.Join([]string{
			last.Timestamp.Format(time.RFC3339Nano),
			last.Summary,
			last.Detail,
		}, "|")
	}
	cacheKey := joinHashParts(
		strconv.Itoa(width),
		strconv.Itoa(height),
		strconv.FormatBool(m.logExpanded),
		strconv.Itoa(m.logScrollOffset),
		strconv.Itoa(len(entries)),
		lastKey,
	)
	if content, ok := m.renderCache.textValue("operation_log", cacheKey); ok {
		return content
	}
	content := m.renderOperationLogPanel(width, height)
	m.renderCache.setTextValue("operation_log", cacheKey, content)
	return content
}

func (m Model) renderObservabilityPanelCachedWithRegions(width, height int) (string, []clickRegion) {
	cacheKey := joinHashParts(
		strconv.Itoa(width),
		strconv.Itoa(height),
		strconv.Itoa(int(m.obsTab)),
		strconv.Itoa(m.obsScroll),
		m.lastCommand.Title,
		m.lastCommand.Status,
		m.lastCommand.Output,
		strconv.Itoa(len(m.mutationLedger)),
		strconv.Itoa(len(m.analysisHistory)),
		strconv.Itoa(len(m.automationLocks)),
		strconv.Itoa(len(m.automationFailures)),
		strconv.FormatBool(m.automationObserveOnly),
		m.lastEscalation.Format(time.RFC3339Nano),
		m.lastRecovery.Format(time.RFC3339Nano),
		m.lastAnalysisFingerprint,
	)
	if content, clicks, ok := m.renderCache.regionValue("observability_panel", cacheKey); ok {
		return content, clicks
	}
	content, clicks := m.renderObservabilityPanelWithRegions(width, height)
	m.renderCache.setRegionValue("observability_panel", cacheKey, content, clicks)
	return content, clicks
}

func (m Model) renderSuggestionCardsCompactWithRegions(width int) (string, []clickRegion) {
	cardWidth := width
	if cardWidth <= 0 {
		cardWidth = m.width
	}
	if cardWidth < 18 {
		cardWidth = 18
	}
	cacheParts := []string{strconv.Itoa(cardWidth), strconv.Itoa(m.suggIdx), strconv.FormatBool(m.expanded), m.llmReason}
	for i, s := range m.suggestions {
		cacheParts = append(cacheParts,
			s.Action,
			s.Reason,
			strconv.Itoa(int(s.RiskLevel)),
			strconv.Itoa(int(s.Interaction)),
			joinCmd(s.Command),
		)
		if s.PlatformOp != nil {
			cacheParts = append(cacheParts, git.PlatformExecIdentity(s.PlatformOp))
		}
		if i < len(m.suggExecState) {
			cacheParts = append(cacheParts, strconv.Itoa(int(m.suggExecState[i])))
		}
		if i < len(m.suggExecMsg) {
			cacheParts = append(cacheParts, m.suggExecMsg[i])
		}
	}
	cacheKey := joinHashParts(cacheParts...)
	if content, clicks, ok := m.renderCache.regionValue("suggestion_cards", cacheKey); ok {
		return content, clicks
	}

	var cards []string
	var clicks []clickRegion
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	done := 0
	for _, st := range m.suggExecState {
		if st != git.ExecPending {
			done++
		}
	}
	title := fmt.Sprintf(i18n.T("suggestions.title"), len(m.suggestions))
	if done > 0 {
		title = fmt.Sprintf(i18n.T("suggestions.title_done"), done, len(m.suggestions))
	}
	cards = append(cards, headerStyle.Render(title))
	lineOffset := lineCount(cards[0])

	doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for i, s := range m.suggestions {
		isActive := i == m.suggIdx
		execState, execMsg := git.ExecPending, ""
		if i < len(m.suggExecState) {
			execState = m.suggExecState[i]
		}
		if i < len(m.suggExecMsg) {
			execMsg = m.suggExecMsg[i]
		}

		reason := strings.TrimSpace(s.Reason)
		if isActive && strings.TrimSpace(m.llmReason) != "" {
			reason = strings.TrimSpace(m.llmReason)
		}
		action := strings.TrimSpace(s.Action)
		if s.Source == git.SourceLLM {
			action = "[AI] " + action
		}
		switch execState {
		case git.ExecDone:
			action = "[done] " + action
		case git.ExecFailed:
			action = "[fail] " + action
		case git.ExecRunning:
			action = "[run] " + action
		}

		cmdDisplay := []string{m.suggestionPrimaryActionText(s)}
		noteDisplay := compactCardNotes(reason, execMsg)
		card := components.NewSuggestionCard(action, reason, cmdDisplay, riskLabel(s.RiskLevel))
		card.Commands = compactStringList(cmdDisplay, 1)
		card.Notes = noteDisplay
		card.Controls = localizedText("click: select", "点击：选中", "click: select")
		if isActive {
			card.Controls = localizedText("selected | /accept /skip /why /refresh /quit", "已选中 | /accept /skip /why /refresh /quit", "selected | /accept /skip /why /refresh /quit")
		}
		if len(cmdDisplay) > 0 {
			card.CommandPrefix = localizedText("next: ", "下一步：", "next: ")
		} else {
			card.CommandPrefix = ""
		}

		if s.PlatformOp != nil {
			pID := m.detectedPlatform()
			meta := platformRequestMeta(pID, s.PlatformOp)
			card.Coverage = strings.TrimSpace(firstNonEmpty(string(meta.Coverage), m.capabilityCoverageLabel(s.PlatformOp.CapabilityID)))
			card.Adapter = string(meta.Adapter)
			card.Rollback = string(meta.Rollback)
			if meta.ApprovalRequired {
				card.Approval = "required"
			}
			card.BoundaryReason = strings.TrimSpace(firstNonEmpty(platformBoundaryReason(pID, s.PlatformOp.CapabilityID), meta.BoundaryReason))
			card.RequestIdentity = git.PlatformExecIdentity(s.PlatformOp)
			card.CommandPrefix = ""
			card.Commands = []string{m.suggestionPrimaryActionText(s)}
		}
		if s.Interaction == git.FileWrite || s.Interaction == git.InfoOnly || s.Interaction == git.NeedsInput {
			card.CommandPrefix = ""
		}
		if isActive && m.expanded {
			card.Expanded = true
		}

		rendered := card.View(cardWidth - 2)
		switch execState {
		case git.ExecDone:
			rendered = doneStyle.Render("[ok] ") + dimStyle.Render(rendered)
		case git.ExecFailed:
			rendered = failStyle.Render("[x] ") + rendered
		case git.ExecRunning:
			rendered = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).Render("[..] ") + rendered
		default:
			if isActive {
				rendered = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true).Render("> ") + rendered
			} else {
				rendered = "  " + rendered
			}
		}
		cards = append(cards, rendered)
		cardHeight := lineCount(rendered)
		clicks = append(clicks, clickRegion{action: "select_suggestion", index: i, x0: 0, y0: lineOffset, x1: cardWidth, y1: lineOffset + cardHeight})
		lineOffset += cardHeight
	}

	content := strings.Join(cards, "\n")
	m.renderCache.setRegionValue("suggestion_cards", cacheKey, content, clicks)
	return content, clicks
}

func compactCardNotes(reason, execMsg string) []string {
	notes := make([]string, 0, 2)
	appendNote := func(text string) {
		text = strings.TrimSpace(oneLine(text))
		if text == "" {
			return
		}
		for _, existing := range notes {
			if existing == text {
				return
			}
		}
		notes = append(notes, text)
	}
	appendNote(reason)
	appendNote(execMsg)
	return compactStringList(notes, 2)
}

func (m Model) suggestionPrimaryActionText(s git.Suggestion) string {
	switch s.Interaction {
	case git.InfoOnly:
		return localizedText("Review advisory details", "查看建议详情", "Review advisory details")
	case git.FileWrite:
		if s.FileOp != nil {
			op := strings.TrimSpace(s.FileOp.Operation)
			if op == "" {
				op = "create"
			}
			return fmt.Sprintf(localizedText("%s file %s", "%s 文件 %s", "%s file %s"), op, s.FileOp.Path)
		}
		return localizedText("Prepare file change", "准备文件修改", "Prepare file change")
	case git.NeedsInput:
		return localizedText("Requires input before execution", "执行前需要补充输入", "Requires input before execution")
	case git.PlatformExec:
		return platformSuggestionCommand(s.PlatformOp)
	default:
		command := strings.TrimSpace(joinCmd(suggestionCommandForExecution(s)))
		if command == "" {
			return localizedText("Review suggested action", "查看建议动作", "Review suggested action")
		}
		return command
	}
}

func (m Model) opLogEntries() []oplog.Entry {
	if m.opLog == nil {
		return nil
	}
	return m.opLog.Entries()
}

func (m Model) renderWorkspaceOverviewPanel(width int) string {
	sections := make([]string, 0, 3)
	if response := m.renderCommandResponsePanelCached(width); response != "" {
		sections = append(sections, response)
	}
	if result := m.renderLatestResultPanel(width); result != "" {
		sections = append(sections, result)
	}
	if len(sections) == 0 && strings.TrimSpace(m.llmAnalysis) != "" {
		sections = append(sections, m.renderAnalysisPanelCached(width))
	}
	if len(sections) == 0 {
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(localizedText(
			"Set a goal or run /refresh to start.",
			"先设置目标，或运行 /refresh 开始。",
			"Set a goal or run /refresh to start.",
		)))
	}
	return strings.Join(sections, "\n\n")
}

func (m Model) renderWorkspaceResultPanel(width int) string {
	sections := make([]string, 0, 2)
	if result := m.renderLatestResultPanel(width); result != "" {
		sections = append(sections, result)
	}
	if strings.TrimSpace(m.lastCommand.Title) != "" {
		sections = append(sections, m.renderCommandInspector(maxInt(18, width-4)))
	}
	if len(sections) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(localizedText(
			"No execution result yet.",
			"当前还没有执行结果。",
			"No execution result yet.",
		))
	}
	return strings.Join(sections, "\n\n")
}
