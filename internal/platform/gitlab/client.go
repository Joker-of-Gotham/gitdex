package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/Joker-of-Gotham/gitdex/internal/platform/contributing"
)

// Client is a GitLab API client for MR and issue operations.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
	projectID  string
}

// New creates a new GitLab API client. projectID can be numeric or URL-encoded path (e.g. namespace%2Fproject).
func New(token, projectID string) *Client {
	return &Client{
		token:      token,
		baseURL:    "https://gitlab.com/api/v4",
		httpClient: &http.Client{},
		projectID:  projectID,
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}
	return c.httpClient.Do(req)
}

type glMRRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	Draft        bool   `json:"draft"`
}

type glMRResponse struct {
	WebURL string `json:"web_url"`
	IID    int    `json:"iid"`
}

// CreatePR creates a merge request. POST /projects/:id/merge_requests
func (c *Client) CreatePR(ctx context.Context, req platform.PRRequest) (*platform.PRResponse, error) {
	glReq := glMRRequest{
		Title:        req.Title,
		Description:  req.Body,
		SourceBranch: req.HeadBranch,
		TargetBranch: req.BaseBranch,
		Draft:        req.Draft,
	}
	path := fmt.Sprintf("/projects/%s/merge_requests", url.PathEscape(c.projectID))
	resp, err := c.doRequest(ctx, http.MethodPost, path, glReq)
	if err != nil {
		return nil, fmt.Errorf("create MR: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create MR: status %d: %s", resp.StatusCode, string(body))
	}
	var glResp glMRResponse
	if err := json.NewDecoder(resp.Body).Decode(&glResp); err != nil {
		return nil, fmt.Errorf("decode MR response: %w", err)
	}
	return &platform.PRResponse{URL: glResp.WebURL, Number: glResp.IID}, nil
}

type glIssue struct {
	IID    int    `json:"iid"`
	Title  string `json:"title"`
	WebURL string `json:"web_url"`
}

// ListIssues lists issues matching the filter. GET /projects/:id/issues
func (c *Client) ListIssues(ctx context.Context, filter platform.IssueFilter) ([]platform.Issue, error) {
	path := fmt.Sprintf("/projects/%s/issues?state=%s", url.PathEscape(c.projectID), url.QueryEscape(filter.State))
	if len(filter.Labels) > 0 {
		path += "&labels=" + url.QueryEscape(strings.Join(filter.Labels, ","))
	}
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list issues: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list issues: status %d: %s", resp.StatusCode, string(body))
	}
	var glIssues []glIssue
	if err := json.NewDecoder(resp.Body).Decode(&glIssues); err != nil {
		return nil, fmt.Errorf("decode issues: %w", err)
	}
	issues := make([]platform.Issue, len(glIssues))
	for i, gi := range glIssues {
		issues[i] = platform.Issue{Number: gi.IID, Title: gi.Title, URL: gi.WebURL}
	}
	return issues, nil
}

// DetectPlatform returns PlatformGitLab for GitLab remote URLs.
func (c *Client) DetectPlatform(ctx context.Context, remoteURL string) (platform.Platform, error) {
	_ = ctx
	_ = c
	return platform.DetectPlatform(remoteURL), nil
}

// GetContributing fetches CONTRIBUTING.md. GET /projects/:id/repository/files/:file_path/raw
func (c *Client) GetContributing(ctx context.Context) (*platform.ContributingSpec, error) {
	paths := []string{"CONTRIBUTING.md", ".github/CONTRIBUTING.md", "docs/CONTRIBUTING.md"}
	for _, p := range paths {
		encodedPath := url.PathEscape(p)
		path := fmt.Sprintf("/projects/%s/repository/files/%s/raw?ref=HEAD", url.PathEscape(c.projectID), encodedPath)
		resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		content, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		return contributing.Parse(string(content)), nil
	}
	return &platform.ContributingSpec{}, nil
}
