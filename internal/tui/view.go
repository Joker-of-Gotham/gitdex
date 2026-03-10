package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

func (m Model) View() tea.View {
	var v tea.View
	if !m.ready {
		v = tea.NewView(i18n.T("ui.loading"))
	} else {
		switch m.screen {
		case screenLanguageSelect:
			v = tea.NewView(m.renderLanguageSelectScreen())
		case screenModelSelect:
			v = tea.NewView(m.renderModelSelectScreen())
		case screenInput:
			v = tea.NewView(m.renderInputScreen())
		case screenGoalInput:
			v = tea.NewView(m.renderGoalInputScreen())
		case screenWorkflowSelect:
			v = tea.NewView(m.renderWorkflowSelectScreen())
		default:
			header := m.renderHeader()
			statusBar := m.renderStatusBar()
			actionBar := m.renderActionBar()
			usedHeight := 3
			contentHeight := m.height - usedHeight
			if contentHeight < 1 {
				contentHeight = 1
			}
			content := m.renderMainContent(contentHeight)
			v = tea.NewView(header + "\n" + content + "\n" + statusBar + "\n" + actionBar)
		}
	}
	v.AltScreen = true
	return v
}

func (m Model) renderLanguageSelectScreen() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		MarginBottom(1)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginBottom(1)
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Background(lipgloss.Color("236")).
		PaddingLeft(1).
		PaddingRight(1)
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(1).
		PaddingRight(1)
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	title := i18n.T("language_select.title")
	if m.startupInfo.FirstRun {
		title = i18n.T("language_select.first_run_title")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(i18n.T("language_select.hint")))
	b.WriteString("\n\n")

	for i, option := range languageOptions() {
		cursor := "  "
		style := normalStyle
		if i == m.languageCursor {
			cursor = "> "
			style = activeStyle
		}
		b.WriteString(cursor + style.Render(option.Label) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(detailStyle.Render(fmt.Sprintf(i18n.T("language_select.current"), m.currentLanguageOptionLabel())))
	b.WriteString("\n")
	b.WriteString(detailStyle.Render(fmt.Sprintf(i18n.T("language_select.system"), m.startupInfo.SystemLang)))

	return m.padContent(b.String(), m.height)
}

func (m Model) renderModelSelectScreen() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		MarginBottom(1)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginBottom(1)
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Background(lipgloss.Color("236")).
		PaddingLeft(1).
		PaddingRight(1)
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(1).
		PaddingRight(1)
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	title := i18n.T("model_select.title")
	hint := i18n.T("model_select.hint")
	if m.modelSelectPhase == selectSecondary {
		title = i18n.T("model_select.secondary_title")
		hint = i18n.T("model_select.secondary_hint")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(hint))
	b.WriteString("\n\n")

	for i, model := range m.availModels {
		cursor := "  "
		style := normalStyle
		if i == m.modelCursor {
			cursor = "> "
			style = activeStyle
		}

		name := style.Render(model.Name)
		var details []string
		if model.ParamSize != "" {
			details = append(details, model.ParamSize)
		}
		if model.Family != "" {
			details = append(details, model.Family)
		}
		if model.Size > 0 {
			details = append(details, humanSize(model.Size))
		}
		if model.Quant != "" {
			details = append(details, model.Quant)
		}

		detailStr := ""
		if len(details) > 0 {
			detailStr = detailStyle.Render("  " + strings.Join(details, " | "))
		}
		b.WriteString(cursor + name + detailStr + "\n")
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(fmt.Sprintf("  "+i18n.T("model_select.found"), len(m.availModels))))

	return m.padContent(b.String(), m.height)
}

