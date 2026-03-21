package repo

import "time"

type StateLabel string

const (
	Healthy  StateLabel = "healthy"
	Drifting StateLabel = "drifting"
	Blocked  StateLabel = "blocked"
	Degraded StateLabel = "degraded"
	Unknown  StateLabel = "unknown"
)

var stateSeverity = map[StateLabel]int{
	Healthy:  0,
	Unknown:  1,
	Drifting: 2,
	Degraded: 3,
	Blocked:  4,
}

func (s StateLabel) WorseThan(other StateLabel) bool {
	return stateSeverity[s] > stateSeverity[other]
}

func WorstLabel(labels ...StateLabel) StateLabel {
	worst := Healthy
	for _, l := range labels {
		if l.WorseThan(worst) {
			worst = l
		}
	}
	return worst
}

type RiskSeverity string

const (
	RiskHigh   RiskSeverity = "high"
	RiskMedium RiskSeverity = "medium"
	RiskLow    RiskSeverity = "low"
)

type Risk struct {
	Severity    RiskSeverity `json:"severity" yaml:"severity"`
	Description string       `json:"description" yaml:"description"`
	Evidence    string       `json:"evidence" yaml:"evidence"`
	Action      string       `json:"suggested_action" yaml:"suggested_action"`
}

type NextAction struct {
	Action       string `json:"action" yaml:"action"`
	Reason       string `json:"reason" yaml:"reason"`
	RiskLevel    string `json:"risk_level" yaml:"risk_level"`
	EvidenceRefs string `json:"evidence_refs,omitempty" yaml:"evidence_refs,omitempty"`
}

type LocalState struct {
	Label         StateLabel `json:"label" yaml:"label"`
	Branch        string     `json:"branch" yaml:"branch"`
	HeadSHA       string     `json:"head_sha" yaml:"head_sha"`
	IsDetached    bool       `json:"is_detached" yaml:"is_detached"`
	IsClean       bool       `json:"is_clean" yaml:"is_clean"`
	StagedCount   int        `json:"staged_count" yaml:"staged_count"`
	DirtyCount    int        `json:"dirty_count" yaml:"dirty_count"`
	Ahead         int        `json:"ahead" yaml:"ahead"`
	Behind        int        `json:"behind" yaml:"behind"`
	DefaultRemote string     `json:"default_remote,omitempty" yaml:"default_remote,omitempty"`
	Detail        string     `json:"detail,omitempty" yaml:"detail,omitempty"`
}

type RemoteState struct {
	Label         StateLabel `json:"label" yaml:"label"`
	FullName      string     `json:"full_name,omitempty" yaml:"full_name,omitempty"`
	Description   string     `json:"description,omitempty" yaml:"description,omitempty"`
	DefaultBranch string     `json:"default_branch,omitempty" yaml:"default_branch,omitempty"`
	IsPrivate     bool       `json:"is_private" yaml:"is_private"`
	Detail        string     `json:"detail,omitempty" yaml:"detail,omitempty"`
}

type PullRequestSummary struct {
	Number      int      `json:"number" yaml:"number"`
	Title       string   `json:"title" yaml:"title"`
	Author      string   `json:"author" yaml:"author"`
	Labels      []string `json:"labels,omitempty" yaml:"labels,omitempty"`
	IsDraft     bool     `json:"is_draft" yaml:"is_draft"`
	NeedsReview bool     `json:"needs_review" yaml:"needs_review"`
	StaleDays   int      `json:"stale_days,omitempty" yaml:"stale_days,omitempty"`
}

