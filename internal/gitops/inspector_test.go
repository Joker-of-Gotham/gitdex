package gitops

import (
	"context"
	"testing"
)

func TestRecommend_Synced(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	rec := ins.Recommend(&RepoInspection{Divergence: DivSynced})
	if rec.Action != "none" {
		t.Errorf("got action %q, want %q", rec.Action, "none")
	}
}

func TestRecommend_Ahead(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	rec := ins.Recommend(&RepoInspection{Divergence: DivAhead, Ahead: 3})
	if rec.Action != "push" {
		t.Errorf("got action %q, want %q", rec.Action, "push")
	}
	if rec.RiskLevel != "low" {
		t.Errorf("got risk %q, want %q", rec.RiskLevel, "low")
	}
}

func TestRecommend_Behind(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	rec := ins.Recommend(&RepoInspection{Divergence: DivBehind, Behind: 5})
	if rec.Action != "fast_forward" {
		t.Errorf("got action %q, want %q", rec.Action, "fast_forward")
	}
}

func TestRecommend_BehindWithUncommitted(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	rec := ins.Recommend(&RepoInspection{Divergence: DivBehind, Behind: 5, HasUncommitted: true})
	if rec.Action != "stash_and_pull" {
		t.Errorf("got action %q, want %q", rec.Action, "stash_and_pull")
	}
	if rec.RiskLevel != "medium" {
		t.Errorf("got risk %q, want %q", rec.RiskLevel, "medium")
	}
}

func TestRecommend_Diverged(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	rec := ins.Recommend(&RepoInspection{Divergence: DivDiverged, Ahead: 2, Behind: 3})
	if rec.Action != "merge_or_rebase" {
		t.Errorf("got action %q, want %q", rec.Action, "merge_or_rebase")
	}
	if rec.RiskLevel != "high" {
		t.Errorf("got risk %q, want %q", rec.RiskLevel, "high")
	}
}

func TestRecommend_Detached(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	rec := ins.Recommend(&RepoInspection{Divergence: DivDetached})
	if rec.Action != "checkout_branch" {
		t.Errorf("got action %q, want %q", rec.Action, "checkout_branch")
	}
}

func TestRecommend_NoUpstream(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	rec := ins.Recommend(&RepoInspection{Divergence: DivNoUpstream})
	if rec.Action != "set_upstream" {
		t.Errorf("got action %q, want %q", rec.Action, "set_upstream")
	}
}

func TestRecommend_NilInspection(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	rec := ins.Recommend(nil)
	if rec.Action != "none" {
		t.Errorf("got action %q, want %q", rec.Action, "none")
	}
}

func TestInspect_EmptyPath(t *testing.T) {
	ins := NewInspector(NewGitExecutor())
	_, err := ins.Inspect(context.TODO(), "")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}
