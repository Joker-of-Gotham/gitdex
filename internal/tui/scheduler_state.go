package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

const automationStateFileName = "automation-state.json"

const checkpointResumeTTL = 72 * time.Hour

type automationCheckpoint struct {
	UpdatedAt          time.Time                      `json:"updated_at"`
	RepoFingerprint    string                         `json:"repo_fingerprint,omitempty"`
	ActiveGoal         string                         `json:"active_goal,omitempty"`
	Workflow           *prompt.WorkflowOrchestration  `json:"workflow,omitempty"`
	Flow               *workflowFlowState             `json:"flow,omitempty"`
	ScheduleLastRun    map[string]time.Time           `json:"schedule_last_run,omitempty"`
	AutomationLocks    map[string]string              `json:"automation_locks,omitempty"`
	AutomationFailures map[string]int                 `json:"automation_failures,omitempty"`
	ObserveOnly        bool                           `json:"observe_only,omitempty"`
	EscalatedAt        time.Time                      `json:"escalated_at,omitempty"`
	RecoveredAt        time.Time                      `json:"recovered_at,omitempty"`
	Ledger             []platform.MutationLedgerEntry `json:"ledger,omitempty"`
}

func loadAutomationCheckpoint() automationCheckpoint {
	path, err := automationStatePath()
	if err != nil || path == "" {
		return automationCheckpoint{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return automationCheckpoint{}
	}
	var state automationCheckpoint
	if json.Unmarshal(data, &state) != nil {
		return automationCheckpoint{}
	}
	if state.ScheduleLastRun == nil {
		state.ScheduleLastRun = map[string]time.Time{}
	}
	if state.AutomationLocks == nil {
		state.AutomationLocks = map[string]string{}
	}
	if state.AutomationFailures == nil {
		state.AutomationFailures = map[string]int{}
	}
	if !state.UpdatedAt.IsZero() && time.Since(state.UpdatedAt) > checkpointResumeTTL {
		state.ActiveGoal = ""
		state.Workflow = nil
		state.Flow = nil
	}
	return state
}

func (m *Model) persistAutomationCheckpoint() {
	path, err := automationStatePath()
	if err != nil || path == "" {
		return
	}
	state := automationCheckpoint{
		UpdatedAt:          time.Now(),
		RepoFingerprint:    strings.TrimSpace(m.repoFingerprint()),
		ActiveGoal:         m.session.ActiveGoal,
		Workflow:           m.workflowPlan,
		Flow:               m.workflowFlow,
		ScheduleLastRun:    map[string]time.Time{},
		AutomationLocks:    map[string]string{},
		AutomationFailures: map[string]int{},
		ObserveOnly:        m.automationObserveOnly,
		EscalatedAt:        m.lastEscalation,
		RecoveredAt:        m.lastRecovery,
		Ledger:             append([]platform.MutationLedgerEntry(nil), m.mutationLedger...),
	}
	for key, value := range m.scheduleLastRun {
		state.ScheduleLastRun[key] = value
	}
	for key, value := range m.automationLocks {
		state.AutomationLocks[key] = value
	}
	for key, value := range m.automationFailures {
		state.AutomationFailures[key] = value
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	hash := string(data)
	if hash == m.lastCheckpointHash {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
	m.lastCheckpointHash = hash
	m.exportAuditReports()
}

func automationStatePath() (string, error) {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, automationStateFileName), nil
}
