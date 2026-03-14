package context

import (
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

const maxRetrievedFragments = 3

// Retriever matches current GitState against knowledge scenarios.
type Retriever struct {
	kb *KnowledgeBase
}

// NewRetriever creates a scenario retriever with an embedded knowledge base.
func NewRetriever() *Retriever {
	return &Retriever{kb: LoadKnowledgeBase()}
}

// Retrieve returns the most relevant knowledge fragments for the given state.
func (r *Retriever) Retrieve(state *status.GitState) []prompt.KnowledgeFragment {
	return r.RetrieveWithGoal(state, "")
}

// RetrieveWithGoal returns the most relevant knowledge fragments for the given
// state and active goal.
func (r *Retriever) RetrieveWithGoal(state *status.GitState, goal string) []prompt.KnowledgeFragment {
	if r.kb == nil || state == nil {
		return nil
	}

	type scored struct {
		scenario Scenario
		score    int
	}
	var matches []scored

	for _, s := range r.kb.Scenarios {
		score := matchScore(s.Triggers, state, goal)
		if score > 0 {
			matches = append(matches, scored{s, score})
		}
	}

	// Sort by score descending (insertion sort)
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0 && matches[j].score > matches[j-1].score; j-- {
			matches[j], matches[j-1] = matches[j-1], matches[j]
		}
	}

	var results []prompt.KnowledgeFragment
	for i, m := range matches {
		if i >= maxRetrievedFragments {
			break
		}
		results = append(results, prompt.KnowledgeFragment{
			ScenarioID: m.scenario.Source + "#" + m.scenario.ID,
			Content:    FormatScenario(m.scenario),
		})
	}
	return results
}

// FetchByIDs returns knowledge fragments matching the given scenario IDs.
// Accepts both full IDs (e.g. "branch#conflict_resolution") and short IDs (e.g. "conflict_resolution").
func (r *Retriever) FetchByIDs(ids []string) []prompt.KnowledgeFragment {
	if r.kb == nil || len(ids) == 0 {
		return nil
	}
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[strings.TrimSpace(id)] = true
	}
	var results []prompt.KnowledgeFragment
	for _, s := range r.kb.Scenarios {
		fullID := s.Source + "#" + s.ID
		if idSet[fullID] || idSet[s.ID] {
			results = append(results, prompt.KnowledgeFragment{
				ScenarioID: fullID,
				Content:    FormatScenario(s),
			})
		}
	}
	return results
}

// Catalog returns the knowledge base catalog for prompt inclusion.
func (r *Retriever) Catalog() []KnowledgeCatalogEntry {
	if r.kb == nil {
		return nil
	}
	return r.kb.Catalog()
}

func matchScore(triggers map[string]any, state *status.GitState, goal string) int {
	if len(triggers) == 0 {
		return 0
	}

	score := 0
	for key, val := range triggers {
		matched := matchTrigger(key, val, state, goal)
		if matched > 0 {
			score += matched
		} else if matched < 0 {
			return 0 // required trigger failed
		}
	}
	return score
}

func matchTrigger(key string, val any, state *status.GitState, goal string) int {
	boolVal, isBool := val.(bool)
	goal = normalizeText(goal)

	switch key {
	case "always":
		if isBool && boolVal {
			return 1
		}
		return -1

	case "is_initial":
		if isBool && boolVal == state.IsInitial {
			return 3
		}
		return -1

	case "has_gitignore":
		if isBool && boolVal == state.HasGitIgnore {
			return 2
		}
		return -1

	case "identity_configured":
		configured := state.ConfigInfo != nil && state.ConfigInfo.IdentityConfigured
		if isBool && boolVal == configured {
			return 3
		}
		return -1

	case "ssh_keys_empty":
		empty := state.ConfigInfo == nil || len(state.ConfigInfo.SSHKeyFiles) == 0
		if isBool && boolVal == empty {
			return 2
		}
		return -1

	case "has_remote":
		if isBool && boolVal == (len(state.Remotes) > 0) {
			return 1
		}
		return -1

	case "has_upstream":
		if isBool && boolVal == (state.UpstreamState != nil) {
			return 1
		}
		return -1

	case "detached_head":
		if isBool && boolVal == state.LocalBranch.IsDetached {
			return 3
		}
		return -1

	case "merge_in_progress":
		if isBool && boolVal == state.MergeInProgress {
			return 5
		}
		return -1

	case "rebase_in_progress":
		if isBool && boolVal == state.RebaseInProgress {
			return 5
		}
		return -1

	case "cherry_in_progress":
		if isBool && boolVal == state.CherryInProgress {
			return 5
		}
		return -1

	case "commit_count":
		if n, ok := toInt(val); ok && state.CommitCount == n {
			return 2
		}
		return -1

	case "working_count_gt":
		if n, ok := toInt(val); ok && len(state.WorkingTree) > n {
			return 2
		}
		return 0

	case "staged_count_gt":
		if n, ok := toInt(val); ok && len(state.StagingArea) > n {
			return 2
		}
		return 0

	case "ahead_gt":
		if n, ok := toInt(val); ok && len(state.AheadCommits) > n {
			return 2
		}
		return 0

	case "behind_gt":
		if n, ok := toInt(val); ok && len(state.BehindCommits) > n {
			return 2
		}
		return 0

	case "ahead":
		if n, ok := toInt(val); ok && len(state.AheadCommits) == n {
			return 1
		}
		return 0

	case "commit_count_gt":
		if n, ok := toInt(val); ok && state.CommitCount > n {
			return 1
		}
		return 0

	case "branch_count_gt":
		if n, ok := toInt(val); ok && len(state.LocalBranches) > n {
			return 1
		}
		return 0

	case "platform":
		want := toStringSlice(val)
		if len(want) == 0 {
			return -1
		}
		current := platformName(state)
		for _, candidate := range want {
			if normalizeText(candidate) == current {
				return 3
			}
		}
		return -1

	case "goal_contains":
		tokens := toStringSlice(val)
		if len(tokens) == 0 {
			return -1
		}
		if goal == "" {
			return -1
		}
		for _, token := range tokens {
			token = normalizeText(token)
			if token != "" && strings.Contains(goal, token) {
				return 4
			}
		}
		return -1
	}

	return 0
}

func platformName(state *status.GitState) string {
	if state == nil {
		return ""
	}
	for _, info := range state.RemoteInfos {
		if info.Name == "origin" {
			if info.PushURL != "" {
				return platformToName(platform.DetectPlatform(info.PushURL))
			}
			if info.FetchURL != "" {
				return platformToName(platform.DetectPlatform(info.FetchURL))
			}
		}
	}
	for _, info := range state.RemoteInfos {
		if info.PushURL != "" {
			return platformToName(platform.DetectPlatform(info.PushURL))
		}
		if info.FetchURL != "" {
			return platformToName(platform.DetectPlatform(info.FetchURL))
		}
	}
	return ""
}

func platformToName(p platform.Platform) string {
	switch p {
	case platform.PlatformGitHub:
		return "github"
	case platform.PlatformGitLab:
		return "gitlab"
	case platform.PlatformBitbucket:
		return "bitbucket"
	default:
		return ""
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	}
	return 0, false
}

func toStringSlice(v any) []string {
	switch value := v.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return nil
		}
		return []string{value}
	case []string:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if strings.TrimSpace(item) != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func normalizeText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
