// Package ports defines the stable interfaces (hexagonal architecture ports)
// that connect domain logic to infrastructure adapters. All dependencies
// point inward: TUI -> Application -> Domain <- Infrastructure.
package ports

import (
	"context"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/contract"
)

// GitPort abstracts Git CLI operations.
type GitPort interface {
	Exec(ctx context.Context, args ...string) (stdout, stderr string, err error)
	StatusPorcelain(ctx context.Context) (string, error)
	CurrentBranch(ctx context.Context) (string, error)
	BranchList(ctx context.Context) ([]BranchInfo, error)
	CommitLog(ctx context.Context, count int) ([]CommitInfo, error)
	StashList(ctx context.Context) ([]StashEntry, error)
	RemoteList(ctx context.Context) ([]RemoteInfo, error)
	DiffStat(ctx context.Context, ref string) (string, error)
}

// GitHubPort abstracts GitHub CLI operations.
type GitHubPort interface {
	Exec(ctx context.Context, args ...string) (stdout, stderr string, err error)
	IsAuthenticated(ctx context.Context) bool
	ListIssues(ctx context.Context, state string, limit int) ([]IssueInfo, error)
	ListPRs(ctx context.Context, state string, limit int) ([]PRInfo, error)
	ListNotifications(ctx context.Context, limit int) ([]NotificationInfo, error)
	ListReleases(ctx context.Context, limit int) ([]ReleaseInfo, error)
	ListWorkflowRuns(ctx context.Context, limit int) ([]WorkflowRunInfo, error)
}

// FileSystemPort abstracts file operations within the repo.
type FileSystemPort interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path, content string) error
	AppendFile(path, content string) error
	DeleteFile(path string) error
	MkdirAll(path string) error
	FileExists(path string) bool
	ListDir(path string) ([]string, error)
	ResolveInsideRepo(path string) (string, error)
}

// LLMPort abstracts LLM provider operations.
type LLMPort interface {
	Generate(ctx context.Context, prompt string) (string, error)
	Name() string
	Model() string
	IsAvailable() bool
	ContextLength() int
}

// ClockPort abstracts time for testability.
type ClockPort interface {
	Now() time.Time
	Since(t time.Time) time.Duration
}

// LoggerPort abstracts structured logging.
type LoggerPort interface {
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	Debug(msg string, fields ...any)
	WithTraceID(traceID string) LoggerPort
}

// PlannerPort defines the planning boundary.
type PlannerPort interface {
	Plan(ctx context.Context, payload contract.PlannerPayload) (contract.PlannerResult, error)
}

// ExecutorPort defines the execution boundary.
type ExecutorPort interface {
	Execute(ctx context.Context, seqID int, item contract.SuggestionItem) *contract.ActionResult
	ResetRound()
}

// --- Data Types ---

// BranchInfo describes a git branch.
type BranchInfo struct {
	Name       string
	IsCurrent  bool
	Upstream   string
	Ahead      int
	Behind     int
	LastCommit string
}

// CommitInfo describes a git commit.
type CommitInfo struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
}

// StashEntry describes a git stash entry.
type StashEntry struct {
	Index   int
	Message string
	Date    time.Time
}

// RemoteInfo describes a git remote.
type RemoteInfo struct {
	Name     string
	FetchURL string
	PushURL  string
}

// IssueInfo describes a GitHub issue.
type IssueInfo struct {
	Number    int
	State     string
	Title     string
	Author    string
	Labels    []string
	Assignees []string
	Updated   time.Time
	URL       string
}

// PRInfo describes a GitHub pull request.
type PRInfo struct {
	Number   int
	State    string
	Title    string
	Author   string
	CI       string
	Reviews  int
	Updated  time.Time
	URL      string
	HeadRef  string
	BaseRef  string
	IsDraft  bool
	Merged   bool
}

// NotificationInfo describes a GitHub notification.
type NotificationInfo struct {
	Type    string
	Title   string
	Repo    string
	Updated time.Time
	Reason  string
	URL     string
	Unread  bool
}

// ReleaseInfo describes a GitHub release.
type ReleaseInfo struct {
	Tag        string
	Name       string
	Date       time.Time
	Draft      bool
	PreRelease bool
	URL        string
}

// WorkflowRunInfo describes a GitHub Actions workflow run.
type WorkflowRunInfo struct {
	Workflow string
	Status   string
	Branch   string
	Event    string
	Duration time.Duration
	Updated  time.Time
	URL      string
}
