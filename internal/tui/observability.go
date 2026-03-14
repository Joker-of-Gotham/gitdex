package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

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
	Title                string
	Status               string
	Output               string
	At                   time.Time
	Command              []string
	ResultKind           string
	FilePath             string
	FileOperation        string
	BeforeContent        string
	AfterContent         string
	PlatformCapability   string
	PlatformFlow         string
	PlatformOperation    string
	PlatformResourceID   string
	PlatformAdapter      string
	PlatformRollback     string
	PlatformBoundary     string
	PlatformLedgerID     string
	PlatformCompensation string
	PlatformApproval     bool
	PlatformInspect      json.RawMessage
	PlatformBefore       json.RawMessage
	PlatformAfter        json.RawMessage
	PlatformSnapshot     json.RawMessage
}

type tabSegment struct {
	label    string
	rendered string
	tab      observabilityTab
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

func (m *Model) rememberOperationEvent(event string) {
	event = strings.TrimSpace(event)
	if m.memoryStore == nil || event == "" {
		return
	}
	m.memoryStore.RecordOperationEvent(m.repoFingerprint(), event)
	_ = m.memoryStore.Save()
}

func (m *Model) rememberArtifactNote(note string) {
	note = strings.TrimSpace(note)
	if m.memoryStore == nil || note == "" {
		return
	}
	m.memoryStore.RecordArtifactNote(m.repoFingerprint(), note)
	_ = m.memoryStore.Save()
}

func (m *Model) syncTaskMemory() {
	if m.memoryStore == nil {
		return
	}
	workflowID := ""
	if m.workflowPlan != nil {
		workflowID = strings.TrimSpace(firstNonEmpty(m.workflowPlan.WorkflowID, m.workflowPlan.WorkflowLabel))
	}
	status := m.currentGoalStatus()
	if status == "" {
		status = string(m.workflowStage)
	}
	var constraints []string
	if m.workflowPlan != nil {
		for _, capability := range m.workflowPlan.Capabilities {
			constraints = append(constraints, "capability:"+strings.TrimSpace(capability))
		}
	}
	var pending []string
	if m.workflowFlow != nil {
		for _, step := range m.workflowFlow.Steps {
			switch step.Status {
			case workflowFlowDone, workflowFlowCompensated, workflowFlowSkipped:
				continue
			default:
				pending = append(pending, strings.TrimSpace(step.Step.Title)+" ["+string(step.Status)+"]")
			}
		}
	}
	m.memoryStore.UpdateTaskState(m.repoFingerprint(), m.session.ActiveGoal, workflowID, status, constraints, pending)
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

func (m Model) observabilityBody(width int) string {
	body := m.renderWorkflowInspector(width)
	switch m.obsTab {
	case observabilityTimeline:
		body = m.renderTimelineInspector(width)
	case observabilityContext:
		body = m.renderContextInspector(width)
	case observabilityMemory:
		body = m.renderMemoryInspector(width)
	case observabilityRaw:
		body = m.renderRawInspector(width)
	case observabilityCommand:
		body = m.renderCommandInspector(width)
	case observabilityThinking:
		body = m.renderThinkingInspector(width)
	}
	return body
}

func (m Model) renderObservabilityPanel(width, height int) string {
	content, _ := m.renderObservabilityPanelWithRegions(width, height)
	return content
}

func (m Model) renderObservabilityPanelWithRegions(width, height int) (string, []clickRegion) {
	if width < 18 {
		width = 18
	}
	if height < 7 {
		height = 7
	}
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6FC3DF")).
		Padding(0, 1)
	innerWidth, innerHeight := panelInnerSize(panelStyle, width, height)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6FC3DF"))
	tabOn := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F2C572"))
	tabOff := lipgloss.NewStyle().Foreground(lipgloss.Color("#9FB4C4"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99"))

	fullBody := m.observabilityBody(innerWidth)
	chromeHeight := 6
	bodyHeight := innerHeight - chromeHeight
	if bodyHeight < 4 {
		bodyHeight = 4
	}
	totalLines := lineCount(fullBody)
	body := sliceVisibleLines(fullBody, bodyHeight, m.obsScroll)
	maxPage := 1
	if totalLines > bodyHeight {
		maxPage = totalLines - bodyHeight + 1
	}
	scrollHint := hintStyle.Render(fmt.Sprintf("scroll %d/%d", m.obsScroll+1, maxPage))
	tabsLine, tabRegions := m.renderObservabilityTabsWithRegions(innerWidth, tabOn, tabOff, hintStyle)

	content := strings.Join([]string{
		headerStyle.Render(i18n.T("observability.title")),
		tabsLine,
		body,
		scrollHint + "  " + hintStyle.Render(i18n.T("observability.hint")),
	}, "\n\n")

	for i := range tabRegions {
		tabRegions[i].x0 += 2
		tabRegions[i].x1 += 2
		tabRegions[i].y0 += 3
		tabRegions[i].y1 += 3
	}
	return renderBoundedPanel(panelStyle, width, height, content), tabRegions
}

func (m Model) renderObservabilityTabs(width int, activeStyle, idleStyle, hintStyle lipgloss.Style) string {
	line, _ := m.renderObservabilityTabsWithRegions(width, activeStyle, idleStyle, hintStyle)
	return line
}

func (m Model) renderObservabilityTabsWithRegions(width int, activeStyle, idleStyle, hintStyle lipgloss.Style) (string, []clickRegion) {
	tabs := observabilityTabs()
	segments := make([]tabSegment, 0, len(tabs))
	current := 0
	for i, tab := range tabs {
		style := idleStyle
		if tab == m.obsTab {
			style = activeStyle
			current = i
		}
		label := tab.label()
		if tab == m.obsTab {
			label = "[" + label + "]"
		}
		rendered := style.Render(label)
		segments = append(segments, tabSegment{
			label:    label,
			rendered: rendered,
			tab:      tab,
		})
	}

	fullLine, fullRegions := renderTabWindowWithRegions(segments, 0, len(segments)-1, hintStyle)
	if lipgloss.Width(fullLine) <= width {
		return fullLine, fullRegions
	}

	bestLine := activeStyle.Render(segments[current].label)
	bestRegions := []clickRegion{{
		action: "observability_tab",
		index:  int(segments[current].tab),
		x0:     0,
		y0:     0,
		x1:     lipgloss.Width(bestLine),
		y1:     1,
	}}
	bestCount := 1
	bestWidth := lipgloss.Width(bestLine)
	for start := 0; start <= current; start++ {
		for end := current; end < len(segments); end++ {
			line, regions := renderTabWindowWithRegions(segments, start, end, hintStyle)
			lineWidth := lipgloss.Width(line)
			if lineWidth > width {
				continue
			}
			count := end - start + 1
			if count > bestCount || (count == bestCount && lineWidth > bestWidth) {
				bestLine = line
				bestRegions = regions
				bestCount = count
				bestWidth = lineWidth
			}
		}
	}

	if bestWidth <= width {
		return bestLine, bestRegions
	}

	label := segments[current].label
	if width > 0 {
		label = runewidth.Truncate(label, maxInt(1, width-2), "")
	}
	line := activeStyle.Render(label)
	return line, []clickRegion{{
		action: "observability_tab",
		index:  int(segments[current].tab),
		x0:     0,
		y0:     0,
		x1:     lipgloss.Width(line),
		y1:     1,
	}}
}

func renderTabWindow(segments []tabSegment, start, end int, hintStyle lipgloss.Style) string {
	line, _ := renderTabWindowWithRegions(segments, start, end, hintStyle)
	return line
}

func renderTabWindowWithRegions(segments []tabSegment, start, end int, hintStyle lipgloss.Style) (string, []clickRegion) {
	parts := make([]string, 0, end-start+3)
	regions := make([]clickRegion, 0, end-start+1)
	x := 0
	appendPart := func(part string, region *clickRegion) {
		if len(parts) > 0 {
			x++
		}
		startX := x
		parts = append(parts, part)
		x += lipgloss.Width(part)
		if region != nil {
			region.x0 = startX
			region.x1 = x
			regions = append(regions, *region)
		}
	}
	if start > 0 {
		appendPart(hintStyle.Render("<"), nil)
	}
	for idx := start; idx <= end; idx++ {
		appendPart(segments[idx].rendered, &clickRegion{
			action: "observability_tab",
			index:  int(segments[idx].tab),
			y0:     0,
			y1:     1,
		})
	}
	if end < len(segments)-1 {
		appendPart(hintStyle.Render(">"), nil)
	}
	return strings.Join(parts, " "), regions
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
		workflowPerceive: i18n.T("observability.workflow_step_perceive"),
		workflowAnalyze:  i18n.T("observability.workflow_step_analyze"),
		workflowSuggest:  i18n.T("observability.workflow_step_suggest"),
		workflowConfirm:  i18n.T("observability.workflow_step_confirm"),
		workflowExecute:  i18n.T("observability.workflow_step_execute"),
	}

	stageLabel := valueOr(labels[m.workflowStage], i18n.T("observability.workflow_waiting"))
	lines := renderWrappedField(i18n.T("observability.workflow_current_label"), keyStyle(), stageLabel, statusStyleForText(stageLabel), width)
	if m.automation.Enabled || strings.TrimSpace(m.automation.Mode) != "" {
		mode := localizedAutomationModeLabel(m.automationMode())
		if m.automationObserveOnly {
			mode += localizedText(" (observe-only)", "（仅观察）", " (observe-only)")
		}
		lines = append(lines, renderWrappedField("Automation: ", keyStyle(), fmt.Sprintf("%s every %ds", mode, m.automation.MonitorInterval), infoStyle(), width)...)
		if len(m.automation.Schedules) > 0 {
			lines = append(lines, renderWrappedField("Schedules: ", keyStyle(), fmt.Sprintf("%d configured", len(m.automation.Schedules)), infoStyle(), width)...)
		}
		if len(m.automation.MaintenanceWindows) > 0 {
			windows := make([]string, 0, len(m.automation.MaintenanceWindows))
			for _, window := range m.automation.MaintenanceWindows {
				days := strings.Join(window.Days, ",")
				if strings.TrimSpace(days) == "" {
					days = "daily"
				}
				windows = append(windows, fmt.Sprintf("%s %s-%s", days, window.Start, window.End))
			}
			lines = append(lines, renderWrappedField("Windows: ", keyStyle(), strings.Join(windows, " | "), infoStyle(), width)...)
		}
		trustMode := "untrusted"
		if m.automation.TrustedMode {
			trustMode = "trusted"
		}
		trustSummary := trustMode
		if len(m.automation.TrustPolicy.TrustedCapabilities) > 0 {
			trustSummary += " | caps=" + strings.Join(m.automation.TrustPolicy.TrustedCapabilities, ", ")
		}
		if m.automation.TrustPolicy.AllowDangerousGit {
			trustSummary += " | dangerous_git"
		}
		lines = append(lines, renderWrappedField("Trust: ", keyStyle(), trustSummary, infoStyle(), width)...)
		approvalSummary := fmt.Sprintf("partial=%t composed=%t adapter=%t irreversible=%t",
			m.automation.ApprovalPolicy.RequireForPartial,
			m.automation.ApprovalPolicy.RequireForComposed,
			m.automation.ApprovalPolicy.RequireForAdapterBacked,
			m.automation.ApprovalPolicy.RequireForIrreversible,
		)
		lines = append(lines, renderWrappedField("Approval policy: ", keyStyle(), approvalSummary, infoStyle(), width)...)
		lines = append(lines, renderWrappedField("Concurrency: ", keyStyle(), fmt.Sprintf("enabled=%t", m.automation.Concurrency.Enabled), infoStyle(), width)...)
		lines = append(lines, renderWrappedField("Dead-letter policy: ", keyStyle(), fmt.Sprintf("pause_after=%d", m.automation.DeadLetter.PauseAfter), infoStyle(), width)...)
		if len(m.automationLocks) > 0 {
			lines = append(lines, renderWrappedField("Locks: ", keyStyle(), fmt.Sprintf("%d active", len(m.automationLocks)), warnStyle(), width)...)
		}
		if !m.lastEscalation.IsZero() {
			lines = append(lines, renderWrappedField("Escalated: ", keyStyle(), m.lastEscalation.Format("15:04:05"), dangerStyle(), width)...)
		}
		if !m.lastRecovery.IsZero() {
			lines = append(lines, renderWrappedField("Recovered: ", keyStyle(), m.lastRecovery.Format("15:04:05"), successStyle(), width)...)
		}
	}
	currentIdx := workflowStageIndex(m.workflowStage)
	for _, step := range steps {
		idx := workflowStageIndex(step)
		prefix := "[ ]"
		style := mutedStyle()
		switch {
		case currentIdx >= 0 && idx < currentIdx:
			prefix = "[x]"
			style = successStyle()
		case idx == currentIdx:
			prefix = "[>]"
			style = warnStyle()
		}
		lines = append(lines, style.Render(fmt.Sprintf("%s %s", prefix, labels[step])))
	}
	if goal := strings.TrimSpace(m.session.ActiveGoal); goal != "" {
		lines = append(lines, "")
		lines = append(lines, renderWrappedField(i18n.T("observability.workflow_goal_label"), keyStyle(), goal, valueStyle(), width)...)
	}
	if status := strings.TrimSpace(m.llmGoalStatus); status != "" {
		lines = append(lines, renderWrappedField(i18n.T("observability.workflow_goal_status_label"), keyStyle(), status, statusStyleForText(status), width)...)
	}
	if m.workflowPlan != nil && len(m.workflowPlan.Steps) > 0 {
		lines = append(lines, "")
		lines = append(lines, keyStyle().Render("Workflow orchestration"))
		lines = append(lines, renderWrappedField("Template: ", keyStyle(), valueOr(m.workflowPlan.WorkflowLabel, m.workflowPlan.WorkflowID), valueStyle(), width)...)
		if coverage := strings.TrimSpace(m.capabilityCoverageSummary(m.workflowPlan.Capabilities)); coverage != "" {
			lines = append(lines, renderWrappedField("Coverage: ", keyStyle(), coverage, infoStyle(), width)...)
		}
		if m.workflowFlow != nil && len(m.workflowFlow.Steps) > 0 {
			total, pending, running, done, failed := m.workflowFlowCounts()
			if strings.TrimSpace(m.workflowFlow.RunID) != "" {
				lines = append(lines, renderWrappedField("Run: ", keyStyle(), m.workflowFlow.RunID, tsStyle(), width)...)
			}
			lines = append(lines, renderWrappedField("Health: ", keyStyle(), valueOr(m.workflowFlow.Health, "pending"), statusStyleForText(valueOr(m.workflowFlow.Health, "pending")), width)...)
			lines = append(lines, renderWrappedField("Approval: ", keyStyle(), valueOr(m.workflowFlow.ApprovalState, "clear"), statusStyleForText(valueOr(m.workflowFlow.ApprovalState, "clear")), width)...)
			if strings.TrimSpace(m.workflowFlow.ApprovalDetail) != "" {
				lines = append(lines, renderWrappedField("Approval detail: ", keyStyle(), m.workflowFlow.ApprovalDetail, infoStyle(), width)...)
			}
			lines = append(lines, renderWrappedField("Flow: ", keyStyle(), fmt.Sprintf("total=%d pending=%d running=%d done=%d failed=%d", total, pending, running, done, failed), infoStyle(), width)...)
			if !m.workflowFlow.NextRetryAt.IsZero() {
				nextRetry := m.workflowFlow.NextRetryAt.Format("15:04:05")
				if strings.TrimSpace(m.workflowFlow.NextRetryStep) != "" {
					nextRetry += " | " + m.workflowFlow.NextRetryStep
				}
				lines = append(lines, renderWrappedField("Next retry: ", keyStyle(), nextRetry, tsStyle(), width)...)
			}
			if strings.TrimSpace(m.workflowFlow.PausedReason) != "" {
				lines = append(lines, renderWrappedField("Paused: ", keyStyle(), m.workflowFlow.PausedReason, warnStyle(), width)...)
			}
			if len(m.workflowFlow.ActiveLocks) > 0 {
				lockParts := make([]string, 0, len(m.workflowFlow.ActiveLocks))
				for key, owner := range m.workflowFlow.ActiveLocks {
					lockParts = append(lockParts, key+"="+owner)
				}
				sort.Strings(lockParts)
				lines = append(lines, renderWrappedField("Locks: ", keyStyle(), strings.Join(lockParts, "; "), infoStyle(), width)...)
			}
			if len(m.workflowFlow.DeadLetterEntries) > 0 {
				dead := make([]string, 0, len(m.workflowFlow.DeadLetterEntries))
				for _, item := range m.workflowFlow.DeadLetterEntries {
					label := item.Identity
					if item.Acked {
						label += " (acked)"
					}
					if item.Reason != "" {
						label += ": " + item.Reason
					}
					dead = append(dead, label)
				}
				lines = append(lines, renderWrappedField("Dead-letter: ", keyStyle(), strings.Join(dead, " | "), dangerStyle(), width)...)
			}
			ops := "< prev | > next | Y approve | P pause | R resume | X retry | A ack | K skip | C compensate | u clear-lock"
			if m.automationObserveOnly {
				ops += " | H recover-auto"
			}
			lines = append(lines, renderWrappedField("Ops: ", keyStyle(), ops, infoStyle(), width)...)
			for idx, step := range m.workflowFlow.Steps {
				stepStyle := mutedStyle()
				prefix := "[ ]"
				selectedMarker := " "
				if idx == m.workflowFlow.SelectedStepIndex {
					selectedMarker = ">"
				}
				switch step.Status {
				case workflowFlowReady:
					prefix = "[·]"
					stepStyle = valueStyle()
				case workflowFlowSuggested:
					prefix = "[~]"
					stepStyle = warnStyle()
				case workflowFlowRunning:
					prefix = "[>]"
					stepStyle = commandStyle()
				case workflowFlowWaitingValidation:
					prefix = "[v]"
					stepStyle = infoStyle()
				case workflowFlowRetrying:
					prefix = "[r]"
					stepStyle = warnStyle()
				case workflowFlowPaused:
					prefix = "[p]"
					stepStyle = mutedStyle()
				case workflowFlowDone:
					prefix = "[x]"
					stepStyle = successStyle()
				case workflowFlowCompensated:
					prefix = "[c]"
					stepStyle = warnStyle()
				case workflowFlowSkipped:
					prefix = "[-]"
					stepStyle = mutedStyle()
				case workflowFlowFailed:
					prefix = "[!]"
					stepStyle = dangerStyle()
				case workflowFlowDeadLetter:
					prefix = "[!]"
					stepStyle = dangerStyle()
				}
				lines = append(lines, renderWrappedField(fmt.Sprintf(" %s%d. ", selectedMarker, step.Index+1), mutedStyle(), prefix+" "+step.Step.Title, stepStyle, width)...)
				if strings.TrimSpace(step.SuggestionRef) != "" {
					lines = append(lines, renderWrappedField("     linked ", mutedStyle(), step.SuggestionRef, infoStyle(), width)...)
				}
				if strings.TrimSpace(step.LastDetail) != "" {
					lines = append(lines, renderWrappedField("     detail ", mutedStyle(), step.LastDetail, infoStyle(), width)...)
				}
				if !step.NextRetryAt.IsZero() && step.Status == workflowFlowRetrying {
					lines = append(lines, renderWrappedField("     retry  ", mutedStyle(), step.NextRetryAt.Format("15:04:05"), tsStyle(), width)...)
				}
				if step.Policy.TimeoutSecs > 0 || step.Policy.ConcurrencyKey != "" || strings.TrimSpace(step.ApprovalState) != "" {
					policySummary := fmt.Sprintf("timeout=%ds", step.Policy.TimeoutSecs)
					if step.Policy.ConcurrencyKey != "" {
						policySummary += " | lock=" + step.Policy.ConcurrencyKey
						if owner := strings.TrimSpace(m.workflowFlow.ActiveLocks[step.Policy.ConcurrencyKey]); owner != "" {
							policySummary += " | owner=" + owner
						}
					}
					if step.Policy.ApprovalRequired {
						policySummary += " | approval"
					}
					if strings.TrimSpace(step.ApprovalState) != "" {
						policySummary += " | approval_state=" + step.ApprovalState
					}
					lines = append(lines, renderWrappedField("     policy ", mutedStyle(), policySummary, infoStyle(), width)...)
				}
				if len(step.LedgerRefs) > 0 {
					lines = append(lines, renderWrappedField("     ledger ", mutedStyle(), strings.Join(step.LedgerRefs, ", "), tsStyle(), width)...)
				}
			}
		} else {
			for idx, step := range m.workflowPlan.Steps {
				lines = append(lines, renderWrappedField(fmt.Sprintf("  %d. ", idx+1), mutedStyle(), step.Title, commandStyle(), width)...)
				if strings.TrimSpace(step.Rationale) != "" {
					lines = append(lines, renderWrappedField("     ", mutedStyle(), step.Rationale, infoStyle(), width)...)
				}
			}
		}
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
	lines = append(lines, renderWrappedField(i18n.T("observability.workflow_suggestions_label"), keyStyle(), fmt.Sprintf("total=%d pending=%d done=%d", len(m.suggestions), pending, done), infoStyle(), width)...)
	if m.execSuggIdx >= 0 && m.execSuggIdx < len(m.suggestions) {
		lines = append(lines, renderWrappedField(i18n.T("observability.workflow_executing_label"), keyStyle(), m.suggestions[m.execSuggIdx].Action, commandStyle(), width)...)
	}
	if m.lastCommand.Title != "" {
		lines = append(lines, renderWrappedField(i18n.T("observability.workflow_last_result_label"), keyStyle(), m.lastCommand.Status+" | "+m.lastCommand.Title, statusStyleForText(m.lastCommand.Status), width)...)
	}
	if !m.workflowAt.IsZero() {
		lines = append(lines, renderWrappedField(i18n.T("observability.workflow_updated_label"), keyStyle(), m.workflowAt.Format("15:04:05"), tsStyle(), width)...)
	}
	if entries := m.latestLogEntries(4); len(entries) > 0 {
		lines = append(lines, "", keyStyle().Render(i18n.T("observability.workflow_recent_events")))
		for _, entry := range entries {
			prefix := "  " + entry.Timestamp.Format("15:04:05") + " "
			lines = append(lines, renderWrappedField(prefix, tsStyle(), trimInspectorLine(entry.Summary, width-10), eventTypeStyle(entry.Type), width)...)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderTimelineInspector(width int) string {
	lines := []string{keyStyle().Render(i18n.T("observability.timeline_title"))}
	entries := m.latestLogEntries(10)
	if len(entries) == 0 {
		lines = append(lines, mutedStyle().Render(i18n.T("observability.timeline_empty")))
		return strings.Join(lines, "\n")
	}
	for _, entry := range entries {
		prefix := entry.Timestamp.Format("15:04:05") + " " + entry.Icon() + " "
		lines = append(lines, renderWrappedField(prefix, tsStyle(), entry.Summary, eventTypeStyle(entry.Type), width)...)
		if detail := strings.TrimSpace(entry.Detail); detail != "" {
			for _, detailLine := range wrapPlainText(detail, maxInt(8, width-2)) {
				lines = append(lines, infoStyle().Render("  "+detailLine))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderContextInspector(width int) string {
	lines := []string{}
	lines = append(lines, renderWrappedField("Mode: ", keyStyle(), strings.TrimSpace(m.analysisTrace.Mode), valueStyle(), width)...)
	lines = append(lines, renderWrappedField("Models: ", keyStyle(), fmt.Sprintf("%s | verify=%s", valueOr(m.analysisTrace.PrimaryModel, "-"), valueOr(m.analysisTrace.SecondaryModel, "off")), valueStyle(), width)...)
	lines = append(lines, renderWrappedField("Budget: ", keyStyle(), fmt.Sprintf("total=%d reserved=%d usable=%d", m.analysisTrace.Budget, m.analysisTrace.Reserved, m.analysisTrace.Available), infoStyle(), width)...)
	if info := strings.TrimSpace(m.llmDebugInfo); info != "" {
		lines = append(lines, renderWrappedField("Summary: ", keyStyle(), info, infoStyle(), width)...)
	}
	if len(m.analysisTrace.Partitions) > 0 {
		lines = append(lines, "", keyStyle().Render(i18n.T("observability.context_partitions")))
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
			lines = append(lines, infoStyle().Render(fmt.Sprintf("  %s %-16s %4dt%s", mark, part.Name, part.Tokens, extra)))
		}
	}
	if len(m.analysisTrace.Knowledge) > 0 {
		lines = append(lines, "", keyStyle().Render(i18n.T("observability.context_knowledge")))
		for _, item := range m.analysisTrace.Knowledge {
			lines = append(lines, valueStyle().Render("  - "+item.ScenarioID))
		}
	}
	if m.analysisTrace.Workflow != nil && len(m.analysisTrace.Workflow.Steps) > 0 {
		lines = append(lines, "", keyStyle().Render("Workflow orchestration"))
		lines = append(lines, renderWrappedField("Workflow: ", keyStyle(), valueOr(m.analysisTrace.Workflow.WorkflowLabel, m.analysisTrace.Workflow.WorkflowID), valueStyle(), width)...)
		for idx, step := range m.analysisTrace.Workflow.Steps {
			lines = append(lines, renderWrappedField(fmt.Sprintf("  %d. ", idx+1), mutedStyle(), step.Title, commandStyle(), width)...)
		}
	}
	if m.workflowFlow != nil && len(m.workflowFlow.Steps) > 0 {
		lines = append(lines, "", keyStyle().Render("Workflow execution flow"))
		for _, step := range m.workflowFlow.Steps {
			lines = append(lines, renderWrappedField(fmt.Sprintf("  %d. ", step.Index+1), mutedStyle(), string(step.Status)+" | "+step.Step.Title, statusStyleForText(string(step.Status)), width)...)
		}
	}
	if m.analysisTrace.PlatformState != nil {
		if len(m.analysisTrace.PlatformState.Capabilities) > 0 {
			lines = append(lines, "", keyStyle().Render("Platform capabilities"))
			lines = append(lines, renderWrappedField("Count: ", keyStyle(), fmt.Sprintf("%d", len(m.analysisTrace.PlatformState.Capabilities)), valueStyle(), width)...)
		}
		if len(m.analysisTrace.PlatformState.Playbooks) > 0 {
			lines = append(lines, "", keyStyle().Render("Recommended playbooks"))
			for _, playbook := range m.analysisTrace.PlatformState.Playbooks {
				lines = append(lines, renderWrappedField("  "+playbook.ID+": ", keyStyle(), playbook.Label+" ["+playbook.Category+"]", valueStyle(), width)...)
				if len(playbook.Inspect) > 0 {
					lines = append(lines, renderWrappedField("    inspect ", mutedStyle(), playbook.Inspect[0], infoStyle(), width)...)
				}
				if len(playbook.Apply) > 0 {
					lines = append(lines, renderWrappedField("    apply   ", mutedStyle(), playbook.Apply[0], commandStyle(), width)...)
				}
				if len(playbook.Verify) > 0 {
					lines = append(lines, renderWrappedField("    verify  ", mutedStyle(), playbook.Verify[0], successStyle(), width)...)
				}
			}
		}
		if len(m.analysisTrace.PlatformState.SurfaceStates) > 0 {
			lines = append(lines, keyStyle().Render("Admin surfaces"))
			for _, item := range m.analysisTrace.PlatformState.SurfaceStates {
				lines = append(lines, infoStyle().Render("  - "+item))
			}
		}
	}
	if len(m.analysisTrace.RecentOps) > 0 {
		lines = append(lines, "")
		lines = append(lines, renderWrappedField("Recent ops: ", keyStyle(), fmt.Sprintf("%d", len(m.analysisTrace.RecentOps)), valueStyle(), width)...)
	}
	lines = append(lines, "", keyStyle().Render(i18n.T("observability.context_prompt_excerpt")))
	lines = append(lines, clipMultiline(m.analysisTrace.UserPrompt, 6, width)...)
	if system := strings.TrimSpace(m.analysisTrace.SystemPrompt); system != "" {
		lines = append(lines, "", keyStyle().Render(i18n.T("observability.context_system_excerpt")))
		lines = append(lines, clipMultiline(system, 4, width)...)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderMemoryInspector(width int) string {
	lines := renderWrappedField("Repo memory: ", keyStyle(), m.repoFingerprint(), valueStyle(), width)
	if m.memoryStore != nil {
		if path := strings.TrimSpace(m.memoryStore.Path()); path != "" {
			lines = append(lines, renderWrappedField("Path: ", keyStyle(), path, infoStyle(), width)...)
		}
		snapshot := m.memoryStore.Snapshot()
		if !snapshot.UpdatedAt.IsZero() {
			lines = append(lines, renderWrappedField("Updated: ", keyStyle(), snapshot.UpdatedAt.Format("2006-01-02 15:04:05"), tsStyle(), width)...)
		}
	}
	mem := m.currentPromptMemory()
	if mem == nil {
		lines = append(lines, mutedStyle().Render(i18n.T("observability.memory_none")))
	} else {
		if len(mem.UserPreferences) > 0 {
			lines = append(lines, "", keyStyle().Render(i18n.T("observability.memory_preferences")))
			keys := make([]string, 0, len(mem.UserPreferences))
			for key := range mem.UserPreferences {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				lines = append(lines, infoStyle().Render(fmt.Sprintf("  %s=%s", key, mem.UserPreferences[key])))
			}
		}
		if len(mem.RepoPatterns) > 0 {
			lines = append(lines, "", keyStyle().Render(i18n.T("observability.memory_patterns")))
			for _, item := range mem.RepoPatterns {
				lines = append(lines, valueStyle().Render("  - "+item))
			}
		}
		if len(mem.ResolvedIssues) > 0 {
			lines = append(lines, "", keyStyle().Render(i18n.T("observability.memory_resolved")))
			for _, item := range mem.ResolvedIssues {
				lines = append(lines, valueStyle().Render("  - "+item))
			}
		}
		if len(mem.RecentEvents) > 0 {
			lines = append(lines, "", keyStyle().Render("Recent events"))
			for _, item := range mem.RecentEvents {
				lines = append(lines, infoStyle().Render("  - "+item))
			}
		}
		if len(mem.ArtifactNotes) > 0 {
			lines = append(lines, "", keyStyle().Render("Artifact notes"))
			for _, item := range mem.ArtifactNotes {
				lines = append(lines, valueStyle().Render("  - "+item))
			}
		}
		if len(mem.Episodes) > 0 {
			lines = append(lines, "", keyStyle().Render("Episodic memory"))
			for _, item := range mem.Episodes {
				label := strings.TrimSpace(firstNonEmpty(item.Action, item.Kind, item.Surface) + " | " + item.Summary)
				if item.CapabilityID != "" {
					label += " | capability=" + item.CapabilityID
				}
				if item.Flow != "" {
					label += " | flow=" + item.Flow
				}
				if item.Result != "" {
					label += " [" + item.Result + "]"
				}
				lines = append(lines, infoStyle().Render("  - "+label))
			}
		}
		if len(mem.SemanticFacts) > 0 {
			lines = append(lines, "", keyStyle().Render("Semantic memory"))
			for _, item := range mem.SemanticFacts {
				line := fmt.Sprintf("  - %s (confidence %.2f, score %.2f)", item.Fact, item.Confidence, item.CurrentScore)
				if item.Stale {
					line += " [stale]"
				}
				lines = append(lines, valueStyle().Render(line))
			}
		}
		if mem.TaskState != nil {
			lines = append(lines, "", keyStyle().Render("Task memory"))
			lines = append(lines, renderWrappedField("  Goal: ", mutedStyle(), mem.TaskState.Goal, valueStyle(), width)...)
			if mem.TaskState.WorkflowID != "" {
				lines = append(lines, renderWrappedField("  Flow: ", mutedStyle(), mem.TaskState.WorkflowID, infoStyle(), width)...)
			}
			if mem.TaskState.Status != "" {
				lines = append(lines, renderWrappedField("  Status: ", mutedStyle(), mem.TaskState.Status, statusStyleForText(mem.TaskState.Status), width)...)
			}
			if len(mem.TaskState.Pending) > 0 {
				lines = append(lines, renderWrappedField("  Pending: ", mutedStyle(), strings.Join(mem.TaskState.Pending, "; "), infoStyle(), width)...)
			}
		}
	}
	if len(m.session.GoalHistory) > 0 {
		lines = append(lines, "", keyStyle().Render(i18n.T("observability.memory_session_goals")))
		for i := len(m.session.GoalHistory) - 1; i >= 0 && i >= len(m.session.GoalHistory)-5; i-- {
			entry := m.session.GoalHistory[i]
			lines = append(lines, valueStyle().Render(fmt.Sprintf("  - %s [%s]", entry.Goal, entry.Status)))
		}
	}
	if len(m.session.SkippedActions) > 0 {
		lines = append(lines, "", keyStyle().Render(i18n.T("observability.memory_skipped")))
		start := len(m.session.SkippedActions) - 5
		if start < 0 {
			start = 0
		}
		for _, item := range m.session.SkippedActions[start:] {
			lines = append(lines, valueStyle().Render("  - "+item))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderRawInspector(width int) string {
	lines := []string{keyStyle().Render(i18n.T("observability.raw_response"))}
	if raw := strings.TrimSpace(m.analysisTrace.RawResponse); raw != "" {
		lines = append(lines, clipMultiline(raw, 8, width)...)
	} else {
		lines = append(lines, mutedStyle().Render(i18n.T("observability.raw_none")))
	}
	lines = append(lines, "", keyStyle().Render(i18n.T("observability.raw_structured")))
	if cleaned := strings.TrimSpace(m.analysisTrace.CleanedResponse); cleaned != "" {
		lines = append(lines, clipMultiline(cleaned, 8, width)...)
	} else {
		lines = append(lines, mutedStyle().Render(i18n.T("observability.raw_cleaned_none")))
	}
	if len(m.analysisTrace.Rejected) > 0 {
		lines = append(lines, "", keyStyle().Render(i18n.T("observability.raw_rejected")))
		for _, item := range m.analysisTrace.Rejected {
			lines = append(lines, dangerStyle().Render("  - "+item))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderCommandInspector(width int) string {
	lines := []string{}
	if m.lastCommand.Title == "" {
		if req := m.editablePlatformRequest(); req != nil {
			lines = append(lines, renderPlatformRequestPreview(m.detectedPlatform(), req, width)...)
			lines = append(lines, "", infoStyle().Render("Press e to edit and retry this platform request."))
			return strings.Join(lines, "\n")
		}
		lines = append(lines, mutedStyle().Render(i18n.T("observability.result_none")))
		if entries := m.latestResultEntries(5); len(entries) > 0 {
			lines = append(lines, "", keyStyle().Render(i18n.T("observability.result_recent")))
			for _, entry := range entries {
				lines = append(lines, renderWrappedField("  "+entry.Timestamp.Format("15:04:05")+" ", tsStyle(), entry.Summary, eventTypeStyle(entry.Type), width)...)
			}
		}
		return strings.Join(lines, "\n")
	}
	lines = append(lines, renderWrappedField(i18n.T("observability.result_status_label"), keyStyle(), m.lastCommand.Status, statusStyleForText(m.lastCommand.Status), width)...)
	lines = append(lines, renderWrappedField(i18n.T("observability.result_target_label"), keyStyle(), m.lastCommand.Title, commandStyle(), width)...)
	if m.lastCommand.ResultKind == resultKindPlatformAdmin && m.lastPlatform != nil && m.lastPlatform.Mutation != nil {
		lines = append(lines, renderWrappedField("Next: ", keyStyle(), "e edit request | v validate latest mutation | b rollback latest mutation", infoStyle(), width)...)
	} else if m.lastCommand.ResultKind == resultKindPlatformAdmin && m.lastPlatformOp != nil {
		lines = append(lines, renderWrappedField("Next: ", keyStyle(), "e edit request and retry", infoStyle(), width)...)
	}
	if !m.lastCommand.At.IsZero() {
		lines = append(lines, renderWrappedField(i18n.T("observability.result_time_label"), keyStyle(), m.lastCommand.At.Format("15:04:05"), tsStyle(), width)...)
	}
	lines = append(lines, "", keyStyle().Render(i18n.T("observability.result_output")))
	lines = append(lines, m.renderStructuredCommandResult(width)...)
	if entries := m.latestResultEntries(5); len(entries) > 0 {
		lines = append(lines, "", keyStyle().Render(i18n.T("observability.result_recent")))
		for _, entry := range entries {
			lines = append(lines, renderWrappedField("  "+entry.Timestamp.Format("15:04:05")+" ", tsStyle(), entry.Summary, eventTypeStyle(entry.Type), width)...)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderThinkingInspector(width int) string {
	if strings.TrimSpace(m.llmThinking) == "" {
		return strings.Join([]string{mutedStyle().Render(i18n.T("thinking.unavailable"))}, "\n")
	}
	lines := []string{keyStyle().Render(i18n.T("thinking.text"))}
	lines = append(lines, clipMultiline(m.llmThinking, 16, width)...)
	return strings.Join(lines, "\n")
}

func clipMultiline(text string, maxLines, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{"(empty)"}
	}
	return wrapInspectorText(text, width, maxLines)
}

func trimInspectorLine(s string, width int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\t", "  "))
	return s
}

func joinInspectorLines(lines []string, width int) string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrapInspectorText(line, width, 0)...)
	}
	return strings.Join(out, "\n")
}

func wrapInspectorText(text string, width, _ int) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(strings.ReplaceAll(raw, "\t", "  "))
		if line == "" {
			out = append(out, "")
			continue
		}
		if width <= 0 {
			out = append(out, line)
			continue
		}
		wrapped := runewidth.Wrap(line, width)
		out = append(out, strings.Split(wrapped, "\n")...)
	}
	return out
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
