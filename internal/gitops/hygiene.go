package gitops

// HygieneAction identifies a supported low-risk maintenance task.
type HygieneAction string

const (
	HygienePruneRemoteBranches  HygieneAction = "prune_remote_branches"
	HygieneGCAggressive         HygieneAction = "gc_aggressive"
	HygieneCleanUntracked       HygieneAction = "clean_untracked"
	HygieneRemoveMergedBranches HygieneAction = "remove_merged_branches"
)

// HygieneTask describes a supported hygiene action and its metadata.
type HygieneTask struct {
	Action          HygieneAction `json:"action" yaml:"action"`
	Description     string        `json:"description" yaml:"description"`
	RiskLevel       string        `json:"risk_level" yaml:"risk_level"`
	Reversible      bool          `json:"reversible" yaml:"reversible"`
	EstimatedImpact string        `json:"estimated_impact" yaml:"estimated_impact"`
}

// HygieneResult summarizes the outcome of a hygiene execution.
type HygieneResult struct {
	Success          bool          `json:"success" yaml:"success"`
	Action           HygieneAction `json:"action" yaml:"action"`
	FilesAffected    int           `json:"files_affected" yaml:"files_affected"`
	BranchesAffected int           `json:"branches_affected" yaml:"branches_affected"`
	ErrorMessage     string        `json:"error_message,omitempty" yaml:"error_message,omitempty"`
	Summary          string        `json:"summary" yaml:"summary"`
	DiskReclaimed    string        `json:"disk_reclaimed,omitempty" yaml:"disk_reclaimed,omitempty"`
	DeletedFiles     []string      `json:"deleted_files,omitempty" yaml:"deleted_files,omitempty"`
	DeletedBranches  []string      `json:"deleted_branches,omitempty" yaml:"deleted_branches,omitempty"`
}

// SupportedHygieneTasks returns all supported low-risk hygiene tasks.
func SupportedHygieneTasks() []HygieneTask {
	return []HygieneTask{
		{
			Action:          HygienePruneRemoteBranches,
			Description:     "Remove remote-tracking branches that no longer exist on the remote",
			RiskLevel:       "low",
			Reversible:      true,
			EstimatedImpact: "Reduces local ref clutter; branches can be re-fetched",
		},
		{
			Action:          HygieneGCAggressive,
			Description:     "Run aggressive git garbage collection to reclaim disk space",
			RiskLevel:       "low",
			Reversible:      false,
			EstimatedImpact: "Reclaims disk space from unreachable objects",
		},
		{
			Action:          HygieneCleanUntracked,
			Description:     "Remove untracked files and directories",
			RiskLevel:       "low",
			Reversible:      false,
			EstimatedImpact: "Removes untracked files; use with caution",
		},
		{
			Action:          HygieneRemoveMergedBranches,
			Description:     "Delete local branches that have been merged into the current branch",
			RiskLevel:       "low",
			Reversible:      true,
			EstimatedImpact: "Cleans up merged branches; can be recreated from remote",
		},
	}
}
