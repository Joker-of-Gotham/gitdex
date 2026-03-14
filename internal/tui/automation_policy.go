package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) shouldRunUnattended() bool {
	mode := m.automationMode()
	if !(m.automation.Enabled && m.automation.Unattended && m.automation.AutoAcceptSafe) {
		return false
	}
	if mode != config.AutomationModeAuto && mode != config.AutomationModeCruise {
		return false
	}
	if m.automationObserveOnly {
		return false
	}
	return m.automationWithinMaintenanceWindow(time.Now())
}

func (m Model) automationWithinMaintenanceWindow(now time.Time) bool {
	windows := m.automation.MaintenanceWindows
	if len(windows) == 0 {
		return true
	}
	day := strings.ToLower(now.Weekday().String())
	minutes := now.Hour()*60 + now.Minute()
	for _, window := range windows {
		if !windowMatchesDay(window.Days, day) {
			continue
		}
		start, okStart := parseWindowMinutes(window.Start)
		end, okEnd := parseWindowMinutes(window.End)
		if !okStart || !okEnd {
			continue
		}
		if start <= end {
			if minutes >= start && minutes <= end {
				return true
			}
			continue
		}
		if minutes >= start || minutes <= end {
			return true
		}
	}
	return false
}

func windowMatchesDay(days []string, current string) bool {
	if len(days) == 0 {
		return true
	}
	for _, day := range days {
		day = strings.ToLower(strings.TrimSpace(day))
		if day == "" {
			continue
		}
		if day == current || strings.HasPrefix(current, day) {
			return true
		}
	}
	return false
}

func parseWindowMinutes(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func (m Model) automationTrustedCapability(capabilityID string) bool {
	if m.automation.TrustedMode {
		return true
	}
	capabilityID = strings.TrimSpace(capabilityID)
	if capabilityID == "" {
		return false
	}
	for _, trusted := range m.automation.TrustPolicy.TrustedCapabilities {
		if strings.EqualFold(strings.TrimSpace(trusted), capabilityID) {
			return true
		}
	}
	return false
}

func (m Model) automationRequiresApproval(meta platform.ExecutionMeta) bool {
	if meta.ApprovalRequired {
		return true
	}
	if m.automation.ApprovalPolicy.RequireForAdapterBacked && meta.Adapter != "" && meta.Adapter != platform.AdapterAPI {
		return true
	}
	if m.automation.ApprovalPolicy.RequireForPartial && meta.Coverage == platform.CoveragePartial {
		return true
	}
	if m.automation.ApprovalPolicy.RequireForComposed && meta.Coverage == platform.CoverageComposed {
		return true
	}
	if m.automation.ApprovalPolicy.RequireForIrreversible && meta.Rollback == platform.RollbackNotSupported {
		return true
	}
	return false
}

func (m Model) shouldAllowAutomationSuggestion(s git.Suggestion) (bool, string) {
	trusted := m.automation.TrustPolicy.AllowDangerousGit || m.automation.TrustedMode

	operatorApproved := false
	if !trusted && s.Interaction == git.PlatformExec && s.PlatformOp != nil {
		if step := m.findWorkflowFlowStep(s.PlatformOp); step != nil && step.Policy.ApprovalRequired {
			operatorApproved = strings.EqualFold(strings.TrimSpace(step.ApprovalState), "approved")
		}
	}

	if trusted || operatorApproved {
		if m.automation.Concurrency.Enabled && s.Interaction == git.PlatformExec && s.PlatformOp != nil {
			if key := m.workflowConcurrencyKey(s.PlatformOp); key != "" {
				if owner, locked := m.automationLocks[key]; locked && owner != "" {
					return false, "concurrency lock active for " + key
				}
			}
		}
		return true, ""
	}

	if !isAutomationSafeSuggestion(s, false) {
		return false, "suggestion is outside safe unattended policy"
	}
	if s.Interaction != git.PlatformExec || s.PlatformOp == nil {
		return true, ""
	}
	diagnostics, _ := platform.DiagnosePlatformOperation(m.detectedPlatform(), m.gitState, clonePlatformExecInfo(s.PlatformOp))
	if diagnostics.Decision == platform.DiagnosticBlocked {
		return false, "diagnostic blocked unattended execution: " + summarizeDiagnostics(diagnostics)
	}
	return true, ""
}

func (m Model) workflowConcurrencyKey(op *git.PlatformExecInfo) string {
	if op == nil {
		return ""
	}
	if step := m.findWorkflowFlowStep(op); step != nil {
		return strings.TrimSpace(step.Concurrency)
	}
	return strings.TrimSpace(git.PlatformExecIdentity(op))
}

func (m *Model) acquireAutomationLock(key, owner string) bool {
	key = strings.TrimSpace(key)
	owner = strings.TrimSpace(owner)
	if key == "" || !m.automation.Concurrency.Enabled {
		return true
	}
	if m.automationLocks == nil {
		m.automationLocks = map[string]string{}
	}
	if current, exists := m.automationLocks[key]; exists && strings.TrimSpace(current) != "" && !strings.EqualFold(current, owner) {
		return false
	}
	m.automationLocks[key] = owner
	m.refreshWorkflowRunState("")
	return true
}

func (m *Model) releaseAutomationLock(key, owner string) {
	key = strings.TrimSpace(key)
	if key == "" || len(m.automationLocks) == 0 {
		return
	}
	current := strings.TrimSpace(m.automationLocks[key])
	if current == "" || strings.EqualFold(current, owner) {
		delete(m.automationLocks, key)
	}
	m.refreshWorkflowRunState("")
}

func (m *Model) clearAutomationLock(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" || len(m.automationLocks) == 0 {
		return false
	}
	if _, ok := m.automationLocks[key]; !ok {
		return false
	}
	delete(m.automationLocks, key)
	m.refreshWorkflowRunState("")
	return true
}

