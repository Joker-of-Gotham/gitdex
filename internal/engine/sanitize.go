package engine

import (
	"strings"

	gitctx "github.com/Joker-of-Gotham/gitdex/internal/engine/context"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	gitplatform "github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func sanitizeSuggestedArgv(argv []string) []string {
	out := make([]string, len(argv))
	for i, token := range argv {
		out[i] = sanitizeSuggestedToken(token)
	}
	return out
}

func sanitizeSuggestedToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) < 2 {
		return token
	}
	if (strings.HasPrefix(token, `"`) && strings.HasSuffix(token, `"`)) ||
		(strings.HasPrefix(token, `'`) && strings.HasSuffix(token, `'`)) {
		token = token[1 : len(token)-1]
	}
	token = strings.ReplaceAll(token, `\"`, `"`)
	token = strings.ReplaceAll(token, `\'`, `'`)
	return strings.TrimSpace(token)
}

func detectPlaceholdersInArgv(argv []string) []llmInputJSON {
	var found []llmInputJSON
	for i := 2; i < len(argv); i++ {
		token := argv[i]
		if strings.HasPrefix(token, "-") {
			continue
		}
		if looksLikePlaceholder(token) {
			label := guessLabelForPlaceholder(token, argv, i)
			found = append(found, llmInputJSON{
				Key:      token,
				Label:    label,
				ArgIndex: i,
			})
		}
	}
	return found
}

func normalizeGitSuggestion(argv []string, inputs []llmInputJSON, interaction git.InteractionMode) ([]string, []llmInputJSON, git.InteractionMode) {
	if len(argv) < 2 || !strings.EqualFold(argv[0], "git") {
		return argv, inputs, interaction
	}
	sub := strings.ToLower(strings.TrimSpace(argv[1]))
	info, hasInfo := gitctx.Get().Subcommands[sub]
	if !hasInfo || !info.RequiresMessage {
		return argv, inputs, interaction
	}

	msgIdx, hasMsg, msgLooksPlaceholder := findCommitMessageArg(argv)
	if !hasMsg {
		argv = append(argv, "-m", "<commit-message>")
		msgIdx = len(argv) - 1
		msgLooksPlaceholder = true
	}

	needInput := interaction == git.NeedsInput || msgLooksPlaceholder
	if !needInput {
		return argv, inputs, interaction
	}
	interaction = git.NeedsInput

	commitInputIdx := -1
	for i := range inputs {
		if inputLooksLikeCommitMessage(inputs[i]) {
			commitInputIdx = i
			break
		}
	}
	if commitInputIdx == -1 {
		for i := range inputs {
			if inputs[i].ArgIndex < 2 {
				commitInputIdx = i
				break
			}
		}
	}
	if commitInputIdx == -1 {
		inputs = append(inputs, llmInputJSON{})
		commitInputIdx = len(inputs) - 1
	}
	inputs[commitInputIdx].Key = "commit_message"
	inputs[commitInputIdx].Label = info.DefaultInputLabel
	if strings.TrimSpace(inputs[commitInputIdx].Placeholder) == "" {
		inputs[commitInputIdx].Placeholder = info.DefaultInputPlaceholder
	}
	inputs[commitInputIdx].ArgIndex = msgIdx
	return argv, inputs, interaction
}

func findCommitMessageArg(argv []string) (idx int, hasMessage bool, placeholder bool) {
	ctx := gitctx.Get()
	for i := 2; i < len(argv); i++ {
		a := argv[i]
		if ctx.IsMessageFlag(a) {
			if i+1 >= len(argv) {
				return -1, false, false
			}
			val := strings.TrimSpace(argv[i+1])
			if val == "" {
				return i + 1, true, true
			}
			return i + 1, true, looksLikePlaceholder(val)
		}
		if strings.HasPrefix(a, "--message=") {
			val := strings.TrimSpace(strings.TrimPrefix(a, "--message="))
			if val == "" {
				return i, true, true
			}
			return i, true, looksLikePlaceholder(val)
		}
		if strings.HasPrefix(a, "-m") && len(a) > 2 {
			val := strings.TrimSpace(a[2:])
			if val == "" {
				return i, true, true
			}
			return i, true, looksLikePlaceholder(val)
		}
	}
	return -1, false, false
}

func inputLooksLikeCommitMessage(in llmInputJSON) bool {
	combined := strings.ToLower(strings.TrimSpace(in.Key + " " + in.Label + " " + in.Placeholder))
	return strings.Contains(combined, "commit") || strings.Contains(combined, "message")
}

func inferInputArgIndex(argv []string, in llmInputJSON) (int, bool) {
	if len(argv) < 3 {
		return -1, false
	}
	key := strings.TrimSpace(in.Key)
	if key != "" {
		for i := 2; i < len(argv); i++ {
			if argv[i] == key {
				return i, true
			}
		}
	}
	for i := 2; i < len(argv); i++ {
		if looksLikePlaceholder(argv[i]) {
			return i, true
		}
	}
	sub := strings.ToLower(strings.TrimSpace(argv[1]))
	combined := strings.ToLower(strings.TrimSpace(in.Key + " " + in.Label + " " + in.Placeholder))
	if sub == "commit" && (strings.Contains(combined, "message") || strings.Contains(combined, "commit")) {
		if idx, has, _ := findCommitMessageArg(argv); has {
			return idx, true
		}
	}
	if (sub == "remote" || sub == "push" || sub == "pull" || sub == "fetch") &&
		(strings.Contains(combined, "url") || strings.Contains(combined, "remote")) {
		return len(argv) - 1, true
	}
	return -1, false
}

func looksLikePlaceholder(token string) bool {
	return gitctx.Get().IsPlaceholder(token)
}

func guessLabelForPlaceholder(token string, argv []string, idx int) string {
	sub := ""
	if idx >= 2 && len(argv) > 1 {
		sub = strings.ToLower(argv[1])
	}
	return gitctx.Get().GuessLabel(token, sub)
}

func defaultInputPlaceholder(key, label, existing string) string {
	if existing != "" {
		return existing
	}
	return gitctx.Get().DefaultPlaceholder(key, label, gitplatform.HasSSHKeys())
}
