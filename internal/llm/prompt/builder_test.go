package prompt

import (
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/stretchr/testify/assert"
)

func TestBuildAnalyzeRichIncludesKnowledgeMemoryAndOps(t *testing.T) {
	builder := NewBuilderWithBudget(4096)

	system, user := builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{
			LocalBranch: git.BranchInfo{Name: "main", Upstream: "origin/main"},
			Remotes:     []string{"origin"},
			WorkingTree: []git.FileStatus{{Path: "README.md", WorktreeCode: git.StatusModified}},
		},
		Mode: "zen",
		RecentOps: []OperationRecord{{
			Type:    "executed",
			Command: "git status -sb",
			Result:  "success",
		}},
		Session: &SessionContext{
			ActiveGoal:  "整理发布前变更",
			Preferences: map[string]string{"remote_protocol": "ssh", "language": "zh"},
		},
		AnalysisHistory: []string{"上次分析认为应先检查远端状态"},
		PlatformState: &PlatformState{
			Detected:      "github",
			DefaultBranch: "main",
			CIStatus:      "passing",
		},
		Memory: &MemoryContext{
			UserPreferences: map[string]string{"language": "zh"},
			RepoPatterns:    []string{"release branch required"},
		},
		Knowledge: []KnowledgeFragment{{
			ScenarioID: "inspect#daily_check",
			Content:    "先看 git status -sb，再看 git diff --staged。",
		}},
		FileContext: &FileContext{
			Files: map[string]string{"README.md": "# title"},
		},
	})

	assert.NotEmpty(t, system)
	assert.Contains(t, user, "Relevant Git SOP/knowledge")
	assert.Contains(t, user, "Recently executed operations")
	assert.Contains(t, user, "ACTIVE GOAL")
	assert.Contains(t, user, "PREFERRED RESPONSE LANGUAGE: zh")
	assert.Contains(t, user, "Platform API state")
	assert.Contains(t, user, "README.md")

	trace := builder.LastBuildTrace()
	assert.NotEmpty(t, trace.Partitions)

	names := make([]string, 0, len(trace.Partitions))
	for _, part := range trace.Partitions {
		if part.Included {
			names = append(names, part.Name)
		}
	}
	assert.True(t, strings.Contains(strings.Join(names, ","), "knowledge"))
	assert.True(t, strings.Contains(strings.Join(names, ","), "recent_ops"))
	assert.True(t, strings.Contains(strings.Join(names, ","), "long_term_memory"))
}

func TestBuildAnalyzeRichUsesPreferredLanguage(t *testing.T) {
	builder := NewBuilderWithBudget(4096)

	system, _ := builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}},
		Session: &SessionContext{
			Preferences: map[string]string{"language": "en"},
		},
	})
	assert.Contains(t, system, "Output all text in English")
	assert.Contains(t, system, "situational analysis in English")

	system, _ = builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}},
		Session: &SessionContext{
			Preferences: map[string]string{"language": "zh"},
		},
	})
	assert.Contains(t, system, "Output all text in Simplified Chinese")
	assert.Contains(t, system, "situational analysis in Chinese")
}

func TestBuildAnalyzeRichIncludesActiveGoalStatus(t *testing.T) {
	builder := NewBuilderWithBudget(4096)

	_, user := builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}},
		Session: &SessionContext{
			ActiveGoal:       "Stabilize release flow",
			ActiveGoalStatus: "in_progress",
		},
	})

	assert.Contains(t, user, "ACTIVE GOAL: Stabilize release flow")
	assert.Contains(t, user, "GOAL STATUS: in_progress")
}

func TestSystemPromptHasNoConstraintKeywords(t *testing.T) {
	sys := analyzeSystem("en")
	forbidden := []string{
		"MUST",
		"MUST NOT",
		"Do NOT",
		"PRIORITY RULES",
		"Nothing else",
		"default step ladder",
	}
	for _, kw := range forbidden {
		assert.NotContains(t, sys, kw, "system prompt should not contain constraint keyword %q", kw)
	}
}

func TestSessionContextHasNoInstructionalLanguage(t *testing.T) {
	builder := NewBuilderWithBudget(4096)
	_, user := builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}},
		Session: &SessionContext{
			ActiveGoal: "Deploy to prod",
			SkippedActions: []string{"git push"},
		},
	})
	assert.NotContains(t, user, "Do NOT suggest unrelated actions")
	assert.NotContains(t, user, "do NOT repeat")
	assert.Contains(t, user, "ACTIVE GOAL: Deploy to prod")
	assert.Contains(t, user, "Previously skipped actions: git push")
}

func TestWorkflowOrchestrationIsLowPriority(t *testing.T) {
	builder := NewBuilderWithBudget(4096)

	_, user := builder.BuildAnalyzeRich(AnalyzeInput{
		State: &status.GitState{LocalBranch: git.BranchInfo{Name: "main"}},
		Session: &SessionContext{ActiveGoal: "Setup pages"},
		Workflow: &WorkflowOrchestration{
			WorkflowID: "pages_setup",
			Steps: []WorkflowOrchestrationStep{
				{Title: "Inspect pages", Capability: "pages", Flow: "inspect"},
			},
		},
	})
	assert.Contains(t, user, "Available platform operations")
	assert.NotContains(t, user, "Workflow orchestration hints")
	assert.NotContains(t, user, "Use them to shape the final suggestions")
}