type IssueSummary struct {
	Number    int      `json:"number" yaml:"number"`
	Title     string   `json:"title" yaml:"title"`
	Author    string   `json:"author" yaml:"author"`
	Labels    []string `json:"labels,omitempty" yaml:"labels,omitempty"`
	State     string   `json:"state" yaml:"state"`
	Comments  int      `json:"comments" yaml:"comments"`
	CreatedAt string   `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

type CollaborationSignals struct {
	Label          StateLabel           `json:"label" yaml:"label"`
	OpenPRCount    int                  `json:"open_pr_count" yaml:"open_pr_count"`
	OpenIssueCount int                  `json:"open_issue_count" yaml:"open_issue_count"`
	PullRequests   []PullRequestSummary `json:"pull_requests,omitempty" yaml:"pull_requests,omitempty"`
	Detail         string               `json:"detail,omitempty" yaml:"detail,omitempty"`
}

type WorkflowRunSummary struct {
	RunID      int64  `json:"run_id,omitempty" yaml:"run_id,omitempty"`
	WorkflowID int64  `json:"workflow_id,omitempty" yaml:"workflow_id,omitempty"`
	Name       string `json:"name" yaml:"name"`
	Status     string `json:"status" yaml:"status"`
	Conclusion string `json:"conclusion" yaml:"conclusion"`
	Branch     string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Event      string `json:"event,omitempty" yaml:"event,omitempty"`
	CreatedAt  string `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	URL        string `json:"url,omitempty" yaml:"url,omitempty"`
}

type WorkflowState struct {
	Label  StateLabel           `json:"label" yaml:"label"`
	Runs   []WorkflowRunSummary `json:"runs,omitempty" yaml:"runs,omitempty"`
	Detail string               `json:"detail,omitempty" yaml:"detail,omitempty"`
}

type DeploymentSummary struct {
	ID          int64  `json:"id,omitempty" yaml:"id,omitempty"`
	Environment string `json:"environment" yaml:"environment"`
	State       string `json:"state" yaml:"state"`
	Ref         string `json:"ref,omitempty" yaml:"ref,omitempty"`
	CreatedAt   string `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	URL         string `json:"url,omitempty" yaml:"url,omitempty"`
}

type DeploymentState struct {
	Label       StateLabel          `json:"label" yaml:"label"`
	Deployments []DeploymentSummary `json:"deployments,omitempty" yaml:"deployments,omitempty"`
	Detail      string              `json:"detail,omitempty" yaml:"detail,omitempty"`
}

type RepoContext struct {
	Owner             string            `json:"owner" yaml:"owner"`
	Name              string            `json:"name" yaml:"name"`
	FullName          string            `json:"full_name" yaml:"full_name"`
	LocalPaths        []string          `json:"local_paths,omitempty" yaml:"local_paths,omitempty"`
	IsLocal           bool              `json:"is_local" yaml:"is_local"`
	IsReadOnly        bool              `json:"is_read_only" yaml:"is_read_only"`
	RemoteTopology    map[string]string `json:"remote_topology,omitempty" yaml:"remote_topology,omitempty"` // remoteName -> URL
	IsFork            bool              `json:"is_fork,omitempty" yaml:"is_fork,omitempty"`
	UpstreamURL       string            `json:"upstream_url,omitempty" yaml:"upstream_url,omitempty"`
	DefaultBranch     string            `json:"default_branch,omitempty" yaml:"default_branch,omitempty"`
	ProtectedBranches []string          `json:"protected_branches,omitempty" yaml:"protected_branches,omitempty"`
}

func (rc *RepoContext) LocalPath() string {
	if len(rc.LocalPaths) > 0 {
		return rc.LocalPaths[0]
	}
	return ""
}

type RepoSummary struct {
	Owner         string               `json:"owner" yaml:"owner"`
	Repo          string               `json:"repo" yaml:"repo"`
	OverallLabel  StateLabel           `json:"overall_label" yaml:"overall_label"`
	Timestamp     time.Time            `json:"timestamp" yaml:"timestamp"`
	Local         LocalState           `json:"local" yaml:"local"`
	Remote        RemoteState          `json:"remote" yaml:"remote"`
	Collaboration CollaborationSignals `json:"collaboration" yaml:"collaboration"`
	Workflows     WorkflowState        `json:"workflows" yaml:"workflows"`
	Deployments   DeploymentState      `json:"deployments" yaml:"deployments"`
	Risks         []Risk               `json:"risks,omitempty" yaml:"risks,omitempty"`
	NextActions   []NextAction         `json:"next_actions,omitempty" yaml:"next_actions,omitempty"`
}
