package git

import "encoding/json"

type FileStatusCode byte

const (
	StatusUnmodified  FileStatusCode = ' '
	StatusModified    FileStatusCode = 'M'
	StatusAdded       FileStatusCode = 'A'
	StatusDeleted     FileStatusCode = 'D'
	StatusRenamed     FileStatusCode = 'R'
	StatusCopied      FileStatusCode = 'C'
	StatusUntracked   FileStatusCode = '?'
	StatusIgnored     FileStatusCode = '!'
	StatusTypeChanged FileStatusCode = 'T'
	StatusUnmerged    FileStatusCode = 'U'
)

type FileStatus struct {
	Path         string
	StagingCode  FileStatusCode
	WorktreeCode FileStatusCode
	OrigPath     string
}

type BranchInfo struct {
	Name       string
	Upstream   string
	Ahead      int
	Behind     int
	IsDetached bool
}

type RemoteInfo struct {
	Name                string
	URL                 string // legacy summary: prefer PushURL, fallback to FetchURL
	FetchURL            string
	PushURL             string
	FetchURLValid       bool
	PushURLValid        bool
	ReachabilityChecked bool
	Reachable           bool
	LastError           string
}

type UpstreamInfo struct {
	Name   string
	Ahead  int
	Behind int
}

type StashEntry struct {
	Index   int
	Message string
}

type SubmoduleInfo struct {
	Name   string
	Path   string
	URL    string
	Commit string
	Status string
}

type RepoConfig struct {
	DefaultBranch string
	MergeStrategy string
}

type RiskLevel int

const (
	RiskSafe RiskLevel = iota
	RiskCaution
	RiskDangerous
)

// InteractionMode defines how a suggestion should be handled by the TUI.
type InteractionMode int

const (
	// AutoExec: command is complete, can be executed directly (git add ., git push)
	AutoExec InteractionMode = iota
	// NeedsInput: command has placeholders that require user input before execution
	NeedsInput
	// InfoOnly: advisory suggestion, no git command to execute (e.g. "create .gitignore file")
	InfoOnly
	// FileWrite: create/overwrite a file with given content (e.g. .gitignore)
	FileWrite
	// CommitMessage: auto-generated commit message that the user can edit/accept/regenerate
	CommitMessage
	// ConflictGuide: step-by-step conflict resolution guide
	ConflictGuide
	// RecoveryGuide: undo/restore decision guide
	RecoveryGuide
	// PlatformExec: execute a platform admin flow through a first-class executor
	PlatformExec
)

// FileWriteInfo describes a file operation (create/update/delete/read).
type FileWriteInfo struct {
	Path      string
	Content   string   // for create/update
	Operation string   // "create", "update", "delete", "read", "append"
	Backup    bool     // create backup before modify/delete
	Lines     []string // for line-based operations
	LineStart int      // for update lines
	LineEnd   int      // for update lines
}

// ExecState tracks per-suggestion execution status inside the TUI.
type ExecState int

const (
	ExecPending ExecState = iota
	ExecRunning
	ExecDone
	ExecFailed
	ExecSkipped
)

// InputField defines a user-input parameter required by a NeedsInput suggestion.
type InputField struct {
	Key          string // optional human-readable placeholder token, e.g. "<url>"
	Label        string // display label for the TUI input prompt
	Placeholder  string // hint text shown in the input box
	ArgIndex     int    // argv index to replace with the user's value
	DefaultValue string // optional initial value shown to the user
}

type PlatformExecInfo struct {
	CapabilityID    string            `json:"capability_id"`
	Flow            string            `json:"flow"` // inspect|mutate|validate|rollback
	Operation       string            `json:"operation,omitempty"`
	ResourceID      string            `json:"resource_id,omitempty"`
	Scope           map[string]string `json:"scope,omitempty"`
	Query           map[string]string `json:"query,omitempty"`
	Payload         json.RawMessage   `json:"payload,omitempty"`
	ValidatePayload json.RawMessage   `json:"validate_payload,omitempty"`
	RollbackPayload json.RawMessage   `json:"rollback_payload,omitempty"`
}

type Suggestion struct {
	ID          string
	Action      string
	Command     []string
	Steps       [][]string // Chained commands for multi-step operations
	Reason      string
	RiskLevel   RiskLevel
	Impact      ImpactPreview
	Source      SuggestionSource
	Confidence  float64
	Interaction InteractionMode // how TUI should handle this suggestion
	Inputs      []InputField    // required inputs for NeedsInput mode
	FileOp      *FileWriteInfo  // non-nil when Interaction == FileWrite
	PlatformOp  *PlatformExecInfo
}

type SuggestionSource int

const (
	SourceLLM SuggestionSource = iota
)

type ImpactPreview struct {
	RiskLevel        RiskLevel
	AffectedFiles    []string
	AffectedBranches []string
	Description      string
}

type ExecutionResult struct {
	Command  []string
	Stdout   string
	Stderr   string
	ExitCode int
	Success  bool
}

type AppError struct {
	Code    string
	Message string
	Detail  string
	Cause   error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}
