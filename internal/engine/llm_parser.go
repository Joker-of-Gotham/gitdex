package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

type llmResponseJSON struct {
	Analysis     string              `json:"analysis"`
	PlanOverview string              `json:"plan_overview,omitempty"`
	GoalStatus   string              `json:"goal_status,omitempty"`
	Suggestions  []llmSuggestionJSON `json:"suggestions"`
}

type llmInputJSON struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Placeholder  string `json:"placeholder"`
	DefaultValue string `json:"default_value"`
	ArgIndex     int    `json:"arg_index"`
}

type llmSuggestionJSON struct {
	Action        string         `json:"action"`
	Argv          []string       `json:"argv"`
	Command       string         `json:"command"`
	Reason        string         `json:"reason"`
	Risk          string         `json:"risk"`
	Interaction   string         `json:"interaction"`
	Inputs        []llmInputJSON `json:"inputs"`
	FilePath      string         `json:"file_path,omitempty"`
	FileContent   string         `json:"file_content,omitempty"`
	FileOperation string         `json:"file_operation,omitempty"` // "create", "update", "delete", "append"
}

type parsedLLMResult struct {
	analysis     string
	planOverview string
	goalStatus   string
	suggestions  []git.Suggestion
	rejected     []string
}

func parseLLMResponse(state *status.GitState, text string) (parsedLLMResult, error) {
	text = normalizeStructuredResponseText(text)
	if text == "" {
		return parsedLLMResult{}, fmt.Errorf("empty AI response")
	}

	// Try parsing as-is first
	result, err := tryParseJSON(state, text)
	if err == nil {
		return result, nil
	}

	// If that fails, try repairing JSON
	repaired := repairJSON(text)
	result, err = tryParseJSON(state, repaired)
	if err == nil {
		return result, nil
	}

	// Last resort: try to extract just the suggestions array
	return tryParseSuggestionsArray(state, text)
}

func tryParseJSON(state *status.GitState, text string) (parsedLLMResult, error) {
	firstBrace := strings.Index(text, "{")
	firstBracket := strings.Index(text, "[")

	if firstBrace >= 0 && (firstBracket < 0 || firstBrace < firstBracket) {
		candidate := text[firstBrace:]
		if end := findMatchingBrace(candidate); end > 0 {
			candidate = candidate[:end+1]
			var resp llmResponseJSON
			if err := json.Unmarshal([]byte(candidate), &resp); err == nil {
				if len(resp.Suggestions) == 0 &&
					strings.TrimSpace(resp.Analysis) == "" &&
					strings.TrimSpace(resp.PlanOverview) == "" &&
					strings.TrimSpace(resp.GoalStatus) == "" {
					return parsedLLMResult{}, fmt.Errorf("json object does not match response schema")
				}
				suggestions, rejected, convErr := convertSuggestions(state, resp.Suggestions)
				if convErr != nil {
					return parsedLLMResult{}, convErr
				}
				analysis := strings.TrimSpace(resp.Analysis)
				if analysis == "" {
					analysis = "AI analysis completed."
				}
				return parsedLLMResult{
					analysis:     analysis,
					planOverview: strings.TrimSpace(resp.PlanOverview),
					goalStatus:   strings.TrimSpace(resp.GoalStatus),
					suggestions:  suggestions,
					rejected:     rejected,
				}, nil
			}
		}
	}
	return parsedLLMResult{}, fmt.Errorf("failed to parse JSON")
}

func tryParseSuggestionsArray(state *status.GitState, text string) (parsedLLMResult, error) {
	firstBracket := strings.Index(text, "[")
	if firstBracket < 0 {
		return parsedLLMResult{}, fmt.Errorf("no JSON array found")
	}
	arrText := text[firstBracket:]
	if end := strings.LastIndex(arrText, "]"); end >= 0 {
		arrText = arrText[:end+1]
		var items []llmSuggestionJSON
		if err := json.Unmarshal([]byte(arrText), &items); err == nil {
			suggestions, rejected, convErr := convertSuggestions(state, items)
			if convErr != nil {
				return parsedLLMResult{}, convErr
			}
			return parsedLLMResult{
				analysis:    "AI returned suggestions.",
				suggestions: suggestions,
				rejected:    rejected,
			}, nil
		}
	}
	return parsedLLMResult{}, fmt.Errorf("response is not valid JSON")
}

func findMatchingBrace(s string) int {
	depth := 0
	inStr := false
	escape := false
	for i, c := range s {
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inStr {
			escape = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func normalizeStructuredResponseText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = stripMarkdownCodeFences(text)
	text = strings.TrimSpace(text)
	if text == "```" {
		return ""
	}
	return text
}

func stripMarkdownCodeFences(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizedRawResponseForDisplay(raw string, cleaned string) string {
	raw = strings.TrimSpace(raw)
	cleaned = normalizeStructuredResponseText(cleaned)
	if cleaned != "" {
		return cleaned
	}
	raw = stripMarkdownCodeFences(raw)
	raw = strings.TrimSpace(raw)
	if raw != "" {
		return raw
	}
	return "(empty response after stripping thinking blocks and markdown fences)"
}

func truncateForDisplay(text string, max int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "(empty response)"
	}
	if len(text) <= max {
		return text
	}
	return text[:max] + "\n... (truncated)"
}

func shellSplit(s string) []string {
	var args []string
	var current strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == ' ' && !inSingle && !inDouble:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
