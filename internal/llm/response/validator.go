package response

import (
	"regexp"
	"strings"
)

// thinkRegex matches <think>...</think> blocks from thinking models (qwen3, deepseek-r1, etc.)
var thinkRegex = regexp.MustCompile(`(?s)<think>.*?</think>`)

// ExtractThinking separates the <think>...</think> reasoning from the output.
// Returns (thinking content, clean output). For non-thinking models, thinking is empty.
func ExtractThinking(text string) (thinking string, output string) {
	matches := thinkRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return "", strings.TrimSpace(text)
	}
	var thinkParts []string
	for _, m := range matches {
		inner := m
		inner = strings.TrimPrefix(inner, "<think>")
		inner = strings.TrimSuffix(inner, "</think>")
		inner = strings.TrimSpace(inner)
		if inner != "" {
			thinkParts = append(thinkParts, inner)
		}
	}
	thinking = strings.Join(thinkParts, "\n")
	output = thinkRegex.ReplaceAllString(text, "")
	output = strings.TrimSpace(output)
	return thinking, output
}

// StripThinking removes <think>...</think> blocks, returning only the output.
func StripThinking(text string) string {
	_, output := ExtractThinking(text)
	return output
}
