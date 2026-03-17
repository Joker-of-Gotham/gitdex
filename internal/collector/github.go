package collector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

// IssueSummary is a lightweight snapshot of a GitHub issue.
type IssueSummary struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author string `json:"author"`
	Labels string `json:"labels,omitempty"`
}

// PRSummary is a lightweight snapshot of a GitHub pull request.
type PRSummary struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author string `json:"author"`
	Base   string `json:"base"`
	Head   string `json:"head"`
}

// GitHubContext holds sampled issue/PR data and README content for creative flow.
type GitHubContext struct {
	Issues         []IssueSummary `json:"issues"`
	PullRequests   []PRSummary    `json:"pull_requests"`
	LocalREADME    string         `json:"local_readme,omitempty"`
	UpstreamREADME string         `json:"upstream_readme,omitempty"`
}

// GitHubCollector fetches GitHub context for the creative flow.
type GitHubCollector struct {
	// In a full implementation this would hold a GitHub API client.
	// For now it provides a stub that can be wired up later.
}

// NewGitHubCollector creates a new GitHubCollector.
func NewGitHubCollector() *GitHubCollector {
	return &GitHubCollector{}
}

// Collect gathers GitHub issues, PRs, and README content using the gh CLI.
// The sampling strategy is: newest 10 + oldest 10 + middle 10 for both issues and PRs.
func (c *GitHubCollector) Collect(ctx context.Context, state *status.GitState) (*GitHubContext, error) {
	if state == nil {
		return &GitHubContext{}, nil
	}

	if _, err := exec.LookPath(ghBinary()); err != nil {
		return &GitHubContext{}, nil
	}

	result := &GitHubContext{}

	result.Issues = c.collectIssues(ctx)
	result.PullRequests = c.collectPRs(ctx)
	result.LocalREADME = c.readLocalREADME()
	result.UpstreamREADME = c.readUpstreamREADME(ctx, state)

	return result, nil
}

func (c *GitHubCollector) collectIssues(ctx context.Context) []IssueSummary {
	type ghIssue struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	newest := ghListJSON[ghIssue](ctx, "issue", "list", "--limit", "10", "--sort", "created", "--order", "desc", "--state", "all")
	oldest := ghListJSON[ghIssue](ctx, "issue", "list", "--limit", "10", "--sort", "created", "--order", "asc", "--state", "all")

	// Middle segment: get total count, then skip to midpoint
	var middle []ghIssue
	countOut, err := exec.CommandContext(ctx, ghBinary(), "issue", "list", "--state", "all", "--limit", "1", "--json", "number").Output()
	if err == nil {
		var countItems []struct{ Number int }
		_ = json.Unmarshal(countOut, &countItems)
		// gh doesn't support --offset, so we fetch by page if there are enough issues
		total := ghCountItems(ctx, "issue")
		if total > 30 {
			// Use --limit with a midpoint search (fetch from middle by sorting)
			mid := total / 2
			if mid > 10 {
				allMid := ghListJSON[ghIssue](ctx, "issue", "list", "--limit", fmt.Sprintf("%d", mid+10), "--sort", "created", "--order", "desc", "--state", "all")
				if len(allMid) > 10 {
					start := len(allMid) - 10
					if start < 0 {
						start = 0
					}
					middle = allMid[start:]
				}
			}
		}
	}

	seen := make(map[int]bool)
	var result []IssueSummary
	for _, list := range [][]ghIssue{newest, oldest, middle} {
		for _, iss := range list {
			if seen[iss.Number] {
				continue
			}
			seen[iss.Number] = true
			labels := ""
			for _, l := range iss.Labels {
				if labels != "" {
					labels += ","
				}
				labels += l.Name
			}
			result = append(result, IssueSummary{
				Number: iss.Number,
				Title:  iss.Title,
				State:  iss.State,
				Author: iss.Author.Login,
				Labels: labels,
			})
		}
	}
	return result
}

func (c *GitHubCollector) collectPRs(ctx context.Context) []PRSummary {
	type ghPR struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		BaseRefName string `json:"baseRefName"`
		HeadRefName string `json:"headRefName"`
	}

	open := ghListJSON[ghPR](ctx, "pr", "list", "--limit", "10", "--state", "open")
	closed := ghListJSON[ghPR](ctx, "pr", "list", "--limit", "10", "--state", "closed")
	// oldest (by created time)
	oldestOpen := ghListJSON[ghPR](ctx, "pr", "list", "--limit", "10", "--state", "merged", "--sort", "created", "--order", "asc")

	seen := make(map[int]bool)
	var result []PRSummary
	for _, list := range [][]ghPR{open, closed, oldestOpen} {
		for _, pr := range list {
			if seen[pr.Number] {
				continue
			}
			seen[pr.Number] = true
			result = append(result, PRSummary{
				Number: pr.Number,
				Title:  pr.Title,
				State:  pr.State,
				Author: pr.Author.Login,
				Base:   pr.BaseRefName,
				Head:   pr.HeadRefName,
			})
		}
	}
	return result
}