func (m *Model) clearSelectedAutomationLock() (string, bool) {
	if m.workflowFlow != nil {
		if step := m.selectedWorkflowStep(); step != nil {
			key := strings.TrimSpace(step.Policy.ConcurrencyKey)
			if m.clearAutomationLock(key) {
				return key, true
			}
		}
	}
	if len(m.automationLocks) == 1 {
		for key := range m.automationLocks {
			if m.clearAutomationLock(key) {
				return key, true
			}
		}
	}
	return "", false
}

func (m *Model) recordAutomationOutcome(key string, ok bool, failure platform.FailureTaxonomy) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if m.automationFailures == nil {
		m.automationFailures = map[string]int{}
	}
	if ok {
		delete(m.automationFailures, key)
		return
	}
	m.automationFailures[key]++
	threshold := m.automation.Escalation.FailureThreshold
	if threshold <= 0 {
		threshold = 3
	}
	if m.automationFailures[key] >= threshold {
		m.automationObserveOnly = true
		m.lastEscalation = time.Now()
		m.statusMsg = "Automation degraded to observe-only after repeated failures"
		*m = m.addLog(oplog.Entry{
			Type:    oplog.EntryLLMError,
			Summary: "Automation escalation triggered",
			Detail:  fmt.Sprintf("%s reached %d failures (%s)", key, m.automationFailures[key], failure),
		})
	}
}

func (m *Model) recoverAutomationEscalation(reason string) bool {
	if !m.automationObserveOnly && len(m.automationFailures) == 0 {
		return false
	}
	m.automationObserveOnly = false
	if len(m.automationFailures) > 0 {
		m.automationFailures = map[string]int{}
	}
	m.lastRecovery = time.Now()
	m.refreshWorkflowRunState("")
	m.statusMsg = "Automation recovered from observe-only"
	*m = m.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: "Automation recovery path activated",
		Detail:  strings.TrimSpace(firstNonEmpty(reason, "operator cleared observe-only state and failure counters")),
	})
	return true
}

func (m *Model) applyDeadLetterPolicy() {
	if m.workflowFlow == nil {
		return
	}
	threshold := m.automation.DeadLetter.PauseAfter
	if threshold <= 0 {
		threshold = 2
	}
	deadLetters := 0
	for _, step := range m.workflowFlow.Steps {
		if step.Status == workflowFlowDeadLetter {
			deadLetters++
		}
	}
	if deadLetters < threshold {
		return
	}
	m.pauseWorkflowFlow("dead-letter threshold reached")
	m.automationObserveOnly = true
	m.lastEscalation = time.Now()
	*m = m.addLog(oplog.Entry{
		Type:    oplog.EntryLLMError,
		Summary: "Workflow paused after repeated dead-letter failures",
		Detail:  fmt.Sprintf("%d dead-letter step(s)", deadLetters),
	})
}
