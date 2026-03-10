package oplog

import (
	"strings"
	"time"
)

// EntryType describes the kind of timeline event.
type EntryType string

const (
	EntryLLMStart     EntryType = "llm_start"
	EntryLLMOutput    EntryType = "llm_output"
	EntryLLMError     EntryType = "llm_error"
	EntryCmdExec      EntryType = "cmd_exec"
	EntryCmdSuccess   EntryType = "cmd_success"
	EntryCmdFail      EntryType = "cmd_fail"
	EntryStateRefresh EntryType = "state_refresh"
	EntryUserAction   EntryType = "user_action"
)

// Entry is a single operation-log timeline item.
type Entry struct {
	Timestamp time.Time
	Type      EntryType
	Summary   string
	Detail    string
}

// Normalized ensures empty fields have safe defaults.
func (e Entry) Normalized(now time.Time) Entry {
	if e.Timestamp.IsZero() {
		e.Timestamp = now
	}
	e.Summary = strings.TrimSpace(e.Summary)
	e.Detail = strings.TrimSpace(e.Detail)
	return e
}

// Icon returns a compact symbol for timeline display.
func (e Entry) Icon() string {
	switch e.Type {
	case EntryLLMStart:
		return "[AI]"
	case EntryLLMOutput:
		return "[out]"
	case EntryLLMError:
		return "[err]"
	case EntryCmdExec:
		return "[run]"
	case EntryCmdSuccess:
		return "[ok]"
	case EntryCmdFail:
		return "[x]"
	case EntryStateRefresh:
		return "[sync]"
	case EntryUserAction:
		return "[ui]"
	default:
		return "[.]"
	}
}
