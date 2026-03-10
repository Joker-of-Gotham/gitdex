package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/memory"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

type workflowStage string

const (
	workflowPerceive workflowStage = "perceive"
	workflowAnalyze  workflowStage = "analyze"
	workflowSuggest  workflowStage = "suggest"
	workflowConfirm  workflowStage = "confirm"
	workflowExecute  workflowStage = "execute"
)

type observabilityTab int

const (
	observabilityWorkflow observabilityTab = iota
	observabilityTimeline
	observabilityContext
	observabilityMemory
	observabilityRaw
	observabilityCommand
	observabilityThinking
)

type commandTrace struct {
	Title  string
	Status string
	Output string
	At     time.Time
}

func (m *Model) setWorkflowStage(stage workflowStage) {
	m.workflowStage = stage
	m.workflowAt = time.Now()
}

func (m *Model) rememberPreference(key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if m.memoryStore == nil || key == "" || value == "" {
		return
	}
	m.memoryStore.SetPreference(key, value)
	_ = m.memoryStore.Save()
}

func (m *Model) rememberResolvedGoal(goal string) {
	goal = strings.TrimSpace(goal)
	if m.memoryStore == nil || goal == "" {
		return
	}
	m.memoryStore.RecordResolvedIssue(m.repoFingerprint(), goal)
	_ = m.memoryStore.Save()
}

func (m *Model) rememberRepoPattern(pattern string) {
	pattern = strings.TrimSpace(pattern)
	if m.memoryStore == nil || pattern == "" {
		return
	}
	m.memoryStore.RecordRepoPattern(m.repoFingerprint(), pattern)
	_ = m.memoryStore.Save()
}

func (m Model) repoFingerprint() string {
	if m.gitState == nil {
		return "unknown"
	}
	remoteURL := ""
	for _, info := range m.gitState.RemoteInfos {
		if strings.EqualFold(info.Name, "origin") {
			if strings.TrimSpace(info.PushURL) != "" {
				remoteURL = info.PushURL
			} else {
				remoteURL = info.FetchURL
			}
			break
		}
	}
	if remoteURL == "" && len(m.gitState.RemoteInfos) > 0 {
		remoteURL = m.gitState.RemoteInfos[0].PushURL
		if strings.TrimSpace(remoteURL) == "" {
			remoteURL = m.gitState.RemoteInfos[0].FetchURL
		}
	}
	return memory.RepoFingerprint(remoteURL, m.gitState.LocalBranch.Name)
}

func (m Model) currentPromptMemory() *prompt.MemoryContext {
	if m.memoryStore == nil {
		return nil
	}
	return m.memoryStore.ToPromptMemory(m.repoFingerprint())
}

func (t observabilityTab) label() string {
	switch t {
	case observabilityTimeline:
		return i18n.T("observability.tab_timeline")
	case observabilityContext:
		return i18n.T("observability.tab_context")
	case observabilityMemory:
		return i18n.T("observability.tab_memory")
	case observabilityRaw:
		return i18n.T("observability.tab_raw")
	case observabilityCommand:
		return i18n.T("observability.tab_result")
	case observabilityThinking:
		return i18n.T("observability.tab_thinking")
	default:
		return i18n.T("observability.tab_workflow")
	}
}

func (t observabilityTab) next() observabilityTab {
	switch t {
	case observabilityWorkflow:
		return observabilityTimeline
	case observabilityTimeline:
		return observabilityContext
	case observabilityContext:
		return observabilityMemory
	case observabilityMemory:
		return observabilityRaw
	case observabilityRaw:
		return observabilityCommand
	case observabilityCommand:
		return observabilityThinking
	default:
		return observabilityWorkflow
	}
}

func (t observabilityTab) prev() observabilityTab {
	switch t {
	case observabilityTimeline:
		return observabilityWorkflow
	case observabilityContext:
		return observabilityTimeline
	case observabilityMemory:
		return observabilityContext
	case observabilityRaw:
		return observabilityMemory
	case observabilityCommand:
		return observabilityRaw
	case observabilityThinking:
		return observabilityCommand
	default:
		return observabilityThinking
	}
}

func observabilityTabs() []observabilityTab {
	return []observabilityTab{
		observabilityWorkflow,
		observabilityTimeline,
		observabilityContext,
		observabilityMemory,
		observabilityRaw,
		observabilityCommand,
		observabilityThinking,
	}
}

