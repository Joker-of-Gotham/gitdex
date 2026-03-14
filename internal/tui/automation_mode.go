package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) automationMode() string {
	if strings.TrimSpace(m.automation.Mode) != "" {
		return config.NormalizeAutomationMode(m.automation.Mode)
	}
	return config.AutomationModeFromFlags(m.automation)
}

func (m Model) automationDraftMode() string {
	if strings.TrimSpace(m.automationDraft.Mode) != "" {
		return config.NormalizeAutomationMode(m.automationDraft.Mode)
	}
	return config.AutomationModeFromFlags(m.automationDraft)
}

func localizedAutomationModeLabel(mode string) string {
	switch config.NormalizeAutomationMode(mode) {
	case config.AutomationModeAuto:
		return localizedText("auto", "自动", "auto")
	case config.AutomationModeCruise:
		return localizedText("cruise", "巡航", "cruise")
	default:
		return localizedText("manual", "手动", "manual")
	}
}

func localizedAutomationModeDescription(mode string) string {
	switch config.NormalizeAutomationMode(mode) {
	case config.AutomationModeAuto:
		return localizedText(
			"Auto: fully automatic with zero human intervention. Gitdex loops: analyze → suggest → execute → re-analyze. On error it re-plans and retries. Continues until the repo is clean and any goal is completed.",
			"自动：全自动无需人工介入。gitdex 循环执行：分析→建议→执行→重新分析。遇错则重新规划重试。直到仓库干净且目标完成。",
			"Auto: fully automatic with zero human intervention. Gitdex loops: analyze → suggest → execute → re-analyze. On error it re-plans and retries. Continues until the repo is clean and any goal is completed.",
		)
	case config.AutomationModeCruise:
		return localizedText(
			"Cruise: auto mode plus scheduled self-checks. Gitdex monitors the repo on a timer, creates goals when issues are found, and executes them autonomously.",
			"巡航：在自动模式基础上加入定时自检。gitdex 按定时监控仓库，发现问题时自主创建目标并执行。",
			"Cruise: auto mode plus scheduled self-checks. Gitdex monitors the repo on a timer, creates goals when issues are found, and executes them autonomously.",
		)
	default:
		return localizedText(
			"Manual: you review AI suggestions and approve them with /run accept (one by one) or /run all (batch). Nothing executes without your explicit command.",
			"手动：你审阅 AI 建议，通过 /run accept 逐条批准或 /run all 批量批准。未经你明确指示不会执行任何操作。",
			"Manual: you review AI suggestions and approve them with /run accept (one by one) or /run all (batch). Nothing executes without your explicit command.",
		)
	}
}

func (m Model) automationHasActiveGoal() bool {
	return strings.TrimSpace(m.session.ActiveGoal) != "" || m.workflowPlan != nil
}

func (m Model) shouldAutoAnalyzeOnTick() bool {
	if !(m.automation.Enabled && m.automation.AutoAnalyze) {
		return false
	}
	return true
}

func (m Model) shouldAllowBatchRun() bool {
	for idx, candidate := range m.batchRunCandidates(true) {
		if idx < len(m.suggExecState) && m.suggExecState[idx] != git.ExecPending {
			continue
		}
		return candidate.Runnable
	}
	return false
}

func (m Model) applyCruiseGoalIfNeeded() (Model, tea.Cmd, bool) {
	if !config.AutomationModeAllowsSelfDirectedGoals(m.automationMode()) {
		return m, nil, false
	}
	if m.automationHasActiveGoal() {
		return m, nil, false
	}
	goal := strings.TrimSpace(m.cruiseAuditGoal())
	if goal == "" {
		return m, nil, false
	}
	next, cmd := m.applyActiveGoal(goal)
	updated, ok := next.(Model)
	if !ok {
		return m, cmd, false
	}
	updated = updated.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: localizedText("Cruise mode created an audit goal", "自主巡航已生成巡检目标", "Cruise mode created an audit goal"),
		Detail:  goal,
	})
	return updated, cmd, true
}

func (m Model) automationSummaryText() string {
	mode := m.automationMode()
	return fmt.Sprintf(
		localizedText(
			"Mode: %s | Every %ds | Trusted: %t | Click or run /mode to configure",
			"模式：%s | 每 %d 秒 | 信任：%t | 点击或运行 /mode 进行配置",
			"Mode: %s | Every %ds | Trusted: %t | Click or run /mode to configure",
		),
		localizedAutomationModeLabel(mode),
		m.automation.MonitorInterval,
		m.automation.TrustedMode,
	)
}

func (m Model) automationGoalRequirementText() string {
	mode := m.automationMode()
	switch config.NormalizeAutomationMode(mode) {
	case config.AutomationModeCruise:
		return localizedText(
			"Fully autonomous: scheduled self-checks, self-directed goals, and automatic execution.",
			"完全自治：定时自检、自主设定目标并自动执行。",
			"Fully autonomous: scheduled self-checks, self-directed goals, and automatic execution.",
		)
	case config.AutomationModeAuto:
		return localizedText(
			"Fully automatic: analyze → suggest → execute → re-analyze. Use /goal to focus on a specific objective.",
			"全自动：分析→建议→执行→重新分析。可通过 /goal 聚焦特定目标。",
			"Fully automatic: analyze → suggest → execute → re-analyze. Use /goal to focus on a specific objective.",
		)
	default:
		return localizedText(
			"Manual: review suggestions, then /run accept or /run all to execute.",
			"手动：审阅建议后通过 /run accept 或 /run all 执行。",
			"Manual: review suggestions, then /run accept or /run all to execute.",
		)
	}
}

func (m Model) cruiseAuditGoal() string {
	if m.gitState == nil {
		return localizedText(
			"Audit repository sync state, local changes, and platform health",
			"巡检仓库同步状态、本地改动与平台健康",
			"Audit repository sync state, local changes, and platform health",
		)
	}
	return localizedCruiseGoalForState(m.gitState)
}

func localizedCruiseGoalForState(state *status.GitState) string {
	if state == nil {
		return localizedText(
			"Audit repository sync state, local changes, and platform health",
			"巡检仓库同步状态、本地改动与平台健康",
			"Audit repository sync state, local changes, and platform health",
		)
	}
	branch := strings.TrimSpace(state.LocalBranch.Name)
	if branch == "" {
		branch = localizedText("current branch", "当前分支", "current branch")
	}
	if len(state.WorkingTree) > 0 || len(state.StagingArea) > 0 {
		return localizedText(
			fmt.Sprintf("Review local changes on %s, remote sync status, and deployment/platform health", branch),
			fmt.Sprintf("检查 %s 上的本地改动、远端同步状态以及部署与平台健康", branch),
			fmt.Sprintf("Review local changes on %s, remote sync status, and deployment/platform health", branch),
		)
	}
	return localizedText(
		fmt.Sprintf("Audit %s remote sync, deployments, Pages, and platform health", branch),
		fmt.Sprintf("巡检 %s 的远端同步、部署、Pages 与平台健康", branch),
		fmt.Sprintf("Audit %s remote sync, deployments, Pages, and platform health", branch),
	)
}
