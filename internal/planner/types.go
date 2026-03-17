package planner

import (
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/contract"
)

// SuggestionItem and ActionSpec are canonical cross-layer contracts.
type SuggestionItem = contract.SuggestionItem
type ActionSpec = contract.ActionSpec

type plannerResponse struct {
	Analysis    string           `json:"analysis"`
	Suggestions []SuggestionItem `json:"suggestions"`
}

func sanitizeSuggestions(items []SuggestionItem) {
	for i := range items {
		if strings.TrimSpace(items[i].Version) == "" {
			items[i].Version = contract.ProtocolVersion
		}
		items[i].Name = sanitizeField(items[i].Name)
		items[i].Reason = sanitizeField(items[i].Reason)
		if strings.TrimSpace(items[i].Action.Version) == "" {
			items[i].Action.Version = contract.ProtocolVersion
		}
	}
}

func sanitizeField(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.SplitN(s, "\n", 2)[0]
	s = strings.TrimSpace(s)
	return s
}

// CreativeOutput is the Planner's output for creative goal generation.
type CreativeOutput struct {
	Analysis      string   `json:"analysis"`
	GitdexGoals   []string `json:"gitdex_goals"`
	CreativeGoals []string `json:"creative_goals"`
}
