package executor

import (
	"context"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/contract"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

type chaosGitAdapter struct {
	result *ExecutionResult
}

func (a chaosGitAdapter) ExecGit(context.Context, string) *ExecutionResult {
	return a.result
}

func TestChaosDrill_NetworkFailure(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	r.gitAdapter = chaosGitAdapter{
		result: &ExecutionResult{
			Command: "git fetch --all",
			Stderr:  "connection timeout to remote",
			Success: false,
		},
	}

	got := r.ExecuteSuggestion(context.Background(), 1, planner.SuggestionItem{
		Name:   "sync remotes",
		Reason: "refresh state",
		Action: planner.ActionSpec{Type: "git_command", Command: "git fetch --all"},
	})
	if got.RecoverBy.Type != contract.RecoveryRetry {
		t.Fatalf("expected retry recovery for network failure, got %+v", got.RecoverBy)
	}
}

func TestChaosDrill_AuthFailure(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	r.gitAdapter = chaosGitAdapter{
		result: &ExecutionResult{
			Command: "git push origin main",
			Stderr:  "HTTP 401 unauthorized",
			Success: false,
		},
	}

	got := r.ExecuteSuggestion(context.Background(), 1, planner.SuggestionItem{
		Name:   "publish branch",
		Reason: "share changes",
		Action: planner.ActionSpec{Type: "git_command", Command: "git push origin main"},
	})
	if got.RecoverBy.Type != contract.RecoveryManual {
		t.Fatalf("expected manual recovery for auth failure, got %+v", got.RecoverBy)
	}
}
