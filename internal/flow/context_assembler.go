package flow

import (
	"github.com/Joker-of-Gotham/gitdex/internal/llm/budget"
)

type contextProfile struct {
	outputRounds int
	gitMax       int
	outputMax    int
	indexMax     int
	knowledgeMax int
	goalMax      int
	todoMax      int
	githubMax    int
}

// ContextAssembler centralizes V3 context budgeting and section allocation.
type ContextAssembler struct {
	limit         int
	outputReserve int
	contentBudget int
	profile       contextProfile
	budget        *budget.ContextBudget
}

func NewContextAssembler(flow string, contextLimit int) *ContextAssembler {
	limit := contextLimit
	if limit <= 0 {
		limit = 32768
	}
	outputReserve := limit * 2 / 5
	contentBudget := limit - outputReserve
	profile := profileForFlow(flow)
	return &ContextAssembler{
		limit:         limit,
		outputReserve: outputReserve,
		contentBudget: contentBudget,
		profile:       profile,
		budget:        budget.NewBudget(limit, outputReserve),
	}
}

func profileForFlow(flow string) contextProfile {
	switch flow {
	case "creative":
		return contextProfile{
			outputRounds: 1,
			gitMax:       1536,
			outputMax:    512,
			indexMax:     512,
			goalMax:      512,
			todoMax:      512,
			githubMax:    1536,
		}
	case "goal":
		return contextProfile{
			outputRounds: 2,
			gitMax:       2048,
			outputMax:    1536,
			indexMax:     512,
			knowledgeMax: 1024,
			goalMax:      768,
			todoMax:      1024,
		}
	default: // maintain
		return contextProfile{
			outputRounds: 2,
			gitMax:       2048,
			outputMax:    1536,
			indexMax:     512,
			knowledgeMax: 1024,
		}
	}
}

func (a *ContextAssembler) OutputRounds() int { return a.profile.outputRounds }

func (a *ContextAssembler) TokensUsed() int { return a.budget.Used() }

func (a *ContextAssembler) TokensBudget() int { return a.contentBudget }

func (a *ContextAssembler) SectionUsage() map[string]int {
	out := make(map[string]int)
	for _, s := range a.budget.Snapshot() {
		out[s.Name] = s.Tokens
	}
	return out
}

func (a *ContextAssembler) AddGit(text string) string {
	return a.budget.Add("git_context", budget.CompressGitContent(text, a.profile.gitMax))
}

func (a *ContextAssembler) AddOutput(text string) string {
	return a.budget.Add("output_log", budget.CompressOutputLog(text, a.profile.outputMax))
}

func (a *ContextAssembler) AddIndex(text string) string {
	return a.budget.Add("index", budget.TruncateToTokens(text, a.profile.indexMax))
}

func (a *ContextAssembler) AddKnowledge(text string) string {
	if a.profile.knowledgeMax <= 0 {
		return a.budget.Add("knowledge", text)
	}
	return a.budget.Add("knowledge", budget.TruncateToTokens(text, a.profile.knowledgeMax))
}

func (a *ContextAssembler) AddGoal(text string) string {
	if a.profile.goalMax <= 0 {
		return a.budget.Add("goal", text)
	}
	return a.budget.Add("goal", budget.TruncateToTokens(text, a.profile.goalMax))
}

func (a *ContextAssembler) AddTodo(text string) string {
	if a.profile.todoMax <= 0 {
		return a.budget.Add("todo_list", text)
	}
	return a.budget.Add("todo_list", budget.TruncateToTokens(text, a.profile.todoMax))
}

func (a *ContextAssembler) AddGitHub(text string) string {
	if a.profile.githubMax <= 0 {
		return a.budget.Add("github_context", text)
	}
	return a.budget.Add("github_context", budget.TruncateToTokens(text, a.profile.githubMax))
}