func (m Model) renderObservabilityPanel(width int) string {
	if width < 24 {
		width = 24
	}
	panelWidth := width - 2
	if panelWidth < 24 {
		panelWidth = 24
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6FC3DF"))
	tabOn := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#10212B")).
		Background(lipgloss.Color("#F2C572")).
		Padding(0, 1)
	tabOff := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C7D5E0")).
		Background(lipgloss.Color("#233645")).
		Padding(0, 1)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99"))

	tabs := make([]string, 0, len(observabilityTabs()))
	for _, tab := range observabilityTabs() {
		style := tabOff
		if tab == m.obsTab {
			style = tabOn
		}
		tabs = append(tabs, style.Render(tab.label()))
	}

	body := m.renderWorkflowInspector(panelWidth - 4)
	switch m.obsTab {
	case observabilityTimeline:
		body = m.renderTimelineInspector(panelWidth - 4)
	case observabilityContext:
		body = m.renderContextInspector(panelWidth - 4)
	case observabilityMemory:
		body = m.renderMemoryInspector(panelWidth - 4)
	case observabilityRaw:
		body = m.renderRawInspector(panelWidth - 4)
	case observabilityCommand:
		body = m.renderCommandInspector(panelWidth - 4)
	case observabilityThinking:
		body = m.renderThinkingInspector(panelWidth - 4)
	}

	content := strings.Join([]string{
		headerStyle.Render(i18n.T("observability.title")),
		strings.Join(tabs, " "),
		body,
		hintStyle.Render(i18n.T("observability.hint")),
	}, "\n\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6FC3DF")).
		Padding(0, 1).
		Width(panelWidth).
		Render(content)
}

func (m Model) renderWorkflowInspector(width int) string {
	steps := []workflowStage{
		workflowPerceive,
		workflowAnalyze,
		workflowSuggest,
		workflowConfirm,
		workflowExecute,
	}
	labels := map[workflowStage]string{
		workflowPerceive: "Perceive repo state",
		workflowAnalyze:  "Analyze with LLM",
		workflowSuggest:  "Build suggestions",
		workflowConfirm:  "Wait for human choice",
		workflowExecute:  "Execute or write",
	}

	stageLabel := valueOr(labels[m.workflowStage], i18n.T("observability.workflow_waiting"))
	lines := []string{
		fmt.Sprintf(i18n.T("observability.workflow_current"), stageLabel),
	}
	currentIdx := workflowStageIndex(m.workflowStage)
	for _, step := range steps {
		idx := workflowStageIndex(step)
		prefix := "[ ]"
		switch {
		case currentIdx >= 0 && idx < currentIdx:
			prefix = "[x]"
		case idx == currentIdx:
			prefix = "[>]"
		}
		lines = append(lines, fmt.Sprintf("%s %s", prefix, labels[step]))
	}
	if goal := strings.TrimSpace(m.session.ActiveGoal); goal != "" {
		lines = append(lines, "", fmt.Sprintf(i18n.T("observability.workflow_goal"), goal))
	}
	if status := strings.TrimSpace(m.llmGoalStatus); status != "" {
		lines = append(lines, fmt.Sprintf(i18n.T("observability.workflow_goal_status"), status))
	}
	pending := 0
	done := 0
	for _, state := range m.suggExecState {
		if state == 0 {
			pending++
			continue
		}
		done++
	}
	lines = append(lines, fmt.Sprintf(i18n.T("observability.workflow_suggestions"), len(m.suggestions), pending, done))
	if m.execSuggIdx >= 0 && m.execSuggIdx < len(m.suggestions) {
		lines = append(lines, fmt.Sprintf(i18n.T("observability.workflow_executing"), m.suggestions[m.execSuggIdx].Action))
	}
	if m.lastCommand.Title != "" {
		lines = append(lines, fmt.Sprintf(i18n.T("observability.workflow_last_result"), m.lastCommand.Status, m.lastCommand.Title))
	}
	if !m.workflowAt.IsZero() {
		lines = append(lines, fmt.Sprintf(i18n.T("observability.workflow_updated"), m.workflowAt.Format("15:04:05")))
	}
	if entries := m.latestLogEntries(4); len(entries) > 0 {
		lines = append(lines, "", i18n.T("observability.workflow_recent_events"))
		for _, entry := range entries {
			lines = append(lines, fmt.Sprintf("  %s %s", entry.Timestamp.Format("15:04:05"), trimInspectorLine(entry.Summary, width-10)))
		}
	}
	return joinInspectorLines(lines, width)
}

func (m Model) renderTimelineInspector(width int) string {
	lines := []string{i18n.T("observability.timeline_title")}
	entries := m.latestLogEntries(10)
	if len(entries) == 0 {
		lines = append(lines, "(no events yet)")
		return joinInspectorLines(lines, width)
	}
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s [%s] %s", entry.Timestamp.Format("15:04:05"), entry.Type, entry.Summary))
		if detail := strings.TrimSpace(entry.Detail); detail != "" {
			for _, detailLine := range clipMultiline(detail, 2, width-4) {
				lines = append(lines, "  "+detailLine)
			}
		}
	}
	return joinInspectorLines(lines, width)
}

