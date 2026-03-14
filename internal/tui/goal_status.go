package tui

import "strings"

func normalizeGoalStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "unknown":
		return ""
	case "in_progress", "running", "active", "pending":
		return "in_progress"
	case "completed", "done", "resolved", "success":
		return "completed"
	case "blocked", "failed", "error":
		return "blocked"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func (m Model) currentGoalStatus() string {
	if status := normalizeGoalStatus(m.llmGoalStatus); status != "" {
		return status
	}
	if strings.TrimSpace(m.session.ActiveGoal) != "" {
		return "in_progress"
	}
	return ""
}

func localizedGoalStatusText(status string) string {
	switch normalizeGoalStatus(status) {
	case "in_progress":
		return localizedText("in progress", "进行中", "in progress")
	case "completed":
		return localizedText("completed", "已完成", "completed")
	case "blocked":
		return localizedText("blocked", "已阻塞", "blocked")
	default:
		return strings.TrimSpace(status)
	}
}
