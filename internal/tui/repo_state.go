package tui

import (
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m *Model) reconcileRepoScopedState() {
	current := strings.TrimSpace(m.repoFingerprint())
	if current == "" {
		return
	}
	loaded := strings.TrimSpace(m.loadedCheckpointRepo)
	if loaded == current {
		hasPending := m.pendingCheckpointGoal != "" || m.pendingCheckpointWf != nil || m.pendingCheckpointFlow != nil
		if hasPending {
			if m.pendingCheckpointGoal != "" {
				m.session.ActiveGoal = m.pendingCheckpointGoal
				m.llmGoalStatus = "in_progress"
			}
			if m.pendingCheckpointWf != nil {
				m.workflowPlan = m.pendingCheckpointWf
			}
			if m.pendingCheckpointFlow != nil {
				m.workflowFlow = m.pendingCheckpointFlow
			}
			m.pendingCheckpointGoal = ""
			m.pendingCheckpointWf = nil
			m.pendingCheckpointFlow = nil
		}
		m.syncTaskMemory()
		return
	}
	if loaded == "" {
		m.loadedCheckpointRepo = current
		m.pendingCheckpointGoal = ""
		m.pendingCheckpointWf = nil
		m.pendingCheckpointFlow = nil
		m.restoreGoalFromMemoryStore()
		m.syncTaskMemory()
		return
	}
	m.pendingCheckpointGoal = ""
	m.pendingCheckpointWf = nil
	m.pendingCheckpointFlow = nil
	m.session.ActiveGoal = ""
	m.llmGoalStatus = ""
	m.workflowPlan = nil
	m.workflowFlow = nil
	m.suggestions = nil
	m.suggExecState = nil
	m.suggExecMsg = nil
	m.suggIdx = 0
	m.llmPlanOverview = ""
	m.llmReason = ""
	m.lastPlatform = nil
	m.lastPlatformOp = nil
	m.lastCommand = commandTrace{}
	m.mutationLedger = nil
	m.automationLocks = map[string]string{}
	m.automationFailures = map[string]int{}
	m.automationObserveOnly = false
	m.lastEscalation = time.Time{}
	m.lastRecovery = time.Time{}
	m.lastAnalysisFingerprint = ""
	m.loadedCheckpointRepo = current
	m.restoreGoalFromMemoryStore()
	m.syncTaskMemory()
	m.persistAutomationCheckpoint()
	m.commandResponseTitle = localizedText("Repository context reset", "仓库上下文已重置", "Repository context reset")
	m.commandResponseBody = localizedText(
		"Switched to a different repository. Previous session state has been cleared.",
		"已切换到不同仓库，之前的会话状态已清除。",
		"Switched to a different repository. Previous session state has been cleared.",
	)
	m.statusMsg = localizedText("Repository context reset", "仓库上下文已重置", "Repository context reset")
	updated := m.addLog(oplog.Entry{
		Type:    oplog.EntryStateRefresh,
		Summary: m.statusMsg,
		Detail:  current,
	})
	*m = updated
}

func (m *Model) restoreGoalFromMemoryStore() {
	if m.memoryStore == nil {
		return
	}
	mem := m.memoryStore.ToPromptMemory(m.repoFingerprint())
	if mem != nil && mem.TaskState != nil && strings.TrimSpace(mem.TaskState.Goal) != "" {
		m.session.ActiveGoal = strings.TrimSpace(mem.TaskState.Goal)
		m.llmGoalStatus = "in_progress"
	}
}