func (m Model) renderContextInspector(width int) string {
	lines := []string{
		fmt.Sprintf(i18n.T("observability.context_mode"), strings.TrimSpace(m.analysisTrace.Mode)),
		fmt.Sprintf(i18n.T("observability.context_models"), valueOr(m.analysisTrace.PrimaryModel, "-"), valueOr(m.analysisTrace.SecondaryModel, "off")),
		fmt.Sprintf(i18n.T("observability.context_budget"), m.analysisTrace.Budget, m.analysisTrace.Reserved, m.analysisTrace.Available),
	}
	if info := strings.TrimSpace(m.llmDebugInfo); info != "" {
		lines = append(lines, fmt.Sprintf(i18n.T("observability.context_summary"), info))
	}
	if len(m.analysisTrace.Partitions) > 0 {
		lines = append(lines, "", i18n.T("observability.context_partitions"))
		for _, part := range m.analysisTrace.Partitions {
			mark := "-"
			if part.Included {
				mark = "+"
			}
			if part.Truncated {
				mark = "~"
			}
			extra := ""
			if part.Required {
				extra = " req"
			}
			lines = append(lines, fmt.Sprintf("  %s %-16s %4dt%s", mark, part.Name, part.Tokens, extra))
		}
	}
	if len(m.analysisTrace.Knowledge) > 0 {
		lines = append(lines, "", i18n.T("observability.context_knowledge"))
		for _, item := range m.analysisTrace.Knowledge {
			lines = append(lines, "  - "+item.ScenarioID)
		}
	}
	if len(m.analysisTrace.RecentOps) > 0 {
		lines = append(lines, "", fmt.Sprintf(i18n.T("observability.context_recent_ops"), len(m.analysisTrace.RecentOps)))
	}
	lines = append(lines, "", i18n.T("observability.context_prompt_excerpt"))
	lines = append(lines, clipMultiline(m.analysisTrace.UserPrompt, 6, width)...)
	if system := strings.TrimSpace(m.analysisTrace.SystemPrompt); system != "" {
		lines = append(lines, "", i18n.T("observability.context_system_excerpt"))
		lines = append(lines, clipMultiline(system, 4, width)...)
	}
	return joinInspectorLines(lines, width)
}

func (m Model) renderMemoryInspector(width int) string {
	lines := []string{
		fmt.Sprintf(i18n.T("observability.memory_repo"), m.repoFingerprint()),
	}
	if m.memoryStore != nil {
		if path := strings.TrimSpace(m.memoryStore.Path()); path != "" {
			lines = append(lines, fmt.Sprintf(i18n.T("observability.memory_path"), path))
		}
		snapshot := m.memoryStore.Snapshot()
		if !snapshot.UpdatedAt.IsZero() {
			lines = append(lines, fmt.Sprintf(i18n.T("observability.memory_updated"), snapshot.UpdatedAt.Format("2006-01-02 15:04:05")))
		}
	}
	mem := m.currentPromptMemory()
	if mem == nil {
		lines = append(lines, i18n.T("observability.memory_none"))
	} else {
		if len(mem.UserPreferences) > 0 {
			lines = append(lines, "", i18n.T("observability.memory_preferences"))
			keys := make([]string, 0, len(mem.UserPreferences))
			for key := range mem.UserPreferences {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				lines = append(lines, fmt.Sprintf("  %s=%s", key, mem.UserPreferences[key]))
			}
		}
		if len(mem.RepoPatterns) > 0 {
			lines = append(lines, "", i18n.T("observability.memory_patterns"))
			for _, item := range mem.RepoPatterns {
				lines = append(lines, "  - "+item)
			}
		}
		if len(mem.ResolvedIssues) > 0 {
			lines = append(lines, "", i18n.T("observability.memory_resolved"))
			for _, item := range mem.ResolvedIssues {
				lines = append(lines, "  - "+item)
			}
		}
	}
	if len(m.session.GoalHistory) > 0 {
		lines = append(lines, "", i18n.T("observability.memory_session_goals"))
		for i := len(m.session.GoalHistory) - 1; i >= 0 && i >= len(m.session.GoalHistory)-5; i-- {
			entry := m.session.GoalHistory[i]
			lines = append(lines, fmt.Sprintf("  - %s [%s]", entry.Goal, entry.Status))
		}
	}
	if len(m.session.SkippedActions) > 0 {
		lines = append(lines, "", i18n.T("observability.memory_skipped"))
		start := len(m.session.SkippedActions) - 5
		if start < 0 {
			start = 0
		}
		for _, item := range m.session.SkippedActions[start:] {
			lines = append(lines, "  - "+item)
		}
	}
	return joinInspectorLines(lines, width)
}

