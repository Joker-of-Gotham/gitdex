package views

import ghp "github.com/your-org/gitdex/internal/platform/github"

type StreamChunkMsg struct {
	Content string
	Done    bool
}

type StreamErrorMsg struct {
	Error error
}

type RepoListMsg struct {
	Repos []RepoListItem
}

type RepoSelectMsg struct {
	Repo RepoListItem
}

type CloneRepoRequestMsg struct {
	Repo       RepoListItem
	TargetPath string
}

type CloneProgressMsg struct {
	Repo    string
	Percent int
	Done    bool
	Err     error
}

type CloneRepoResultMsg struct {
	Repo       RepoListItem
	TargetPath string
	Err        error
}

// CloneCompleteMsg is emitted when a background git clone finishes (e.g. dashboard "c" on remote-only).
type CloneCompleteMsg struct {
	Repo       RepoListItem
	TargetPath string
	URL        string
	Err        error
}

type RepoDetailMsg struct {
	Description   string
	Stars         int
	Forks         int
	Language      string
	License       string
	Topics        []string
	DefaultBranch string
	IsPrivate     bool
	CreatedAt     string
	HTMLURL       string
	OpenPRs       int
	OpenIssues    int
}

type RequestPRDetailMsg struct {
	Number int
}

type RequestIssueDetailMsg struct {
	Number int
}

type PRActionKind string

const (
	PRActionComment        PRActionKind = "comment"
	PRActionApprove        PRActionKind = "approve"
	PRActionRequestChanges PRActionKind = "request-changes"
	PRActionMerge          PRActionKind = "merge"
	PRActionClose          PRActionKind = "close"
)

type RequestPRActionMsg struct {
	Number      int
	Kind        PRActionKind
	Body        string
	MergeMethod string
}

type PRActionResultMsg struct {
	Number  int
	Kind    PRActionKind
	Message string
	Err     error
}

type IssueActionKind string

const (
	IssueActionComment IssueActionKind = "comment"
	IssueActionClose   IssueActionKind = "close"
	IssueActionReopen  IssueActionKind = "reopen"
	IssueActionLabel   IssueActionKind = "label"
	IssueActionAssign  IssueActionKind = "assign"
)

type RequestIssueActionMsg struct {
	Number int
	Kind   IssueActionKind
	Body   string
	Values []string
}

type IssueActionResultMsg struct {
	Number  int
	Kind    IssueActionKind
	Message string
	Err     error
}

type WorkflowRunEntry struct {
	RunID      int64
	WorkflowID int64
	Name       string
	Status     string
	Conclusion string
	Branch     string
	Event      string
	CreatedAt  string
	URL        string
}

type WorkflowRunsDataMsg struct {
	Runs []WorkflowRunEntry
}

type WorkflowActionKind string

const (
	WorkflowActionRerun  WorkflowActionKind = "rerun"
	WorkflowActionCancel WorkflowActionKind = "cancel"
)

type RequestWorkflowActionMsg struct {
	RunID int64
	Kind  WorkflowActionKind
}

type WorkflowActionResultMsg struct {
	RunID   int64
	Kind    WorkflowActionKind
	Message string
	Err     error
}

type RequestWorkflowDispatchMsg struct {
	WorkflowID int64
	Ref        string
}

type WorkflowDispatchResultMsg struct {
	WorkflowID int64
	Ref        string
	Message    string
	Err        error
}

type WorkflowSelectedMsg struct {
	Run WorkflowRunEntry
}

type DeploymentEntry struct {
	ID          int64
	Environment string
	State       string
	Ref         string
	CreatedAt   string
	URL         string
}

type DeploymentDataMsg struct {
	Deployments []DeploymentEntry
}

type DeploymentSelectedMsg struct {
	Deployment DeploymentEntry
}

type RequestCommitDetailMsg struct {
	Hash string
}

type CommitActionKind string

const (
	CommitActionCherryPick CommitActionKind = "cherry-pick"
	CommitActionRevert     CommitActionKind = "revert"
)

type RequestCommitActionMsg struct {
	Hash string
	Kind CommitActionKind
}

type CommitActionResultMsg struct {
	Hash    string
	Kind    CommitActionKind
	Message string
	Err     error
}

