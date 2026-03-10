package engine

import "github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"

// AnalysisTrace keeps the full observable state for one analysis pass.
type AnalysisTrace struct {
	Mode            string                     `json:"mode"`
	PrimaryModel    string                     `json:"primary_model,omitempty"`
	SecondaryModel  string                     `json:"secondary_model,omitempty"`
	Budget          int                        `json:"budget,omitempty"`
	Reserved        int                        `json:"reserved,omitempty"`
	Available       int                        `json:"available,omitempty"`
	SystemPrompt    string                     `json:"system_prompt,omitempty"`
	UserPrompt      string                     `json:"user_prompt,omitempty"`
	Partitions      []prompt.PartitionTrace    `json:"partitions,omitempty"`
	RecentOps       []prompt.OperationRecord   `json:"recent_ops,omitempty"`
	Knowledge       []prompt.KnowledgeFragment `json:"knowledge,omitempty"`
	Memory          *prompt.MemoryContext      `json:"memory,omitempty"`
	PlatformState   *prompt.PlatformState      `json:"platform_state,omitempty"`
	RawResponse     string                     `json:"raw_response,omitempty"`
	CleanedResponse string                     `json:"cleaned_response,omitempty"`
	Rejected        []string                   `json:"rejected,omitempty"`
}
