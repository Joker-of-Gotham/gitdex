package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/memory"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type FlowReport struct {
	GeneratedAt        time.Time                     `json:"generated_at"`
	ActiveGoal         string                        `json:"active_goal,omitempty"`
	WorkflowID         string                        `json:"workflow_id,omitempty"`
	Label              string                        `json:"label,omitempty"`
	Stage              string                        `json:"stage,omitempty"`
	Health             string                        `json:"health,omitempty"`
	Approval           string                        `json:"approval,omitempty"`
	ApprovalRef        string                        `json:"approval_detail,omitempty"`
	Paused             string                        `json:"paused_reason,omitempty"`
	Selected           int                           `json:"selected_step_index,omitempty"`
	NextRetryAt        time.Time                     `json:"next_retry_at,omitempty"`
	NextRetry          string                        `json:"next_retry_step,omitempty"`
	ObserveOnly        bool                          `json:"observe_only,omitempty"`
	EscalatedAt        time.Time                     `json:"escalated_at,omitempty"`
	RecoveredAt        time.Time                     `json:"recovered_at,omitempty"`
	RecoveryPath       string                        `json:"recovery_path,omitempty"`
	TrustMode          string                        `json:"trust_mode,omitempty"`
	TrustPolicy        string                        `json:"trust_policy,omitempty"`
	ApprovalPol        string                        `json:"approval_policy,omitempty"`
	DeadLetterPol      string                        `json:"dead_letter_policy,omitempty"`
	Maintenance        []string                      `json:"maintenance_windows,omitempty"`
	AutomationFailures map[string]int                `json:"automation_failures,omitempty"`
	ActiveLocks        map[string]string             `json:"active_locks,omitempty"`
	DeadLetter         []DeadLetterEntry             `json:"dead_letter,omitempty"`
	Steps              []FlowReportStep              `json:"steps,omitempty"`
	Boundaries         []platform.CapabilityBoundary `json:"boundaries,omitempty"`
}

