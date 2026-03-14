package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

type PRSummary struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type PlatformState struct {
	Detected      string               `json:"detected"`
	DefaultBranch string               `json:"default_branch,omitempty"`
	CIStatus      string               `json:"ci_status,omitempty"`
	OpenPRs       []PRSummary          `json:"open_prs,omitempty"`
	Capabilities  []string             `json:"capabilities,omitempty"`
	AdminSummary  []string             `json:"admin_summary,omitempty"`
	SurfaceStates []string             `json:"surface_states,omitempty"`
	Playbooks     []CapabilityPlaybook `json:"playbooks,omitempty"`
	LastError     string               `json:"last_error,omitempty"`
}

type Collector struct {
	githubToken    string
	gitlabToken    string
	bitbucketToken string
	httpClient     *http.Client
}

func NewCollector(githubToken, gitlabToken, bitbucketToken string) *Collector {
	return &Collector{
		githubToken:    strings.TrimSpace(githubToken),
		gitlabToken:    strings.TrimSpace(gitlabToken),
		bitbucketToken: strings.TrimSpace(bitbucketToken),
		httpClient:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Collector) Collect(ctx context.Context, state *status.GitState) *PlatformState {
	if c == nil || state == nil || len(state.RemoteInfos) == 0 {
		return nil
	}
	remoteURL := preferredRemoteURL(state.RemoteInfos[0])
	for _, r := range state.RemoteInfos {
		if strings.EqualFold(r.Name, "origin") {
			remoteURL = preferredRemoteURL(r)
			break
		}
	}
	if strings.TrimSpace(remoteURL) == "" {
		return nil
	}

	p := DetectPlatform(remoteURL)
	out := &PlatformState{
		Detected: p.String(),
		CIStatus: "unknown",
	}
	out.Capabilities = CapabilityIDs(p)
	if len(out.Capabilities) > 0 {
		out.AdminSummary = append(out.AdminSummary, fmt.Sprintf("capability_catalog=%d", len(out.Capabilities)))
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	switch p {
	case PlatformGitHub:
		if c.githubToken == "" {
			return out
		}
		if err := c.collectGitHub(ctx, remoteURL, state.LocalBranch.Name, out); err != nil {
			out.LastError = err.Error()
		}
	case PlatformGitLab:
		if c.gitlabToken == "" {
			return out
		}
		if err := c.collectGitLab(ctx, remoteURL, state.LocalBranch.Name, out); err != nil {
			out.LastError = err.Error()
		}
	case PlatformBitbucket:
		if c.bitbucketToken == "" {
			return out
		}
		if err := c.collectBitbucket(ctx, remoteURL, state.LocalBranch.Name, out); err != nil {
			out.LastError = err.Error()
		}
	default:
	}
	return out
}

func (c *Collector) collectGitHub(ctx context.Context, remoteURL, branch string, out *PlatformState) error {
	owner, repo, err := parseGitHubOwnerRepo(remoteURL)
	if err != nil {
		return err
	}
	repoPath := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	var repoMeta struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := c.getJSON(ctx, repoPath, map[string]string{
		"Authorization": "Bearer " + c.githubToken,
		"Accept":        "application/vnd.github+json",
	}, &repoMeta); err == nil {
		out.DefaultBranch = strings.TrimSpace(repoMeta.DefaultBranch)
	}

	var pulls []struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		HTMLURL string `json:"html_url"`
	}
	if err := c.getJSON(ctx, repoPath+"/pulls?state=open&per_page=5", map[string]string{
		"Authorization": "Bearer " + c.githubToken,
		"Accept":        "application/vnd.github+json",
	}, &pulls); err == nil {
		for _, pr := range pulls {
			out.OpenPRs = append(out.OpenPRs, PRSummary{
				Number: pr.Number,
				Title:  pr.Title,
				URL:    pr.HTMLURL,
			})
		}
	}

	ref := strings.TrimSpace(branch)
	if ref == "" {
		ref = strings.TrimSpace(out.DefaultBranch)
	}
	if ref != "" {
		var statusResp struct {
			State string `json:"state"`
		}
		if err := c.getJSON(ctx, repoPath+"/commits/"+url.PathEscape(ref)+"/status", map[string]string{
			"Authorization": "Bearer " + c.githubToken,
			"Accept":        "application/vnd.github+json",
		}, &statusResp); err == nil {
			out.CIStatus = normalizeCIStatus(statusResp.State)
		}
	}
	c.collectSurfaceStates(ctx, PlatformGitHub, repoPath, map[string]string{
		"Authorization": "Bearer " + c.githubToken,
		"Accept":        "application/vnd.github+json",
	}, out)
	return nil
}

func (c *Collector) collectGitLab(ctx context.Context, remoteURL, branch string, out *PlatformState) error {
	projectPath, err := parseGitLabProjectPath(remoteURL)
	if err != nil {
		return err
	}
	encoded := url.PathEscape(projectPath)
	base := "https://gitlab.com/api/v4/projects/" + encoded

	var project struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := c.getJSON(ctx, base, map[string]string{
		"PRIVATE-TOKEN": c.gitlabToken,
	}, &project); err == nil {
		out.DefaultBranch = strings.TrimSpace(project.DefaultBranch)
	}

	var mrs []struct {
		IID    int    `json:"iid"`
		Title  string `json:"title"`
		WebURL string `json:"web_url"`
	}
	if err := c.getJSON(ctx, base+"/merge_requests?state=opened&per_page=5", map[string]string{
		"PRIVATE-TOKEN": c.gitlabToken,
	}, &mrs); err == nil {
		for _, pr := range mrs {
			out.OpenPRs = append(out.OpenPRs, PRSummary{
				Number: pr.IID,
				Title:  pr.Title,
				URL:    pr.WebURL,
			})
		}
	}

	ref := strings.TrimSpace(branch)
	if ref == "" {
		ref = strings.TrimSpace(out.DefaultBranch)
	}
	if ref != "" {
		var pipelines []struct {
			Status string `json:"status"`
		}
		if err := c.getJSON(ctx, base+"/pipelines?ref="+url.QueryEscape(ref)+"&per_page=1", map[string]string{
			"PRIVATE-TOKEN": c.gitlabToken,
		}, &pipelines); err == nil && len(pipelines) > 0 {
			out.CIStatus = normalizeCIStatus(pipelines[0].Status)
		}
	}

	c.collectSurfaceStates(ctx, PlatformGitLab, base, map[string]string{
		"PRIVATE-TOKEN": c.gitlabToken,
	}, out)

	return nil
}

func (c *Collector) probeEndpoint(ctx context.Context, endpoint string, headers map[string]string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "unavailable"
	}
	for k, v := range headers {
		if strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "unavailable"
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 128))
	if len(strings.TrimSpace(string(body))) == 0 {
		return "empty"
	}
	return "available"
}

