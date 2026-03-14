package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/Joker-of-Gotham/gitdex/internal/platform/contributing"
)

// Client is a Bitbucket API client for PR and issue operations.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
	workspace  string
	repo       string
}

// New creates a new Bitbucket API client.
func New(token, workspace, repo string) *Client {
	return &Client{
		token:      token,
		baseURL:    "https://api.bitbucket.org/2.0",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		workspace:  workspace,
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
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.httpClient.Do(req)
}

type bbPRRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Source      struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
}

type bbPRResponse struct {
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	ID int `json:"id"`
}

// CreatePR creates a pull request. POST /repositories/:workspace/:repo/pullrequests
func (c *Client) CreatePR(ctx context.Context, req platform.PRRequest) (*platform.PRResponse, error) {
	bbReq := bbPRRequest{
		Title:       req.Title,
		Description: req.Body,
	}
	bbReq.Source.Branch.Name = req.HeadBranch
	bbReq.Destination.Branch.Name = req.BaseBranch

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", url.PathEscape(c.workspace), url.PathEscape(c.repo))
	resp, err := c.doRequest(ctx, http.MethodPost, path, bbReq)
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create PR: status %d: %s", resp.StatusCode, string(body))
	}
	var bbResp bbPRResponse
	if err := json.NewDecoder(resp.Body).Decode(&bbResp); err != nil {
		return nil, fmt.Errorf("decode PR response: %w", err)
	}
	return &platform.PRResponse{URL: bbResp.Links.HTML.Href, Number: bbResp.ID}, nil
}

type bbIssue struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

type bbIssuesResponse struct {
	Values []bbIssue `json:"values"`
}

// ListIssues lists issues matching the filter. GET /repositories/:workspace/:repo/issues
func (c *Client) ListIssues(ctx context.Context, filter platform.IssueFilter) ([]platform.Issue, error) {
	path := fmt.Sprintf("/repositories/%s/%s/issues?q=state=%q", url.PathEscape(c.workspace), url.PathEscape(c.repo), filter.State)
	if len(filter.Labels) > 0 {
		path += "+AND+labels.name=" + url.QueryEscape(strings.Join(filter.Labels, ","))
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
	var bbResp bbIssuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&bbResp); err != nil {
		return nil, fmt.Errorf("decode issues: %w", err)
	}
	issues := make([]platform.Issue, len(bbResp.Values))
	for i, bi := range bbResp.Values {
		issues[i] = platform.Issue{Number: bi.ID, Title: bi.Title, URL: bi.Links.HTML.Href}
	}
	return issues, nil
}

// DetectPlatform returns PlatformBitbucket for Bitbucket remote URLs.
func (c *Client) DetectPlatform(ctx context.Context, remoteURL string) (platform.Platform, error) {
	_ = ctx
	_ = c
	return platform.DetectPlatform(remoteURL), nil
}

// GetContributing fetches CONTRIBUTING.md. GET /repositories/:workspace/:repo/src/:ref/CONTRIBUTING.md
func (c *Client) GetContributing(ctx context.Context) (*platform.ContributingSpec, error) {
	paths := []string{"CONTRIBUTING.md", ".github/CONTRIBUTING.md", "docs/CONTRIBUTING.md"}
	refs := []string{"main", "master", "HEAD"}
	for _, ref := range refs {
		for _, p := range paths {
			encodedPath := strings.ReplaceAll(p, "/", "%2F")
			path := fmt.Sprintf("/repositories/%s/%s/src/%s/%s", url.PathEscape(c.workspace), url.PathEscape(c.repo), ref, encodedPath)
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
	}
	return &platform.ContributingSpec{}, nil
}
