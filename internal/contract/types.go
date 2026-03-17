package contract

import "time"

// ProtocolVersion is the schema version of cross-layer DTOs.
const ProtocolVersion = "v3"

// SuggestionItem is the canonical action proposal DTO shared across layers.
type SuggestionItem struct {
	Version string     `json:"version,omitempty"`
	Name    string     `json:"name"`
	Action  ActionSpec `json:"action"`
	Reason  string     `json:"reason"`
}

// ActionSpec describes a single executable action.
type ActionSpec struct {
	Version     string `json:"version,omitempty"`
	Type        string `json:"type"`
	Command     string `json:"command,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	FileContent string `json:"file_content,omitempty"`
	FileOp      string `json:"file_operation,omitempty"` // create, update, delete, append, mkdir
}

// ToolLabel returns a short UI label for an action type.
func (a ActionSpec) ToolLabel() string {
	switch a.Type {
	case "git_command":
		return "GIT"
	case "shell_command":
		return "SHELL"
	case "file_write":
		return "FILE"
	case "file_read":
		return "READ"
	case "github_op":
		return "GITHUB"
	default:
		return "UNKNOWN"
	}
}

// TraceMetadata links flow/runtime/tui events in a single trace.
type TraceMetadata struct {
	TraceID   string `json:"trace_id,omitempty"`
	RoundID   string `json:"round_id,omitempty"`
	AttemptID string `json:"attempt_id,omitempty"`
	SliceID   string `json:"slice_id,omitempty"`
}

type RecoveryActionType string

const (
	RecoveryRetry  RecoveryActionType = "retry"
	RecoverySkip   RecoveryActionType = "skip"
	RecoveryManual RecoveryActionType = "manual"
	RecoveryAbort  RecoveryActionType = "abort"
)

// RecoveryAction standardizes how a failed action can be recovered.
type RecoveryAction struct {
	Type   RecoveryActionType `json:"type"`
	Reason string             `json:"reason,omitempty"`
}

// ActionResult is the canonical execution outcome DTO.
type ActionResult struct {
	Command   string        `json:"command,omitempty"`
	FilePath  string        `json:"file_path,omitempty"`
	Stdout    string        `json:"stdout"`
	Stderr    string        `json:"stderr"`
	ExitCode  int           `json:"exit_code"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	Trace     TraceMetadata `json:"trace,omitempty"`
	RecoverBy RecoveryAction `json:"recover_by,omitempty"`
}

