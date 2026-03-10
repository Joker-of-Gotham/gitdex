package tui

import (
	"strings"

	gitctx "github.com/Joker-of-Gotham/gitdex/internal/engine/context"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

type workflowDefinition struct {
	ID            string
	Label         string
	Goal          string
	Prerequisites []string
}

func loadWorkflowDefinitions() []workflowDefinition {
	defs := gitctx.Get().WorkflowList()
	out := make([]workflowDefinition, 0, len(defs))
	for _, d := range defs {
		out = append(out, workflowDefinition{
			ID:            d.ID,
			Label:         d.Label,
			Goal:          d.Goal,
			Prerequisites: append([]string(nil), d.Prerequisites...),
		})
	}
	return out
}

func checkWorkflowPrerequisites(state *status.GitState, wf workflowDefinition) (bool, string) {
	if state == nil || len(wf.Prerequisites) == 0 {
		return true, ""
	}
	for _, rule := range wf.Prerequisites {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		if strings.Contains(rule, "|") {
			parts := strings.Split(rule, "|")
			any := false
			for _, p := range parts {
				if workflowConditionMet(state, strings.TrimSpace(p)) {
					any = true
					break
				}
			}
			if !any {
				return false, rule
			}
			continue
		}
		if !workflowConditionMet(state, rule) {
			return false, rule
		}
	}
	return true, ""
}

func workflowConditionMet(state *status.GitState, cond string) bool {
	cond = strings.TrimSpace(cond)
	switch cond {
	case "has_remote":
		return len(state.RemoteInfos) > 0
	case "has_commits_ahead":
		if len(state.AheadCommits) > 0 {
			return true
		}
		ahead := state.LocalBranch.Ahead
		if state.UpstreamState != nil {
			ahead = state.UpstreamState.Ahead
		}
		return ahead > 0
	case "has_upstream_remote":
		for _, r := range state.RemoteInfos {
			if strings.EqualFold(r.Name, "upstream") {
				return true
			}
		}
		return false
	case "merge_in_progress":
		return state.MergeInProgress
	case "rebase_in_progress":
		return state.RebaseInProgress
	case "has_staged":
		return len(state.StagingArea) > 0
	case "has_working_changes":
		return len(state.WorkingTree) > 0
	case "has_commits":
		return state.CommitCount > 0
	case "has_stash":
		return len(state.StashStack) > 0
	default:
		return false
	}
}