func ghCountItems(ctx context.Context, itemType string) int {
	out, err := exec.CommandContext(ctx, ghBinary(), itemType, "list", "--state", "all", "--limit", "1000", "--json", "number").Output()
	if err != nil {
		return 0
	}
	var items []struct{ Number int }
	_ = json.Unmarshal(out, &items)
	return len(items)
}

func ghListJSON[T any](ctx context.Context, args ...string) []T {
	fullArgs := append(args, "--json", ghJSONFields[T]())
	cmd := exec.CommandContext(ctx, ghBinary(), fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var result []T
	if err := json.Unmarshal(out, &result); err != nil {
		return nil
	}
	return result
}

func ghJSONFields[T any]() string {
	var zero T
	data, _ := json.Marshal(zero)
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	fields := make([]string, 0, len(m))
	for k := range m {
		fields = append(fields, k)
	}
	return strings.Join(fields, ",")
}

func (c *GitHubCollector) readLocalREADME() string {
	for _, name := range []string{"README.md", "README", "readme.md"} {
		data, err := os.ReadFile(name)
		if err == nil {
			return string(data)
		}
	}
	return ""
}

func (c *GitHubCollector) readUpstreamREADME(ctx context.Context, state *status.GitState) string {
	if state == nil || state.RemoteURLs == nil {
		return ""
	}
	if url, ok := state.RemoteURLs["upstream"]; ok && url != "" {
		return c.fetchREADMEFromRemote(ctx, url)
	}
	return ""
}

func (c *GitHubCollector) fetchREADMEFromRemote(ctx context.Context, remoteURL string) string {
	// Extract owner/repo from remote URL
	repo := extractGHRepo(remoteURL)
	if repo == "" {
		return ""
	}
	out, err := exec.CommandContext(ctx, ghBinary(), "api", fmt.Sprintf("repos/%s/readme", repo),
		"--jq", ".content", "-H", "Accept: application/vnd.github.v3+json").Output()
	if err != nil {
		return ""
	}
	return decodeGitHubREADMEContent(string(out))
}

func extractGHRepo(url string) string {
	url = strings.TrimSuffix(url, ".git")
	// Handle https://github.com/owner/repo
	if idx := strings.Index(url, "github.com/"); idx >= 0 {
		return url[idx+len("github.com/"):]
	}
	// Handle git@github.com:owner/repo
	if idx := strings.Index(url, "github.com:"); idx >= 0 {
		return url[idx+len("github.com:"):]
	}
	return ""
}

func ghBinary() string {
	if cfg := config.Get(); cfg != nil {
		if bin := strings.TrimSpace(cfg.Adapters.GitHub.GH.Binary); bin != "" {
			return bin
		}
	}
	return "gh"
}

func decodeGitHubREADMEContent(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	compact := strings.ReplaceAll(raw, "\n", "")
	compact = strings.ReplaceAll(compact, "\r", "")
	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		// Fallback for already-decoded content or non-standard output.
		return raw
	}
	return string(decoded)
}

// FormatForPrompt renders the GitHub context as text suitable for LLM consumption.
func (ghCtx *GitHubContext) FormatForPrompt() string {
	if ghCtx == nil {
		return ""
	}
	var b strings.Builder

	if len(ghCtx.Issues) > 0 {
		b.WriteString("## GitHub Issues\n")
		for _, iss := range ghCtx.Issues {
			b.WriteString(fmt.Sprintf("  #%d [%s] %s (by %s)\n", iss.Number, iss.State, iss.Title, iss.Author))
		}
	}

	if len(ghCtx.PullRequests) > 0 {
		b.WriteString("\n## GitHub Pull Requests\n")
		for _, pr := range ghCtx.PullRequests {
			b.WriteString(fmt.Sprintf("  #%d [%s] %s (by %s, %s -> %s)\n",
				pr.Number, pr.State, pr.Title, pr.Author, pr.Head, pr.Base))
		}
	}

	if ghCtx.LocalREADME != "" {
		b.WriteString("\n## Local README\n")
		b.WriteString(ghCtx.LocalREADME)
		b.WriteString("\n")
	}

	if ghCtx.UpstreamREADME != "" {
		b.WriteString("\n## Upstream README\n")
		b.WriteString(ghCtx.UpstreamREADME)
		b.WriteString("\n")
	}

	return b.String()
}
