package collector

import (
	"context"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func TestNewGitHubCollector_ReturnsNonNil(t *testing.T) {
	c := NewGitHubCollector()
	if c == nil {
		t.Fatal("NewGitHubCollector returned nil")
	}
}

func TestGitHubCollector_Collect_NilState_ReturnsEmptyContext(t *testing.T) {
	c := NewGitHubCollector()
	ctx := context.Background()

	ghCtx, err := c.Collect(ctx, nil)
	if err != nil {
		t.Fatalf("Collect with nil state: got err %v", err)
	}
	if ghCtx == nil {
		t.Fatal("Collect returned nil GitHubContext")
	}
	if len(ghCtx.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(ghCtx.Issues))
	}
	if len(ghCtx.PullRequests) != 0 {
		t.Errorf("expected 0 pull requests, got %d", len(ghCtx.PullRequests))
	}
	if ghCtx.LocalREADME != "" {
		t.Errorf("expected empty LocalREADME, got %q", ghCtx.LocalREADME)
	}
	if ghCtx.UpstreamREADME != "" {
		t.Errorf("expected empty UpstreamREADME, got %q", ghCtx.UpstreamREADME)
	}
}

func TestGitHubContext_FormatForPrompt_EmptyContext_ReturnsEmptyString(t *testing.T) {
	ghCtx := &GitHubContext{}
	got := ghCtx.FormatForPrompt()
	if got != "" {
		t.Errorf("FormatForPrompt with empty context: got %q, want \"\"", got)
	}
}

func TestGitHubContext_FormatForPrompt_NilContext_ReturnsEmptyString(t *testing.T) {
	var ghCtx *GitHubContext
	got := ghCtx.FormatForPrompt()
	if got != "" {
		t.Errorf("FormatForPrompt with nil context: got %q, want \"\"", got)
	}
}

func TestGitHubContext_FormatForPrompt_WithIssues_RendersThem(t *testing.T) {
	ghCtx := &GitHubContext{
		Issues: []IssueSummary{
			{Number: 1, Title: "Fix bug", State: "open", Author: "alice"},
			{Number: 2, Title: "Add feature", State: "closed", Author: "bob", Labels: "enhancement"},
		},
	}
	got := ghCtx.FormatForPrompt()

	want := "## GitHub Issues\n  #1 [open] Fix bug (by alice)\n  #2 [closed] Add feature (by bob)\n"
	if got != want {
		t.Errorf("FormatForPrompt with issues:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestGitHubContext_FormatForPrompt_WithPRs_RendersThem(t *testing.T) {
	ghCtx := &GitHubContext{
		PullRequests: []PRSummary{
			{Number: 10, Title: "Merge feature", State: "open", Author: "carol", Base: "main", Head: "feature-x"},
			{Number: 11, Title: "Hotfix", State: "merged", Author: "dave", Base: "main", Head: "hotfix-1"},
		},
	}
	got := ghCtx.FormatForPrompt()

	want := "\n## GitHub Pull Requests\n  #10 [open] Merge feature (by carol, feature-x -> main)\n  #11 [merged] Hotfix (by dave, hotfix-1 -> main)\n"
	if got != want {
		t.Errorf("FormatForPrompt with PRs:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestGitHubContext_FormatForPrompt_WithIssuesAndPRs_RendersBoth(t *testing.T) {
	ghCtx := &GitHubContext{
		Issues: []IssueSummary{
			{Number: 1, Title: "Bug", State: "open", Author: "alice"},
		},
		PullRequests: []PRSummary{
			{Number: 5, Title: "PR", State: "open", Author: "bob", Base: "main", Head: "dev"},
		},
	}
	got := ghCtx.FormatForPrompt()

	if !contains(got, "## GitHub Issues") {
		t.Error("expected ## GitHub Issues section")
	}
	if !contains(got, "#1 [open] Bug (by alice)") {
		t.Error("expected issue line")
	}
	if !contains(got, "## GitHub Pull Requests") {
		t.Error("expected ## GitHub Pull Requests section")
	}
	if !contains(got, "#5 [open] PR (by bob, dev -> main)") {
		t.Error("expected PR line")
	}
}

func TestGitHubContext_FormatForPrompt_WithLocalREADME_RendersIt(t *testing.T) {
	readme := "# My Project\n\nHello world."
	ghCtx := &GitHubContext{LocalREADME: readme}
	got := ghCtx.FormatForPrompt()

	want := "\n## Local README\n# My Project\n\nHello world.\n"
	if got != want {
		t.Errorf("FormatForPrompt with LocalREADME:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestGitHubContext_FormatForPrompt_WithUpstreamREADME_RendersIt(t *testing.T) {
	readme := "# Upstream README content"
	ghCtx := &GitHubContext{UpstreamREADME: readme}
	got := ghCtx.FormatForPrompt()

	want := "\n## Upstream README\n# Upstream README content\n"
	if got != want {
		t.Errorf("FormatForPrompt with UpstreamREADME:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestGitHubContext_FormatForPrompt_WithBothREADMEs_RendersBoth(t *testing.T) {
	local := "local readme"
	upstream := "upstream readme"
	ghCtx := &GitHubContext{LocalREADME: local, UpstreamREADME: upstream}
	got := ghCtx.FormatForPrompt()

	if !contains(got, "## Local README") || !contains(got, local) {
		t.Error("expected Local README section with content")
	}
	if !contains(got, "## Upstream README") || !contains(got, upstream) {
		t.Error("expected Upstream README section with content")
	}
}

func TestNewGitCollector_ReturnsNonNil(t *testing.T) {
	watcher := (*status.StatusWatcher)(nil)
	store := dotgitdex.New("/tmp/test-repo")
	c := NewGitCollector(watcher, store)
	if c == nil {
		t.Fatal("NewGitCollector returned nil")
	}
}

func TestDecodeGitHubREADMEContent_Base64(t *testing.T) {
	raw := "IyBUaXRsZQoKQm9keQo="
	got := decodeGitHubREADMEContent(raw)
	want := "# Title\n\nBody\n"
	if got != want {
		t.Fatalf("decoded README mismatch: got %q, want %q", got, want)
	}
}

func TestDecodeGitHubREADMEContent_FallbackPlainText(t *testing.T) {
	raw := "# Plain README\nLine 2"
	got := decodeGitHubREADMEContent(raw)
	if got != raw {
		t.Fatalf("plain text should be returned unchanged: got %q", got)
	}
}

func TestGhBinary_UsesConfigValue(t *testing.T) {
	orig := config.Get()
	t.Cleanup(func() {
		config.Set(orig)
	})
	cfg := config.DefaultConfig()
	cfg.Adapters.GitHub.GH.Binary = "gh-custom"
	config.Set(cfg)
	if got := ghBinary(); got != "gh-custom" {
		t.Fatalf("ghBinary() = %q, want gh-custom", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (s[:len(sub)] == sub || contains(s[1:], sub))))
}
