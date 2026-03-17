package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

// GoalMaintainer updates goal-list.md completion status based on execution results.
type GoalMaintainer struct {
	LLM      llm.LLMProvider
	Store    *dotgitdex.Manager
	Language string
}

type goalUpdateResponse struct {
	Goals []goalItem `json:"goals"`
}

type goalItem struct {
	Title     string     `json:"title"`
	Completed bool       `json:"completed"`
	Todos     []todoItem `json:"todos,omitempty"`
}

type todoItem struct {
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// GoalTriageResult is the output of TriageGoal: classifies whether a goal
// is achievable by Gitdex and, if so, decomposes it into sub-tasks.
type GoalTriageResult struct {
	Achievable bool           `json:"achievable"`
	Reason     string         `json:"reason"`
	Category   string         `json:"category"` // "gitdex", "creative", "discard"
	Todos      []dotgitdex.Todo `json:"-"`
}

// TriageAndDecomposeGoal first classifies whether a goal is achievable by Gitdex
// (git commands, file operations, GitHub API), then decomposes achievable goals
// into sub-tasks. Non-achievable goals are categorized for creative-proposal or discard.
func (gm *GoalMaintainer) TriageAndDecomposeGoal(ctx context.Context, goalTitle, gitContent string) (*GoalTriageResult, error) {
	system := `You are a goal-triage and decomposition assistant for Gitdex, an AI-native Git workbench.

STEP 1 - TRIAGE: Determine whether the given goal can be achieved by Gitdex.
Gitdex CAN do: git commands, file read/write/create/delete, GitHub API operations
(branches, commits, PRs, issues, pages, actions, deployments, tags, releases, README edits).
Gitdex CANNOT do: write application code logic, run external build systems it doesn't know about,
interact with non-Git services, make subjective creative decisions, or perform actions
requiring human judgment beyond git/GitHub scope.

STEP 2 - CLASSIFY:
- "gitdex": The goal is achievable through git/file/GitHub operations → decompose into sub-tasks.
- "creative": The goal is inspirational/strategic (e.g. "improve code quality") → save as creative proposal.
- "discard": The goal is impossible, nonsensical, dangerous, or a duplicate.

STEP 3 - DECOMPOSE (only if category is "gitdex"):
Break the goal into 3-10 concrete, ordered, actionable sub-tasks.

OUTPUT FORMAT (strict JSON, no markdown fences):
{
  "achievable": true/false,
  "category": "gitdex" | "creative" | "discard",
  "reason": "brief explanation of the classification",
  "todos": [{"title": "Sub-task description"}]
}

For "creative" or "discard" categories, "todos" should be an empty array.
Consider the current repository state when making your decision.`

	user := fmt.Sprintf(`## Goal
%s

## Current Repository State
%s

Triage this goal and decompose if achievable.`, goalTitle, gitContent)

	resp, err := gm.LLM.Generate(ctx, llm.GenerateRequest{
		Role:   llm.RoleSecondary,
		System: system,
		Prompt: user,
	})
	if err != nil {
		return nil, err
	}

	text := cleanJSON(resp.Text)
	var raw struct {
		Achievable bool   `json:"achievable"`
		Category   string `json:"category"`
		Reason     string `json:"reason"`
		Todos      []struct {
			Title string `json:"title"`
		} `json:"todos"`
	}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, err
	}

	result := &GoalTriageResult{
		Achievable: raw.Achievable,
		Reason:     raw.Reason,
		Category:   raw.Category,
	}

	if result.Category == "" {
		if result.Achievable {
			result.Category = "gitdex"
		} else {
			result.Category = "creative"
		}
	}

	if result.Category == "gitdex" {
		for _, t := range raw.Todos {
			if t.Title != "" {
				result.Todos = append(result.Todos, dotgitdex.Todo{Title: t.Title})
			}
		}
	}

	return result, nil
}

// DecomposeGoal is a backward-compatible wrapper that calls TriageAndDecomposeGoal
// and returns only the todos (for goals already known to be achievable).
func (gm *GoalMaintainer) DecomposeGoal(ctx context.Context, goalTitle, gitContent string) ([]dotgitdex.Todo, error) {
	result, err := gm.TriageAndDecomposeGoal(ctx, goalTitle, gitContent)
	if err != nil {
		return nil, err
	}
	return result.Todos, nil
}