func (m Model) renderInputScreen() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).MarginBottom(1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	activeBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(0, 1).
		Width(60)
	inactiveBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(60)

	var b strings.Builder
	action := ""
	if m.inputSuggRef != nil {
		action = m.inputSuggRef.Action
	}
	b.WriteString(titleStyle.Render(action))
	b.WriteString("\n\n")

	if m.inputSuggRef != nil {
		previewArgs := make([]string, len(m.inputSuggRef.Command))
		copy(previewArgs, m.inputSuggRef.Command)
		for i, field := range m.inputFields {
			val := m.inputValues[i]
			if val != "" && field.ArgIndex >= 0 && field.ArgIndex < len(previewArgs) {
				previewArgs[field.ArgIndex] = val
			} else if val != "" && field.Key != "" {
				for j := range previewArgs {
					previewArgs[j] = strings.ReplaceAll(previewArgs[j], field.Key, val)
				}
			}
		}
		cmdPreview := joinCmd(previewArgs)
		b.WriteString(hintStyle.Render("  "+fmt.Sprintf(i18n.T("input.preview"), cmdPreview)) + "\n\n")
	}

	for i, field := range m.inputFields {
		isActive := i == m.inputIdx
		b.WriteString(labelStyle.Render("  "+field.Label+":") + "\n")

		val := m.inputValues[i]
		display := val
		if display == "" && !isActive {
			display = field.Placeholder
		}

		if isActive {
			before := val[:m.inputCursorAt]
			after := val[m.inputCursorAt:]
			cursor := "|"
			if display == "" {
				display = cursor + hintStyle.Render(field.Placeholder)
			} else {
				display = before + cursor + after
			}
			b.WriteString("  " + activeBoxStyle.Render(display) + "\n")
		} else {
			if val == "" {
				display = hintStyle.Render(field.Placeholder)
			}
			b.WriteString("  " + inactiveBoxStyle.Render(display) + "\n")
		}
		b.WriteString("\n")
	}

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	footer := i18n.T("input.footer")
	if len(m.inputFields) > 1 {
		footer += "  " + i18n.T("input.footer_tab")
	}
	footer += "  " + i18n.T("input.footer_paste")
	b.WriteString(footerStyle.Render("  " + footer))

	return m.padContent(b.String(), m.height)
}

func (m Model) renderGoalInputScreen() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).MarginBottom(1)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(0, 1).
		Width(72)

	val := m.goalInput
	before := val
	after := ""
	if m.goalCursorAt >= 0 && m.goalCursorAt <= len(val) {
		before = val[:m.goalCursorAt]
		after = val[m.goalCursorAt:]
	}
	cursor := "|"
	display := before + cursor + after
	if strings.TrimSpace(val) == "" {
		display = cursor + hintStyle.Render(i18n.T("goal.placeholder"))
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("goal.title")))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render(i18n.T("goal.hint")))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(display))
	return m.padContent(b.String(), m.height)
}

func (m Model) renderWorkflowSelectScreen() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).MarginBottom(1)
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginBottom(1)
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Background(lipgloss.Color("236")).
		PaddingLeft(1).
		PaddingRight(1)
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(1).
		PaddingRight(1)
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("workflow_menu.title")))
	b.WriteString("\n")
	b.WriteString(subStyle.Render(i18n.T("workflow_menu.hint")))
	b.WriteString("\n\n")

	for i, wf := range m.workflows {
		cursor := "  "
		style := normalStyle
		if i == m.workflowCursor {
			cursor = "> "
			style = activeStyle
		}
		b.WriteString(cursor + style.Render(wf.Label) + "\n")
		b.WriteString(detailStyle.Render("   "+wf.Goal) + "\n")
		if len(wf.Prerequisites) > 0 {
			b.WriteString(detailStyle.Render("   prerequisites: "+strings.Join(wf.Prerequisites, ", ")) + "\n")
		}
		b.WriteString("\n")
	}

	return m.padContent(b.String(), m.height)
}

