package github

import "time"

// Release is a GitHub repository release (REST API subset).
type Release struct {
	ID          int64     `json:"id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
}

// BranchProtection summarizes branch protection for a branch.
type BranchProtection struct {
	RequiredReviews      int  `json:"required_approving_review_count"`
	RequireSignedCommits bool `json:"required_signatures"`
	EnforceAdmins        bool `json:"enforce_admins"`
	RequireLinearHistory bool `json:"required_linear_history"`
}

// BranchProtectionSettings is used to update branch protection via the REST API.
type BranchProtectionSettings struct {
	RequiredReviews      int  `json:"required_approving_review_count"`
	RequireSignedCommits bool `json:"required_signatures"`
	EnforceAdmins        bool `json:"enforce_admins"`
	RequireLinearHistory bool `json:"required_linear_history"`
}

// Ruleset is a repository ruleset summary (REST API subset).
type Ruleset struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Target      string `json:"target"`
	Enforcement string `json:"enforcement"`
}
