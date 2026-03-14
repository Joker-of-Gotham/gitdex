package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) handleUIClick(msg uiClickMsg) (tea.Model, tea.Cmd) {
	switch msg.action {
	case "focus_prompt":
		m.composerFocused = true
		m.statusMsg = localizedText("Prompt focused", "输入框已聚焦", "Prompt focused")
		return m, nil
	case "open_settings":
		m.composerFocused = false
		m = m.openAutomationConfig()
		return m, nil
	case "pick_slash_suggestion":
		if !m.applySlashSuggestion(msg.index) {
			return m, nil
		}
		return m, nil
	case "toggle_log":
		return m.toggleLogPanel(), nil
	case "observability_tab":
		tabs := observabilityTabs()
		if msg.index < 0 || msg.index >= len(tabs) {
			return m, nil
		}
		m.obsTab = tabs[msg.index]
		m.obsScroll = 0
		m.statusMsg = fmt.Sprintf(i18n.T("observability.inspector_status"), m.obsTab.label())
		return m, nil
	case "workspace_tab":
		tab := workspaceTab(msg.index)
		if !tab.valid() {
			return m, nil
		}
		m.workspaceTab = tab
		m.leftScroll = 0
		m.statusMsg = localizedText("Workspace view: ", "工作区视图：", "Workspace view: ") + tab.label()
		return m, nil
	case "select_suggestion":
		if msg.index < 0 || msg.index >= len(m.suggestions) {
			return m, nil
		}
		m.suggIdx = msg.index
		m.workspaceTab = workspaceTabSuggestions
		m.expanded = false
		m.llmReason = ""
		m.statusMsg = localizedText("Selected suggestion: ", "已选择建议：", "Selected suggestion: ") + m.suggestions[msg.index].Action
		m.showSelectedSuggestionGuidance()
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) toggleLogPanel() Model {
	m.logExpanded = !m.logExpanded
	if !m.logExpanded {
		m.logScrollOffset = 0
	}
	if m.logExpanded {
		m.statusMsg = i18n.T("oplog.title") + " (" + i18n.T("oplog.expanded") + ")"
	} else {
		m.statusMsg = i18n.T("oplog.title") + " (" + i18n.T("oplog.collapsed") + ")"
	}
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: m.statusMsg,
	})
	return m
}

func (m Model) runSlashCommand(input string) (tea.Model, tea.Cmd) {
	command := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "/"))
	if command == "" {
		m.statusMsg = localizedText("Use /help to see available commands.", "输入 /help 查看可用命令。", "Use /help to see available commands.")
		return m, nil
	}

	fields := strings.Fields(command)
	if len(fields) == 0 {
		m.statusMsg = localizedText("Use /help to see available commands.", "输入 /help 查看可用命令。", "Use /help to see available commands.")
		return m, nil
	}

	switch strings.ToLower(fields[0]) {
	case "help":
		lines := []string{
			"/help - " + localizedText("show the command palette", "显示命令面板", "show the command palette"),
			"/goal <text> - " + localizedText("set the active goal", "设置当前目标", "set the active goal"),
			"/mode show|manual|auto|cruise - " + localizedText("choose how automation behaves", "选择自动化模式", "choose how automation behaves"),
			"/interval <seconds> - " + localizedText("set the background check interval", "设置后台巡检间隔", "set the background check interval"),
			"/trust on|off - " + localizedText("toggle trusted unattended execution", "切换信任无人值守执行", "toggle trusted unattended execution"),
			"/settings - " + localizedText("open automation settings", "打开自动化设置", "open automation settings"),
			"/run accept|all|skip|why|refresh - " + localizedText("execute or inspect suggestions", "执行或查看建议", "execute or inspect suggestions"),
			"/view overview|suggestions|result|analysis|log|observability - " + localizedText("switch the main workspace view", "切换主工作区视图", "switch the main workspace view"),
			"/config status|llm|provider|automation|platform - " + localizedText("open the relevant configuration flow", "打开相应配置流程", "open the relevant configuration flow"),
			"/language auto|en|zh|ja - " + localizedText("switch UI language immediately", "立即切换界面语言", "switch UI language immediately"),
			"/quit - " + localizedText("exit gitdex", "退出 gitdex", "exit gitdex"),
		}
		m.statusMsg = localizedText("Slash commands ready.", "斜杠命令已就绪。", "Slash commands ready.")
		m.setCommandResponse(localizedText("Slash commands", "斜杠命令", "Slash commands"), strings.Join(lines, "\n"))
		return m, nil
	case "goal":
		return m.applyActiveGoal(strings.TrimSpace(strings.TrimPrefix(command, fields[0])))
	case "mode":
		return m.runModeSlashCommand(fields[1:])
	case "interval":
		return m.runAutomationSlashCommand([]string{"interval"}, fields[1:]...)
	case "trust":
		return m.runAutomationSlashCommand([]string{"trust"}, fields[1:]...)
	case "automation", "settings":
		if len(fields) == 1 {
			m.composerFocused = false
			return m.openAutomationConfig(), nil
		}
		return m.runAutomationSlashCommand(fields[1:])
	case "config":
		return m.runConfigSlashCommand(fields[1:])
	case "llm", "model":
		m.composerFocused = false
		return m.openModelSetup(selectPrimary), nil
	case "provider":
		m.composerFocused = false
		return m.openProviderConfig(selectPrimary), nil
	case "workflow":
		return m.dispatchMainCommand("f")
	case "refresh":
		return m.dispatchMainCommand("r")
	case "log":
		return m.toggleLogPanel(), nil
	case "run":
		return m.runExecutionSlashCommand(fields[1:])
	case "accept":
		return m.runExecutionSlashCommand([]string{"accept"})
	case "skip":
		return m.runExecutionSlashCommand([]string{"skip"})
	case "why":
		return m.runExecutionSlashCommand([]string{"why"})
	case "validate":
		return m.runExecutionSlashCommand([]string{"validate"})
	case "rollback":
		return m.runExecutionSlashCommand([]string{"rollback"})
	case "edit":
		return m.runExecutionSlashCommand([]string{"edit"})
	case "view":
		return m.runViewSlashCommand(fields[1:])
	case "language", "lang":
		if len(fields) == 1 {
			return m.dispatchMainCommand("L")
		}
		next, err := m.applyLanguagePreference(fields[1])
		if err != nil {
			m.statusMsg = localizedText("Unknown language.", "未知语言。", "Unknown language.")
			m.setCommandResponse(localizedAssistantTitle(), localizedText("Supported values: /language auto|en|zh|ja", "支持的值：/language auto|en|zh|ja", "Supported values: /language auto|en|zh|ja"))
			return m, nil
		}
		return next, nil
	case "quit", "exit":
		return m, tea.Quit
	default:
		m.statusMsg = localizedText("Unknown slash command: /", "未知斜杠命令：/", "Unknown slash command: /") + fields[0]
		m.setCommandResponse(localizedAssistantTitle(), m.statusMsg)
		return m, nil
	}
}

