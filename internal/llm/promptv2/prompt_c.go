package promptv2

import "fmt"

// BuildPromptC constructs the Helper LLM prompt for goal-completion knowledge selection.
// Same as Prompt A but also includes the active goal and its to-do list.
func BuildPromptC(gitContent, output, index, goal, todoList, language string) (system, user string) {
	system = fmt.Sprintf(`You are a knowledge-index assistant for a Git repository goal-completion system.
Your job: given the repository state, execution logs, knowledge index, the user's goal,
and the current to-do list, select the knowledge files most relevant to making progress.

RULES:
- Select 1-5 files that help accomplish the goal while maintaining repository health.
- Consider both the goal's domain (PR, deployment, etc.) and any maintenance issues.
- Only select files relevant to the current goal and maintenance needs.

OUTPUT FORMAT (strict JSON, no markdown fences, no prose):
{"selected_knowledge": ["<full path 1>", "<full path 2>", ...]}

Respond in %s.`, languageName(language))

	user = fmt.Sprintf(`## Git Context
%s

## Recent Execution Log
%s

## Knowledge Index
%s

## Active Goal
%s

## To-Do List (pending items only)
%s

Based on the above, select the most relevant knowledge files.`,
		gitContent, output, index, goal, todoList)

	return system, user
}
