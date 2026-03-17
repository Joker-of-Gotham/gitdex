package budget

import (
	"strings"
	"unicode/utf8"
)

// EstimateTokens provides a rough token count estimation.
// Uses the heuristic: ~4 characters per token for English, ~2 for CJK.
// This avoids requiring a full tokenizer dependency.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	asciiChars := 0
	cjkChars := 0
	for _, r := range text {
		if r <= 0x7F {
			asciiChars++
		} else {
			cjkChars++
		}
	}
	return (asciiChars+3)/4 + (cjkChars+1)/2
}

// ContextBudget tracks token usage across prompt sections.
type ContextBudget struct {
	MaxTokens int
	Reserved  int // tokens reserved for LLM output
	sections  []Section
}

// Section represents one named section of the prompt.
type Section struct {
	Name   string
	Tokens int
	Text   string
}

// NewBudget creates a context budget with the given model context limit.
// reserveForOutput is the number of tokens to reserve for model output.
func NewBudget(maxTokens, reserveForOutput int) *ContextBudget {
	if reserveForOutput <= 0 {
		reserveForOutput = maxTokens / 4
	}
	return &ContextBudget{
		MaxTokens: maxTokens,
		Reserved:  reserveForOutput,
	}
}

// Available returns how many tokens are still available for content.
func (b *ContextBudget) Available() int {
	used := b.Used()
	avail := b.MaxTokens - b.Reserved - used
	if avail < 0 {
		return 0
	}
	return avail
}

// Used returns the total tokens used across all sections.
func (b *ContextBudget) Used() int {
	total := 0
	for _, s := range b.sections {
		total += s.Tokens
	}
	return total
}

// Add registers a section and returns the (possibly truncated) text.
// If the text would exceed the budget, it's truncated.
func (b *ContextBudget) Add(name, text string) string {
	tokens := EstimateTokens(text)
	avail := b.Available()

	if tokens <= avail {
		b.sections = append(b.sections, Section{Name: name, Tokens: tokens, Text: text})
		return text
	}

	truncated := TruncateToTokens(text, avail)
	actualTokens := EstimateTokens(truncated)
	b.sections = append(b.sections, Section{Name: name, Tokens: actualTokens, Text: truncated})
	return truncated
}

// Snapshot returns a copy of all sections for display.
func (b *ContextBudget) Snapshot() []Section {
	out := make([]Section, len(b.sections))
	copy(out, b.sections)
	return out
}

// Summary returns a human-readable summary: "used/max tokens (N sections)".
func (b *ContextBudget) Summary() string {
	used := b.Used()
	avail := b.MaxTokens - b.Reserved
	return itoa(used) + "/" + itoa(avail) + " tokens (" + itoa(len(b.sections)) + " sections)"
}

// FormatUsage returns "[used/max]" string for TUI display.
func (b *ContextBudget) FormatUsage() string {
	used := b.Used()
	return "[" + itoa(used) + "/" + itoa(b.MaxTokens-b.Reserved) + "]"
}

func itoa(n int) string {
	if n < 0 {
		n = 0
	}
	if n >= 1000 {
		k := n / 1000
		rem := (n % 1000) / 100
		return strings.Join([]string{itoaSimple(k), ".", itoaSimple(rem), "k"}, "")
	}
	return itoaSimple(n)
}

