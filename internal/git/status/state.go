package status

import (
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

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
	RemoteBranches   []string          // remote branch list with tracking hints
	HasGitIgnore     bool
	MergeInProgress  bool
	RebaseInProgress bool
	CherryInProgress bool
	BisectInProgress bool
	CommitCount      int // total commits on current branch (-1 if unknown)
	Tags             []string

	// Upstream context
	AheadCommits  []string
	BehindCommits []string
	LastFetchTime time.Time

	// Rich inspection data
	FileInspect       *FileInspection `json:"file_inspect,omitempty"`
	CommitSummaryInfo *CommitSummary  `json:"commit_summary,omitempty"`
	ConfigInfo        *ConfigState    `json:"config_info,omitempty"`
}
