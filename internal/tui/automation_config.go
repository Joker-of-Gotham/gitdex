package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

type automationField int

const (
	automationFieldMode automationField = iota
	automationFieldMonitorInterval
	automationFieldTrustedMode
	automationFieldMaxAutoSteps
)

func automationFields() []automationField {
	return []automationField{
		automationFieldMode,
		automationFieldMonitorInterval,
		automationFieldTrustedMode,
		automationFieldMaxAutoSteps,
	}
}

func automationModeChoices() []string {
	return []string{
		config.AutomationModeManual,
		config.AutomationModeAssist,
		config.AutomationModeAuto,
		config.AutomationModeCruise,
	}
}

func (m Model) openAutomationConfig() Model {
	m.screen = screenAutomationConfig
	m.automationDraft = m.automation
	config.ApplyAutomationMode(&m.automationDraft)
	m.automationField = automationFieldMode
	m.composerFocused = false
	m.statusMsg = localizedText(
		"Adjust automation mode and cadence, then press Enter to save.",
		"调整自动化模式与节奏后，按 Enter 保存。",
		"Adjust automation mode and cadence, then press Enter to save.",
	)
	m.setCommandResponse(localizedAutomationTitle(), strings.Join([]string{
		m.automationSummaryText(),
		m.automationGoalRequirementText(),
		localizedAutomationModeDescription(m.automationMode()),
	}, "\n"))
	return m
}

func (m Model) automationFieldLabel(field automationField) string {
	switch field {
	case automationFieldMode:
		return localizedText("Automation mode", "自动化模式", "Automation mode")
	case automationFieldMonitorInterval:
		return localizedText("Check interval", "检查间隔", "Check interval")
	case automationFieldTrustedMode:
		return localizedText("Trusted mode", "信任模式", "Trusted mode")
	case automationFieldMaxAutoSteps:
		return localizedText("Max auto steps", "最大自动步数", "Max auto steps")
	default:
		return localizedText("Field", "字段", "Field")
	}
}

func (m Model) automationFieldHelp(field automationField) string {
	switch field {
	case automationFieldMode:
		return localizedText(
			"manual = human approves one by one; assist = AI keeps suggestions fresh and you batch-approve; auto = AI executes your goal; cruise = AI also creates scheduled audit goals.",
			"manual = 逐条人工批准；assist = AI 持续刷新建议并由你批量批准；auto = AI 自动执行你的目标；cruise = AI 还会按周期自主生成巡检目标。",
			"manual = human approves one by one; assist = AI keeps suggestions fresh and you batch-approve; auto = AI executes your goal; cruise = AI also creates scheduled audit goals.",
		)
	case automationFieldMonitorInterval:
		return localizedText(
			"Longer intervals reduce token usage. Cruise uses this cadence for full repo and platform audits.",
			"间隔越长，token 消耗越低。cruise 会按这个周期进行完整仓库与平台巡检。",
			"Longer intervals reduce token usage. Cruise uses this cadence for full repo and platform audits.",
		)
	case automationFieldTrustedMode:
		return localizedText(
			"Trusted mode broadens what unattended execution may mutate after policy checks pass.",
			"信任模式会在策略检查通过后，放宽无人值守可执行的变更范围。",
			"Trusted mode broadens what unattended execution may mutate after policy checks pass.",
		)
	case automationFieldMaxAutoSteps:
		return localizedText(
			"Caps how many unattended or batch-run steps gitdex may execute before yielding control.",
			"限制 gitdex 在一次无人值守或批量执行中，最多执行多少步后主动让出控制权。",
			"Caps how many unattended or batch-run steps gitdex may execute before yielding control.",
		)
	default:
		return ""
	}
}

func (m Model) automationFieldValue(field automationField) string {
	switch field {
	case automationFieldMode:
		return localizedAutomationModeLabel(m.automationDraftMode())
	case automationFieldMonitorInterval:
		return fmt.Sprintf(localizedText("%d seconds", "%d 秒", "%d seconds"), maxInt(30, m.automationDraft.MonitorInterval))
	case automationFieldTrustedMode:
		if m.automationDraft.TrustedMode {
			return localizedText("on", "开启", "on")
		}
		return localizedText("off", "关闭", "off")
	case automationFieldMaxAutoSteps:
		return fmt.Sprintf(localizedText("%d steps", "%d 步", "%d steps"), maxInt(1, m.automationDraft.MaxAutoSteps))
	default:
		return ""
	}
}

