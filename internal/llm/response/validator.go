package response

import (
	"regexp"
	"strings"
	"unicode"
)

var thinkingBlockPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?is)<think>(.*?)</think>`),
	regexp.MustCompile(`(?is)<thinking>(.*?)</thinking>`),
	regexp.MustCompile(`(?is)<reasoning>(.*?)</reasoning>`),
	regexp.MustCompile("(?is)```(?:thinking|reasoning|thoughts?)\\s*(.*?)```"),
}

var leadingThinkingLabel = regexp.MustCompile(`(?is)^(?:thinking|reasoning|thoughts?|thought process)\s*:\s*(.+)$`)

// ExtractThinking separates the <think>...</think> reasoning from the output.
// Returns (thinking content, clean output). For non-thinking models, thinking is empty.
func ExtractThinking(text string) (thinking string, output string) {
	working := strings.TrimSpace(text)
	if working == "" {
		return "", ""
	}

	var parts []string
	for _, pattern := range thinkingBlockPatterns {
		for {
			match := pattern.FindStringSubmatchIndex(working)
			if match == nil {
				break
			}
			if len(match) >= 4 {
				if inner := strings.TrimSpace(working[match[2]:match[3]]); inner != "" {
					parts = append(parts, inner)
				}
			}
			replacement := ""
			if needsInlineSpace(working, match[0], match[1]) {
				replacement = " "
			}
			working = strings.TrimSpace(working[:match[0]] + replacement + working[match[1]:])
		}
	}

	if leading, remainder, ok := extractLeadingThinkingPrefix(working); ok {
		parts = append(parts, leading)
		working = remainder
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n")), strings.TrimSpace(working)
}

func needsInlineSpace(text string, start, end int) bool {
	if start <= 0 || end >= len(text) {
		return false
	}
	prev := []rune(text[:start])
	next := []rune(text[end:])
	if len(prev) == 0 || len(next) == 0 {
		return false
	}
	return !unicode.IsSpace(prev[len(prev)-1]) && !unicode.IsSpace(next[0])
}

// StripThinking removes <think>...</think> blocks, returning only the output.
func StripThinking(text string) string {
	_, output := ExtractThinking(text)
	return output
}

func extractLeadingThinkingPrefix(text string) (thinking, remainder string, ok bool) {
	jsonStart := firstJSONStart(text)
	if jsonStart <= 0 {
		return "", text, false
	}
	prefix := strings.TrimSpace(text[:jsonStart])
	if prefix == "" {
		return "", text, false
	}
	match := leadingThinkingLabel.FindStringSubmatch(prefix)
	if len(match) < 2 {
		return "", text, false
	}
	return strings.TrimSpace(match[1]), strings.TrimSpace(text[jsonStart:]), true
}

func firstJSONStart(text string) int {
	firstBrace := strings.Index(text, "{")
	firstBracket := strings.Index(text, "[")
	switch {
	case firstBrace < 0:
		return firstBracket
	case firstBracket < 0:
		return firstBrace
	case firstBrace < firstBracket:
		return firstBrace
	default:
		return firstBracket
	}
}
