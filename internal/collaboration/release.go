package collaboration

import (
	"context"
	"fmt"
	"time"

	ghp "github.com/your-org/gitdex/internal/platform/github"
)

// ReleaseStatus represents release readiness status.
type ReleaseStatus string

const (
	ReleaseReady   ReleaseStatus = "ready"
	ReleaseBlocked ReleaseStatus = "blocked"
	ReleasePending ReleaseStatus = "pending"
)

// CheckStatus represents a check result status.
type CheckStatus string

const (
	CheckPassed  CheckStatus = "passed"
	CheckFailed  CheckStatus = "failed"
	CheckPending CheckStatus = "pending"
)

// CheckResult holds the result of a release check.
type CheckResult struct {
	Name    string      `json:"name" yaml:"name"`
	Status  CheckStatus `json:"status" yaml:"status"`
	Details string      `json:"details" yaml:"details"`
}

// ReleaseReadiness holds the assessment of a release.
type ReleaseReadiness struct {
	RepoOwner      string        `json:"repo_owner" yaml:"repo_owner"`
	RepoName       string        `json:"repo_name" yaml:"repo_name"`
	Tag            string        `json:"tag" yaml:"tag"`
	Status         ReleaseStatus `json:"status" yaml:"status"`
	Blockers       []string      `json:"blockers" yaml:"blockers"`
	IncludedPRs    []int         `json:"included_prs" yaml:"included_prs"`
	CheckResults   []CheckResult `json:"check_results" yaml:"check_results"`
	ApprovalStatus string        `json:"approval_status" yaml:"approval_status"`
	Notes          string        `json:"notes" yaml:"notes"`
	AssessedAt     time.Time     `json:"assessed_at" yaml:"assessed_at"`
}

// ReleaseInfo holds basic release metadata for listing.
type ReleaseInfo struct {
	Tag         string    `json:"tag" yaml:"tag"`
	Status      string    `json:"status" yaml:"status"`
	PublishedAt time.Time `json:"published_at" yaml:"published_at"`
}

// ReleaseEngine assesses release readiness and lists releases.
type ReleaseEngine interface {
	Assess(ctx context.Context, owner, repo, tag string) (*ReleaseReadiness, error)
	ListReleases(ctx context.Context, owner, repo string) ([]ReleaseInfo, error)
}

// GitHubReleaseEngine assesses release readiness and lists releases via the GitHub API.
type GitHubReleaseEngine struct {
	client *ghp.Client
}

// NewGitHubReleaseEngine creates a new GitHubReleaseEngine.
func NewGitHubReleaseEngine(client *ghp.Client) *GitHubReleaseEngine {
	return &GitHubReleaseEngine{client: client}
}

// Assess evaluates release readiness using combined status and check runs for the tag.
func (e *GitHubReleaseEngine) Assess(ctx context.Context, owner, repo, tag string) (*ReleaseReadiness, error) {
	if e.client == nil {
		return nil, fmt.Errorf("GitHub client is required")
	}
	if owner == "" || repo == "" || tag == "" {
		return nil, fmt.Errorf("owner, repo, and tag are required")
	}
	ref := tag
	if ref != "" && len(ref) < 40 && (len(ref) < 5 || ref[:5] != "refs/") {
		ref = "refs/tags/" + tag
	}

	var blockers []string
	var checkResults []CheckResult

	status, err := e.client.GetCombinedStatus(ctx, owner, repo, ref)
	if err == nil && status != nil {
		s := status.GetState()
		if s == "failure" || s == "error" {
			blockers = append(blockers, "commit status: "+s)
		}
		for _, st := range status.Statuses {
			cs := CheckPassed
			if st.GetState() == "failure" || st.GetState() == "error" {
				cs = CheckFailed
			} else if st.GetState() == "pending" {
				cs = CheckPending
			}
			checkResults = append(checkResults, CheckResult{
				Name:    st.GetContext(),
				Status:  cs,
				Details: st.GetDescription(),
			})
		}
	}

	runs, err := e.client.ListCheckRuns(ctx, owner, repo, ref)
	if err == nil {
		for _, r := range runs {
			cs := CheckPassed
			if r.Conclusion == "failure" || r.Conclusion == "cancelled" {
				cs = CheckFailed
				blockers = append(blockers, "check "+r.Name+": "+r.Conclusion)
			} else if r.Status != "completed" {
				cs = CheckPending
			}
			checkResults = append(checkResults, CheckResult{
				Name:    r.Name,
				Status:  cs,
				Details: r.Conclusion,
			})
		}
	}

	st := ReleaseReady
	if len(blockers) > 0 {
		st = ReleaseBlocked
	} else if len(checkResults) > 0 {
		for _, c := range checkResults {
			if c.Status == CheckPending {
				st = ReleasePending
				break
			}
		}
	}

	return &ReleaseReadiness{
		RepoOwner:      owner,
		RepoName:       repo,
		Tag:            tag,
		Status:         st,
		Blockers:       blockers,
		CheckResults:   checkResults,
		ApprovalStatus: "unknown",
		Notes:          "assessed via GitHub API",
		AssessedAt:     time.Now().UTC(),
	}, nil
}

// ListReleases returns releases from the GitHub API.
func (e *GitHubReleaseEngine) ListReleases(ctx context.Context, owner, repo string) ([]ReleaseInfo, error) {
	if e.client == nil {
		return nil, fmt.Errorf("GitHub client is required")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}
	releases, err := e.client.ListReleases(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("list releases: %w", err)
	}
	result := make([]ReleaseInfo, 0, len(releases))
	for _, r := range releases {
		if r == nil {
			continue
		}
		status := "published"
		if r.Draft {
			status = "draft"
		} else if r.Prerelease {
			status = "prerelease"
		}
		result = append(result, ReleaseInfo{
			Tag:         r.TagName,
			Status:      status,
			PublishedAt: r.PublishedAt,
		})
	}
	return result, nil
}