func (m Model) renderHeader() string {
	branchInfo := ""
	if m.gitState != nil {
		b := m.gitState.LocalBranch
		branchInfo = fmt.Sprintf("  branch:%s", b.Name)
		if b.Ahead > 0 || b.Behind > 0 {
			branchInfo += fmt.Sprintf("  ahead:%d behind:%d", b.Ahead, b.Behind)
		}

		var parts []string
		untrackedCount, modifiedCount := 0, 0
		for _, f := range m.gitState.WorkingTree {
			if f.WorktreeCode == git.StatusUntracked {
				untrackedCount++
			} else {
				modifiedCount++
			}
		}
		stagedCount := len(m.gitState.StagingArea)
		if modifiedCount > 0 {
			parts = append(parts, fmt.Sprintf(i18n.T("header.modified"), modifiedCount))
		}
		if untrackedCount > 0 {
			parts = append(parts, fmt.Sprintf(i18n.T("header.untracked"), untrackedCount))
		}
		if stagedCount > 0 {
			parts = append(parts, fmt.Sprintf(i18n.T("header.staged"), stagedCount))
		}
		stashCount := len(m.gitState.StashStack)
		if stashCount > 0 {
			parts = append(parts, fmt.Sprintf(i18n.T("header.stash"), stashCount))
		}
		if len(parts) > 0 {
			branchInfo += "  " + strings.Join(parts, " | ")
		}
	}

	title := i18n.T("app.name") + branchInfo
	if theme.Current != nil {
		return theme.Current.Header.Width(m.width).Render(title)
	}
	return title
}

func (m Model) renderStatusBar() string {
	msg := m.statusMsg
	if msg == "" && m.startupInfo.GitAvailable {
		msg = fmt.Sprintf(
			i18n.T("status_bar.format"),
			m.startupInfo.GitVersion,
			m.startupInfo.OllamaStatus,
			m.currentResolvedLanguage(),
			m.mode,
		)
		if modelInfo := m.renderModelSummary(); modelInfo != "" {
			msg += " | " + modelInfo
		}
	}
	if msg == "" {
		msg = i18n.T("ui.ready")
	}
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(m.width)
	return style.Render(msg)
}

func (m Model) renderActionBar() string {
	bar := i18n.T("action_bar.main")
	bar += "  g:goal  f:flow  L:lang  o/O:inspect"
	if m.llmThinking != "" {
		bar += "  " + i18n.T("action_bar.thinking")
	}
	if m.logExpanded {
		bar += "  " + i18n.T("action_bar.log_scroll")
	}
	if theme.Current != nil {
		return theme.Current.ActionBar.Width(m.width).Render(bar)
	}
	return bar
}

func (m Model) renderModelSummary() string {
	if strings.TrimSpace(m.selectedPrimary) == "" {
		return ""
	}
	if m.secondaryEnabled && strings.TrimSpace(m.selectedSecondary) != "" {
		return fmt.Sprintf(i18n.T("model_select.summary_on"), m.selectedPrimary, m.selectedSecondary)
	}
	return fmt.Sprintf(i18n.T("model_select.summary_off"), m.selectedPrimary)
}

func (m Model) renderMainContent(height int) string {
	if m.gitState == nil {
		welcome := i18n.T("welcome.title") + "\n\n"
		if m.startupInfo.GitAvailable {
			welcome += fmt.Sprintf(i18n.T("welcome.git_ok"), m.startupInfo.GitVersion) + "\n"
		} else {
			welcome += i18n.T("welcome.git_missing") + "\n"
		}
		welcome += fmt.Sprintf(i18n.T("welcome.ollama_status"), m.startupInfo.OllamaStatus) + "\n"
		welcome += "\n" + i18n.T("welcome.waiting")
		return m.padContent(welcome, height)
	}

	if m.width < 80 {
		var parts []string
		parts = append(parts, m.renderCompactAreasSummary())
		parts = append(parts, m.renderLeftColumn(m.width, height))
		parts = append(parts, m.renderObservabilityPanel(m.width))
		return m.padContent(strings.Join(parts, "\n\n"), height)
	}

	gap := 1
	rightWidth := m.width * 28 / 100
	if rightWidth < 26 {
		rightWidth = 26
	}
	if rightWidth > m.width-32 {
		rightWidth = m.width - 32
	}
	leftWidth := m.width - gap - rightWidth
	if leftWidth < 40 {
		return m.padContent(m.renderLeftColumn(m.width, height), height)
	}

	left := lipgloss.NewStyle().Width(leftWidth).Render(m.renderLeftColumn(leftWidth, height))
	right := lipgloss.NewStyle().Width(rightWidth).Render(m.renderRightColumn(rightWidth, height))
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	return m.padContent(content, height)
}