func (m *Model) adjustAutomationField(delta int) {
	switch m.automationField {
	case automationFieldMode:
		choices := automationModeChoices()
		current := m.automationDraftMode()
		index := 0
		for i, choice := range choices {
			if choice == current {
				index = i
				break
			}
		}
		index = (index + delta + len(choices)) % len(choices)
		m.automationDraft.Mode = choices[index]
		config.ApplyAutomationMode(&m.automationDraft)
	case automationFieldMonitorInterval:
		next := m.automationDraft.MonitorInterval + (delta * 300)
		if next < 60 {
			next = 60
		}
		if next > 21600 {
			next = 21600
		}
		m.automationDraft.MonitorInterval = next
	case automationFieldTrustedMode:
		if delta != 0 {
			m.automationDraft.TrustedMode = delta > 0
		}
	case automationFieldMaxAutoSteps:
		next := m.automationDraft.MaxAutoSteps + delta
		if next < 1 {
			next = 1
		}
		if next > 64 {
			next = 64
		}
		m.automationDraft.MaxAutoSteps = next
	}
}

func (m *Model) toggleAutomationField() {
	switch m.automationField {
	case automationFieldMode:
		m.adjustAutomationField(1)
	case automationFieldTrustedMode:
		m.automationDraft.TrustedMode = !m.automationDraft.TrustedMode
	}
}

func (m Model) updateAutomationConfig(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "escape", "esc":
		m.screen = screenMain
		m.statusMsg = localizedText("Automation settings cancelled.", "已取消自动化设置。", "Automation settings cancelled.")
		return m, nil
	case "up", "shift+tab":
		fields := automationFields()
		for i, field := range fields {
			if field == m.automationField && i > 0 {
				m.automationField = fields[i-1]
				return m, nil
			}
		}
		return m, nil
	case "down", "tab":
		fields := automationFields()
		for i, field := range fields {
			if field == m.automationField && i < len(fields)-1 {
				m.automationField = fields[i+1]
				return m, nil
			}
		}
		return m, nil
	case "left":
		m.adjustAutomationField(-1)
		return m, nil
	case "right":
		m.adjustAutomationField(1)
		return m, nil
	case " ":
		m.toggleAutomationField()
		return m, nil
	case "enter":
		return m.persistAutomationState(m.automationDraft)
	default:
		return m, nil
	}
}

func (m Model) persistAutomationState(next config.AutomationConfig) (tea.Model, tea.Cmd) {
	config.ApplyAutomationMode(&next)
	current := config.Get()
	if current == nil {
		current = config.DefaultConfig()
	}
	cfg := *current
	cfg.Automation = next
	if err := config.SaveGlobal(&cfg); err != nil {
		m.statusMsg = localizedText("Failed to save automation settings: ", "保存自动化设置失败：", "Failed to save automation settings: ") + err.Error()
		return m, nil
	}

	config.Set(&cfg)
	m.automation = next
	m.automationDraft = next
	m.screen = screenMain
	m.batchRunRequested = false
	m.statusMsg = fmt.Sprintf(
		localizedText(
			"Automation updated: mode=%s interval=%ds trusted=%t max-steps=%d",
			"自动化已更新：mode=%s interval=%ds trusted=%t max-steps=%d",
			"Automation updated: mode=%s interval=%ds trusted=%t max-steps=%d",
		),
		localizedAutomationModeLabel(next.Mode),
		next.MonitorInterval,
		next.TrustedMode,
		next.MaxAutoSteps,
	)
	m.setCommandResponse(localizedAutomationTitle(), strings.Join([]string{
		m.automationSummaryText(),
		m.automationGoalRequirementText(),
		localizedAutomationModeDescription(next.Mode),
	}, "\n"))
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: localizedText("Automation settings updated", "自动化设置已更新", "Automation settings updated"),
		Detail:  fmt.Sprintf("mode=%s interval=%ds trusted=%t max_steps=%d", next.Mode, next.MonitorInterval, next.TrustedMode, next.MaxAutoSteps),
	})
	m.persistAutomationCheckpoint()
	if next.Enabled && next.MonitorInterval > 0 {
		return m, scheduleAutomationTick(time.Duration(next.MonitorInterval) * time.Second)
	}
	return m, nil
}

func parseOnOff(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "yes":
		return true, true
	case "off", "false", "no":
		return false, true
	default:
		return false, false
	}
}

func parsePositiveInt(value string, min int) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < min {
		return 0, false
	}
	return n, true
}
