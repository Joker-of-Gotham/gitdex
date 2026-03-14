package tui

import "strings"

type slashCommandSpec struct {
	Command     string
	Template    string
	Description string
}

func slashCommandSpecs() []slashCommandSpec {
	return []slashCommandSpec{
		{Command: "help", Template: "/help", Description: localizedText("Show the command palette.", "显示命令面板。", "Show the command palette.")},
		{Command: "goal <text>", Template: "/goal ", Description: localizedText("Set the active goal in natural language.", "用自然语言设置当前目标。", "Set the active goal in natural language.")},
		{Command: "mode manual|assist|auto|cruise", Template: "/mode ", Description: localizedText("Choose the automation operating mode.", "选择自动化运行模式。", "Choose the automation operating mode.")},
		{Command: "interval <seconds>", Template: "/interval ", Description: localizedText("Set the background audit interval.", "设置后台巡检间隔。", "Set the background audit interval.")},
		{Command: "trust on|off", Template: "/trust ", Description: localizedText("Toggle trusted unattended execution.", "切换可信无人值守执行。", "Toggle trusted unattended execution.")},
		{Command: "run accept", Template: "/run accept", Description: localizedText("Accept the selected suggestion.", "接受当前选中的建议。", "Accept the selected suggestion.")},
		{Command: "run all", Template: "/run all", Description: localizedText("Batch-run all currently eligible suggestions.", "批量执行当前可执行的建议。", "Batch-run all currently eligible suggestions.")},
		{Command: "run skip", Template: "/run skip", Description: localizedText("Skip the selected suggestion.", "跳过当前选中的建议。", "Skip the selected suggestion.")},
		{Command: "run why", Template: "/run why", Description: localizedText("Show why the selected suggestion exists.", "查看当前建议的原因。", "Show why the selected suggestion exists.")},
		{Command: "run refresh", Template: "/run refresh", Description: localizedText("Refresh repository and platform state.", "刷新仓库与平台状态。", "Refresh repository and platform state.")},
		{Command: "view overview|suggestions|result|analysis|log|observability", Template: "/view ", Description: localizedText("Switch the main workspace view.", "切换主工作区视图。", "Switch the main workspace view.")},
		{Command: "config status", Template: "/config status", Description: localizedText("Show LLM, automation, and platform readiness.", "查看 LLM、自动化与平台就绪状态。", "Show LLM, automation, and platform readiness.")},
		{Command: "config platform", Template: "/config platform", Description: localizedText("Show platform route readiness and next steps.", "查看平台访问路径与下一步操作。", "Show platform route readiness and next steps.")},
		{Command: "config llm", Template: "/config llm", Description: localizedText("Open the LLM setup flow.", "打开 LLM 设置流程。", "Open the LLM setup flow.")},
		{Command: "config provider", Template: "/provider", Description: localizedText("Open provider credentials and endpoint setup.", "打开 provider 凭证与端点设置。", "Open provider credentials and endpoint setup.")},
		{Command: "config automation", Template: "/settings", Description: localizedText("Open automation settings.", "打开自动化设置。", "Open automation settings.")},
		{Command: "settings", Template: "/settings", Description: localizedText("Alias: open automation settings.", "别名：打开自动化设置。", "Alias: open automation settings.")},
		{Command: "workflow", Template: "/workflow", Description: localizedText("Open workflow presets.", "打开工作流预设。", "Open workflow presets.")},
		{Command: "language auto|en|zh|ja", Template: "/language ", Description: localizedText("Switch the UI language immediately.", "立即切换界面语言。", "Switch the UI language immediately.")},
		{Command: "quit", Template: "/quit", Description: localizedText("Quit gitdex.", "退出 gitdex。", "Quit gitdex.")},
	}
}

func normalizeSlashQuery(input string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "/")))
}

func (m Model) slashCommandSuggestions() []slashCommandSpec {
	if !m.composerFocused {
		return nil
	}
	trimmed := strings.TrimSpace(m.composerInput)
	if !strings.HasPrefix(trimmed, "/") {
		return nil
	}

	query := normalizeSlashQuery(trimmed)
	specs := slashCommandSpecs()
	if query == "" {
		return limitSlashSuggestions(specs, 10)
	}

	prefixMatches := make([]slashCommandSpec, 0, len(specs))
	containsMatches := make([]slashCommandSpec, 0, len(specs))
	for _, spec := range specs {
		command := strings.ToLower(strings.TrimSpace(spec.Command))
		template := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(spec.Template, "/")))
		description := strings.ToLower(strings.TrimSpace(spec.Description))
		switch {
		case strings.HasPrefix(command, query), strings.HasPrefix(template, query):
			prefixMatches = append(prefixMatches, spec)
		case strings.Contains(command, query), strings.Contains(description, query):
			containsMatches = append(containsMatches, spec)
		}
	}
	return limitSlashSuggestions(append(prefixMatches, containsMatches...), 10)
}

func limitSlashSuggestions(specs []slashCommandSpec, max int) []slashCommandSpec {
	if max <= 0 || len(specs) <= max {
		return specs
	}
	return append([]slashCommandSpec(nil), specs[:max]...)
}

func (m *Model) clampSlashCursor() {
	suggestions := m.slashCommandSuggestions()
	if len(suggestions) == 0 {
		m.slashCursor = 0
		return
	}
	if m.slashCursor < 0 {
		m.slashCursor = 0
	}
	if m.slashCursor >= len(suggestions) {
		m.slashCursor = len(suggestions) - 1
	}
}

func (m *Model) moveSlashCursor(delta int) bool {
	suggestions := m.slashCommandSuggestions()
	if len(suggestions) == 0 {
		m.slashCursor = 0
		return false
	}
	m.slashCursor += delta
	if m.slashCursor < 0 {
		m.slashCursor = 0
	}
	if m.slashCursor >= len(suggestions) {
		m.slashCursor = len(suggestions) - 1
	}
	return true
}

func (m Model) selectedSlashCommand() (slashCommandSpec, bool) {
	suggestions := m.slashCommandSuggestions()
	if len(suggestions) == 0 {
		return slashCommandSpec{}, false
	}
	idx := m.slashCursor
	if idx < 0 {
		idx = 0
	}
	if idx >= len(suggestions) {
		idx = len(suggestions) - 1
	}
	return suggestions[idx], true
}

func (m *Model) applySlashSuggestion(index int) bool {
	suggestions := m.slashCommandSuggestions()
	if len(suggestions) == 0 {
		return false
	}
	if index < 0 || index >= len(suggestions) {
		index = 0
	}
	spec := suggestions[index]
	m.composerInput = spec.Template
	m.composerCursor = runeLen(m.composerInput)
	m.slashCursor = index
	m.composerFocused = true
	m.statusMsg = localizedText("Command selected: ", "已选择命令：", "Command selected: ") + spec.Template
	return true
}