type FlowReportStep struct {
	Index      int       `json:"index"`
	Title      string    `json:"title,omitempty"`
	Capability string    `json:"capability,omitempty"`
	Flow       string    `json:"flow,omitempty"`
	Status     string    `json:"status,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
	LastDetail string    `json:"last_detail,omitempty"`
	LedgerRefs []string  `json:"ledger_refs,omitempty"`
}

type AuditExport struct {
	GeneratedAt     time.Time                      `json:"generated_at"`
	Flow            FlowReport                     `json:"flow"`
	Ledger          []platform.MutationLedgerEntry `json:"ledger,omitempty"`
	Memory          memory.MemoryData              `json:"memory"`
	FailureTaxonomy map[string]int                 `json:"failure_taxonomy,omitempty"`
}

type OperatorReport struct {
	GeneratedAt     time.Time                      `json:"generated_at"`
	ObserveOnly     bool                           `json:"observe_only,omitempty"`
	EscalatedAt     time.Time                      `json:"escalated_at,omitempty"`
	RecoveredAt     time.Time                      `json:"recovered_at,omitempty"`
	RecoveryPath    string                         `json:"recovery_path,omitempty"`
	Flow            FlowReport                     `json:"flow"`
	Ledger          []platform.MutationLedgerEntry `json:"ledger,omitempty"`
	FailureTaxonomy map[string]int                 `json:"failure_taxonomy,omitempty"`
	SelectedStep    string                         `json:"selected_step,omitempty"`
}

func (m *Model) exportAuditReports() {
	dir := m.reportExportDir()
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	flow := m.buildFlowReport()
	mem := memory.MemoryData{}
	if m.memoryStore != nil {
		mem = m.memoryStore.Snapshot()
	}
	taxonomy := m.buildFailureTaxonomy()
	operator := m.buildOperatorReport(flow, taxonomy)
	audit := AuditExport{
		GeneratedAt:     time.Now(),
		Flow:            flow,
		Ledger:          append([]platform.MutationLedgerEntry(nil), m.mutationLedger...),
		Memory:          mem,
		FailureTaxonomy: taxonomy,
	}
	hashParts := []string{
		strings.TrimSpace(m.lastCheckpointHash),
		fmt.Sprintf("%d", len(audit.Ledger)),
		fmt.Sprintf("%d", len(audit.Memory.Repos)),
		fmt.Sprintf("%d", len(audit.FailureTaxonomy)),
		valueOr(flow.WorkflowID, flow.Label),
		strings.TrimSpace(flow.ActiveGoal),
	}
	exportHash := strings.Join(hashParts, "|")
	if exportHash == m.lastReportExportHash && time.Since(m.lastReportExportAt) < 5*time.Second {
		return
	}
	writeReportFile(dir, "flow-report", flow, renderFlowReportMarkdown(flow))
	writeReportFile(dir, "platform-mutation-ledger", audit.Ledger, renderLedgerMarkdown(audit.Ledger))
	writeReportFile(dir, "memory-snapshot", mem, renderMemoryMarkdown(mem))
	writeReportFile(dir, "failure-taxonomy", taxonomy, renderFailureTaxonomyMarkdown(taxonomy))
	writeReportFile(dir, "operator-report", operator, renderOperatorReportMarkdown(operator))
	writeReportFile(dir, "audit-export", audit, renderAuditMarkdown(audit))
	m.lastReportExportHash = exportHash
	m.lastReportExportAt = time.Now()
}

func (m Model) reportExportDir() string {
	dir := strings.TrimSpace(m.reportsCfg.ExportDir)
	if dir == "" {
		return ""
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	cwd, err := os.Getwd()
	if err != nil {
		return dir
	}
	return filepath.Join(cwd, dir)
}

func writeReportFile(dir, base string, data any, markdown string) {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, base+".json"), raw, 0o600)
	_ = os.WriteFile(filepath.Join(dir, base+".md"), []byte(markdown), 0o600)
}

func (m Model) buildFlowReport() FlowReport {
	report := FlowReport{
		GeneratedAt: time.Now(),
		ActiveGoal:  strings.TrimSpace(m.session.ActiveGoal),
		Stage:       string(m.workflowStage),
		ObserveOnly: m.automationObserveOnly,
		EscalatedAt: m.lastEscalation,
		RecoveredAt: m.lastRecovery,
	}
	if len(m.automationFailures) > 0 {
		report.AutomationFailures = cloneIntMap(m.automationFailures)
	}
	if report.ObserveOnly || !report.EscalatedAt.IsZero() || len(report.AutomationFailures) > 0 {
		report.RecoveryPath = "H recover-auto -> R resume-step -> X retry-step -> C compensate-step"
	}
	if m.automation.TrustedMode {
		report.TrustMode = "trusted"
	} else {
		report.TrustMode = "untrusted"
	}
	trustSummary := report.TrustMode
	if len(m.automation.TrustPolicy.TrustedCapabilities) > 0 {
		trustSummary += " | caps=" + strings.Join(m.automation.TrustPolicy.TrustedCapabilities, ", ")
	}
	if m.automation.TrustPolicy.AllowDangerousGit {
		trustSummary += " | dangerous_git"
	}
	report.TrustPolicy = trustSummary
	report.ApprovalPol = fmt.Sprintf("partial=%t composed=%t adapter=%t irreversible=%t",
		m.automation.ApprovalPolicy.RequireForPartial,
		m.automation.ApprovalPolicy.RequireForComposed,
		m.automation.ApprovalPolicy.RequireForAdapterBacked,
		m.automation.ApprovalPolicy.RequireForIrreversible,
	)
	report.DeadLetterPol = fmt.Sprintf("pause_after=%d", m.automation.DeadLetter.PauseAfter)
	for _, window := range m.automation.MaintenanceWindows {
		days := strings.Join(window.Days, ",")
		if strings.TrimSpace(days) == "" {
			days = "daily"
		}
		report.Maintenance = append(report.Maintenance, fmt.Sprintf("%s %s-%s", days, window.Start, window.End))
	}
	if m.workflowFlow != nil {
		report.WorkflowID = strings.TrimSpace(m.workflowFlow.WorkflowID)
		report.Label = strings.TrimSpace(m.workflowFlow.WorkflowLabel)
		report.Health = strings.TrimSpace(m.workflowFlow.Health)
		report.Approval = strings.TrimSpace(m.workflowFlow.ApprovalState)
		report.ApprovalRef = strings.TrimSpace(m.workflowFlow.ApprovalDetail)
		report.Paused = strings.TrimSpace(m.workflowFlow.PausedReason)
		report.Selected = m.workflowFlow.SelectedStepIndex
		report.NextRetryAt = m.workflowFlow.NextRetryAt
		report.NextRetry = strings.TrimSpace(m.workflowFlow.NextRetryStep)
		if len(m.workflowFlow.ActiveLocks) > 0 {
			report.ActiveLocks = cloneStringMap(m.workflowFlow.ActiveLocks)
		}
		report.DeadLetter = append([]DeadLetterEntry(nil), m.workflowFlow.DeadLetterEntries...)
		for _, step := range m.workflowFlow.Steps {
			report.Steps = append(report.Steps, FlowReportStep{
				Index:      step.Index,
				Title:      step.Step.Title,
				Capability: step.Step.Capability,
				Flow:       step.Step.Flow,
				Status:     string(step.Status),
				UpdatedAt:  step.UpdatedAt,
				LastDetail: step.LastDetail,
				LedgerRefs: append([]string(nil), step.LedgerRefs...),
			})
		}
	}
	if report.WorkflowID == "" && m.workflowPlan != nil {
		report.WorkflowID = strings.TrimSpace(m.workflowPlan.WorkflowID)
		report.Label = strings.TrimSpace(m.workflowPlan.WorkflowLabel)
	}
	capabilityIDs := make([]string, 0, len(report.Steps))
	for _, step := range report.Steps {
		capabilityIDs = append(capabilityIDs, step.Capability)
	}
	if len(capabilityIDs) == 0 && m.workflowPlan != nil {
		capabilityIDs = append(capabilityIDs, m.workflowPlan.Capabilities...)
	}
	platformID := m.detectedPlatform()
	if platformID == platform.PlatformUnknown {
		platformID = platform.PlatformGitHub
	}
	report.Boundaries = platform.RelevantCapabilityBoundaries(platformID, capabilityIDs)
	return report
}

func (m Model) buildFailureTaxonomy() map[string]int {
	out := map[string]int{}
	for _, entry := range m.mutationLedger {
		key := strings.TrimSpace(string(entry.Failure))
		if key == "" {
			continue
		}
		out[key]++
	}
	if m.workflowFlow != nil {
		for _, step := range m.workflowFlow.Steps {
			if step.Status == workflowFlowDeadLetter {
				out["deadletter"]++
			}
		}
	}
	return out
}

func renderFlowReportMarkdown(report FlowReport) string {
	lines := []string{
		"# Flow Report",
		"",
		fmt.Sprintf("- Generated: %s", report.GeneratedAt.Format(time.RFC3339)),
		fmt.Sprintf("- Goal: %s", valueOr(report.ActiveGoal, "(none)")),
		fmt.Sprintf("- Workflow: %s", valueOr(report.Label, report.WorkflowID)),
		fmt.Sprintf("- Stage: %s", valueOr(report.Stage, "(idle)")),
		fmt.Sprintf("- Health: %s", valueOr(report.Health, "(unknown)")),
		fmt.Sprintf("- Approval: %s", valueOr(report.Approval, "(clear)")),
		fmt.Sprintf("- Observe only: %t", report.ObserveOnly),
		fmt.Sprintf("- Trust mode: %s", valueOr(report.TrustMode, "(unknown)")),
	}
	if !report.EscalatedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("- Escalated: %s", report.EscalatedAt.Format(time.RFC3339)))
	}
	if !report.RecoveredAt.IsZero() {
		lines = append(lines, fmt.Sprintf("- Recovered: %s", report.RecoveredAt.Format(time.RFC3339)))
	}
	if strings.TrimSpace(report.TrustPolicy) != "" {
		lines = append(lines, fmt.Sprintf("- Trust policy: %s", report.TrustPolicy))
	}
	if strings.TrimSpace(report.ApprovalPol) != "" {
		lines = append(lines, fmt.Sprintf("- Approval policy: %s", report.ApprovalPol))
	}
	if strings.TrimSpace(report.DeadLetterPol) != "" {
		lines = append(lines, fmt.Sprintf("- Dead-letter policy: %s", report.DeadLetterPol))
	}
	if len(report.Maintenance) > 0 {
		lines = append(lines, fmt.Sprintf("- Maintenance windows: %s", strings.Join(report.Maintenance, " | ")))
	}
	if strings.TrimSpace(report.ApprovalRef) != "" {
		lines = append(lines, fmt.Sprintf("- Approval detail: %s", report.ApprovalRef))
	}
	if strings.TrimSpace(report.Paused) != "" {
		lines = append(lines, fmt.Sprintf("- Paused reason: %s", report.Paused))
	}
	if !report.NextRetryAt.IsZero() {
		lines = append(lines, fmt.Sprintf("- Next retry: %s (%s)", report.NextRetryAt.Format(time.RFC3339), valueOr(report.NextRetry, "pending step")))
	}
	if len(report.AutomationFailures) > 0 {
		keys := make([]string, 0, len(report.AutomationFailures))
		for key := range report.AutomationFailures {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%d", key, report.AutomationFailures[key]))
		}
		lines = append(lines, fmt.Sprintf("- Automation failures: %s", strings.Join(parts, ", ")))
	}
	if strings.TrimSpace(report.RecoveryPath) != "" {
		lines = append(lines, fmt.Sprintf("- Recovery path: %s", report.RecoveryPath))
	}
	lines = append(lines, "", "## Steps")
	if len(report.Steps) == 0 {
		lines = append(lines, "", "_No workflow steps_")
	} else {
		lines = append(lines, "", "| # | Capability | Flow | Status | Detail |", "| --- | --- | --- | --- | --- |")
		for _, step := range report.Steps {
			lines = append(lines, fmt.Sprintf("| %d | %s | %s | %s | %s |", step.Index+1, valueOr(step.Capability, "-"), valueOr(step.Flow, "-"), valueOr(step.Status, "-"), sanitizeMarkdownCell(step.LastDetail)))
		}
	}
	if len(report.ActiveLocks) > 0 {
		keys := make([]string, 0, len(report.ActiveLocks))
		for key := range report.ActiveLocks {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines = append(lines, "", "## Active Locks")
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- %s: %s", key, valueOr(report.ActiveLocks[key], "(unknown owner)")))
		}
	}
	if len(report.DeadLetter) > 0 {
		lines = append(lines, "", "## Dead Letter")
		for _, item := range report.DeadLetter {
			label := valueOr(item.Identity, fmt.Sprintf("step %d", item.StepIndex+1))
			detail := sanitizeMarkdownCell(item.Reason)
			if item.Acked {
				detail += " | acked"
			}
			if !item.NextRetryAt.IsZero() {
				detail += " | next retry " + item.NextRetryAt.Format(time.RFC3339)
			}
			lines = append(lines, fmt.Sprintf("- %s: %s", label, detail))
		}
	}
	if len(report.Boundaries) > 0 {
		lines = append(lines, "", "## Boundaries", "", "| Capability | Coverage | Reason |", "| --- | --- | --- |")
		for _, boundary := range report.Boundaries {
			lines = append(lines, fmt.Sprintf("| %s | %s | %s |", boundary.CapabilityID, boundary.Mode, sanitizeMarkdownCell(boundary.Reason)))
		}
	}
	return strings.Join(lines, "\n")
}

func renderLedgerMarkdown(entries []platform.MutationLedgerEntry) string {
	lines := []string{"# Platform Mutation Ledger", ""}
	if len(entries) == 0 {
		lines = append(lines, "_No ledger entries_")
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "| Time | Capability | Flow | Operation | Adapter | Adapter Detail | Coverage | Rollback | Diagnostics | Boundary | Summary |", "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |", entry.At.Format(time.RFC3339), entry.CapabilityID, entry.Flow, valueOr(entry.Operation, "-"), adapterDisplayLabel(entry.ExecMeta.Adapter), sanitizeMarkdownCell(adapterMetadataSummary(entry.Metadata)), entry.ExecMeta.Coverage, valueOr(string(entry.ExecMeta.Rollback), "-"), valueOr(string(entry.DiagnosticDecision), "-"), sanitizeMarkdownCell(entry.ExecMeta.BoundaryReason), sanitizeMarkdownCell(entry.Summary)))
	}
	return strings.Join(lines, "\n")
}

func renderMemoryMarkdown(data memory.MemoryData) string {
	lines := []string{"# Memory Snapshot", "", fmt.Sprintf("- Updated: %s", data.UpdatedAt.Format(time.RFC3339))}
	lines = append(lines, "", "## Preferences")
	if len(data.Preferences) == 0 {
		lines = append(lines, "_No preferences_")
	} else {
		keys := make([]string, 0, len(data.Preferences))
		for key := range data.Preferences {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- `%s`: %s", key, data.Preferences[key]))
		}
	}
	lines = append(lines, "", fmt.Sprintf("## Repositories (%d)", len(data.Repos)))
	if len(data.Repos) == 0 {
		lines = append(lines, "_No repository memory_")
		return strings.Join(lines, "\n")
	}
	fingerprints := make([]string, 0, len(data.Repos))
	for key := range data.Repos {
		fingerprints = append(fingerprints, key)
	}
	sort.Strings(fingerprints)
	for _, key := range fingerprints {
		repo := data.Repos[key]
		lines = append(lines, "", "### "+key)
		lines = append(lines, fmt.Sprintf("- Patterns: %d", len(repo.Patterns)))
		lines = append(lines, fmt.Sprintf("- Recent events: %d", len(repo.RecentEvents)))
		lines = append(lines, fmt.Sprintf("- Episodes: %d", len(repo.Episodes)))
		lines = append(lines, fmt.Sprintf("- Semantic facts: %d", len(repo.SemanticFacts)))
		if len(repo.Episodes) > 0 {
			episode := repo.Episodes[len(repo.Episodes)-1]
			lines = append(lines, fmt.Sprintf("- Latest episode: %s [%s]", valueOr(episode.Summary, "(none)"), valueOr(episode.Result, "observed")))
		}
		if len(repo.SemanticFacts) > 0 {
			fact := repo.SemanticFacts[0]
			lines = append(lines, fmt.Sprintf("- Top fact: %s (confidence %.2f)", fact.Fact, fact.Confidence))
		}
		if repo.Task != nil {
			lines = append(lines, fmt.Sprintf("- Task: %s [%s]", valueOr(repo.Task.Goal, "(none)"), valueOr(repo.Task.Status, "-")))
		}
	}
	return strings.Join(lines, "\n")
}

func renderFailureTaxonomyMarkdown(taxonomy map[string]int) string {
	lines := []string{"# Failure Taxonomy", ""}
	if len(taxonomy) == 0 {
		lines = append(lines, "_No failures recorded_")
		return strings.Join(lines, "\n")
	}
	keys := make([]string, 0, len(taxonomy))
	for key := range taxonomy {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines = append(lines, "| Failure | Count |", "| --- | --- |")
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("| %s | %d |", key, taxonomy[key]))
	}
	return strings.Join(lines, "\n")
}

func (m Model) buildOperatorReport(flow FlowReport, taxonomy map[string]int) OperatorReport {
	report := OperatorReport{
		GeneratedAt:     time.Now(),
		ObserveOnly:     m.automationObserveOnly,
		EscalatedAt:     m.lastEscalation,
		RecoveredAt:     m.lastRecovery,
		RecoveryPath:    flow.RecoveryPath,
		Flow:            flow,
		Ledger:          append([]platform.MutationLedgerEntry(nil), m.mutationLedger...),
		FailureTaxonomy: taxonomy,
	}
	if m.workflowFlow != nil && m.workflowFlow.SelectedStepIndex >= 0 && m.workflowFlow.SelectedStepIndex < len(m.workflowFlow.Steps) {
		report.SelectedStep = strings.TrimSpace(firstNonEmpty(
			m.workflowFlow.Steps[m.workflowFlow.SelectedStepIndex].Step.Title,
			m.workflowFlow.Steps[m.workflowFlow.SelectedStepIndex].Identity,
		))
	}
	return report
}

func renderOperatorReportMarkdown(report OperatorReport) string {
	lines := []string{
		"# Operator Report",
		"",
		fmt.Sprintf("- Generated: %s", report.GeneratedAt.Format(time.RFC3339)),
		fmt.Sprintf("- Observe only: %t", report.ObserveOnly),
		fmt.Sprintf("- Workflow: %s", valueOr(report.Flow.Label, report.Flow.WorkflowID)),
		fmt.Sprintf("- Health: %s", valueOr(report.Flow.Health, "(unknown)")),
		fmt.Sprintf("- Approval: %s", valueOr(report.Flow.Approval, "(clear)")),
	}
	if !report.EscalatedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("- Escalated: %s", report.EscalatedAt.Format(time.RFC3339)))
	}
	if !report.RecoveredAt.IsZero() {
		lines = append(lines, fmt.Sprintf("- Recovered: %s", report.RecoveredAt.Format(time.RFC3339)))
	}
	if strings.TrimSpace(report.SelectedStep) != "" {
		lines = append(lines, fmt.Sprintf("- Selected step: %s", report.SelectedStep))
	}
	if strings.TrimSpace(report.Flow.ApprovalRef) != "" {
		lines = append(lines, fmt.Sprintf("- Approval detail: %s", report.Flow.ApprovalRef))
	}
	if !report.Flow.NextRetryAt.IsZero() {
		lines = append(lines, fmt.Sprintf("- Next retry: %s (%s)", report.Flow.NextRetryAt.Format(time.RFC3339), valueOr(report.Flow.NextRetry, "pending step")))
	}
	if len(report.Flow.AutomationFailures) > 0 {
		keys := make([]string, 0, len(report.Flow.AutomationFailures))
		for key := range report.Flow.AutomationFailures {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- Automation failure counter %s: %d", key, report.Flow.AutomationFailures[key]))
		}
	}
	if strings.TrimSpace(report.RecoveryPath) != "" {
		lines = append(lines, fmt.Sprintf("- Recovery path: %s", report.RecoveryPath))
	}
	if len(report.Flow.Boundaries) > 0 {
		lines = append(lines, "", "## Capability Boundaries")
		for _, boundary := range report.Flow.Boundaries {
			lines = append(lines, fmt.Sprintf("- %s: %s | %s", boundary.CapabilityID, boundary.Mode, boundary.Reason))
		}
	}
	if len(report.FailureTaxonomy) > 0 {
		lines = append(lines, "", "## Failure Taxonomy")
		keys := make([]string, 0, len(report.FailureTaxonomy))
		for key := range report.FailureTaxonomy {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- %s: %d", key, report.FailureTaxonomy[key]))
		}
	}
	if len(report.Ledger) > 0 {
		lines = append(lines, "", "## Ledger")
		for _, entry := range report.Ledger {
			detail := adapterMetadataSummary(entry.Metadata)
			if detail != "" {
				detail = " | adapter_detail=" + detail
			}
			lines = append(lines, fmt.Sprintf("- %s %s/%s %s | adapter=%s rollback=%s diagnostics=%s%s", entry.At.Format(time.RFC3339), entry.CapabilityID, entry.Flow, valueOr(entry.Operation, "-"), adapterDisplayLabel(entry.ExecMeta.Adapter), entry.ExecMeta.Rollback, valueOr(string(entry.DiagnosticDecision), "allow"), detail))
		}
	}
	return strings.Join(lines, "\n")
}

func renderAuditMarkdown(audit AuditExport) string {
	lines := []string{
		"# Audit Export",
		"",
		fmt.Sprintf("- Generated: %s", audit.GeneratedAt.Format(time.RFC3339)),
		fmt.Sprintf("- Flow steps: %d", len(audit.Flow.Steps)),
		fmt.Sprintf("- Ledger entries: %d", len(audit.Ledger)),
		fmt.Sprintf("- Repo memories: %d", len(audit.Memory.Repos)),
	}
	if len(audit.FailureTaxonomy) > 0 {
		lines = append(lines, "", "## Failure Taxonomy")
		keys := make([]string, 0, len(audit.FailureTaxonomy))
		for key := range audit.FailureTaxonomy {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- %s: %d", key, audit.FailureTaxonomy[key]))
		}
	}
	return strings.Join(lines, "\n")
}

func sanitizeMarkdownCell(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "|", `\|`)
	v = strings.ReplaceAll(v, "\n", "<br>")
	if v == "" {
		return "-"
	}
	return v
}

func cloneIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func adapterMetadataSummary(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	parts := make([]string, 0, 4)
	if binary := strings.TrimSpace(metadata["adapter_binary"]); binary != "" {
		parts = append(parts, "binary="+binary)
	}
	if driver := strings.TrimSpace(metadata["browser_driver"]); driver != "" {
		parts = append(parts, "driver="+driver)
	}
	if manual := strings.TrimSpace(metadata["manual_completion_required"]); manual != "" {
		parts = append(parts, "manual="+manual)
	}
	if validation := strings.TrimSpace(metadata["operator_validation_required"]); validation != "" {
		parts = append(parts, "operator_validation="+validation)
	}
	return strings.Join(parts, ", ")
}
