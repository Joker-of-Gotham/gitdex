package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/Joker-of-Gotham/gitdex/internal/platform/contributing"
)

// Client is a GitHub API client for PR/MR and issue operations.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
	owner      string
	repo       string
}

// New creates a new GitHub API client.
func New(token, owner, repo string) *Client {
	return &Client{
		token:      token,
		baseURL:    "https://api.github.com",
		httpClient: &http.Client{},
		owner:      owner,
		repo:       repo,
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
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.httpClient.Do(req)
}

type ghPRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Draft bool   `json:"draft"`
}

type ghPRResponse struct {
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
}

// CreatePR creates a pull request. POST /repos/{owner}/{repo}/pulls
func (c *Client) CreatePR(ctx context.Context, req platform.PRRequest) (*platform.PRResponse, error) {
	ghReq := ghPRRequest{
		Title: req.Title,
		Body:  req.Body,
		Head:  req.HeadBranch,
		Base:  req.BaseBranch,
		Draft: req.Draft,
	}
	resp, err := c.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/pulls", c.owner, c.repo), ghReq)
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create PR: status %d: %s", resp.StatusCode, string(body))
	}
	var ghResp ghPRResponse
	if err := json.NewDecoder(resp.Body).Decode(&ghResp); err != nil {
		return nil, fmt.Errorf("decode PR response: %w", err)
	}
	return &platform.PRResponse{URL: ghResp.HTMLURL, Number: ghResp.Number}, nil
}

type ghIssue struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
}

// ListIssues lists issues matching the filter.
func (c *Client) ListIssues(ctx context.Context, filter platform.IssueFilter) ([]platform.Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", c.owner, c.repo, url.QueryEscape(filter.State))
	if len(filter.Labels) > 0 {
		for _, label := range filter.Labels {
			path += "&labels=" + url.QueryEscape(label)
		}
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
	var ghIssues []ghIssue
	if err := json.NewDecoder(resp.Body).Decode(&ghIssues); err != nil {
		return nil, fmt.Errorf("decode issues: %w", err)
	}
	issues := make([]platform.Issue, len(ghIssues))
	for i, gi := range ghIssues {
		issues[i] = platform.Issue{Number: gi.Number, Title: gi.Title, URL: gi.HTMLURL}
	}
	return issues, nil
}

// DetectPlatform returns PlatformGitHub for GitHub remote URLs.
func (c *Client) DetectPlatform(ctx context.Context, remoteURL string) (platform.Platform, error) {
	_ = ctx
	_ = c
	return platform.DetectPlatform(remoteURL), nil
}

// GetContributing fetches CONTRIBUTING.md.
func (c *Client) GetContributing(ctx context.Context) (*platform.ContributingSpec, error) {
	paths := []string{"CONTRIBUTING.md", ".github/CONTRIBUTING.md", "docs/CONTRIBUTING.md"}
	for _, p := range paths {
		resp, err := c.doRequest(ctx, http.MethodGet,
			fmt.Sprintf("/repos/%s/%s/contents/%s", c.owner, c.repo, p), nil)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		var file struct {
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		if file.Encoding == "base64" {
			decoded, err := base64Decode(file.Content)
			if err != nil {
				continue
			}
			return contributing.Parse(decoded), nil
		}
	}
	return &platform.ContributingSpec{}, nil
}

func base64Decode(s string) (string, error) {
	s = strings.ReplaceAll(s, "\n", "")
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
