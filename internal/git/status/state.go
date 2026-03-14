package status

import (
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

// BranchDetail holds rich per-branch metadata for LLM context.
type BranchDetail struct {
	Name       string `json:"name"`
	Upstream   string `json:"upstream,omitempty"`
	Ahead      int    `json:"ahead,omitempty"`
	Behind     int    `json:"behind,omitempty"`
	LastCommit string `json:"last_commit,omitempty"`
	IsMerged   bool   `json:"is_merged"`
	IsCurrent  bool   `json:"is_current"`
}

type GitState struct {
	HeadRef       string // commit hash from branch.oid, or empty (initial)
	WorkingTree   []git.FileStatus
	StagingArea   []git.FileStatus
	LocalBranch   git.BranchInfo
	RemoteState   git.RemoteInfo
	RemoteInfos   []git.RemoteInfo
	UpstreamState *git.UpstreamInfo
	StashStack    []git.StashEntry
	Submodules    []git.SubmoduleInfo
	RepoConfig    git.RepoConfig

	// Extended context
	IsInitial        bool              // true if no commits yet (HeadRef empty)
	Remotes          []string          // remote names (origin, upstream, etc.)
	RemoteURLs       map[string]string // legacy summary URL map, prefer RemoteInfos for rich data
	LocalBranches    []string          // all local branch names
	BranchDetails    []BranchDetail    // rich per-branch info for LLM
	MergedBranches   []string          // branches merged into current HEAD
	RemoteBranches   []string          // remote branch list with tracking hints
	HasGitIgnore     bool
	MergeInProgress  bool
	RebaseInProgress bool
	CherryInProgress bool
	BisectInProgress bool
	CommitCount      int // total commits on current branch (-1 if unknown)
	Tags             []string

	// Additional context
	RecentReflog     []string // recent reflog entries (e.g. "HEAD@{0} checkout: moving from main to dev")
	DescribeTag      string   // output of git describe --tags --always --long
	Worktrees        []string // git worktree list entries (only when multiple worktrees)
	HasGitAttributes bool

	// Upstream context
	AheadCommits  []string
	BehindCommits []string
	LastFetchTime time.Time

	// Rich inspection data
	FileInspect       *FileInspection `json:"file_inspect,omitempty"`
	CommitSummaryInfo *CommitSummary  `json:"commit_summary,omitempty"`
	ConfigInfo        *ConfigState    `json:"config_info,omitempty"`
}
