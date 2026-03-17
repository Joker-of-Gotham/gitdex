package promptv2

import "fmt"

// BuildPromptA constructs the Helper LLM prompt for maintenance knowledge selection.
// Inputs: git-content.txt content, output.txt recent rounds, index.yaml full text.
// Output: the LLM should return JSON {"selected_knowledge": ["path1", "path2", ...]}.
func BuildPromptA(gitContent, output, index, language string) (system, user string) {
	system = fmt.Sprintf(`You are a knowledge-index assistant for a Git repository maintenance system.
Your job: given the repository's current state, recent execution logs, and a knowledge index,
select the knowledge files that are most relevant to the current situation.

RULES:
- Select 1-5 files that are directly relevant to what the repository needs right now.
- Prefer files that address active problems (dirty working tree, sync issues, conflicts).
- Only select files that match current situations.

OUTPUT FORMAT (strict JSON, no markdown fences, no prose):
{"selected_knowledge": ["<full path 1>", "<full path 2>", ...]}

Respond in %s.`, languageName(language))

	user = fmt.Sprintf(`## Git Context
%s

## Recent Execution Log
%s

## Knowledge Index
%s

Based on the above, select the most relevant knowledge files.`, gitContent, output, index)

	return system, user
}
