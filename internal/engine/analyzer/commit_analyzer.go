package analyzer

import (
	"context"
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
)

type LLMProvider interface {
	Generate(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error)
	IsAvailable(ctx context.Context) bool
}

type CommitAnalyzer struct{}

func NewCommitAnalyzer() *CommitAnalyzer { return &CommitAnalyzer{} }

// GenerateMessageWithLLM sends staged files and diff to LLM for commit message generation.
// Falls back to rule-based GenerateMessage if LLM unavailable or fails.
func (a *CommitAnalyzer) GenerateMessageWithLLM(ctx context.Context, staged []git.FileStatus, diffs string, provider LLMProvider) string {
	if len(staged) == 0 {
		return ""
	}
	if provider == nil || !provider.IsAvailable(ctx) {
		return a.GenerateMessage(staged)
	}
	builder := prompt.NewBuilder()
	system, user := builder.BuildCommitMessage(staged, diffs)
	req := llm.GenerateRequest{
		System:      system,
		Prompt:      user,
		Temperature: builder.Temperature(),
	}
	resp, err := provider.Generate(ctx, req)
	if err != nil {
		return a.GenerateMessage(staged)
	}
	msg := strings.TrimSpace(resp.Text)
	if msg == "" {
		return a.GenerateMessage(staged)
	}
	return msg
}

// GenerateMessage analyzes staged files and generates a Conventional Commit message
func (a *CommitAnalyzer) GenerateMessage(staged []git.FileStatus) string {
	if len(staged) == 0 {
		return ""
	}

	// Determine type based on file patterns
	commitType := a.detectType(staged)
	scope := a.detectScope(staged)
	description := a.buildDescription(staged)

	if scope != "" {
		return fmt.Sprintf("%s(%s): %s", commitType, scope, description)
	}
	return fmt.Sprintf("%s: %s", commitType, description)
}

func (a *CommitAnalyzer) detectType(staged []git.FileStatus) string {
	// If all files are new -> "feat"
	// If has deletions -> "refactor"
	// If has test files -> "test"
	// If has doc files -> "docs"
	// Default -> "chore"
	hasNew := false
	hasMod := false
	for _, f := range staged {
		if f.StagingCode == git.StatusAdded {
			hasNew = true
		}
		if f.StagingCode == git.StatusModified {
			hasMod = true
		}
	}
	if hasNew && !hasMod {
		return "feat"
	}
	if hasMod {
		return "fix"
	}
	return "chore"
}

func (a *CommitAnalyzer) detectScope(staged []git.FileStatus) string {
	// Extract common directory prefix
	if len(staged) == 1 {
		parts := strings.Split(staged[0].Path, "/")
		if len(parts) > 1 {
			return parts[0]
		}
	}
	return ""
}

func (a *CommitAnalyzer) buildDescription(staged []git.FileStatus) string {
	if len(staged) == 1 {
		action := "update"
		if staged[0].StagingCode == git.StatusAdded {
			action = "add"
		}
		if staged[0].StagingCode == git.StatusDeleted {
			action = "remove"
		}
		parts := strings.Split(staged[0].Path, "/")
		return fmt.Sprintf("%s %s", action, parts[len(parts)-1])
	}
	return fmt.Sprintf("update %d files", len(staged))
}