func (m Model) renderLeftColumn(width, _ int) string {
	var sections []string

	if m.llmAnalysis != "" {
		sections = append(sections, m.renderAnalysisPanel(width))
	}
	if len(m.gitState.WorkingTree) > 0 || len(m.gitState.StagingArea) > 0 {
		sections = append(sections, m.renderFileStatus())
	}

	if len(m.suggestions) > 0 {
		sections = append(sections, m.renderSuggestionCards(width))
	} else if m.llmProvider == nil {
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		sections = append(sections, hintStyle.Render(i18n.T("analysis.no_ollama")))
	} else if !analysisInProgress(m.llmAnalysis) {
		if analysisHasFailure(m.llmAnalysis) {
			warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
			sections = append(sections, warnStyle.Render(i18n.T("analysis.no_suggestions")))
		} else if repositoryLooksClean(m.gitState) {
			cleanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
			sections = append(sections, cleanStyle.Render(i18n.T("analysis.clean")))
		}
	}

	if m.llmThinking != "" && m.expanded {
		sections = append(sections, m.renderThinkingPanel(width))
	}

	sections = append(sections, m.renderOperationLogPanel(width))
	return strings.Join(sections, "\n\n")
}

func (m Model) renderRightColumn(width, height int) string {
	content := strings.Join([]string{
		m.renderAreasTreePanel(width),
		m.renderObservabilityPanel(width),
	}, "\n\n")
	return m.padContent(content, height)
}

func (m Model) renderCompactAreasSummary() string {
	if m.gitState == nil {
		return ""
	}
	ahead, behind := m.gitState.LocalBranch.Ahead, m.gitState.LocalBranch.Behind
	if m.gitState.UpstreamState != nil {
		ahead = m.gitState.UpstreamState.Ahead
		behind = m.gitState.UpstreamState.Behind
	}
	summary := fmt.Sprintf(
		"[areas] wd:%d | stage:%d | branch:%s | ahead:%d behind:%d | remotes:%d",
		len(m.gitState.WorkingTree),
		len(m.gitState.StagingArea),
		m.gitState.LocalBranch.Name,
		ahead,
		behind,
		len(m.gitState.RemoteInfos),
	)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(summary)
}

