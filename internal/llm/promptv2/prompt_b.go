package promptv2

import "fmt"

// BuildPromptB constructs the Planner prompt for maintenance suggestions.
func BuildPromptB(gitContent, output, knowledgeCtx, language string) (system, user string) {
	system = fmt.Sprintf(`Role: Git repository maintenance planner.

Task: Analyse the repository state and execution log. Produce 1-3 ordered
maintenance actions to make the repository clean, synced, and healthy.

Scope:
- Sync upstream/downstream (no behind commits)
- Clean working tree, staging area, and HEAD
- Resolve conflicts and abnormal states (detached HEAD, failed rebase, etc.)
- Prune merged branches and stale references

%s

Execution: Commands run via Go exec.Command (not a shell interpreter).
No pipes, no &&, no env var expansion, no glob expansion.
Each suggestion = exactly one action.

%s

%s

Rules:
- Review [OK] and [FAIL] entries in the execution log before suggesting any action.
- If repository is already clean, return empty suggestions.
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

Produce the maintenance suggestion sequence.`, gitContent, output, knowledgeCtx)

	return system, user
}
