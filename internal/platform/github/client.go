package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/your-org/gitdex/internal/state/repo"
)

type Client struct {
	gh              *gh.Client
	httpClient      *http.Client
	graphQLEndpoint string
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{
		gh:              gh.NewClient(httpClient),
		httpClient:      httpClient,
		graphQLEndpoint: "https://api.github.com/graphql",
	}
}

func NewClientWithBaseURL(httpClient *http.Client, baseURL string) (*Client, error) {
	c, err := gh.NewClient(httpClient).WithEnterpriseURLs(baseURL, baseURL)
	if err != nil {
		return nil, fmt.Errorf("github: set enterprise URL: %w", err)
	}
	return &Client{
		gh:              c,
		httpClient:      httpClient,
		graphQLEndpoint: deriveGraphQLEndpoint(baseURL),
	}, nil
}

func deriveGraphQLEndpoint(baseURL string) string {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	baseURL = strings.TrimSuffix(baseURL, "/api/v3")
	return baseURL + "/api/graphql"
}

// Milestone represents a GitHub milestone.
type Milestone struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"` // open, closed
	DueOn        *time.Time `json:"due_on,omitempty"`
	OpenIssues   int        `json:"open_issues"`
	ClosedIssues int        `json:"closed_issues"`
}

// Label represents a repository label.
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// ProjectV2 is a GitHub Projects (classic successor) project.
type ProjectV2 struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Number int    `json:"number"`
	URL    string `json:"url"`
	Closed bool   `json:"closed"`
}

// Artifact is a workflow run artifact.
type Artifact struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_in_bytes"`
	CreatedAt time.Time `json:"created_at"`
	Expired   bool      `json:"expired"`
}

// CheckRun is a GitHub Check run for a commit/ref.
type CheckRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
}

// ReviewComment is an inline pull request review comment.
type ReviewComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	User      string    `json:"user"` // login of the author (flattened from API user object)
	CreatedAt time.Time `json:"created_at"`
}

// Environment is a deployment environment.
type Environment struct {
	ID          int64            `json:"id"`
	Name        string           `json:"name"`
	URL         string           `json:"url"`
	Protection  []ProtectionRule `json:"protection_rules,omitempty"`
}

// ProtectionRule describes an environment protection rule.
type ProtectionRule struct {
	Type string `json:"type"` // required_reviewers, wait_timer, branch_policy
}

func (c *Client) GetRepository(ctx context.Context, owner, repoName string) (*repo.RemoteState, error) {
	r, resp, err := c.gh.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		return nil, fmt.Errorf("github: get repository: %w", err)
	}
	logRateLimit(resp)

	return &repo.RemoteState{
		FullName:      r.GetFullName(),
		Description:   r.GetDescription(),
		DefaultBranch: r.GetDefaultBranch(),
		IsPrivate:     r.GetPrivate(),
	}, nil
}

type RepositoryDetail struct {
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
	OpenIssues    int
}

func (c *Client) GetRepositoryDetail(ctx context.Context, owner, repoName string) (*RepositoryDetail, error) {
	r, resp, err := c.gh.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		return nil, fmt.Errorf("github: get repository detail: %w", err)
	}
	logRateLimit(resp)

	license := ""
	if r.GetLicense() != nil {
		license = r.GetLicense().GetName()
	}
	createdAt := ""
	if !r.GetCreatedAt().Time.IsZero() {
		createdAt = r.GetCreatedAt().Time.Format("2006-01-02")
	}

	return &RepositoryDetail{
		Description:   r.GetDescription(),
		Stars:         r.GetStargazersCount(),
		Forks:         r.GetForksCount(),
		Language:      r.GetLanguage(),
		License:       license,
		Topics:        r.Topics,
		DefaultBranch: r.GetDefaultBranch(),
		IsPrivate:     r.GetPrivate(),
		CreatedAt:     createdAt,
		HTMLURL:       r.GetHTMLURL(),
		OpenIssues:    r.GetOpenIssuesCount(),
	}, nil
}

func (c *Client) ListOpenPullRequests(ctx context.Context, owner, repoName string) ([]repo.PullRequestSummary, error) {
	prs, resp, err := c.gh.PullRequests.List(ctx, owner, repoName, &gh.PullRequestListOptions{
		State:       "open",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 25},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list pull requests: %w", err)
	}
	logRateLimit(resp)

	var result []repo.PullRequestSummary
	for _, pr := range prs {
		labels := make([]string, 0, len(pr.Labels))
		for _, l := range pr.Labels {
			labels = append(labels, l.GetName())
		}

		staleDays := 0
		if pr.UpdatedAt != nil {
			staleDays = int(time.Since(pr.UpdatedAt.Time).Hours() / 24)
		}

		result = append(result, repo.PullRequestSummary{
			Number:      pr.GetNumber(),
			Title:       pr.GetTitle(),
			Author:      pr.GetUser().GetLogin(),
			Labels:      labels,
			IsDraft:     pr.GetDraft(),
			NeedsReview: !pr.GetDraft() && len(pr.RequestedReviewers) > 0,
			StaleDays:   staleDays,
		})
	}
	return result, nil
}

