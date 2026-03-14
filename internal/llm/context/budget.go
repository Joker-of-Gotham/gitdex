package context

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Priority levels for context partitions, lower = higher priority.
type Priority int

const (
	PrioSystemPrompt   Priority = 0
	PrioCriticalState  Priority = 1
	PrioUserGoal       Priority = 2
	PrioKnowledge        Priority = 3
	PrioKnowledgeCatalog Priority = 4
	PrioRecentOps        Priority = 5
	PrioFileInspect      Priority = 6
	PrioCommitSummary    Priority = 7
	PrioConfigState      Priority = 8
	PrioExtendedState    Priority = 9
	PrioSessionHistory   Priority = 10
	PrioPlatformState    Priority = 11
	PrioLongTermMemory   Priority = 12
)

// Partition represents a named chunk of context content with priority.
type Partition struct {
	Name     string
	Priority Priority
	Content  string
	Required bool // if true, never truncate
}

// PartitionUsage describes how a partition was handled by the budget manager.
type PartitionUsage struct {
	Name      string
	Priority  Priority
	Tokens    int
	Required  bool
	Included  bool
	Truncated bool
}

// BudgetManager allocates context partitions within a token budget.
type BudgetManager struct {
	TotalBudget int // total tokens available
	Reserved    int // tokens reserved for output
}

// NewBudgetManager creates a budget manager.
// totalTokens is the model's context length; reserveForOutput is tokens
// reserved for the model's response (typically 2048-4096).
func NewBudgetManager(totalTokens, reserveForOutput int) *BudgetManager {
	if totalTokens <= 0 {
		totalTokens = 32768
	}
	if reserveForOutput <= 0 {
		reserveForOutput = 2048
	}
	return &BudgetManager{
		TotalBudget: totalTokens,
		Reserved:    reserveForOutput,
	}
}

// AvailableTokens returns how many tokens are available for prompts.
func (b *BudgetManager) AvailableTokens() int {
	avail := b.TotalBudget - b.Reserved
	if avail < 1024 {
		avail = 1024
	}
	return avail
}

// Assemble takes partitions sorted by priority and fits them within budget.
// Returns the final system prompt and user prompt.
func (b *BudgetManager) Assemble(systemPrompt string, partitions []Partition) (system, user string) {
	system, user, _ = b.AssembleDetailed(systemPrompt, partitions)
	return system, user
}

// AssembleDetailed returns the final prompts plus per-partition usage details.
func (b *BudgetManager) AssembleDetailed(systemPrompt string, partitions []Partition) (system, user string, usage []PartitionUsage) {
	budget := b.AvailableTokens()
	systemTokens := EstimateTokens(systemPrompt)
	remaining := budget - systemTokens

	// Sort partitions by priority (already expected sorted, but stable-sort)
	sorted := make([]Partition, len(partitions))
	copy(sorted, partitions)
	sortPartitions(sorted)

	var included []Partition
	for _, p := range sorted {
		tokens := EstimateTokens(p.Content)
		entry := PartitionUsage{
			Name:     p.Name,
			Priority: p.Priority,
			Tokens:   tokens,
			Required: p.Required,
		}
		if tokens == 0 {
			usage = append(usage, entry)
			continue
		}
		if p.Required || remaining >= tokens {
			included = append(included, p)
			remaining -= tokens
			entry.Included = true
		} else if remaining > 200 {
			// Try to fit a truncated version
			truncated := TruncateToTokens(p.Content, remaining-50)
			if truncated != "" {
				p.Content = truncated
				included = append(included, p)
				entry.Included = true
				entry.Truncated = true
				entry.Tokens = EstimateTokens(truncated)
				remaining -= entry.Tokens
			}
		}
		usage = append(usage, entry)
	}

	var parts []string
	for _, p := range included {
		if strings.TrimSpace(p.Content) != "" {
			parts = append(parts, p.Content)
		}
	}

	return systemPrompt, strings.Join(parts, "\n\n"), usage
}

// EstimateTokens provides a rough token count estimate.
// Uses ~3.5 characters per token for mixed CJK/Latin text.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	chars := len([]rune(text))
	return (chars*10 + 34) / 35 // ~chars/3.5 with integer math
}

// TruncateToTokens truncates text to fit within approximately maxTokens.
func TruncateToTokens(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	runes := []rune(text)
	maxChars := maxTokens * 35 / 10 // inverse of token estimation
	if len(runes) <= maxChars {
		return text
	}
	if maxChars < 20 {
		return ""
	}
	return string(runes[:maxChars-15]) + "\n... (truncated)"
}

// CompressJSON takes a JSON-serializable value and marshals it compactly.
func CompressJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

func sortPartitions(parts []Partition) {
	for i := 1; i < len(parts); i++ {
		for j := i; j > 0 && parts[j].Priority < parts[j-1].Priority; j-- {
			parts[j], parts[j-1] = parts[j-1], parts[j]
		}
	}
}