func (m Model) renderOperationLogPanel(width int) string {
	if width < 20 {
		width = 20
	}
	panelWidth := width - 2
	if panelWidth < 20 {
		panelWidth = 20
	}

	lineBudget := 5
	if m.logExpanded {
		lineBudget = 15
	}
	logBody := i18n.T("oplog.empty")
	if m.opLog != nil {
		logBody = m.opLog.ViewWithOffset(panelWidth-4, lineBudget, m.logScrollOffset)
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	modeLabel := i18n.T("oplog.collapsed")
	if m.logExpanded {
		modeLabel = i18n.T("oplog.expanded")
	}
	hints := i18n.T("oplog.toggle")
	if m.logExpanded {
		hints += "  " + i18n.T("oplog.scroll")
	}

	content := strings.Join([]string{
		headerStyle.Render(i18n.T("oplog.title") + " (" + modeLabel + ")"),
		logBody,
		hintStyle.Render(hints),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(0, 1).
		Width(panelWidth).
		Render(content)
}

func (m Model) renderAreasTreePanel(width int) string {
	if width < 24 {
		width = 24
	}
	panelWidth := width - 2
	if panelWidth < 24 {
		panelWidth = 24
	}
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	tree := components.NewAreasTree(m.gitState).
		SetWidth(panelWidth - 4).
		SetMaxItems(3).
		View()
	content := headerStyle.Render(i18n.T("areas.title")) + "\n" + tree
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(panelWidth).
		Render(content)
}

func (m Model) renderAnalysisPanel(width int) string {
	panelWidth := width
	if panelWidth <= 0 {
		panelWidth = m.width
	}
	if panelWidth < 20 {
		panelWidth = 20
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(panelWidth - 2)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var parts []string
	parts = append(parts, headerStyle.Render(i18n.T("analysis.title")))

	if m.gitState != nil && !analysisInProgress(m.llmAnalysis) {
		if ctx := m.renderContextSummary(); ctx != "" {
			parts = append(parts, dimStyle.Render(ctx))
		}
	}
	if goal := strings.TrimSpace(m.session.ActiveGoal); goal != "" {
		goalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		statusStr := ""
		if s := strings.TrimSpace(m.llmGoalStatus); s != "" {
			statusStr = " (" + s + ")"
		}
		parts = append(parts, goalStyle.Render("Goal: "+goal+statusStr))
	}
	if m.llmDebugInfo != "" && !analysisInProgress(m.llmAnalysis) {
		parts = append(parts, dimStyle.Render("[ctx] "+m.llmDebugInfo))
	}

	parts = append(parts, textStyle.Render(m.llmAnalysis))
	if strings.TrimSpace(m.llmPlanOverview) != "" {
		parts = append(parts, dimStyle.Render("Plan: "+m.llmPlanOverview))
	}

	return borderStyle.Render(strings.Join(parts, "\n"))
}

func (m Model) renderContextSummary() string {
	if m.gitState == nil {
		return ""
	}
	s := m.gitState
	var items []string
	items = append(items, fmt.Sprintf("branch:%s", s.LocalBranch.Name))
	items = append(items, fmt.Sprintf("commits:%d", s.CommitCount))
	if len(s.WorkingTree) > 0 {
		items = append(items, fmt.Sprintf("changes:%d", len(s.WorkingTree)))
	}
	if len(s.StagingArea) > 0 {
		items = append(items, fmt.Sprintf("staged:%d", len(s.StagingArea)))
	}
	for _, r := range s.RemoteInfos {
		tag := "valid"
		if !r.FetchURLValid && !r.PushURLValid {
			tag = "invalid"
		}
		items = append(items, fmt.Sprintf("remote:%s(%s)", r.Name, tag))
	}
	if len(s.Remotes) == 0 {
		items = append(items, "remote:none")
	}
	return "[" + strings.Join(items, " | ") + "]"
}

func (m Model) renderThinkingPanel(width int) string {
	panelWidth := width
	if panelWidth <= 0 {
		panelWidth = m.width
	}
	if panelWidth < 20 {
		panelWidth = 20
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(0, 1).
		Width(panelWidth - 2)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241"))
	thinkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)

	thinking := m.llmThinking
	lines := strings.Split(thinking, "\n")
	if len(lines) > 8 {
		lines = append(lines[:8], i18n.T("thinking.truncated"))
	}
	thinking = strings.Join(lines, "\n")

	content := headerStyle.Render(i18n.T("thinking.title")) + "\n" + thinkStyle.Render(thinking)
	return borderStyle.Render(content)
}

func (m Model) renderFileStatus() string {
	var lines []string

	if len(m.gitState.StagingArea) > 0 {
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
		lines = append(lines, headerStyle.Render(fmt.Sprintf(i18n.T("file_status.staged"), len(m.gitState.StagingArea))))
		for _, f := range m.gitState.StagingArea {
			icon := statusIcon(f.StagingCode)
			style := statusStyle(f.StagingCode)
			lines = append(lines, "  "+style.Render(icon+" "+f.Path))
		}
	}
	if len(m.gitState.WorkingTree) > 0 {
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
		lines = append(lines, headerStyle.Render(fmt.Sprintf(i18n.T("file_status.changes"), len(m.gitState.WorkingTree))))
		for _, f := range m.gitState.WorkingTree {
			icon := statusIcon(f.WorktreeCode)
			style := statusStyle(f.WorktreeCode)
			lines = append(lines, "  "+style.Render(icon+" "+f.Path))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderSuggestionCards(width int) string {
	cardWidth := width
	if cardWidth <= 0 {
		cardWidth = m.width
	}
	if cardWidth < 30 {
		cardWidth = 30
	}

	var cards []string
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

	doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for i, s := range m.suggestions {
		isActive := i == m.suggIdx
		var execState git.ExecState
		var execMsg string
		if i < len(m.suggExecState) {
			execState = m.suggExecState[i]
			execMsg = m.suggExecMsg[i]
		}

		reason := s.Reason
		if isActive && m.llmReason != "" {
			reason = m.llmReason
		}

		action := s.Action
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

		var cmdDisplay []string
		switch s.Interaction {
		case git.InfoOnly:
			cmdDisplay = []string{i18n.T("suggestions.advisory")}
		case git.FileWrite:
			if s.FileOp != nil {
				op := strings.TrimSpace(s.FileOp.Operation)
				if op == "" {
					op = "create"
				}
				cmdDisplay = []string{op + " " + s.FileOp.Path}
			} else {
				cmdDisplay = []string{i18n.T("suggestions.file_write")}
			}
		case git.NeedsInput:
			cmdDisplay = []string{joinCmd(s.Command) + "  " + i18n.T("suggestions.fill_in")}
		default:
			cmdDisplay = []string{joinCmd(s.Command)}
		}
		if execMsg != "" && execState != git.ExecPending {
			cmdDisplay = append(cmdDisplay, execMsg)
		}

		card := components.NewSuggestionCard(action, reason, cmdDisplay, riskLabel(s.RiskLevel))
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
			if isActive {
				marker := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
				rendered = marker.Render("[..] ") + rendered
			} else {
				rendered = "    " + rendered
			}
		default:
			if isActive {
				marker := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
				rendered = marker.Render("> ") + rendered
			} else {
				rendered = "  " + rendered
			}
		}
		cards = append(cards, rendered)
	}

	return strings.Join(cards, "\n")
}

func (m Model) padContent(content string, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func analysisInProgress(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for _, candidate := range []string{
		i18n.T("analysis.analyzing"),
		i18n.T("analysis.analyzing_repo"),
		i18n.T("analysis.reanalyzing"),
		i18n.T("analysis.in_progress_status"),
	} {
		if text == strings.TrimSpace(candidate) {
			return true
		}
	}
	return false
}

func analysisHasFailure(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	errorPrefix := strings.ToLower(strings.TrimSpace(strings.Split(i18n.T("analysis.error_prefix"), "%s")[0]))
	return strings.Contains(lower, errorPrefix) ||
		strings.Contains(lower, "ai error:") ||
		strings.Contains(lower, "invalid structured response") ||
		strings.Contains(lower, "could not be parsed") ||
		strings.Contains(lower, "empty response")
}

func repositoryLooksClean(state *status.GitState) bool {
	if state == nil {
		return false
	}
	if len(state.WorkingTree) > 0 || len(state.StagingArea) > 0 {
		return false
	}
	if state.MergeInProgress || state.RebaseInProgress || state.CherryInProgress || state.BisectInProgress {
		return false
	}
	ahead, behind := state.LocalBranch.Ahead, state.LocalBranch.Behind
	if state.UpstreamState != nil {
		ahead = state.UpstreamState.Ahead
		behind = state.UpstreamState.Behind
	}
	if ahead > 0 || behind > 0 {
		return false
	}
	for _, remote := range state.RemoteInfos {
		if remote.FetchURL != "" && !remote.FetchURLValid {
			return false
		}
		if remote.PushURL != "" && !remote.PushURLValid {
			return false
		}
		if remote.ReachabilityChecked && !remote.Reachable {
			return false
		}
	}
	return true
}

func statusIcon(code git.FileStatusCode) string {
	switch code {
	case git.StatusAdded:
		return "+"
	case git.StatusModified:
		return "~"
	case git.StatusDeleted:
		return "-"
	case git.StatusRenamed:
		return ">"
	case git.StatusUntracked:
		return "?"
	default:
		return " "
	}
}

func statusStyle(code git.FileStatusCode) lipgloss.Style {
	switch code {
	case git.StatusAdded:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	case git.StatusModified:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	case git.StatusDeleted:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	case git.StatusUntracked:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	default:
		return lipgloss.NewStyle()
	}
}