type ReleaseSelectedMsg struct {
	ID          int64
	TagName     string
	Name        string
	Draft       bool
	Prerelease  bool
	CreatedAt   string
	PublishedAt string
	URL         string
	Body        string
}

type RequestBranchCheckoutMsg struct {
	Name string
}

type BranchActionKind string

const (
	BranchActionCreate BranchActionKind = "create"
	BranchActionRename BranchActionKind = "rename"
	BranchActionDelete BranchActionKind = "delete"
)

type RequestBranchActionMsg struct {
	Kind   BranchActionKind
	Name   string
	Target string
	Force  bool
}

type BranchActionResultMsg struct {
	Kind    BranchActionKind
	Name    string
	Target  string
	Message string
	Err     error
}

type RepoListItem struct {
	Owner         string
	Name          string
	FullName      string
	Description   string
	Language      string
	Stars         int
	UpdatedAt     string
	Fork          bool
	DefaultBranch string
	OpenPRs       int
	OpenIssues    int
	LocalPaths    []string
	IsLocal       bool
}

func (r *RepoListItem) LocalPath() string {
	if len(r.LocalPaths) > 0 {
		return r.LocalPaths[0]
	}
	return ""
}

type FileOpKind string

const (
	FileOpCreateFile FileOpKind = "create-file"
	FileOpCreateDir  FileOpKind = "create-dir"
	FileOpMove       FileOpKind = "move"
	FileOpDelete     FileOpKind = "delete"
)

type RequestFileOpMsg struct {
	Kind   FileOpKind
	Path   string
	Target string
}

type FileOpResultMsg struct {
	Kind   FileOpKind
	Path   string
	Target string
	Err    error
}

// RequestBatchFileOpMsg performs batch filesystem ops on repo-relative paths (multi-select).
type RequestBatchFileOpMsg struct {
	Kind      string // rename|copy|move|delete
	Paths     []string
	Pattern   string // rename: "from -> to" glob patterns
	TargetDir string // copy/move destination (repo-relative directory)
}

// BatchFileOpResultMsg reports batch operation outcome to the files view.
type BatchFileOpResultMsg struct {
	Message string
	Err     error
}

// Entry payloads for git-related list messages (submodules, remotes, conflicts, reflog, rebase).

type SubmoduleEntry struct {
	Name   string
	Path   string
	URL    string
	Status string
	Hash   string
}

type RemoteEntry struct {
	Name     string
	FetchURL string
	PushURL  string
}

type ConflictHunk struct {
	OursContent   string
	TheirsContent string
	BaseContent   string
	Resolution    string // "ours", "theirs", "both", ""
	StartLine     int
}

// --- Submodules

type SubmoduleListMsg struct {
	RepoPath string
	Entries  []SubmoduleEntry
	Err      error
}

type SubmoduleOpResultMsg struct {
	Op      string
	Path    string
	Message string
	Err     error
}

// --- Commit graph

type CommitGraphMsg struct {
	RepoPath string
	Lines    []string
	Err      error
}

// RequestCommitGraphDiffMsg asks the app layer to show or run `git diff` between two commits.
type RequestCommitGraphDiffMsg struct {
	RepoPath string
	A        string
	B        string
}

// --- Remotes

type RemoteListMsg struct {
	RepoPath string
	Entries  []RemoteEntry
	Err      error
}

type RemoteOpResultMsg struct {
	Op      string
	Name    string
	Message string
	Err     error
}

// --- Merge conflicts

type ConflictFileMsg struct {
	RepoPath string
	FilePath string
	Prefix   string
	Between  []string
	Suffix   string
	Hunks    []ConflictHunk
	Err      error
}

type ConflictResolvedMsg struct {
	RepoPath string
	FilePath string
	Message  string
	Err      error
}

// --- Reflog (git reflog) ---

type ReflogListMsg struct {
	RepoPath string
	Entries  []ReflogEntry
	Err      error
}

type ReflogOpKind string

const (
	ReflogOpShowDetail ReflogOpKind = "show-detail"
	ReflogOpReset      ReflogOpKind = "reset"
)

type ReflogResetMode string

