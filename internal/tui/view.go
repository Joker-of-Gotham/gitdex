package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
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
		case screenProviderConfig:
			v = tea.NewView(m.renderProviderConfigScreen())
		case screenAutomationConfig:
			v = tea.NewView(m.renderAutomationConfigScreenV2())
		case screenInput:
			v = tea.NewView(m.renderInputScreen())
		case screenGoalInput:
			v = tea.NewView(m.renderGoalInputScreen())
		case screenWorkflowSelect:
			v = tea.NewView(m.renderWorkflowSelectScreen())
		case screenPlatformEdit:
			v = tea.NewView(m.renderPlatformEditScreen())
		case screenFileEdit:
			v = tea.NewView(m.renderFileEditScreen())
		default:
			header := m.renderHeader()
			statusBar := m.renderStatusBar()
			usedHeight := 2
			contentHeight := m.height - usedHeight
			if contentHeight < 1 {
				contentHeight = 1
			}
			content, regions, clickRegions := m.renderMainLayoutWithRegions(contentHeight)
			v = tea.NewView(header + "\n" + content + "\n" + statusBar)
			v.MouseMode = tea.MouseModeCellMotion
			v.OnMouse = func(msg tea.MouseMsg) tea.Cmd {
				mouse := msg.Mouse()
				if _, ok := msg.(tea.MouseClickMsg); ok {
					for _, region := range clickRegions {
						if !region.contains(mouse.X, mouse.Y-1) {
							continue
						}
						return func() tea.Msg {
							return uiClickMsg{action: region.action, index: region.index}
						}
					}
				}
				for _, region := range regions {
					if !region.contains(mouse.X, mouse.Y-1) {
						continue
					}
					switch event := msg.(type) {
					case tea.MouseWheelMsg:
						delta := 0
						switch event.Mouse().Button {
						case tea.MouseWheelDown:
							delta = 3
						case tea.MouseWheelUp:
							delta = -3
						}
						if delta != 0 {
							return func() tea.Msg {
								return paneScrollMsg{pane: region.pane, delta: delta}
							}
						}
					case tea.MouseClickMsg:
						if region.pane == scrollPaneLog {
							return func() tea.Msg {
								return uiClickMsg{action: "toggle_log"}
							}
						}
						return func() tea.Msg {
							return paneFocusMsg{pane: region.pane}
						}
					}
				}
				return nil
			}
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
	if m.modelSelectMode == modelSelectProviders {
		return m.renderProviderSelectScreen()
	}

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
	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("42")).
		PaddingLeft(1).
		PaddingRight(1)
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(1).
		PaddingRight(1)
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedMetaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)

	title := i18n.T("model_select.local_title")
	hint := i18n.T("model_select.local_hint")
	if m.modelSelectPhase == selectSecondary {
		title = i18n.T("model_select.local_secondary_title")
		hint = i18n.T("model_select.local_secondary_hint")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(hint))
	b.WriteString("\n\n")

	currentProvider := m.primaryProvider
	if m.modelSelectPhase == selectSecondary && strings.TrimSpace(m.secondaryProvider) != "" {
		currentProvider = m.secondaryProvider
	}
	if strings.TrimSpace(m.modelListProvider) != "" {
		currentProvider = m.modelListProvider
	}
	if currentProvider == "" {
		currentProvider = "ollama"
	}
	b.WriteString(detailStyle.Render(fmt.Sprintf("  provider: %s  (p: configure provider/API)", currentProvider)))
	b.WriteString("\n\n")

	models := m.currentSelectableModels()
	selectedModel := m.roleModel(m.modelSelectPhase)
	for i, model := range models {
		cursor := "  "
		style := normalStyle
		if i == m.modelCursor {
			cursor = "> "
			style = activeStyle
		} else if strings.EqualFold(strings.TrimSpace(model.Name), selectedModel) {
			style = selectedStyle
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
		selectedMark := ""
		if strings.EqualFold(strings.TrimSpace(model.Name), selectedModel) {
			selectedMark = " " + selectedMetaStyle.Render("[selected]")
		}
		b.WriteString(cursor + name + selectedMark + detailStr + "\n")
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)
	b.WriteString("\n")
	if len(models) == 0 {
		b.WriteString(footerStyle.Render("  " + i18n.T("provider_config.empty_models")))
	} else {
		b.WriteString(footerStyle.Render(fmt.Sprintf("  "+i18n.T("model_select.found"), len(models))))
	}

	return m.padContent(b.String(), m.height)
}

