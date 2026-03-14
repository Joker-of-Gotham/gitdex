package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) applyDueScheduledAutomation(now time.Time) (Model, bool) {
	if m.automationObserveOnly || !m.automationWithinMaintenanceWindow(now) {
		return m, false
	}
	if m.automationHasActiveGoal() {
		return m, false
	}
	schedule, workflow, ok := m.nextDueSchedule(now)
	if !ok {
		return m, false
	}
	identity := scheduledAutomationIdentity(schedule)
	if identity == "" {
		return m, false
	}
	if m.scheduleLastRun == nil {
		m.scheduleLastRun = map[string]time.Time{}
	}
	m.scheduleLastRun[identity] = now

	goal := strings.TrimSpace(schedule.Goal)
	if goal == "" && workflow != nil {
		goal = strings.TrimSpace(workflow.Goal)
	}
	if goal == "" {
		return m, false
	}

	m.session.ActiveGoal = goal
	if workflow != nil {
		m.workflowPlan = buildWorkflowOrchestration(*workflow, m.gitState)
		m.syncWorkflowFlowFromPlan()
	} else {
		m.workflowPlan = nil
		m.workflowFlow = nil
	}
	m.skipNextAnalysis = false
	m.autoSteps = 0
	m.statusMsg = i18n.T("analysis.in_progress_status")
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: "Scheduled automation activated: " + goal,
		Detail:  scheduledAutomationDetail(schedule, workflow),
	})
	m.rememberOperationEvent("automation scheduled: " + identity)
	m.syncTaskMemory()
	m.persistAutomationCheckpoint()
	return m, true
}

func (m Model) nextDueSchedule(now time.Time) (config.AutomationSchedule, *workflowDefinition, bool) {
	if !m.automation.Enabled || len(m.automation.Schedules) == 0 {
		return config.AutomationSchedule{}, nil, false
	}
	if m.screen != screenMain && m.screen != screenLoading {
		return config.AutomationSchedule{}, nil, false
	}
	if m.execSuggIdx >= 0 {
		return config.AutomationSchedule{}, nil, false
	}

	workflows := m.workflows
	if len(workflows) == 0 {
		workflows = loadWorkflowDefinitions()
	}
	workflowByID := make(map[string]workflowDefinition, len(workflows))
	for _, wf := range workflows {
		workflowByID[strings.TrimSpace(wf.ID)] = wf
	}

	for _, schedule := range m.automation.Schedules {
		if !schedule.Enabled || schedule.Interval <= 0 {
			continue
		}
		identity := scheduledAutomationIdentity(schedule)
		if identity == "" {
			continue
		}
		if last, ok := m.scheduleLastRun[identity]; ok && now.Sub(last) < time.Duration(schedule.Interval)*time.Second {
			continue
		}
		var wf *workflowDefinition
		if strings.TrimSpace(schedule.WorkflowID) != "" {
			candidate, ok := workflowByID[strings.TrimSpace(schedule.WorkflowID)]
			if !ok {
				continue
			}
			wf = &candidate
		}
		return schedule, wf, true
	}
	return config.AutomationSchedule{}, nil, false
}

func scheduledAutomationIdentity(schedule config.AutomationSchedule) string {
	switch {
	case strings.TrimSpace(schedule.ID) != "":
		return strings.TrimSpace(schedule.ID)
	case strings.TrimSpace(schedule.WorkflowID) != "":
		return "workflow:" + strings.TrimSpace(schedule.WorkflowID)
	case strings.TrimSpace(schedule.Goal) != "":
		return "goal:" + strings.TrimSpace(schedule.Goal)
	default:
		return ""
	}
}

func scheduledAutomationDetail(schedule config.AutomationSchedule, workflow *workflowDefinition) string {
	parts := []string{fmt.Sprintf("interval=%ds", schedule.Interval)}
	if workflow != nil {
		parts = append(parts, "workflow="+workflow.ID)
	}
	if goal := strings.TrimSpace(schedule.Goal); goal != "" {
		parts = append(parts, "goal="+goal)
	}
	return strings.Join(parts, " | ")
}