func (m Model) dispatchMainCommand(key string) (tea.Model, tea.Cmd) {
	m.composerFocused = false
	return m.updateMain(tea.KeyPressMsg(tea.Key{Text: key}))
}

func (m Model) runModeSlashCommand(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 || strings.EqualFold(strings.TrimSpace(args[0]), "show") {
		m.setCommandResponse(localizedAutomationTitle(), strings.Join([]string{
			m.automationSummaryText(),
			m.automationGoalRequirementText(),
			localizedAutomationModeDescription(m.automationMode()),
		}, "\n"))
		m.statusMsg = localizedText("Automation mode loaded.", "已加载自动化模式。", "Automation mode loaded.")
		return m, nil
	}
	mode := strings.ToLower(strings.TrimSpace(args[0]))
	switch mode {
	case "manual", "auto", "cruise":
	default:
		m.statusMsg = localizedText("Unknown mode. Use: manual, auto, cruise.", "未知模式。可选：manual、auto、cruise。", "Unknown mode. Use: manual, auto, cruise.")
		return m, nil
	}
	next := m.automation
	next.Mode = config.NormalizeAutomationMode(mode)
	config.ApplyAutomationMode(&next)
	return m.persistAutomationState(next)
}

func (m Model) runAutomationSlashCommand(args []string, extra ...string) (tea.Model, tea.Cmd) {
	full := append([]string(nil), args...)
	full = append(full, extra...)
	if len(full) == 0 || strings.EqualFold(strings.TrimSpace(full[0]), "show") {
		m.setCommandResponse(localizedAutomationTitle(), strings.Join([]string{
			m.automationSummaryText(),
			m.automationGoalRequirementText(),
			localizedAutomationModeDescription(m.automationMode()),
		}, "\n"))
		m.statusMsg = localizedText("Automation settings loaded.", "已加载自动化设置。", "Automation settings loaded.")
		return m, nil
	}

	next := m.automation
	switch strings.ToLower(strings.TrimSpace(full[0])) {
	case "mode":
		if len(full) < 2 {
			m.statusMsg = localizedText("Usage: /mode manual|auto|cruise", "用法：/mode manual|auto|cruise", "Usage: /mode manual|auto|cruise")
			return m, nil
		}
		next.Mode = strings.ToLower(strings.TrimSpace(full[1]))
		switch next.Mode {
		case "manual", "auto", "cruise":
		default:
			m.statusMsg = localizedText("Unknown mode. Use: manual, auto, cruise.", "未知模式。可选：manual、auto、cruise。", "Unknown mode. Use: manual, auto, cruise.")
			return m, nil
		}
		config.ApplyAutomationMode(&next)
	case "interval":
		if len(full) < 2 {
			m.statusMsg = localizedText("Usage: /interval <seconds>", "用法：/interval <seconds>", "Usage: /interval <seconds>")
			return m, nil
		}
		seconds, ok := parsePositiveInt(full[1], 60)
		if !ok {
			m.statusMsg = localizedText("Interval must be an integer >= 60 seconds.", "间隔必须是不小于 60 秒的整数。", "Interval must be an integer >= 60 seconds.")
			return m, nil
		}
		next.MonitorInterval = seconds
	case "trust", "trusted":
		if len(full) < 2 {
			m.statusMsg = localizedText("Usage: /trust on|off", "用法：/trust on|off", "Usage: /trust on|off")
			return m, nil
		}
		value, ok := parseOnOff(full[1])
		if !ok {
			m.statusMsg = localizedText("Expected on or off.", "参数必须是 on 或 off。", "Expected on or off.")
			return m, nil
		}
		next.TrustedMode = value
	case "max-steps", "max_steps":
		if len(full) < 2 {
			m.statusMsg = localizedText("Usage: /settings max-steps <count>", "用法：/settings max-steps <count>", "Usage: /settings max-steps <count>")
			return m, nil
		}
		count, ok := parsePositiveInt(full[1], 1)
		if !ok {
			m.statusMsg = localizedText("Max steps must be an integer >= 1.", "最大步数必须是不小于 1 的整数。", "Max steps must be an integer >= 1.")
			return m, nil
		}
		next.MaxAutoSteps = count
	default:
		m.statusMsg = localizedText("Unknown automation command.", "未知自动化命令。", "Unknown automation command.")
		return m, nil
	}
	config.ApplyAutomationMode(&next)
	return m.persistAutomationState(next)
}