// ListPullRequests returns pull requests for a repository filtered by state.
func (c *Client) ListPullRequests(ctx context.Context, owner, repoName, state string) ([]*gh.PullRequest, error) {
	if state == "" {
		state = "open"
	}
	prs, resp, err := c.gh.PullRequests.List(ctx, owner, repoName, &gh.PullRequestListOptions{
		State:       state,
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list pull requests: %w", err)
	}
	logRateLimit(resp)
	return prs, nil
}

func (c *Client) ListOpenIssues(ctx context.Context, owner, repoName string) ([]repo.IssueSummary, error) {
	issues, resp, err := c.gh.Issues.ListByRepo(ctx, owner, repoName, &gh.IssueListByRepoOptions{
		State:       "open",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 25},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list issues: %w", err)
	}
	logRateLimit(resp)

	var result []repo.IssueSummary
	for _, issue := range issues {
		if issue.PullRequestLinks != nil {
			continue
		}
		labels := make([]string, 0, len(issue.Labels))
		for _, l := range issue.Labels {
			labels = append(labels, l.GetName())
		}
		createdAt := ""
		if issue.CreatedAt != nil {
			createdAt = issue.CreatedAt.Time.Format(time.RFC3339)
		}
		updatedAt := ""
		if issue.UpdatedAt != nil {
			updatedAt = issue.UpdatedAt.Time.Format(time.RFC3339)
		}
		result = append(result, repo.IssueSummary{
			Number:    issue.GetNumber(),
			Title:     issue.GetTitle(),
			Author:    issue.GetUser().GetLogin(),
			Labels:    labels,
			State:     issue.GetState(),
			Comments:  issue.GetComments(),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	return result, nil
}

// ListIssues returns issues for a repository filtered by state.
func (c *Client) ListIssues(ctx context.Context, owner, repoName, state string) ([]*gh.Issue, error) {
	if state == "" {
		state = "open"
	}
	issues, resp, err := c.gh.Issues.ListByRepo(ctx, owner, repoName, &gh.IssueListByRepoOptions{
		State:       state,
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list issues: %w", err)
	}
	logRateLimit(resp)
	return issues, nil
}

// EstimateOpenIssueCount returns an approximate count of open issues
// by reading the last page number from a PerPage=1 paginated request.
func (c *Client) EstimateOpenIssueCount(ctx context.Context, owner, repoName string) (int, error) {
	_, resp, err := c.gh.Issues.ListByRepo(ctx, owner, repoName, &gh.IssueListByRepoOptions{
		State:       "open",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 1},
	})
	if err != nil {
		return 0, fmt.Errorf("github: estimate issue count: %w", err)
	}
	logRateLimit(resp)

	return resp.LastPage, nil
}

func (c *Client) ListWorkflowRuns(ctx context.Context, owner, repoName string) ([]repo.WorkflowRunSummary, error) {
	runs, resp, err := c.gh.Actions.ListRepositoryWorkflowRuns(ctx, owner, repoName, &gh.ListWorkflowRunsOptions{
		ListOptions: gh.ListOptions{PerPage: 10},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list workflow runs: %w", err)
	}
	logRateLimit(resp)

	var result []repo.WorkflowRunSummary
	for _, r := range runs.WorkflowRuns {
		createdAt := ""
		if r.CreatedAt != nil {
			createdAt = r.CreatedAt.Time.Format("2006-01-02 15:04")
		}
		result = append(result, repo.WorkflowRunSummary{
			RunID:      r.GetID(),
			WorkflowID: r.GetWorkflowID(),
			Name:       r.GetName(),
			Status:     r.GetStatus(),
			Conclusion: r.GetConclusion(),
			Branch:     r.GetHeadBranch(),
			Event:      r.GetEvent(),
			CreatedAt:  createdAt,
			URL:        r.GetHTMLURL(),
		})
	}
	return result, nil
}

func (c *Client) ListDeployments(ctx context.Context, owner, repoName string) ([]repo.DeploymentSummary, error) {
	deps, resp, err := c.gh.Repositories.ListDeployments(ctx, owner, repoName, &gh.DeploymentsListOptions{
		ListOptions: gh.ListOptions{PerPage: 10},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list deployments: %w", err)
	}
	logRateLimit(resp)

	var result []repo.DeploymentSummary
	for _, d := range deps {
		state := "unknown"
		statuses, _, err := c.gh.Repositories.ListDeploymentStatuses(ctx, owner, repoName, d.GetID(), &gh.ListOptions{PerPage: 1})
		if err == nil && len(statuses) > 0 {
			state = statuses[0].GetState()
		}

		createdAt := ""
		if d.CreatedAt != nil {
			createdAt = d.CreatedAt.Time.Format("2006-01-02 15:04")
		}
		result = append(result, repo.DeploymentSummary{
			ID:          d.GetID(),
			Environment: d.GetEnvironment(),
			State:       state,
			Ref:         d.GetRef(),
			CreatedAt:   createdAt,
			URL:         firstNonEmpty(d.GetURL(), d.GetStatusesURL()),
		})
	}
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func logRateLimit(resp *gh.Response) {
	if resp == nil {
		return
	}
	rate := resp.Rate
	if rate.Limit > 0 && rate.Remaining < 100 {
		_, _ = fmt.Fprintf(os.Stderr, "⚠️  GitHub API rate limit low: %d/%d remaining (resets %s)\n",
			rate.Remaining, rate.Limit, rate.Reset.Time.Format(time.RFC3339))
	}
}

// GetFileContent returns the content of a file from a GitHub repository.
func (c *Client) GetFileContent(ctx context.Context, owner, repoName, path, ref string) (string, error) {
	opts := &gh.RepositoryContentGetOptions{Ref: ref}
	file, _, resp, err := c.gh.Repositories.GetContents(ctx, owner, repoName, path, opts)
	if err != nil {
		return "", fmt.Errorf("github: get file content: %w", err)
	}
	logRateLimit(resp)
	if file == nil {
		return "", fmt.Errorf("github: path is a directory, not a file")
	}
	content, err := file.GetContent()
	if err != nil {
		return "", fmt.Errorf("github: decode file content: %w", err)
	}
	return content, nil
}

// GetTreeRecursive returns all file paths in a repo at a given ref (e.g. "HEAD", "main").
func (c *Client) GetTreeRecursive(ctx context.Context, owner, repoName, ref string) ([]string, error) {
	tree, resp, err := c.gh.Git.GetTree(ctx, owner, repoName, ref, true)
	if err != nil {
		return nil, fmt.Errorf("github: get tree: %w", err)
	}
	logRateLimit(resp)

	var paths []string
	for _, entry := range tree.Entries {
		if entry.GetType() == "blob" {
			paths = append(paths, entry.GetPath())
		}
	}
	return paths, nil
}

// ListUserRepositories returns all repos the authenticated user has access to.
func (c *Client) ListUserRepositories(ctx context.Context) ([]*gh.Repository, error) {
	var all []*gh.Repository
	opts := &gh.RepositoryListByAuthenticatedUserOptions{
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := c.gh.Repositories.ListByAuthenticatedUser(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("github: list user repos: %w", err)
		}
		logRateLimit(resp)
		all = append(all, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// GetPullRequest returns detailed information about a pull request.
func (c *Client) GetPullRequest(ctx context.Context, owner, repoName string, number int) (*gh.PullRequest, error) {
	pr, resp, err := c.gh.PullRequests.Get(ctx, owner, repoName, number)
	if err != nil {
		return nil, fmt.Errorf("github: get pull request: %w", err)
	}
	logRateLimit(resp)
	return pr, nil
}

// GetCommit returns a single commit with file metadata.
func (c *Client) GetCommit(ctx context.Context, owner, repoName, sha string) (*gh.RepositoryCommit, error) {
	commit, resp, err := c.gh.Repositories.GetCommit(ctx, owner, repoName, sha, &gh.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("github: get commit: %w", err)
	}
	logRateLimit(resp)
	return commit, nil
}

// ListPRComments lists review comments on a pull request.
func (c *Client) ListPRComments(ctx context.Context, owner, repoName string, number int) ([]*gh.PullRequestComment, error) {
	comments, resp, err := c.gh.PullRequests.ListComments(ctx, owner, repoName, number, &gh.PullRequestListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 50},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list PR comments: %w", err)
	}
	logRateLimit(resp)
	return comments, nil
}

// ListPRFiles lists files changed in a pull request.
func (c *Client) ListPRFiles(ctx context.Context, owner, repoName string, number int) ([]*gh.CommitFile, error) {
	files, resp, err := c.gh.PullRequests.ListFiles(ctx, owner, repoName, number, &gh.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("github: list PR files: %w", err)
	}
	logRateLimit(resp)
	return files, nil
}

// ListPRReviews lists reviews on a pull request.
func (c *Client) ListPRReviews(ctx context.Context, owner, repoName string, number int) ([]*gh.PullRequestReview, error) {
	reviews, resp, err := c.gh.PullRequests.ListReviews(ctx, owner, repoName, number, &gh.ListOptions{PerPage: 50})
	if err != nil {
		return nil, fmt.Errorf("github: list PR reviews: %w", err)
	}
	logRateLimit(resp)
	return reviews, nil
}

// SubmitPRReview submits a review on a pull request.
func (c *Client) SubmitPRReview(ctx context.Context, owner, repoName string, number int, event, body string) (*gh.PullRequestReview, error) {
	review := &gh.PullRequestReviewRequest{
		Event: gh.String(event),
		Body:  gh.String(body),
	}
	r, resp, err := c.gh.PullRequests.CreateReview(ctx, owner, repoName, number, review)
	if err != nil {
		return nil, fmt.Errorf("github: submit PR review: %w", err)
	}
	logRateLimit(resp)
	return r, nil
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(ctx context.Context, owner, repoName, title, body, head, base string) (*gh.PullRequest, error) {
	pr := &gh.NewPullRequest{
		Title: gh.String(title),
		Body:  gh.String(body),
		Head:  gh.String(head),
		Base:  gh.String(base),
	}
	result, resp, err := c.gh.PullRequests.Create(ctx, owner, repoName, pr)
	if err != nil {
		return nil, fmt.Errorf("github: create pull request: %w", err)
	}
	logRateLimit(resp)
	return result, nil
}

// GetIssue returns detailed information about an issue.
func (c *Client) GetIssue(ctx context.Context, owner, repoName string, number int) (*gh.Issue, error) {
	issue, resp, err := c.gh.Issues.Get(ctx, owner, repoName, number)
	if err != nil {
		return nil, fmt.Errorf("github: get issue: %w", err)
	}
	logRateLimit(resp)
	return issue, nil
}

// ListIssueComments lists comments on an issue.
func (c *Client) ListIssueComments(ctx context.Context, owner, repoName string, number int) ([]*gh.IssueComment, error) {
	comments, resp, err := c.gh.Issues.ListComments(ctx, owner, repoName, number, &gh.IssueListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 50},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list issue comments: %w", err)
	}
	logRateLimit(resp)
	return comments, nil
}

// TriggerWorkflow dispatches a workflow event.
func (c *Client) TriggerWorkflow(ctx context.Context, owner, repoName string, workflowID int64, ref string) error {
	_, _, err := c.gh.Actions.CreateWorkflowDispatchEventByID(ctx, owner, repoName, workflowID, gh.CreateWorkflowDispatchEventRequest{
		Ref: ref,
	})
	if err != nil {
		return fmt.Errorf("github: trigger workflow: %w", err)
	}
	return nil
}

// RerunWorkflowRun re-runs a workflow run by ID.
func (c *Client) RerunWorkflowRun(ctx context.Context, owner, repoName string, runID int64) error {
	resp, err := c.gh.Actions.RerunWorkflowByID(ctx, owner, repoName, runID)
	if err != nil {
		return fmt.Errorf("github: rerun workflow run: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// CancelWorkflowRun cancels a workflow run by ID.
func (c *Client) CancelWorkflowRun(ctx context.Context, owner, repoName string, runID int64) error {
	resp, err := c.gh.Actions.CancelWorkflowRunByID(ctx, owner, repoName, runID)
	if err != nil {
		var accepted *gh.AcceptedError
		if errors.As(err, &accepted) {
			logRateLimit(resp)
			return nil
		}
		return fmt.Errorf("github: cancel workflow run: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// ListBranches lists branches of a repository.
func (c *Client) ListBranches(ctx context.Context, owner, repoName string) ([]*gh.Branch, error) {
	branches, resp, err := c.gh.Repositories.ListBranches(ctx, owner, repoName, &gh.BranchListOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list branches: %w", err)
	}
	logRateLimit(resp)
	return branches, nil
}

// ListCommits lists recent commits of a repository.
func (c *Client) ListCommits(ctx context.Context, owner, repoName string) ([]*gh.RepositoryCommit, error) {
	commits, resp, err := c.gh.Repositories.ListCommits(ctx, owner, repoName, &gh.CommitsListOptions{
		ListOptions: gh.ListOptions{PerPage: 50},
	})
	if err != nil {
		return nil, fmt.Errorf("github: list commits: %w", err)
	}
	logRateLimit(resp)
	return commits, nil
}

// --- Write methods ---

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(ctx context.Context, owner, repo, title, body string, labels, assignees []string) (*gh.Issue, error) {
	req := &gh.IssueRequest{
		Title:     gh.String(title),
		Body:      gh.String(body),
		Labels:    &labels,
		Assignees: &assignees,
	}
	issue, resp, err := c.gh.Issues.Create(ctx, owner, repo, req)
	if err != nil {
		return nil, fmt.Errorf("github: create issue: %w", err)
	}
	logRateLimit(resp)
	return issue, nil
}

// UpdateIssue updates an existing issue.
func (c *Client) UpdateIssue(ctx context.Context, owner, repo string, number int, req *gh.IssueRequest) (*gh.Issue, error) {
	issue, resp, err := c.gh.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		return nil, fmt.Errorf("github: update issue: %w", err)
	}
	logRateLimit(resp)
	return issue, nil
}

// CreateComment adds a comment to an issue or PR.
func (c *Client) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*gh.IssueComment, error) {
	comment := &gh.IssueComment{Body: gh.String(body)}
	cmmt, resp, err := c.gh.Issues.CreateComment(ctx, owner, repo, number, comment)
	if err != nil {
		return nil, fmt.Errorf("github: create comment: %w", err)
	}
	logRateLimit(resp)
	return cmmt, nil
}

// AddLabels adds labels to an issue or PR.
func (c *Client) AddLabels(ctx context.Context, owner, repo string, number int, labels []string) error {
	_, resp, err := c.gh.Issues.AddLabelsToIssue(ctx, owner, repo, number, labels)
	if err != nil {
		return fmt.Errorf("github: add labels: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// SetAssignees replaces assignees on an issue or PR.
func (c *Client) SetAssignees(ctx context.Context, owner, repo string, number int, assignees []string) error {
	req := &gh.IssueRequest{Assignees: &assignees}
	_, resp, err := c.gh.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		return fmt.Errorf("github: set assignees: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// CloseIssue closes an issue or PR.
func (c *Client) CloseIssue(ctx context.Context, owner, repo string, number int) error {
	req := &gh.IssueRequest{State: gh.String("closed")}
	_, resp, err := c.gh.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		return fmt.Errorf("github: close issue: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// ReopenIssue reopens a closed issue or PR.
func (c *Client) ReopenIssue(ctx context.Context, owner, repo string, number int) error {
	req := &gh.IssueRequest{State: gh.String("open")}
	_, resp, err := c.gh.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		return fmt.Errorf("github: reopen issue: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// MergePullRequest merges a pull request.
func (c *Client) MergePullRequest(ctx context.Context, owner, repo string, number int, commitMsg, method string) (*gh.PullRequestMergeResult, error) {
	opts := &gh.PullRequestOptions{MergeMethod: method}
	result, resp, err := c.gh.PullRequests.Merge(ctx, owner, repo, number, commitMsg, opts)
	if err != nil {
		return nil, fmt.Errorf("github: merge pull request: %w", err)
	}
	logRateLimit(resp)
	return result, nil
}

// CreateRelease creates a new release (POST /repos/{owner}/{repo}/releases).
func (c *Client) CreateRelease(ctx context.Context, owner, repo, tag, name, body string, draft, prerelease bool) (*Release, error) {
	rel := &gh.RepositoryRelease{
		TagName:    gh.String(tag),
		Name:       gh.String(name),
		Body:       gh.String(body),
		Draft:      gh.Bool(draft),
		Prerelease: gh.Bool(prerelease),
	}
	release, resp, err := c.gh.Repositories.CreateRelease(ctx, owner, repo, rel)
	if err != nil {
		return nil, fmt.Errorf("github: create release: %w", err)
	}
	logRateLimit(resp)
	return releaseFromGH(release), nil
}

// GetReleaseByTag fetches a release by tag.
func (c *Client) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*Release, error) {
	release, resp, err := c.gh.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("github: get release by tag: %w", err)
	}
	logRateLimit(resp)
	return releaseFromGH(release), nil
}

// UpdateRelease updates an existing release (PATCH /repos/{owner}/{repo}/releases/{release_id}).
func (c *Client) UpdateRelease(ctx context.Context, owner, repo string, releaseID int64, tag, name, body string, draft, prerelease bool) (*Release, error) {
	rel := &gh.RepositoryRelease{
		TagName:    gh.String(tag),
		Name:       gh.String(name),
		Body:       gh.String(body),
		Draft:      gh.Bool(draft),
		Prerelease: gh.Bool(prerelease),
	}
	release, resp, err := c.gh.Repositories.EditRelease(ctx, owner, repo, releaseID, rel)
	if err != nil {
		return nil, fmt.Errorf("github: update release: %w", err)
	}
	logRateLimit(resp)
	return releaseFromGH(release), nil
}

// PublishRelease sets draft=false on a release.
func (c *Client) PublishRelease(ctx context.Context, owner, repo string, releaseID int64) (*Release, error) {
	rel := &gh.RepositoryRelease{
		Draft: gh.Bool(false),
	}
	release, resp, err := c.gh.Repositories.EditRelease(ctx, owner, repo, releaseID, rel)
	if err != nil {
		return nil, fmt.Errorf("github: publish release: %w", err)
	}
	logRateLimit(resp)
	return releaseFromGH(release), nil
}

// DeleteRelease deletes a release by id.
func (c *Client) DeleteRelease(ctx context.Context, owner, repo string, releaseID int64) error {
	resp, err := c.gh.Repositories.DeleteRelease(ctx, owner, repo, releaseID)
	if err != nil {
		return fmt.Errorf("github: delete release: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// UploadReleaseAsset uploads a binary asset to an existing release.
func (c *Client) UploadReleaseAsset(ctx context.Context, owner, repo string, releaseID int64, name string, file io.Reader) error {
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("github: read release asset: %w", err)
	}
	release, resp, err := c.gh.Repositories.GetRelease(ctx, owner, repo, releaseID)
	if err != nil {
		return fmt.Errorf("github: get release for upload: %w", err)
	}
	logRateLimit(resp)
	_, resp, err = c.gh.Repositories.UploadReleaseAssetFromRelease(ctx, release, &gh.UploadOptions{Name: name}, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("github: upload release asset: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// ListReleases lists releases for a repository.
func (c *Client) ListReleases(ctx context.Context, owner, repo string) ([]*Release, error) {
	releases, resp, err := c.gh.Repositories.ListReleases(ctx, owner, repo, &gh.ListOptions{PerPage: 30})
	if err != nil {
		return nil, fmt.Errorf("github: list releases: %w", err)
	}
	logRateLimit(resp)
	out := make([]*Release, 0, len(releases))
	for _, r := range releases {
		out = append(out, releaseFromGH(r))
	}
	return out, nil
}

// GetCombinedStatus returns the combined commit status for a ref.
func (c *Client) GetCombinedStatus(ctx context.Context, owner, repo, ref string) (*gh.CombinedStatus, error) {
	status, resp, err := c.gh.Repositories.GetCombinedStatus(ctx, owner, repo, ref, nil)
	if err != nil {
		return nil, fmt.Errorf("github: get combined status: %w", err)
	}
	logRateLimit(resp)
	return status, nil
}

// rest issues a REST request against the GitHub API using the embedded go-github client.
func (c *Client) rest(ctx context.Context, method, path string, body any, result any) (*gh.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("github: marshal request body: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := c.gh.NewRequest(method, path, rdr)
	if err != nil {
		return nil, fmt.Errorf("github: new request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.gh.Do(ctx, req, result)
	if err != nil {
		return resp, err
	}
	logRateLimit(resp)
	return resp, nil
}

// ListMilestones lists milestones for a repository (all states).
func (c *Client) ListMilestones(ctx context.Context, owner, repo string) ([]Milestone, error) {
	var all []Milestone
	page := 1
	for {
		path := fmt.Sprintf("repos/%s/%s/milestones?state=all&per_page=100&page=%d", url.PathEscape(owner), url.PathEscape(repo), page)
		var batch []Milestone
		resp, err := c.rest(ctx, http.MethodGet, path, nil, &batch)
		if err != nil {
			return nil, fmt.Errorf("github: list milestones: %w", err)
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if resp == nil || resp.NextPage == 0 || len(batch) < 100 {
			break
		}
		page = resp.NextPage
	}
	return all, nil
}

// CreateMilestone creates a milestone.
func (c *Client) CreateMilestone(ctx context.Context, owner, repo, title, description string, dueOn *time.Time) (*Milestone, error) {
	body := map[string]any{
		"title":       title,
		"description": description,
	}
	if dueOn != nil {
		body["due_on"] = dueOn.UTC().Format(time.RFC3339)
	}
	path := fmt.Sprintf("repos/%s/%s/milestones", url.PathEscape(owner), url.PathEscape(repo))
	var out Milestone
	if _, err := c.rest(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, fmt.Errorf("github: create milestone: %w", err)
	}
	return &out, nil
}

// UpdateMilestone updates a milestone by number.
func (c *Client) UpdateMilestone(ctx context.Context, owner, repo string, number int, title, description string, state string) (*Milestone, error) {
	body := map[string]any{
		"title":       title,
		"description": description,
		"state":       state,
	}
	path := fmt.Sprintf("repos/%s/%s/milestones/%d", url.PathEscape(owner), url.PathEscape(repo), number)
	var out Milestone
	if _, err := c.rest(ctx, http.MethodPatch, path, body, &out); err != nil {
		return nil, fmt.Errorf("github: update milestone: %w", err)
	}
	return &out, nil
}

// DeleteMilestone deletes a milestone by number.
func (c *Client) DeleteMilestone(ctx context.Context, owner, repo string, number int) error {
	path := fmt.Sprintf("repos/%s/%s/milestones/%d", url.PathEscape(owner), url.PathEscape(repo), number)
	if _, err := c.rest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("github: delete milestone: %w", err)
	}
	return nil
}

// ListLabels lists all labels in a repository.
func (c *Client) ListLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	var all []Label
	page := 1
	for {
		path := fmt.Sprintf("repos/%s/%s/labels?per_page=100&page=%d", url.PathEscape(owner), url.PathEscape(repo), page)
		var batch []Label
		resp, err := c.rest(ctx, http.MethodGet, path, nil, &batch)
		if err != nil {
			return nil, fmt.Errorf("github: list labels: %w", err)
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if resp == nil || resp.NextPage == 0 || len(batch) < 100 {
			break
		}
		page = resp.NextPage
	}
	return all, nil
}

// CreateLabel creates a label.
func (c *Client) CreateLabel(ctx context.Context, owner, repo, name, color, description string) (*Label, error) {
	body := map[string]any{
		"name":        name,
		"color":       color,
		"description": description,
	}
	path := fmt.Sprintf("repos/%s/%s/labels", url.PathEscape(owner), url.PathEscape(repo))
	var out Label
	if _, err := c.rest(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, fmt.Errorf("github: create label: %w", err)
	}
	return &out, nil
}

// UpdateLabel renames or updates a label.
func (c *Client) UpdateLabel(ctx context.Context, owner, repo, currentName, newName, color, description string) (*Label, error) {
	body := map[string]any{
		"name":        newName,
		"color":       color,
		"description": description,
	}
	path := fmt.Sprintf("repos/%s/%s/labels/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(currentName))
	var out Label
	if _, err := c.rest(ctx, http.MethodPatch, path, body, &out); err != nil {
		return nil, fmt.Errorf("github: update label: %w", err)
	}
	return &out, nil
}

// DeleteLabel deletes a label by name.
func (c *Client) DeleteLabel(ctx context.Context, owner, repo, name string) error {
	path := fmt.Sprintf("repos/%s/%s/labels/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(name))
	if _, err := c.rest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("github: delete label: %w", err)
	}
	return nil
}

// ListProjectsV2 lists GitHub Projects (classic successor) for a repository via GraphQL.
func (c *Client) ListProjectsV2(ctx context.Context, owner, repo string) ([]ProjectV2, error) {
	var payload struct {
		Repository *struct {
			ProjectsV2 struct {
				Nodes []ProjectV2 `json:"nodes"`
			} `json:"projectsV2"`
		} `json:"repository"`
	}
	const query = `
query($owner:String!, $repo:String!, $first:Int!) {
  repository(owner:$owner, name:$repo) {
    projectsV2(first:$first) {
      nodes {
        id
        title
        number
        url
        closed
      }
    }
  }
}`
	if err := c.doGraphQL(ctx, query, map[string]any{
		"owner": owner,
		"repo":  repo,
		"first": 50,
	}, &payload); err != nil {
		return nil, fmt.Errorf("github: list projects v2: %w", err)
	}
	if payload.Repository == nil {
		return nil, nil
	}
	return payload.Repository.ProjectsV2.Nodes, nil
}

// GetWorkflowRunLogs returns a download URL for the workflow run log archive (redirect target).
func (c *Client) GetWorkflowRunLogs(ctx context.Context, owner, repo string, runID int64) (string, error) {
	u, resp, err := c.gh.Actions.GetWorkflowRunLogs(ctx, owner, repo, runID, 1)
	logRateLimit(resp)
	if err != nil {
		return "", fmt.Errorf("github: get workflow run logs: %w", err)
	}
	if u == nil {
		return "", fmt.Errorf("github: workflow run logs url missing")
	}
	return u.String(), nil
}

// ListWorkflowArtifacts lists artifacts for a workflow run.
func (c *Client) ListWorkflowArtifacts(ctx context.Context, owner, repo string, runID int64) ([]Artifact, error) {
	opts := &gh.ListOptions{PerPage: 100}
	var out []Artifact
	for {
		list, resp, err := c.gh.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, runID, opts)
		if err != nil {
			return nil, fmt.Errorf("github: list workflow artifacts: %w", err)
		}
		logRateLimit(resp)
		for _, a := range list.Artifacts {
			out = append(out, artifactFromGH(a))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return out, nil
}

func artifactFromGH(a *gh.Artifact) Artifact {
	if a == nil {
		return Artifact{}
	}
	var created time.Time
	if a.CreatedAt != nil {
		created = a.CreatedAt.Time
	}
	return Artifact{
		ID:        a.GetID(),
		Name:      a.GetName(),
		SizeBytes: a.GetSizeInBytes(),
		CreatedAt: created,
		Expired:   a.GetExpired(),
	}
}

// DownloadArtifact downloads the zip archive for an artifact.
func (c *Client) DownloadArtifact(ctx context.Context, owner, repo string, artifactID int64) ([]byte, error) {
	u, resp, err := c.gh.Actions.DownloadArtifact(ctx, owner, repo, artifactID, 1)
	logRateLimit(resp)
	if err != nil {
		return nil, fmt.Errorf("github: resolve artifact download: %w", err)
	}
	if u == nil {
		return nil, fmt.Errorf("github: artifact download url missing")
	}
	if c.httpClient == nil {
		return nil, fmt.Errorf("github: http client unavailable")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("github: build artifact download request: %w", err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: download artifact: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("github: download artifact: status %s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(res.Body)
}

// ListCheckRuns lists check runs for a ref.
func (c *Client) ListCheckRuns(ctx context.Context, owner, repo, ref string) ([]CheckRun, error) {
	var out []CheckRun
	page := 1
	for {
		path := fmt.Sprintf("repos/%s/%s/commits/%s/check-runs?per_page=100&page=%d",
			url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref), page)
		var payload struct {
			CheckRuns []*struct {
				ID         int64  `json:"id"`
				Name       string `json:"name"`
				Status     string `json:"status"`
				Conclusion string `json:"conclusion"`
				HTMLURL    string `json:"html_url"`
			} `json:"check_runs"`
		}
		resp, err := c.rest(ctx, http.MethodGet, path, nil, &payload)
		if err != nil {
			return nil, fmt.Errorf("github: list check runs: %w", err)
		}
		for _, r := range payload.CheckRuns {
			if r == nil {
				continue
			}
			out = append(out, CheckRun{
				ID:         r.ID,
				Name:       r.Name,
				Status:     r.Status,
				Conclusion: r.Conclusion,
				HTMLURL:    r.HTMLURL,
			})
		}
		if resp == nil || resp.NextPage == 0 || len(payload.CheckRuns) < 100 {
			break
		}
		page = resp.NextPage
	}
	return out, nil
}

// ListReviewComments lists inline review comments on a pull request.
func (c *Client) ListReviewComments(ctx context.Context, owner, repo string, prNumber int) ([]ReviewComment, error) {
	var out []ReviewComment
	page := 1
	for {
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/comments?per_page=100&page=%d",
			url.PathEscape(owner), url.PathEscape(repo), prNumber, page)
		var batch []struct {
			ID        int64     `json:"id"`
			Body      string    `json:"body"`
			Path      string    `json:"path"`
			Line      int       `json:"line"`
			User      struct{ Login string `json:"login"` } `json:"user"`
			CreatedAt time.Time `json:"created_at"`
		}
		resp, err := c.rest(ctx, http.MethodGet, path, nil, &batch)
		if err != nil {
			return nil, fmt.Errorf("github: list review comments: %w", err)
		}
		for _, b := range batch {
			out = append(out, ReviewComment{
				ID:        b.ID,
				Body:      b.Body,
				Path:      b.Path,
				Line:      b.Line,
				User:      b.User.Login,
				CreatedAt: b.CreatedAt,
			})
		}
		if resp == nil || resp.NextPage == 0 || len(batch) < 100 {
			break
		}
		page = resp.NextPage
	}
	return out, nil
}

// CreateReviewComment creates an inline review comment on a pull request diff.
func (c *Client) CreateReviewComment(ctx context.Context, owner, repo string, prNumber int, body, path string, line int) (*ReviewComment, error) {
	pr, err := c.GetPullRequest(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("github: load pull request for review comment: %w", err)
	}
	if pr.GetHead() == nil || pr.GetHead().GetSHA() == "" {
		return nil, fmt.Errorf("github: pull request head sha missing")
	}
	payload := map[string]any{
		"body":      body,
		"path":      path,
		"line":      line,
		"commit_id": pr.GetHead().GetSHA(),
		"side":      "RIGHT",
	}
	apiPath := fmt.Sprintf("repos/%s/%s/pulls/%d/comments", url.PathEscape(owner), url.PathEscape(repo), prNumber)
	var raw struct {
		ID        int64     `json:"id"`
		Body      string    `json:"body"`
		Path      string    `json:"path"`
		Line      int       `json:"line"`
		User      struct{ Login string `json:"login"` } `json:"user"`
		CreatedAt time.Time `json:"created_at"`
	}
	if _, err := c.rest(ctx, http.MethodPost, apiPath, payload, &raw); err != nil {
		return nil, fmt.Errorf("github: create review comment: %w", err)
	}
	return &ReviewComment{
		ID:        raw.ID,
		Body:      raw.Body,
		Path:      raw.Path,
		Line:      raw.Line,
		User:      raw.User.Login,
		CreatedAt: raw.CreatedAt,
	}, nil
}

// ListDeploymentEnvironments lists deployment environments for a repository.
func (c *Client) ListDeploymentEnvironments(ctx context.Context, owner, repo string) ([]Environment, error) {
	var all []Environment
	page := 1
	for {
		path := fmt.Sprintf("repos/%s/%s/environments?per_page=100&page=%d", url.PathEscape(owner), url.PathEscape(repo), page)
		var payload struct {
			Environments []struct {
				ID                int64            `json:"id"`
				Name              string           `json:"name"`
				URL               string           `json:"url"`
				ProtectionRules []ProtectionRule `json:"protection_rules"`
			} `json:"environments"`
		}
		resp, err := c.rest(ctx, http.MethodGet, path, nil, &payload)
		if err != nil {
			return nil, fmt.Errorf("github: list deployment environments: %w", err)
		}
		for _, e := range payload.Environments {
			all = append(all, Environment{
				ID:         e.ID,
				Name:       e.Name,
				URL:        e.URL,
				Protection: e.ProtectionRules,
			})
		}
		if resp == nil || resp.NextPage == 0 || len(payload.Environments) < 100 {
			break
		}
		page = resp.NextPage
	}
	return all, nil
}

// CreateDeploymentStatus creates a deployment status for a deployment.
func (c *Client) CreateDeploymentStatus(ctx context.Context, owner, repo string, deploymentID int64, state, description string) error {
	body := map[string]any{
		"state":       state,
		"description": description,
	}
	path := fmt.Sprintf("repos/%s/%s/deployments/%d/statuses", url.PathEscape(owner), url.PathEscape(repo), deploymentID)
	if _, err := c.rest(ctx, http.MethodPost, path, body, nil); err != nil {
		return fmt.Errorf("github: create deployment status: %w", err)
	}
	return nil
}
