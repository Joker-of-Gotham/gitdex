package gitops

type DivergenceState string

const (
	DivSynced     DivergenceState = "synced"
	DivAhead      DivergenceState = "ahead"
	DivBehind     DivergenceState = "behind"
	DivDiverged   DivergenceState = "diverged"
	DivDetached   DivergenceState = "detached"
	DivNoUpstream DivergenceState = "no_upstream"
)

type RepoInspection struct {
	RepoPath       string          `json:"repo_path" yaml:"repo_path"`
	LocalBranch    string          `json:"local_branch" yaml:"local_branch"`
	RemoteBranch   string          `json:"remote_branch,omitempty" yaml:"remote_branch,omitempty"`
	Ahead          int             `json:"ahead" yaml:"ahead"`
	Behind         int             `json:"behind" yaml:"behind"`
	HasUncommitted bool            `json:"has_uncommitted" yaml:"has_uncommitted"`
	HasUntracked   bool            `json:"has_untracked" yaml:"has_untracked"`
	Divergence     DivergenceState `json:"divergence" yaml:"divergence"`
}

type SyncRecommendation struct {
	Action      string `json:"action" yaml:"action"`
	RiskLevel   string `json:"risk_level" yaml:"risk_level"`
	Description string `json:"description" yaml:"description"`
	Previewable bool   `json:"previewable" yaml:"previewable"`
}

type SyncPreview struct {
	AffectedFiles int    `json:"affected_files" yaml:"affected_files"`
	MergeStrategy string `json:"merge_strategy" yaml:"merge_strategy"`
	ConflictRisk  string `json:"conflict_risk" yaml:"conflict_risk"`
	Description   string `json:"description" yaml:"description"`
}

type SyncResult struct {
	Success      bool   `json:"success" yaml:"success"`
	FilesChanged int    `json:"files_changed" yaml:"files_changed"`
	Conflicts    int    `json:"conflicts" yaml:"conflicts"`
	ErrorMessage string `json:"error_message,omitempty" yaml:"error_message,omitempty"`
	Description  string `json:"description" yaml:"description"`
	StashRef     string `json:"stash_ref,omitempty" yaml:"stash_ref,omitempty"`
}
