package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) editablePlatformRequest() *git.PlatformExecInfo {
	if m.lastPlatformOp != nil {
		return clonePlatformExecInfo(m.lastPlatformOp)
	}
	return nil
}

func (m Model) openPlatformEdit(req *git.PlatformExecInfo) Model {
	if req == nil {
		return m
	}
	m.screen = screenPlatformEdit
	m.platformTitle = platformActionTitle(req)
	m.platformEdit = prettyPlatformRequest(req)
	m.platformCursor = 0
	m.platformScroll = 0
	revision := 1
	if m.lastPlatform != nil && m.lastPlatform.RequestRevision > 0 {
		revision = m.lastPlatform.RequestRevision + 1
	}
	m.statusMsg = fmt.Sprintf("Edit platform request JSON (revision %d), then press Ctrl+S to run", revision)
	return m.syncPlatformEditor()
}

func prettyPlatformRequest(req *git.PlatformExecInfo) string {
	raw, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func parsePlatformRequest(text string) (*git.PlatformExecInfo, error) {
	var req git.PlatformExecInfo
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &req); err != nil {
		return nil, err
	}
	req.CapabilityID = strings.TrimSpace(req.CapabilityID)
	req.Flow = strings.ToLower(strings.TrimSpace(req.Flow))
	req.Operation = strings.TrimSpace(req.Operation)
	req.ResourceID = strings.TrimSpace(req.ResourceID)
	if req.CapabilityID == "" {
		return nil, fmt.Errorf("capability_id is required")
	}
	if req.Flow == "" {
		return nil, fmt.Errorf("flow is required")
	}
	return &req, nil
}

func (m Model) beginPlatformExecution(req platformExecRequest, summary string) (Model, tea.Cmd) {
	if req.Op == nil {
		m.statusMsg = "Platform request is missing operation metadata"
		return m, nil
	}
	preflight, err := m.preflightPlatformRequest(req)
	if err != nil {
		title := platformActionTitle(req.Op)
		summary, detail := m.summarizePlatformFailure(req.Op, err)
		m.statusMsg = summary
		m.setCommandResponse(localizedAccessTitle(), detail)
		m.lastCommand = commandTrace{
			Title:              title,
			Status:             "platform unavailable",
			Output:             summary,
			At:                 time.Now(),
			ResultKind:         resultKindPlatformAdmin,
			PlatformCapability: strings.TrimSpace(req.Op.CapabilityID),
			PlatformFlow:       strings.TrimSpace(req.Op.Flow),
			PlatformOperation:  strings.TrimSpace(req.Op.Operation),
			PlatformResourceID: strings.TrimSpace(req.Op.ResourceID),
		}
		if m.execSuggIdx >= 0 {
			m = m.markSuggExec(m.execSuggIdx, git.ExecPending, summary)
			m.execSuggIdx = -1
		}
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryCmdFail,
			Summary: "Platform action blocked before execution: " + title,
			Detail:  err.Error(),
		})
		return m, nil
	}
	req = preflight.Request
	lockKey := m.workflowConcurrencyKey(req.Op)
	lockOwner := strings.TrimSpace(platformActionTitle(req.Op))
	if !m.acquireAutomationLock(lockKey, lockOwner) {
		m.statusMsg = "Platform action blocked by concurrency policy"
		if m.execSuggIdx >= 0 {
			m = m.markSuggExec(m.execSuggIdx, git.ExecPending, "concurrency lock active")
			m.execSuggIdx = -1
		}
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryLLMError,
			Summary: "Platform action blocked by concurrency lock",
			Detail:  lockKey,
		})
		return m, nil
	}
	m.lastPlatformOp = clonePlatformExecInfo(req.Op)
	m.statusMsg = summary
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryCmdExec,
		Summary: "Platform action: " + platformActionTitle(req.Op),
		Detail:  git.PlatformExecIdentity(req.Op),
	})
	m.markWorkflowFlowRunning(req.Op)
	m.persistAutomationCheckpoint()
	return m, m.executePlatformRequest(req)
}

func (m Model) syncPlatformEditor() Model {
	line := cursorLine(m.platformEdit, m.platformCursor)
	viewport := maxInt(4, m.height-7)
	if line < m.platformScroll {
		m.platformScroll = line
	}
	if line >= m.platformScroll+viewport {
		m.platformScroll = line - viewport + 1
	}
	if m.platformScroll < 0 {
		m.platformScroll = 0
	}
	return m
}