const (
	ReflogResetSoft  ReflogResetMode = "soft"
	ReflogResetMixed ReflogResetMode = "mixed"
	ReflogResetHard  ReflogResetMode = "hard"
)

type ReflogOpResultMsg struct {
	Kind    ReflogOpKind
	Hash    string
	Mode    ReflogResetMode
	Message string
	Err     error
}

// --- Interactive rebase ---

type RebaseCommitsMsg struct {
	RepoPath  string
	TargetRef string
	Entries   []RebaseEntry
	Err       error
}

type RebaseResultMsg struct {
	RepoPath  string
	TargetRef string
	Message   string
	Err       error
}

// --- Bisect ---

type BisectActionKind string

const (
	BisectActionStart BisectActionKind = "start"
	BisectActionGood  BisectActionKind = "good"
	BisectActionBad   BisectActionKind = "bad"
	BisectActionSkip  BisectActionKind = "skip"
	BisectActionReset BisectActionKind = "reset"
	BisectActionLog   BisectActionKind = "log"
)

type BisectResultMsg struct {
	Action   BisectActionKind
	Message  string
	Err      error
	LogLines []string

	CurrentHash string
	Remaining   int
	GoodHash    string
	BadHash     string
}

// StashListMsg carries entries from git stash list (see LoadStashCmd).
type StashListMsg struct {
	Entries []StashEntry
	Err     error
}

// StashOpResultMsg reports apply/pop/drop/branch-from-stash outcomes.
type StashOpResultMsg struct {
	Op      string
	Message string
	Err     error
}

// StashDiffMsg carries patch output from git stash show -p.
type StashDiffMsg struct {
	Diff string
	Err  error
}

// TagsListMsg carries tag list from git for-each-ref refs/tags.
type TagsListMsg struct {
	Tags []TagEntry
	Err  error
}

// TagOpResultMsg reports create/delete/push outcomes.
type TagOpResultMsg struct {
	Op      string
	Message string
	Err     error
}

// WorktreeListMsg carries git worktree list --porcelain parsing.
type WorktreeListMsg struct {
	Entries []WorktreeEntry
	Err     error
}

// WorktreeOpResultMsg reports create/remove/lock/unlock outcomes.
type WorktreeOpResultMsg struct {
	Op      string
	Message string
	Err     error
}

// RequestSwitchWorktreeMsg asks the app to focus or open a worktree path.
type RequestSwitchWorktreeMsg struct {
	Path string
}

// --- GitHub Releases (Explorer tab)

// ReleaseListMsg carries releases loaded from the GitHub API.
type ReleaseListMsg struct {
	Releases []*ghp.Release
	Err      error
}

// ReleaseOpKind identifies a mutating release operation.
type ReleaseOpKind string

const (
	ReleaseOpCreate  ReleaseOpKind = "create"
	ReleaseOpUpdate  ReleaseOpKind = "update"
	ReleaseOpPublish ReleaseOpKind = "publish"
	ReleaseOpDelete  ReleaseOpKind = "delete"
)

// RequestReleaseOpMsg asks the app layer to perform a release mutation.
type RequestReleaseOpMsg struct {
	Kind         ReleaseOpKind
	Tag          string
	Name         string
	Body         string
	Draft        bool
	Prerelease   bool
	ReleaseID    int64
	OwnerHint    string
	RepoHint     string
	RepoPathHint string
}

// ReleaseOpResultMsg reports the outcome of a release mutation.
type ReleaseOpResultMsg struct {
	Kind    ReleaseOpKind
	Message string
	Err     error
}

// RequestBranchProtectionMsg asks the app to load branch protection for inspector display.
type RequestBranchProtectionMsg struct {
	Branch string
}

// BranchProtectionDataMsg carries formatted branch protection lines for the inspector.
type BranchProtectionDataMsg struct {
	Branch string
	Lines  []string
	Err    error
}

// WorkspaceStoresMsg carries plans, tasks, and audit rows loaded from storage for the Workspace view.
type WorkspaceStoresMsg struct {
	Plans    []PlanSummary
	Tasks    []TaskItem
	Evidence []EvidenceEntry
	Err      error
}