func (m Model) runExecutionSlashCommand(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.setCommandResponse(localizedCommandsTitle(), localizedText(
			"/run accept  /run all  /run skip  /run why  /run refresh  /run validate  /run rollback  /run edit  /run quit",
			"/run accept  /run all  /run skip  /run why  /run refresh  /run validate  /run rollback  /run edit  /run quit",
			"/run accept  /run all  /run skip  /run why  /run refresh  /run validate  /run rollback  /run edit  /run quit",
		))
		return m, nil
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "accept":
		return m.dispatchMainCommand("y")
	case "all":
		summary := m.batchRunSummary(true)
		m.setCommandResponse(localizedCommandsTitle(), summary)
		if !m.shouldAllowBatchRun() {
			m.statusMsg = localizedText("No pending suggestions to batch-run.", "当前没有可批量执行的建议。", "No pending suggestions to batch-run.")
			return m, nil
		}
		m.batchRunRequested = true
		m.autoSteps = 0
		if next, cmd, ok := m.autoExecuteNextSafeSuggestion(true); ok {
			next.batchRunRequested = true
			next.workspaceTab = workspaceTabSuggestions
			next.setCommandResponse(localizedCommandsTitle(), localizedText(
				"Batch run started. Gitdex will continue executing eligible pending suggestions until it needs input, hits a policy boundary, or finishes.",
				"批量执行已开始。gitdex 会持续执行当前可执行的待处理建议，直到需要输入、遇到策略边界或全部完成。",
				"Batch run started. Gitdex will continue executing eligible pending suggestions until it needs input, hits a policy boundary, or finishes.",
			)+"\n\n"+summary)
			return next, cmd
		}
		m.batchRunRequested = false
		m.setCommandResponse(localizedCommandsTitle(), summary)
		return m, nil
	case "skip":
		return m.dispatchMainCommand("n")
	case "why":
		return m.dispatchMainCommand("w")
	case "refresh":
		return m.dispatchMainCommand("r")
	case "validate":
		return m.dispatchMainCommand("v")
	case "rollback":
		return m.dispatchMainCommand("b")
	case "edit":
		return m.dispatchMainCommand("e")
	case "quit", "exit":
		return m, tea.Quit
	default:
		m.statusMsg = localizedText("Unknown /run command.", "未知 /run 命令。", "Unknown /run command.")
		return m, nil
	}
}

func (m Model) runViewSlashCommand(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.setCommandResponse(localizedText("Workspace views", "工作区视图", "Workspace views"), localizedText(
			"/view overview\n/view suggestions\n/view result\n/view analysis\n/view log\n/view observability",
			"/view overview\n/view suggestions\n/view result\n/view analysis\n/view log\n/view observability",
			"/view overview\n/view suggestions\n/view result\n/view analysis\n/view log\n/view observability",
		))
		return m, nil
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "overview":
		m.workspaceTab = workspaceTabOverview
	case "suggestions":
		m.workspaceTab = workspaceTabSuggestions
	case "result":
		m.workspaceTab = workspaceTabResult
	case "analysis":
		m.workspaceTab = workspaceTabAnalysis
	case "log":
		return m.toggleLogPanel(), nil
	case "observability":
		m.scrollFocus = scrollPaneObservability
		m.statusMsg = localizedText("Observability focused.", "已聚焦可观测性面板。", "Observability focused.")
		return m, nil
	default:
		m.statusMsg = localizedText("Unknown view name.", "未知视图名称。", "Unknown view name.")
		return m, nil
	}
	m.leftScroll = 0
	m.statusMsg = localizedText("Workspace view: ", "工作区视图：", "Workspace view: ") + m.workspaceTab.label()
	return m, nil
}
