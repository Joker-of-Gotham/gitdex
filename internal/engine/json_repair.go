package engine

import "strings"

// repairJSON attempts to fix common JSON issues from LLM output.
func repairJSON(text string) string {
	text = strings.TrimSpace(text)

	for _, prefix := range []string{"```json", "```JSON", "```"} {
		if strings.HasPrefix(text, prefix) {
			text = strings.TrimPrefix(text, prefix)
			if idx := strings.LastIndex(text, "```"); idx >= 0 {
				text = text[:idx]
			}
			text = strings.TrimSpace(text)
			break
		}
	}

	firstBrace := strings.Index(text, "{")
	firstBracket := strings.Index(text, "[")
	switch {
	case firstBrace < 0 && firstBracket < 0:
		return text
	case firstBracket >= 0 && (firstBrace < 0 || firstBracket < firstBrace):
		text = text[firstBracket:]
	default:
		text = text[firstBrace:]
	}

	// LLMs frequently put literal newlines inside JSON string values.
	// This is invalid JSON and the #1 cause of parse failures.
	text = escapeNewlinesInStrings(text)

	text = fixTrailingCommas(text)

	if strings.HasPrefix(text, "[") {
		if end := findMatchingBracket(text); end > 0 {
			text = text[:end+1]
		} else {
			lastBracket := strings.LastIndex(text, "]")
			if lastBracket > 0 {
				text = text[:lastBracket+1]
			}
		}
	} else {
		end := findMatchingBrace(text)
		if end > 0 {
			text = text[:end+1]
		} else {
			lastBrace := strings.LastIndex(text, "}")
			if lastBrace > 0 {
				text = text[:lastBrace+1]
			}
		}
	}

	text = balanceBrackets(text)
	return text
}

// escapeNewlinesInStrings replaces literal \n \r inside JSON string values
// with their escaped forms. This is the most common LLM JSON error.
func escapeNewlinesInStrings(text string) string {
	var b strings.Builder
	b.Grow(len(text) + 64)
	inStr := false
	esc := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if esc {
			esc = false
			b.WriteByte(c)
			continue
		}
		if c == '\\' && inStr {
			esc = true
			b.WriteByte(c)
			continue
		}
		if c == '"' {
			inStr = !inStr
			b.WriteByte(c)
			continue
		}
		if inStr && c == '\n' {
			b.WriteString("\\n")
			continue
		}
		if inStr && c == '\r' {
			if i+1 < len(text) && text[i+1] == '\n' {
				i++
			}
			b.WriteString("\\n")
			continue
		}
		if inStr && c == '\t' {
			b.WriteString("\\t")
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func fixTrailingCommas(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	inStr := false
	esc := false
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if esc {
			esc = false
			b.WriteRune(c)
			continue
		}
		if c == '\\' && inStr {
			esc = true
			b.WriteRune(c)
			continue
		}
		if c == '"' {
			inStr = !inStr
			b.WriteRune(c)
			continue
		}
		if inStr {
			b.WriteRune(c)
			continue
		}
		if c == ',' {
			// Look ahead past whitespace for } or ]
			j := i + 1
			for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t' || runes[j] == '\n' || runes[j] == '\r') {
				j++
			}
			if j < len(runes) && (runes[j] == '}' || runes[j] == ']') {
				continue // skip trailing comma
			}
		}
		b.WriteRune(c)
	}
	return b.String()
}

func balanceBrackets(text string) string {
	inStr := false
	esc := false
	var stack []rune
	for _, c := range text {
		if esc {
			esc = false
			continue
		}
		if c == '\\' && inStr {
			esc = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		switch c {
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	// Close any unclosed brackets/braces in reverse order
	for i := len(stack) - 1; i >= 0; i-- {
		text += string(stack[i])
	}
	return text
}

func findMatchingBracket(s string) int {
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
		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