func itoaSimple(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// TruncateToTokens truncates text to approximately maxTokens.
func TruncateToTokens(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	estimated := EstimateTokens(text)
	if estimated <= maxTokens {
		return text
	}

	ratio := float64(maxTokens) / float64(estimated)
	targetChars := int(float64(utf8.RuneCountInString(text)) * ratio * 0.95)
	if targetChars <= 0 {
		return ""
	}

	runes := []rune(text)
	if targetChars >= len(runes) {
		return text
	}

	truncated := string(runes[:targetChars])
	if idx := strings.LastIndex(truncated, "\n"); idx > len(truncated)/2 {
		truncated = truncated[:idx]
	}

	return strings.TrimRight(truncated, "\n") + "\n\n[context truncated: budget exceeded]"
}

// lowPrioritySections lists git context sections that can be removed first
// without losing critical state information.
var lowPrioritySections = []string{
	"## Recent Reflog",
	"## Worktrees",
	"## Submodules",
	"## Stash",
	"## Config",
	"## Commit Summary",
	"## File Inspection",
	"## Tags",
	"## Merged Branches",
	"## Ahead Commits",
	"## Behind Commits",
}

// CompressGitContent reduces git context to essentials: branch, remotes,
// working tree changes, staging area, and upstream status.
func CompressGitContent(content string, maxTokens int) string {
	tokens := EstimateTokens(content)
	if tokens <= maxTokens {
		return content
	}

	lines := strings.Split(content, "\n")

	for _, dropSection := range lowPrioritySections {
		var kept []string
		skip := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == dropSection || strings.HasPrefix(trimmed, dropSection) {
				skip = true
				continue
			}
			if skip && strings.HasPrefix(trimmed, "## ") {
				skip = false
			}
			if skip {
				continue
			}
			kept = append(kept, line)
		}
		lines = kept
		if EstimateTokens(strings.Join(lines, "\n")) <= maxTokens {
			break
		}
	}

	result := strings.Join(lines, "\n")
	if EstimateTokens(result) > maxTokens {
		return TruncateToTokens(result, maxTokens)
	}
	return result
}

const failedCommandsBudget = 200

// CompressOutputLog reduces output log size by keeping only the most recent
// round and summarizing older ones. FAILED COMMANDS block is preserved at top
// with an independent token budget.
func CompressOutputLog(output string, maxTokens int) string {
	tokens := EstimateTokens(output)
	if tokens <= maxTokens {
		return output
	}

	failedBlock, body := splitFailedBlock(output)
	failedTokens := EstimateTokens(failedBlock)
	if failedTokens > failedCommandsBudget {
		failedBlock = TruncateToTokens(failedBlock, failedCommandsBudget)
		failedTokens = EstimateTokens(failedBlock)
	}
	bodyBudget := maxTokens - failedTokens

	sections := strings.Split(body, "--- Round ")
	if len(sections) <= 1 {
		compressed := TruncateToTokens(body, bodyBudget)
		if failedBlock != "" {
			return failedBlock + "\n" + compressed
		}
		return compressed
	}

	var result strings.Builder
	for i, sec := range sections {
		if i == 0 && strings.TrimSpace(sec) == "" {
			continue
		}
		if i == len(sections)-1 {
			result.WriteString("--- Round " + sec)
		} else {
			lines := strings.SplitN(sec, "\n", 5)
			for j, l := range lines {
				if j < 3 && (i > 0 || j == 0) {
					if j == 0 && i > 0 {
						result.WriteString("--- Round ")
					}
					result.WriteString(l + "\n")
				}
			}
			result.WriteString("  [earlier round compressed]\n\n")
		}
	}

	text := result.String()
	if EstimateTokens(text) > bodyBudget {
		text = TruncateToTokens(text, bodyBudget)
	}

	if failedBlock != "" {
		return failedBlock + "\n" + text
	}
	return text
}

// splitFailedBlock extracts the FAILED COMMANDS block from output.
// Returns (failedBlock, remainingBody).
func splitFailedBlock(output string) (string, string) {
	startMarker := "=== FAILED COMMANDS"
	endMarker := "=== END FAILED COMMANDS ==="

	startIdx := strings.Index(output, startMarker)
	if startIdx < 0 {
		return "", output
	}

	endIdx := strings.Index(output[startIdx:], endMarker)
	if endIdx < 0 {
		block := output[startIdx:]
		body := strings.TrimSpace(output[:startIdx])
		return block, body
	}

	blockEnd := startIdx + endIdx + len(endMarker)
	block := output[startIdx:blockEnd]
	body := strings.TrimSpace(output[:startIdx] + output[blockEnd:])
	return block, body
}
