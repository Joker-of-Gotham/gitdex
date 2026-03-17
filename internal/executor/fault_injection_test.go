package executor

import (
	"context"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/contract"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

type injectedGitAdapter struct {
	result *ExecutionResult
}

func (a injectedGitAdapter) ExecGit(context.Context, string) *ExecutionResult {
	return a.result
}

func TestFaultInjection_TransientFailureGetsRetryRecovery(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	r.gitAdapter = injectedGitAdapter{
		result: &ExecutionResult{
			Command: "git fetch --prune",
			Stderr:  "HTTP 503: temporarily unavailable",
			Success: false,
		},
	}
	item := planner.SuggestionItem{
		Name:   "sync remotes",
		Reason: "refresh refs",
		Action: planner.ActionSpec{Type: "git_command", Command: "git fetch --prune"},
	}
	got := r.ExecuteSuggestion(context.Background(), 1, item)
	if got.RecoverBy.Type != contract.RecoveryRetry {
		t.Fatalf("expected retry recovery, got %+v", got.RecoverBy)
	}
}

func TestFaultInjection_AuthFailureGetsManualRecovery(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	r.gitAdapter = injectedGitAdapter{
		result: &ExecutionResult{
			Command: "git push origin main",
			Stderr:  "HTTP 403 forbidden",
			Success: false,
		},
	}
	item := planner.SuggestionItem{
		Name:   "push branch",
		Reason: "publish change",
		Action: planner.ActionSpec{Type: "git_command", Command: "git push origin main"},
	}
	got := r.ExecuteSuggestion(context.Background(), 1, item)
	if got.RecoverBy.Type != contract.RecoveryManual {
		t.Fatalf("expected manual recovery, got %+v", got.RecoverBy)
	}
}