func (m Model) renderProviderSelectScreen() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7BD8FF"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#97A9B8"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C572")).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#DCE7EF"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99"))
	activeBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6FC3DF")).
		Padding(0, 1).
		Width(maxInt(28, minInt(84, m.width-8)))
	selectedBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#57D48A")).
		Padding(0, 1).
		Width(maxInt(28, minInt(84, m.width-8)))
	idleBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#31556F")).
		Padding(0, 1).
		Width(maxInt(28, minInt(84, m.width-8)))

	roleLabel := i18n.T("model_select.role_primary")
	if m.modelSelectPhase == selectSecondary {
		roleLabel = i18n.T("model_select.role_secondary")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf(i18n.T("model_select.setup_title"), roleLabel)))
	b.WriteString("\n")
	b.WriteString(subStyle.Render(i18n.T("model_select.setup_hint")))
	b.WriteString("\n\n")

	for i, spec := range llm.ProviderSpecs() {
		lines := []string{
			labelStyle.Render(spec.Label + "  [" + spec.ID + "]"),
			bodyStyle.Render(i18n.T("model_select.kind_" + string(spec.Kind))),
			metaStyle.Render("Base URL: " + spec.DefaultBaseURL),
		}
		if spec.APIKeyEnv != "" {
			lines = append(lines, metaStyle.Render("API Key env: "+spec.APIKeyEnv))
		} else {
			lines = append(lines, metaStyle.Render(i18n.T("model_select.local_provider_hint")))
		}
		if len(spec.RecommendedModels) > 0 {
			lines = append(lines, bodyStyle.Render(i18n.T("model_select.recommended")+": "+strings.Join(spec.RecommendedModels, ", ")))
		}
		if spec.DocsURL != "" {
			lines = append(lines, metaStyle.Render("Docs: "+spec.DocsURL))
		}

		currentMark := ""
		if spec.ID == m.roleProvider(m.modelSelectPhase) {
			modelName := m.roleModel(m.modelSelectPhase)
			if modelName == "" {
				currentMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#57D48A")).Bold(true).Render(i18n.T("model_select.current_provider"))
			} else {
				currentMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#57D48A")).Bold(true).Render(fmt.Sprintf(i18n.T("model_select.current_provider_model"), modelName))
			}
			lines = append(lines, currentMark)
		}

		style := idleBox
		cursor := "  "
		if i == m.modelCursor {
			style = activeBox
			cursor = "> "
		} else if spec.ID == m.roleProvider(m.modelSelectPhase) {
			style = selectedBox
		}
		b.WriteString(cursor + style.Render(strings.Join(lines, "\n")))
		b.WriteString("\n\n")
	}

	b.WriteString(metaStyle.Render(i18n.T("model_select.setup_footer")))
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
		Width(maxInt(24, minInt(60, m.width-8)))
	inactiveBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(maxInt(24, minInt(60, m.width-8)))

	var b strings.Builder
	action := ""
	if m.inputSuggRef != nil {
		action = m.inputSuggRef.Action
	}
	b.WriteString(titleStyle.Render(action))
	b.WriteString("\n\n")

	if m.inputSuggRef != nil {
		if m.inputSuggRef.Interaction == git.PlatformExec {
			preview := platformActionTitle(applyPlatformInputs(m.inputSuggRef.PlatformOp, m.inputFields, m.inputValues))
			b.WriteString(hintStyle.Render("  "+fmt.Sprintf(i18n.T("input.preview"), preview)) + "\n\n")
		} else {
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
			before, after := splitAtRune(val, m.inputCursorAt)
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
		Width(maxInt(24, minInt(72, m.width-8)))

	val := m.goalInput
	before, after := splitAtRune(val, m.goalCursorAt)
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

func (m Model) renderInlineComposer(width int) string {
	content, _ := m.renderInlineComposerWithRegions(width)
	return content
}

func (m Model) renderInlineComposerWithRegions(width int) (string, []clickRegion) {
	panelWidth := width
	if panelWidth < 28 {
		panelWidth = 18
	}
	borderColor := lipgloss.Color("#31556F")
	if m.composerFocused {
		borderColor = lipgloss.Color("#6FC3DF")
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)
	innerWidth, _ := panelInnerSize(borderStyle, panelWidth, 1)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7BD8FF")).Render(localizedPromptTitle())
	hintText := localizedPromptHintIdle()
	if m.composerFocused {
		hintText = localizedPromptHintActive()
	}
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")).Render(hintText)

	val := m.composerInput
	display := val
	if display == "" {
		display = lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")).Render(localizedPromptPlaceholder())
	}
	if m.screen == screenMain && m.composerFocused {
		before, after := splitAtRune(val, m.composerCursor)
		cursor := "|"
		if val == "" {
			display = cursor + lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")).Render(localizedPromptPlaceholder())
		} else {
			display = before + cursor + after
		}
	}
	body := wrapPlainText(display, innerWidth)
	lines := []string{title}
	lines = append(lines, body...)
	lines = append(lines, hint)

	clicks := make([]clickRegion, 0, 10)
	suggestions := m.slashCommandSuggestions()
	if len(suggestions) > 0 {
		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6FC3DF")).Bold(true)
		activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C572")).Bold(true)
		commandStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F4F7FA")).Bold(true)
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8EA5B6"))

		lines = append(lines, "")
		lines = append(lines, headerStyle.Render(localizedCommandsTitle()))
		for i, suggestion := range suggestions {
			prefix := "  "
			cmdText := commandStyle.Render("/" + suggestion.Command)
			if i == m.slashCursor {
				prefix = activeStyle.Render("> ")
				cmdText = activeStyle.Render("/" + suggestion.Command)
			}
			line := prefix + truncateLine("/"+suggestion.Command+"  "+suggestion.Description, innerWidth)
			if innerWidth > 30 {
				line = prefix + cmdText + "  " + descStyle.Render(suggestion.Description)
			}
			lines = append(lines, line)
			contentLine := len(lines) - 1
			clicks = append(clicks, clickRegion{
				action: "pick_slash_suggestion",
				index:  i,
				x0:     0,
				y0:     contentLine + 1,
				x1:     panelWidth,
				y1:     contentLine + 2,
			})
		}
	}

	clicks = append(clicks, clickRegion{
		action: "focus_prompt",
		x0:     0,
		y0:     0,
		x1:     panelWidth,
		y1:     len(lines) + 2,
	})

	content := strings.Join(lines, "\n")
	return borderStyle.Width(panelWidth).Render(content), clicks
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

	heightBudget := maxInt(4, m.height-6)
	linesPerItem := 5
	visibleCount := maxInt(1, heightBudget/linesPerItem)
	start := m.workflowScroll
	if start < 0 {
		start = 0
	}
	if m.workflowCursor < start {
		start = m.workflowCursor
	}
	if m.workflowCursor >= start+visibleCount {
		start = m.workflowCursor - visibleCount + 1
	}
	if maxStart := maxInt(0, len(m.workflows)-visibleCount); start > maxStart {
		start = maxStart
	}
	end := minInt(len(m.workflows), start+visibleCount)

	for i := start; i < end; i++ {
		wf := m.workflows[i]
		cursor := "  "
		style := normalStyle
		if i == m.workflowCursor {
			cursor = "> "
			style = activeStyle
		}
		b.WriteString(cursor + style.Render(wf.Label) + "\n")
		for _, line := range wrapPlainText("   "+wf.Goal, maxInt(16, m.width-6)) {
			b.WriteString(detailStyle.Render(line) + "\n")
		}
		if len(wf.Prerequisites) > 0 {
			for _, line := range wrapPlainText("   prerequisites: "+strings.Join(wf.Prerequisites, ", "), maxInt(16, m.width-6)) {
				b.WriteString(detailStyle.Render(line) + "\n")
			}
		}
		if len(wf.Capabilities) > 0 {
			for _, line := range wrapPlainText("   capabilities: "+strings.Join(wf.Capabilities, ", "), maxInt(16, m.width-6)) {
				b.WriteString(detailStyle.Render(line) + "\n")
			}
			if coverage := strings.TrimSpace(m.capabilityCoverageSummary(wf.Capabilities)); coverage != "" {
				for _, line := range wrapPlainText("   coverage: "+coverage, maxInt(16, m.width-6)) {
					b.WriteString(detailStyle.Render(line) + "\n")
				}
			}
		}
		if len(wf.Prefill) > 0 {
			b.WriteString(detailStyle.Render(fmt.Sprintf("   orchestration hints: %d", len(wf.Prefill))) + "\n")
		}
		b.WriteString("\n")
	}
	if len(m.workflows) > 0 {
		footer := fmt.Sprintf("scroll %d/%d", start+1, maxInt(1, len(m.workflows)-visibleCount+1))
		b.WriteString(detailStyle.Render(footer))
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
	title = truncateLine(title, m.width)
	if theme.Current != nil {
		return theme.Current.Header.Width(m.width).Render(title)
	}
	return title
}

func (m Model) renderStatusBar() string {
	msg := strings.TrimSpace(m.statusMsg)
	if msg == "" {
		msg = m.latestResultStatusSummary()
	}
	if msg == "" && m.startupInfo.GitAvailable {
		msg = fmt.Sprintf(
			i18n.T("status_bar.format"),
			m.startupInfo.GitVersion,
			m.startupInfo.AIStatus,
			m.currentResolvedLanguage(),
			m.mode,
		)
		if modelInfo := m.renderModelSummary(); modelInfo != "" {
			msg += " | " + modelInfo
		}
		if m.automation.Enabled {
			msg += fmt.Sprintf(" | %s:%s/%ds", localizedAutomationTitle(), localizedAutomationModeLabel(m.automationMode()), m.automation.MonitorInterval)
		} else {
			msg += fmt.Sprintf(" | %s:%s", localizedAutomationTitle(), localizedAutomationModeLabel(m.automationMode()))
		}
	}
	if msg == "" {
		msg = i18n.T("ui.ready")
	}
	msg = truncateLine(msg, m.width)
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(m.width)
	return style.Render(msg)
}

func (m Model) latestResultStatusSummary() string {
	if strings.TrimSpace(m.lastCommand.Title) == "" {
		return ""
	}
	switch m.lastCommand.ResultKind {
	case resultKindPlatformAdmin:
		parts := []string{localizedStatusText(m.lastCommand.Status), humanCapabilityLabel(strings.TrimSpace(m.lastCommand.PlatformCapability))}
		if flow := strings.TrimSpace(m.lastCommand.PlatformFlow); flow != "" {
			parts = append(parts, flow)
		}
		if operation := strings.TrimSpace(m.lastCommand.PlatformOperation); operation != "" {
			parts = append(parts, operation)
		}
		if summary := firstNonEmptyLine(m.lastCommand.Output); summary != "" {
			parts = append(parts, summary)
		}
		return strings.Join(compactStringList(parts, 8), " | ")
	case resultKindFileWrite:
		fileLabel := strings.TrimSpace(firstNonEmpty(m.lastCommand.FilePath, m.lastCommand.Title))
		return strings.TrimSpace(fmt.Sprintf("%s | %s | %s", localizedStatusText(m.lastCommand.Status), m.lastCommand.FileOperation, fileLabel))
	default:
		if summary := firstNonEmptyLine(m.lastCommand.Output); summary != "" {
			return strings.TrimSpace(fmt.Sprintf("%s | %s | %s", localizedStatusText(m.lastCommand.Status), m.lastCommand.Title, summary))
		}
		return strings.TrimSpace(fmt.Sprintf("%s | %s", localizedStatusText(m.lastCommand.Status), m.lastCommand.Title))
	}
}

func (m Model) renderActionBar() string {
	bar := "/help  /goal <text>  /settings show|mode|interval  /workflow  /refresh  /accept  /skip  /why  click: suggestions/log"
	if m.lastPlatform != nil && m.lastPlatform.Mutation != nil {
		bar += "  /validate  /rollback"
	}
	if m.editableFileRequest() != nil || m.editablePlatformRequest() != nil {
		bar += "  /edit"
	}
	bar = truncateLine(bar, m.width)
	if theme.Current != nil {
		return theme.Current.ActionBar.Width(m.width).Render(bar)
	}
	return bar
}

func (m Model) renderModelSummary() string {
	if strings.TrimSpace(m.selectedPrimary) == "" {
		return ""
	}
	primary := strings.TrimSpace(m.selectedPrimary)
	if provider := strings.TrimSpace(m.primaryProvider); provider != "" {
		primary = provider + "/" + primary
	}
	if m.secondaryEnabled && strings.TrimSpace(m.selectedSecondary) != "" {
		secondary := strings.TrimSpace(m.selectedSecondary)
		if provider := strings.TrimSpace(m.secondaryProvider); provider != "" {
			secondary = provider + "/" + secondary
		}
		return fmt.Sprintf(i18n.T("model_select.summary_on"), primary, secondary)
	}
	return fmt.Sprintf(i18n.T("model_select.summary_off"), primary)
}

func (m Model) renderMainContent(height int) string {
	content, _, _ := m.renderMainLayoutWithRegions(height)
	return content
}

func (m Model) renderMainLayout(height int) (string, []panelRegion) {
	content, regions, _ := m.renderMainLayoutWithRegions(height)
	return content, regions
}

func (m Model) renderMainLayoutWithRegions(height int) (string, []panelRegion, []clickRegion) {
	if m.gitState == nil {
		welcome := i18n.T("welcome.title") + "\n\n"
		if m.startupInfo.GitAvailable {
			welcome += fmt.Sprintf(i18n.T("welcome.git_ok"), m.startupInfo.GitVersion) + "\n"
		} else {
			welcome += i18n.T("welcome.git_missing") + "\n"
		}
		welcome += fmt.Sprintf(i18n.T("welcome.ollama_status"), m.startupInfo.AIStatus) + "\n"
		welcome += "\n" + i18n.T("welcome.waiting")
		return m.padContent(welcome, height), []panelRegion{{pane: scrollPaneWorkspace, x0: 0, y0: 0, x1: m.width, y1: height}}, nil
	}

	if m.width < 80 {
		summaryHeight := 0
		summary := strings.TrimSpace(m.renderCompactAreasSummary())
		if summary != "" {
			summaryHeight = 1
		}
		remaining := height - summaryHeight
		if remaining < 9 {
			remaining = 9
		}

		logHeight := 1
		logGap := 0
		if m.logExpanded {
			logHeight = maxInt(6, remaining/4)
			logGap = 1
		}
		mainHeight := remaining - logGap - logHeight
		if mainHeight < 8 {
			mainHeight = 8
		}
		obsHeight := maxInt(7, mainHeight/3)
		workspaceHeight := mainHeight - obsHeight
		if workspaceHeight < 4 {
			deficit := 4 - workspaceHeight
			reduceObs := minInt(deficit, maxInt(0, obsHeight-6))
			obsHeight -= reduceObs
			deficit -= reduceObs
			if deficit > 0 {
				reduceLog := minInt(deficit, maxInt(0, logHeight-5))
				logHeight -= reduceLog
				deficit -= reduceLog
			}
			workspaceHeight = mainHeight - obsHeight
			if workspaceHeight < 4 {
				workspaceHeight = 4
			}
		}

		topParts := make([]string, 0, 4)
		regions := make([]panelRegion, 0, 3)
		clicks := make([]clickRegion, 0, 4)
		y := 0
		if summaryHeight > 0 {
			topParts = append(topParts, summary)
			y++
		}

		workspaceFull, workspaceClicks := m.renderLeftWorkspaceWithRegions(m.width)
		workspace := sliceVisibleLines(workspaceFull, workspaceHeight, m.leftScroll)
		topParts = append(topParts, workspace)
		regions = append(regions, panelRegion{pane: scrollPaneWorkspace, x0: 0, y0: y, x1: m.width, y1: y + workspaceHeight})
		clicks = append(clicks, translateClickRegions(workspaceClicks, 0, y, m.leftScroll, workspaceHeight)...)
		y += workspaceHeight

		obsPanel, obsClicks := m.renderObservabilityPanelCachedWithRegions(m.width, obsHeight)
		topParts = append(topParts, obsPanel)
		regions = append(regions, panelRegion{pane: scrollPaneObservability, x0: 0, y0: y, x1: m.width, y1: minInt(height, y+obsHeight)})
		clicks = append(clicks, offsetClickRegions(obsClicks, 0, y)...)
		logTop := height - logGap - logHeight
		logPanel := m.renderOperationLogPanelCached(m.width, logHeight)
		regions = append(regions, panelRegion{pane: scrollPaneLog, x0: 0, y0: y, x1: m.width, y1: y + logHeight})
		regions[len(regions)-1].y0 = logTop
		regions[len(regions)-1].y1 = logTop + logHeight
		clicks = append(clicks, clickRegion{action: "toggle_log", x0: 0, y0: logTop, x1: m.width, y1: logTop + logHeight})

		topContent := m.padContent(joinLayoutBlocks(topParts), logTop)
		contentParts := []string{topContent}
		if logGap > 0 {
			contentParts = append(contentParts, "")
		}
		contentParts = append(contentParts, logPanel)
		return joinLayoutBlocks(contentParts), regions, clicks
	}

	gap := 1
	leftWidth, rightWidth, narrow := m.columnWidths()
	if narrow || leftWidth < 40 {
		workspaceFull, workspaceClicks := m.renderLeftWorkspaceWithRegions(m.width)
		logHeight := 1
		logGap := 0
		if m.logExpanded {
			logHeight = maxInt(8, height/3)
			logGap = 1
		}
		workspaceHeight := height - logGap - logHeight
		if workspaceHeight < 4 {
			workspaceHeight = 4
		}
		topContent := m.padContent(sliceVisibleLines(workspaceFull, workspaceHeight, m.leftScroll), height-logGap-logHeight)
		contentParts := []string{topContent}
		if logGap > 0 {
			contentParts = append(contentParts, "")
		}
		contentParts = append(contentParts, m.renderOperationLogPanelCached(m.width, logHeight))
		content := joinLayoutBlocks(contentParts)
		regions := []panelRegion{
			{pane: scrollPaneWorkspace, x0: 0, y0: 0, x1: m.width, y1: workspaceHeight},
			{pane: scrollPaneLog, x0: 0, y0: workspaceHeight + logGap, x1: m.width, y1: workspaceHeight + logGap + logHeight},
		}
		clicks := translateClickRegions(workspaceClicks, 0, 0, m.leftScroll, workspaceHeight)
		clicks = append(clicks, clickRegion{action: "toggle_log", x0: 0, y0: workspaceHeight + logGap, x1: m.width, y1: workspaceHeight + logGap + logHeight})
		return m.padContent(content, height), regions, clicks
	}

	logHeight := 1
	logGap := 0
	if m.logExpanded {
		logHeight = maxInt(10, height/3)
		logGap = 1
	}
	topHeight := height - logGap - logHeight
	if topHeight < 12 {
		topHeight = 12
		logHeight = maxInt(1, height-logGap-topHeight)
	}
	rightTopHeight := topHeight * 42 / 100
	if rightTopHeight < 12 {
		rightTopHeight = 12
	}
	if rightTopHeight > topHeight-12 {
		rightTopHeight = topHeight - 12
	}
	rightBottomHeight := topHeight - gap - rightTopHeight
	if rightBottomHeight < 10 {
		rightBottomHeight = 10
		rightTopHeight = topHeight - gap - rightBottomHeight
	}
	workspaceHeight := topHeight
	workspaceFull, workspaceClicks := m.renderLeftWorkspaceWithRegions(leftWidth)
	workspace := lipgloss.NewStyle().Width(leftWidth).Render(sliceVisibleLines(workspaceFull, workspaceHeight, m.leftScroll))
	areas := lipgloss.NewStyle().Width(rightWidth).Render(m.renderAreasTreePanel(rightWidth, rightTopHeight))
	obsPanel, obsClicks := m.renderObservabilityPanelCachedWithRegions(rightWidth, rightBottomHeight)
	obs := lipgloss.NewStyle().Width(rightWidth).Render(obsPanel)
	right := lipgloss.JoinVertical(lipgloss.Left, areas, "", obs)
	topContent := lipgloss.JoinHorizontal(lipgloss.Top, workspace, " ", right)
	contentParts := []string{m.padContent(topContent, topHeight)}
	if logGap > 0 {
		contentParts = append(contentParts, "")
	}
	contentParts = append(contentParts, lipgloss.NewStyle().Width(m.width).Render(m.renderOperationLogPanelCached(m.width, logHeight)))
	content := joinLayoutBlocks(contentParts)
	regions := []panelRegion{
		{pane: scrollPaneWorkspace, x0: 0, y0: 0, x1: leftWidth, y1: workspaceHeight},
		{pane: scrollPaneAreas, x0: leftWidth + gap, y0: 0, x1: leftWidth + gap + rightWidth, y1: rightTopHeight},
		{pane: scrollPaneObservability, x0: leftWidth + gap, y0: rightTopHeight + gap, x1: leftWidth + gap + rightWidth, y1: topHeight},
		{pane: scrollPaneLog, x0: 0, y0: topHeight + logGap, x1: m.width, y1: topHeight + logGap + logHeight},
	}
	clicks := translateClickRegions(workspaceClicks, 0, 0, m.leftScroll, workspaceHeight)
	clicks = append(clicks, offsetClickRegions(obsClicks, leftWidth+gap, rightTopHeight+gap)...)
	clicks = append(clicks, clickRegion{action: "toggle_log", x0: 0, y0: topHeight + logGap, x1: m.width, y1: topHeight + logGap + logHeight})
	return m.padContent(content, height), regions, clicks
}

func (m Model) renderLeftWorkspace(width int) string {
	content, _ := m.renderLeftWorkspaceWithRegions(width)
	return content
}

func (m Model) renderLeftWorkspaceWithRegions(width int) (string, []clickRegion) {
	var sections []string
	var clicks []clickRegion
	lineOffset := 0
	appendSection := func(section string, sectionClicks []clickRegion) {
		if section == "" {
			return
		}
		if len(sections) > 0 {
			sections = append(sections, "")
			lineOffset++
		}
		sections = append(sections, section)
		clicks = append(clicks, offsetClickRegions(sectionClicks, 0, lineOffset)...)
		lineOffset += lineCount(section)
	}

	if composer, composerClicks := m.renderInlineComposerWithRegions(width); composer != "" {
		appendSection(composer, composerClicks)
	}
	if automation := m.renderAutomationPanelCached(width); automation != "" {
		appendSection(automation, []clickRegion{{action: "open_settings", x0: 0, y0: 0, x1: width, y1: lineCount(automation)}})
	}
	tabsLine, tabClicks := m.renderWorkspaceTabsWithRegions(width)
	appendSection(tabsLine, tabClicks)

	primarySection, primaryClicks := m.renderWorkspacePrimarySection(width)
	appendSection(primarySection, primaryClicks)

	if m.llmThinking != "" && m.expanded {
		appendSection(m.renderThinkingPanel(width), nil)
	}
	return strings.Join(sections, "\n"), clicks
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

func (m Model) renderOperationLogPanel(width, height int) string {
	if width < 16 {
		width = 16
	}
	if !m.logExpanded {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#A98CFF")).Bold(true).Render("> " + i18n.T("oplog.title") + " (" + i18n.T("oplog.collapsed") + ")")
	}
	if m.logExpanded && height < 8 {
		height = 8
	}
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#8A6BFF")).
		Padding(0, 1)
	innerWidth, innerHeight := panelInnerSize(panelStyle, width, height)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A98CFF"))
	modeLabel := i18n.T("oplog.collapsed")
	if m.logExpanded {
		modeLabel = i18n.T("oplog.expanded")
	}
	hints := i18n.T("oplog.toggle")
	hints += "  " + i18n.T("oplog.scroll")
	chromeHeight := 2
	bodyHeight := innerHeight - chromeHeight
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	logBody := m.renderOperationLogBody(innerWidth, bodyHeight)

	content := strings.Join([]string{
		headerStyle.Render(i18n.T("oplog.title") + " (" + modeLabel + ")"),
		logBody,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")).Render(hints),
	}, "\n")

	return renderBoundedPanel(panelStyle, width, height, content)
}

func (m Model) renderOperationLogBody(width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := m.operationLogLines(width)
	if len(lines) == 0 {
		return sliceVisibleLinesFromEnd(i18n.T("oplog.empty"), height, 0)
	}
	return sliceVisibleLinesFromEnd(strings.Join(lines, "\n"), height, m.logScrollOffset)
}

func (m Model) operationLogLines(width int) []string {
	if m.opLog == nil {
		return nil
	}
	entries := m.opLog.Entries()
	if len(entries) == 0 {
		return nil
	}
	lines := make([]string, 0, len(entries)*3)
	for _, entry := range entries {
		prefix := fmt.Sprintf("%s %s ", entry.Timestamp.Format("15:04:05"), entry.Icon())
		summaryStyle := eventTypeStyle(entry.Type)
		summary := strings.TrimSpace(entry.Summary)
		if summary == "" {
			summary = strings.TrimSpace(entry.Detail)
		}
		lines = append(lines, renderWrappedField(prefix, tsStyle(), summary, summaryStyle, width)...)
		if detail := strings.TrimSpace(entry.Detail); detail != "" {
			for _, detailLine := range wrapPlainText(detail, maxInt(8, width-2)) {
				lines = append(lines, infoStyle().Render("  "+detailLine))
			}
		}
	}
	return lines
}

func (m Model) renderAreasTreePanel(width, height int) string {
	if width < 24 {
		width = 18
	}
	if height < 10 {
		height = 10
	}
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6FC3DF")).
		Padding(0, 1)
	innerWidth, innerHeight := panelInnerSize(panelStyle, width, height)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	tree := components.NewAreasTree(m.gitState).
		SetWidth(innerWidth).
		SetMaxItems(0).
		View()
	chromeHeight := 2
	bodyHeight := innerHeight - chromeHeight
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	body := sliceVisibleLines(tree, bodyHeight, m.areasScroll)
	maxPage := 1
	if totalLines := lineCount(tree); totalLines > bodyHeight {
		maxPage = totalLines - bodyHeight + 1
	}
	scrollHint := mutedStyle().Render(fmt.Sprintf("scroll %d/%d", m.areasScroll+1, maxPage))
	content := headerStyle.Render(i18n.T("areas.title")) + "\n" + body + "\n" + scrollHint
	return renderBoundedPanel(panelStyle, width, height, content)
}

func (m Model) renderAnalysisPanel(width int) string {
	panelWidth := width
	if panelWidth <= 0 {
		panelWidth = m.width
	}
	if panelWidth < 18 {
		panelWidth = 18
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#31556F")).
		Padding(0, 1)
	innerWidth, _ := panelInnerSize(borderStyle, panelWidth, 1)
	borderStyle = borderStyle.Width(innerWidth)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7BD8FF"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5EEF5"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C73")).Bold(true)

	var parts []string
	parts = append(parts, headerStyle.Render(i18n.T("analysis.title")))

	if goal := strings.TrimSpace(m.session.ActiveGoal); goal != "" {
		goalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		statusStr := ""
		if s := m.currentGoalStatus(); s != "" {
			statusStr = " (" + localizedGoalStatusText(s) + ")"
		}
		parts = append(parts, goalStyle.Render(localizedGoalLabel()+": "+goal+statusStr))
	}

	if analysisHasFailure(m.llmAnalysis) {
		parts = append(parts, errorStyle.Render(m.llmAnalysis))
	} else {
		parts = append(parts, textStyle.Render(m.llmAnalysis))
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
	if panelWidth < 18 {
		panelWidth = 18
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(0, 1)
	innerWidth, _ := panelInnerSize(borderStyle, panelWidth, 1)
	borderStyle = borderStyle.Width(innerWidth)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241"))
	thinkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	content := headerStyle.Render(i18n.T("thinking.title")) + "\n" + thinkStyle.Render(strings.TrimSpace(m.llmThinking))
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
	content, _ := m.renderSuggestionCardsWithRegions(width)
	return content
}

func (m Model) renderSuggestionCardsWithRegions(width int) (string, []clickRegion) {
	cardWidth := width
	if cardWidth <= 0 {
		cardWidth = m.width
	}
	if cardWidth < 30 {
		cardWidth = 30
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
		var execState git.ExecState
		var execMsg string
		if i < len(m.suggExecState) {
			execState = m.suggExecState[i]
		}
		if i < len(m.suggExecMsg) {
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
		var noteDisplay []string
		switch s.Interaction {
		case git.InfoOnly:
			cmdDisplay = []string{"Review advisory details"}
		case git.FileWrite:
			if s.FileOp != nil {
				op := strings.TrimSpace(s.FileOp.Operation)
				if op == "" {
					op = "create"
				}
				cmdDisplay = []string{fmt.Sprintf("%s file %s", op, s.FileOp.Path)}
			} else {
				cmdDisplay = []string{"Prepare file change"}
			}
		case git.NeedsInput:
			cmdDisplay = []string{"Requires input before execution"}
		case git.PlatformExec:
			cmdDisplay = []string{platformSuggestionCommand(s.PlatformOp)}
		default:
			command := strings.TrimSpace(joinCmd(s.Command))
			if command == "" {
				command = "Review suggested action"
			}
			cmdDisplay = []string{command}
		}
		if strings.TrimSpace(execMsg) != "" && execMsg != "done" && execMsg != "success" && execMsg != "running..." {
			noteDisplay = append(noteDisplay, execMsg)
		}

		card := components.NewSuggestionCard(action, reason, cmdDisplay, riskLabel(s.RiskLevel))
		card.Notes = noteDisplay
		card.Controls = "click: select  /accept  /skip  /why"
		if isActive {
			card.Controls = "selected  click: details  /accept  /skip  /why"
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
		} else if s.Interaction == git.FileWrite || s.Interaction == git.InfoOnly || s.Interaction == git.NeedsInput {
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
		cardHeight := lineCount(rendered)
		clicks = append(clicks, clickRegion{action: "select_suggestion", index: i, x0: 0, y0: lineOffset, x1: cardWidth, y1: lineOffset + cardHeight})
		lineOffset += cardHeight
	}

	return strings.Join(cards, "\n"), clicks
}

func offsetClickRegions(regions []clickRegion, xOffset, yOffset int) []clickRegion {
	if len(regions) == 0 {
		return nil
	}
	out := make([]clickRegion, 0, len(regions))
	for _, region := range regions {
		region.x0 += xOffset
		region.x1 += xOffset
		region.y0 += yOffset
		region.y1 += yOffset
		out = append(out, region)
	}
	return out
}

func translateClickRegions(regions []clickRegion, xOffset, yOffset, scroll, visibleHeight int) []clickRegion {
	if len(regions) == 0 {
		return nil
	}
	out := make([]clickRegion, 0, len(regions))
	for _, region := range regions {
		top := region.y0 - scroll
		bottom := region.y1 - scroll
		if bottom <= 0 || top >= visibleHeight {
			continue
		}
		if top < 0 {
			top = 0
		}
		if bottom > visibleHeight {
			bottom = visibleHeight
		}
		out = append(out, clickRegion{
			action: region.action,
			index:  region.index,
			x0:     region.x0 + xOffset,
			x1:     region.x1 + xOffset,
			y0:     top + yOffset,
			y1:     bottom + yOffset,
		})
	}
	return out
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

func joinLayoutBlocks(blocks []string) string {
	lines := make([]string, 0, len(blocks)*4)
	for _, block := range blocks {
		if block == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, strings.Split(block, "\n")...)
	}
	return strings.Join(lines, "\n")
}

func truncateLine(text string, width int) string {
	if width <= 0 {
		return text
	}
	return runewidth.Truncate(text, width, "")
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
