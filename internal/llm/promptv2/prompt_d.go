package promptv2

import "fmt"

// BuildPromptD constructs the Planner prompt for goal-completion suggestions.
func BuildPromptD(gitContent, output, knowledgeCtx, goal, todoList, language string) (system, user string) {
	system = fmt.Sprintf(`Role: Goal-execution planner for a Git repository.

Task: Produce 1-3 ordered actions that advance the user's goal while
maintaining repository health. Focus on the current active goal and
address one to-do item at a time.

Scope: All maintenance operations (sync, clean, resolve, prune) plus
goal-specific operations (PR, CI, docs, deployment, file creation, etc.).

%s

Execution: Commands run via Go exec.Command (not a shell interpreter).
No pipes, no &&, no env var expansion, no glob expansion.
Each suggestion = exactly one action.

%s

%s

Rules:
- Review [OK] and [FAIL] entries in the execution log before suggesting any action.
- If goal and all to-do items are done and repo is clean, return empty suggestions.
- Respond with concrete, executable values only.

Respond in %s.`,
		RenderToolDefs(),
		platformGuidance(),
		outputSchema(),
		languageName(language))

	user = fmt.Sprintf(`## Git Context
%s

## Recent Execution Log
%s

## Knowledge Reference
%s

## Active Goal
%s

## To-Do List (pending items only)
%s

Produce the goal-completion suggestion sequence.`,
		gitContent, output, knowledgeCtx, goal, todoList)

	return system, user
}