// UpdateGoalCompletion asks the helper LLM to evaluate which goals/todos
// are now complete based on git state and recent execution output.
func (gm *GoalMaintainer) UpdateGoalCompletion(ctx context.Context, gitContent, output string) error {
	goals, err := gm.Store.ReadGoalList()
	if err != nil || len(goals) == 0 {
		return err
	}

	// Send ALL goals (both completed and pending) so the LLM has full picture
	goalText := formatAllGoalsForUpdate(goals)
	if goalText == "" {
		return nil
	}

	system := `You are a goal-tracking assistant. You evaluate sub-task completion based on EVIDENCE from the execution log and git state.

RULES:
1. Read the execution log carefully. If a step marked [OK] accomplished a sub-task, mark that sub-task as completed.
2. Check the git state for evidence: if a file was supposed to be created and it exists in the working tree, that sub-task is done.
3. A goal is completed when ALL of its sub-tasks are completed.
4. Do NOT mark items as completed without evidence. Do NOT unmark items that were already completed.
5. Return ALL goals (both completed and pending) with their current status.

OUTPUT FORMAT (strict JSON, no markdown fences):
{"goals": [{"title": "...", "completed": false, "todos": [{"title": "...", "completed": true}]}]}`

	user := fmt.Sprintf(`## Git Context (current state)
%s

## Recent Execution Log (evidence of completed work)
%s

## Goal List (evaluate each sub-task against the evidence above)
%s

For each sub-task, check: does the execution log show [OK] entries that accomplish it? Does the git state confirm it?`, gitContent, output, goalText)

	resp, err := gm.LLM.Generate(ctx, llm.GenerateRequest{
		Role:   llm.RoleSecondary,
		System: system,
		Prompt: user,
	})
	if err != nil {
		return err
	}

	text := cleanJSON(resp.Text)
	var result goalUpdateResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return err
	}

	updated := mergeGoalUpdates(goals, result.Goals)
	return gm.Store.WriteGoalList(updated)
}

// formatAllGoalsForUpdate renders ALL goals with their completion status
// so the LLM can see what was already done and evaluate new completions.
func formatAllGoalsForUpdate(goals []dotgitdex.Goal) string {
	if len(goals) == 0 {
		return ""
	}
	var b strings.Builder
	for _, g := range goals {
		mark := " "
		if g.Completed {
			mark = "x"
		}
		b.WriteString(fmt.Sprintf("- [%s] %s\n", mark, g.Title))
		for _, t := range g.Todos {
			tMark := " "
			if t.Completed {
				tMark = "x"
			}
			b.WriteString(fmt.Sprintf("  - [%s] %s\n", tMark, t.Title))
		}
	}
	return b.String()
}

func mergeGoalUpdates(existing []dotgitdex.Goal, updates []goalItem) []dotgitdex.Goal {
	updateMap := make(map[string]*goalItem, len(updates))
	for i := range updates {
		updateMap[normalizeTitle(updates[i].Title)] = &updates[i]
	}

	for i := range existing {
		u, ok := updateMap[normalizeTitle(existing[i].Title)]
		if !ok {
			continue
		}
		// Only mark complete, never un-complete (monotonic progress)
		if u.Completed && !existing[i].Completed {
			existing[i].Completed = true
		}
		todoMap := make(map[string]bool, len(u.Todos))
		for _, t := range u.Todos {
			todoMap[normalizeTitle(t.Title)] = t.Completed
		}
		for j := range existing[i].Todos {
			if done, ok := todoMap[normalizeTitle(existing[i].Todos[j].Title)]; ok {
				// Only mark complete, never un-complete
				if done && !existing[i].Todos[j].Completed {
					existing[i].Todos[j].Completed = true
				}
			}
		}
	}
	return existing
}

// normalizeTitle strips punctuation and lowercases for fuzzy matching.
func normalizeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || r == '.' || r == ':' || r == ',' {
			return ' '
		}
		return r
	}, s)
	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}
