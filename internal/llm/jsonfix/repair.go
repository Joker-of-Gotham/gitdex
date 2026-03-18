package jsonfix

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kaptinlin/jsonrepair"
)

var (
	fencedJSONPattern = regexp.MustCompile("(?s)```(?:json)?\\s*\n?(.*?)\\s*```")
	pythonTrue        = regexp.MustCompile(`\bTrue\b`)
	pythonFalse       = regexp.MustCompile(`\bFalse\b`)
	pythonNone        = regexp.MustCompile(`\bNone\b`)
)

// Repair attempts to fix malformed JSON from LLM output.
// Pipeline: extract fenced -> strip JSONP -> fix Python constants ->
// jsonrepair library -> validate.
func Repair(raw string) (string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", fmt.Errorf("empty input")
	}

	text = extractJSON(text)
	text = fixPythonConstants(text)

	repaired, err := jsonrepair.JSONRepair(text)
	if err != nil {
		if json.Valid([]byte(text)) {
			return text, nil
		}
		return "", fmt.Errorf("jsonrepair failed: %w", err)
	}

	if !json.Valid([]byte(repaired)) {
		return "", fmt.Errorf("repaired JSON is still invalid")
	}

	return repaired, nil
}

// RepairAndUnmarshal repairs JSON then unmarshals into target.
func RepairAndUnmarshal(raw string, target any) error {
	repaired, err := Repair(raw)
	if err != nil {
		return fmt.Errorf("json repair: %w", err)
	}
	if err := json.Unmarshal([]byte(repaired), target); err != nil {
		return fmt.Errorf("json unmarshal after repair: %w", err)
	}
	return nil
}

func extractJSON(text string) string {
	if m := fencedJSONPattern.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	if idx := strings.Index(text, "{"); idx >= 0 {
		lastBrace := strings.LastIndex(text, "}")
		if lastBrace > idx {
			return text[idx : lastBrace+1]
		}
	}

	if idx := strings.Index(text, "["); idx >= 0 {
		lastBracket := strings.LastIndex(text, "]")
		if lastBracket > idx {
			return text[idx : lastBracket+1]
		}
	}

	return text
}

func fixPythonConstants(text string) string {
	text = pythonTrue.ReplaceAllString(text, "true")
	text = pythonFalse.ReplaceAllString(text, "false")
	text = pythonNone.ReplaceAllString(text, "null")
	return text
}
