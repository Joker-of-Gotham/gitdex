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
	"time"

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
	transport  githubTransport
}

// New creates a new GitHub API client.
func New(token, owner, repo string) *Client {
	return &Client{
		token:      token,
		baseURL:    "https://api.github.com",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		owner:      owner,
		repo:       repo,
	}
}

func NewCLI(binary, owner, repo string) *Client {
	client := New("", owner, repo)
	client.transport = ghCLITransport{binary: binary}
	return client
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
	return c.doRequestAbsolute(ctx, method, reqURL, reqBody, map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
		"Content-Type":         "application/json",
	})
}

func (c *Client) doRequestAbsolute(ctx context.Context, method, reqURL string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.httpClient.Do(req)
}

func (c *Client) doBinaryUpload(ctx context.Context, reqURL string, contentType string, body []byte, expected ...int) (json.RawMessage, error) {
	if c.transport != nil {
		return c.transport.binaryUpload(ctx, c, reqURL, contentType, body, expected...)
	}
	resp, err := c.doRequestAbsolute(ctx, http.MethodPost, reqURL, bytes.NewReader(body), map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
		"Content-Type":         firstNonEmpty(contentType, "application/octet-stream"),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if !matchesStatus(resp.StatusCode, expected...) {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("POST %s: status %d: %s", reqURL, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	return json.RawMessage(data), nil
}

func (c *Client) downloadBytes(ctx context.Context, reqURL string, accept string, expected ...int) ([]byte, error) {
	if c.transport != nil {
		return c.transport.downloadBytes(ctx, c, reqURL, accept, expected...)
	}
	resp, err := c.doRequestAbsolute(ctx, http.MethodGet, reqURL, nil, map[string]string{
		"Accept":               firstNonEmpty(accept, "application/octet-stream"),
		"X-GitHub-Api-Version": "2022-11-28",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if !matchesStatus(resp.StatusCode, expected...) {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("GET %s: status %d: %s", reqURL, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return data, nil
}

func (c *Client) doGraphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	body := map[string]any{
		"query":     query,
		"variables": variables,
	}
	raw, err := c.doRaw(ctx, http.MethodPost, "/graphql", body, http.StatusOK)
	if err != nil {
		return err
	}
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode graphql envelope: %w", err)
	}
	if len(envelope.Errors) > 0 {
		messages := make([]string, 0, len(envelope.Errors))
		for _, item := range envelope.Errors {
			if strings.TrimSpace(item.Message) != "" {
				messages = append(messages, strings.TrimSpace(item.Message))
			}
		}
		return fmt.Errorf("graphql: %s", strings.Join(messages, "; "))
	}
	if out == nil || len(envelope.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode graphql data: %w", err)
	}
	return nil
}

func (c *Client) repoPath(suffix string) string {
	base := fmt.Sprintf("/repos/%s/%s", c.owner, c.repo)
	if suffix == "" {
		return base
	}
	if strings.HasPrefix(suffix, "/") {
		return base + suffix
	}
	return base + "/" + suffix
}

func (c *Client) orgPath(suffix string) string {
	base := fmt.Sprintf("/orgs/%s", c.owner)
	if suffix == "" {
		return base
	}
	if strings.HasPrefix(suffix, "/") {
		return base + suffix
	}
	return base + "/" + suffix
}

func (c *Client) userPath(suffix string) string {
	base := "/user"
	if suffix == "" {
		return base
	}
	if strings.HasPrefix(suffix, "/") {
		return base + suffix
	}
	return base + "/" + suffix
}

func (c *Client) doJSON(ctx context.Context, method, path string, body interface{}, out interface{}, expected ...int) error {
	raw, err := c.doRaw(ctx, method, path, body, expected...)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (c *Client) doRaw(ctx context.Context, method, path string, body interface{}, expected ...int) (json.RawMessage, error) {
	if c.transport != nil {
		return c.transport.raw(ctx, c, method, path, body, expected...)
	}
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if !matchesStatus(resp.StatusCode, expected...) {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	return json.RawMessage(data), nil
}

func matchesStatus(code int, expected ...int) bool {
	if len(expected) == 0 {
		return code >= 200 && code < 300
	}
	for _, item := range expected {
		if code == item {
			return true
		}
	}
	return false
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
