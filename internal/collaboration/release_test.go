package collaboration

import (
	"context"
	"testing"

	ghp "github.com/your-org/gitdex/internal/platform/github"
)

func TestGitHubReleaseEngine_NilClient(t *testing.T) {
	engine := NewGitHubReleaseEngine(nil)
	ctx := context.Background()

	_, err := engine.Assess(ctx, "owner", "repo", "v1.0.0")
	if err == nil {
		t.Fatal("expected error with nil client")
	}
}

func TestGitHubReleaseEngine_Assess_InvalidInput(t *testing.T) {
	engine := NewGitHubReleaseEngine(&ghp.Client{})
	ctx := context.Background()

	_, err := engine.Assess(ctx, "", "repo", "v1.0.0")
	if err == nil {
		t.Error("expected error for empty owner")
	}
	_, err = engine.Assess(ctx, "owner", "", "v1.0.0")
	if err == nil {
		t.Error("expected error for empty repo")
	}
	_, err = engine.Assess(ctx, "owner", "repo", "")
	if err == nil {
		t.Error("expected error for empty tag")
	}
}

func TestGitHubReleaseEngine_ListReleases_InvalidInput(t *testing.T) {
	engine := NewGitHubReleaseEngine(&ghp.Client{})
	ctx := context.Background()

	_, err := engine.ListReleases(ctx, "", "repo")
	if err == nil {
		t.Error("expected error for empty owner")
	}
}

func TestReleaseReadiness_Types(t *testing.T) {
	r := &ReleaseReadiness{
		RepoOwner: "owner",
		RepoName:  "repo",
		Tag:       "v1.0.0",
		Status:    ReleaseReady,
	}
	if r.Status != ReleaseReady {
		t.Errorf("status = %q, want %q", r.Status, ReleaseReady)
	}
	r.Status = ReleaseBlocked
	if r.Status != ReleaseBlocked {
		t.Errorf("status = %q, want %q", r.Status, ReleaseBlocked)
	}
}