func (c *Collector) collectSurfaceStates(ctx context.Context, platform Platform, basePath string, headers map[string]string, out *PlatformState) {
	probes := CapabilityProbes(platform)
	for _, probe := range probes {
		state := c.probeEndpoint(ctx, basePath+probe.RelativePath, headers)
		out.SurfaceStates = append(out.SurfaceStates, fmt.Sprintf("%s=%s", probe.CapabilityID, state))
	}
	if len(out.SurfaceStates) > 0 {
		out.AdminSummary = append(out.AdminSummary, out.SurfaceStates...)
	}
}

func (c *Collector) collectBitbucket(ctx context.Context, remoteURL, branch string, out *PlatformState) error {
	workspace, repo, err := parseBitbucketWorkspaceRepo(remoteURL)
	if err != nil {
		return err
	}

	base := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s", url.PathEscape(workspace), url.PathEscape(repo))

	var repository struct {
		Mainbranch struct {
			Name string `json:"name"`
		} `json:"mainbranch"`
	}
	if err := c.getJSON(ctx, base, map[string]string{
		"Authorization": "Bearer " + c.bitbucketToken,
	}, &repository); err == nil {
		out.DefaultBranch = strings.TrimSpace(repository.Mainbranch.Name)
	}

	var pulls struct {
		Values []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Links struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"values"`
	}
	if err := c.getJSON(ctx, base+"/pullrequests?state=OPEN&pagelen=5", map[string]string{
		"Authorization": "Bearer " + c.bitbucketToken,
	}, &pulls); err == nil {
		for _, pr := range pulls.Values {
			out.OpenPRs = append(out.OpenPRs, PRSummary{
				Number: pr.ID,
				Title:  pr.Title,
				URL:    pr.Links.HTML.Href,
			})
		}
	}

	ref := strings.TrimSpace(branch)
	if ref == "" {
		ref = strings.TrimSpace(out.DefaultBranch)
	}
	if ref != "" {
		var pipelines struct {
			Values []struct {
				State struct {
					Name   string `json:"name"`
					Result struct {
						Name string `json:"name"`
					} `json:"result"`
				} `json:"state"`
			} `json:"values"`
		}
		if err := c.getJSON(ctx, base+"/pipelines/?sort=-created_on&target.ref_name="+url.QueryEscape(ref)+"&pagelen=1", map[string]string{
			"Authorization": "Bearer " + c.bitbucketToken,
		}, &pipelines); err == nil && len(pipelines.Values) > 0 {
			stateName := pipelines.Values[0].State.Result.Name
			if strings.TrimSpace(stateName) == "" {
				stateName = pipelines.Values[0].State.Name
			}
			out.CIStatus = normalizeCIStatus(stateName)
		}
	}

	c.collectSurfaceStates(ctx, PlatformBitbucket, base, map[string]string{
		"Authorization": "Bearer " + c.bitbucketToken,
	}, out)

	return nil
}

func (c *Collector) getJSON(ctx context.Context, endpoint string, headers map[string]string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		if strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
		return fmt.Errorf("platform api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func parseGitHubOwnerRepo(remoteURL string) (owner, repo string, err error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		path := strings.TrimPrefix(remoteURL, "git@github.com:")
		return splitOwnerRepo(path)
	}
	if strings.HasPrefix(remoteURL, "https://github.com/") || strings.HasPrefix(remoteURL, "http://github.com/") {
		u, parseErr := url.Parse(remoteURL)
		if parseErr != nil {
			return "", "", parseErr
		}
		return splitOwnerRepo(strings.TrimPrefix(u.Path, "/"))
	}
	return "", "", fmt.Errorf("unsupported github remote url")
}

func parseGitLabProjectPath(remoteURL string) (string, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if strings.HasPrefix(remoteURL, "git@gitlab.com:") {
		return normalizeRepoPath(strings.TrimPrefix(remoteURL, "git@gitlab.com:")), nil
	}
	if strings.HasPrefix(remoteURL, "https://gitlab.com/") || strings.HasPrefix(remoteURL, "http://gitlab.com/") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", err
		}
		return normalizeRepoPath(strings.TrimPrefix(u.Path, "/")), nil
	}
	return "", fmt.Errorf("unsupported gitlab remote url")
}

func parseBitbucketWorkspaceRepo(remoteURL string) (workspace, repo string, err error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if strings.HasPrefix(remoteURL, "git@bitbucket.org:") {
		path := strings.TrimPrefix(remoteURL, "git@bitbucket.org:")
		return splitOwnerRepo(path)
	}
	if strings.HasPrefix(remoteURL, "https://bitbucket.org/") || strings.HasPrefix(remoteURL, "http://bitbucket.org/") {
		u, parseErr := url.Parse(remoteURL)
		if parseErr != nil {
			return "", "", parseErr
		}
		return splitOwnerRepo(strings.TrimPrefix(u.Path, "/"))
	}
	return "", "", fmt.Errorf("unsupported bitbucket remote url")
}

func splitOwnerRepo(path string) (string, string, error) {
	path = normalizeRepoPath(path)
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository path")
	}
	return parts[0], parts[1], nil
}

func normalizeRepoPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".git")
	return path
}

func normalizeCIStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "success", "passed", "passing":
		return "passing"
	case "failed", "failure":
		return "failing"
	case "pending", "running":
		return "pending"
	default:
		return "unknown"
	}
}

func preferredRemoteURL(info git.RemoteInfo) string {
	push := strings.TrimSpace(info.PushURL)
	if push != "" {
		return push
	}
	return strings.TrimSpace(info.FetchURL)
}