func (m Model) renderRawInspector(width int) string {
	lines := []string{i18n.T("observability.raw_response")}
	if raw := strings.TrimSpace(m.analysisTrace.RawResponse); raw != "" {
		lines = append(lines, clipMultiline(raw, 8, width)...)
	} else {
		lines = append(lines, i18n.T("observability.raw_none"))
	}
	lines = append(lines, "", i18n.T("observability.raw_structured"))
	if cleaned := strings.TrimSpace(m.analysisTrace.CleanedResponse); cleaned != "" {
		lines = append(lines, clipMultiline(cleaned, 8, width)...)
	} else {
		lines = append(lines, i18n.T("observability.raw_cleaned_none"))
	}
	if len(m.analysisTrace.Rejected) > 0 {
		lines = append(lines, "", i18n.T("observability.raw_rejected"))
		for _, item := range m.analysisTrace.Rejected {
			lines = append(lines, "  - "+item)
		}
	}
	return joinInspectorLines(lines, width)
}

func (m Model) renderCommandInspector(width int) string {
	lines := []string{}
	if m.lastCommand.Title == "" {
		lines = append(lines, i18n.T("observability.result_none"))
		if entries := m.latestResultEntries(5); len(entries) > 0 {
			lines = append(lines, "", i18n.T("observability.result_recent"))
			for _, entry := range entries {
				lines = append(lines, fmt.Sprintf("  %s %s", entry.Timestamp.Format("15:04:05"), entry.Summary))
			}
		}
		return joinInspectorLines(lines, width)
	}
	lines = append(lines,
		fmt.Sprintf(i18n.T("observability.result_status"), m.lastCommand.Status),
		fmt.Sprintf(i18n.T("observability.result_target"), m.lastCommand.Title),
	)
	if !m.lastCommand.At.IsZero() {
		lines = append(lines, fmt.Sprintf(i18n.T("observability.result_time"), m.lastCommand.At.Format("15:04:05")))
	}
	lines = append(lines, "", i18n.T("observability.result_output"))
	lines = append(lines, clipMultiline(m.lastCommand.Output, 12, width)...)
	if entries := m.latestResultEntries(5); len(entries) > 0 {
		lines = append(lines, "", i18n.T("observability.result_recent"))
		for _, entry := range entries {
			lines = append(lines, fmt.Sprintf("  %s %s", entry.Timestamp.Format("15:04:05"), entry.Summary))
		}
	}
	return joinInspectorLines(lines, width)
}

func (m Model) renderThinkingInspector(width int) string {
	if strings.TrimSpace(m.llmThinking) == "" {
		return joinInspectorLines([]string{i18n.T("thinking.unavailable")}, width)
	}
	lines := []string{i18n.T("thinking.text")}
	lines = append(lines, clipMultiline(m.llmThinking, 16, width)...)
	return joinInspectorLines(lines, width)
}

func clipMultiline(text string, maxLines, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{"(empty)"}
	}
	lines := strings.Split(text, "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = append(lines[:maxLines], "... (truncated)")
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, trimInspectorLine(line, width))
	}
	return out
}

func trimInspectorLine(s string, width int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\t", "  "))
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func joinInspectorLines(lines []string, width int) string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		out = append(out, trimInspectorLine(line, width))
	}
	return strings.Join(out, "\n")
}

func valueOr(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func workflowStageIndex(stage workflowStage) int {
	switch stage {
	case workflowPerceive:
		return 0
	case workflowAnalyze:
		return 1
	case workflowSuggest:
		return 2
	case workflowConfirm:
		return 3
	case workflowExecute:
		return 4
	default:
		return -1
	}
}

func (m Model) latestLogEntries(limit int) []oplog.Entry {
	if limit <= 0 || m.opLog == nil {
		return nil
	}
	entries := m.opLog.Entries()
	if len(entries) == 0 {
		return nil
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries
}

func (m Model) latestResultEntries(limit int) []oplog.Entry {
	if limit <= 0 || m.opLog == nil {
		return nil
	}
	entries := m.opLog.Entries()
	if len(entries) == 0 {
		return nil
	}
	var filtered []oplog.Entry
	for _, entry := range entries {
		switch entry.Type {
		case oplog.EntryCmdSuccess, oplog.EntryCmdFail, oplog.EntryUserAction:
			filtered = append(filtered, entry)
		}
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered
}
