package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	gitctx "github.com/Joker-of-Gotham/gitdex/internal/engine/context"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type workflowPrefillDefinition struct {
	CapabilityID string
	Flow         string
	Operation    string
	ResourceID   string
	Scope        map[string]string
	Query        map[string]string
	Payload      json.RawMessage
	Validate     json.RawMessage
	Rollback     json.RawMessage
}

type workflowDefinition struct {
	ID            string
	Label         string
	Goal          string
	Prerequisites []string
	Capabilities  []string
	Prefill       []workflowPrefillDefinition
}

func loadWorkflowDefinitions() []workflowDefinition {
	defs := gitctx.Get().WorkflowList()
	out := make([]workflowDefinition, 0, len(defs))
	for _, d := range defs {
		prefill := make([]workflowPrefillDefinition, 0, len(d.Prefill))
		for _, item := range d.Prefill {
			payload, _ := marshalWorkflowJSON(item.Payload)
			validate, _ := marshalWorkflowJSON(item.Validate)
			rollback, _ := marshalWorkflowJSON(item.Rollback)
			if string(payload) == "null" {
				payload = nil
			}
			if string(validate) == "null" {
				validate = nil
			}
			if string(rollback) == "null" {
				rollback = nil
			}
			prefill = append(prefill, workflowPrefillDefinition{
				CapabilityID: item.CapabilityID,
				Flow:         item.Flow,
				Operation:    item.Operation,
				ResourceID:   item.ResourceID,
				Scope:        cloneStringMap(item.Scope),
				Query:        cloneStringMap(item.Query),
				Payload:      payload,
				Validate:     validate,
				Rollback:     rollback,
			})
		}
		out = append(out, workflowDefinition{
			ID:            d.ID,
			Label:         d.Label,
			Goal:          d.Goal,
			Prerequisites: append([]string(nil), d.Prerequisites...),
			Capabilities:  append([]string(nil), d.Capabilities...),
			Prefill:       prefill,
		})
	}
	return out
}

func marshalWorkflowJSON(value any) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return json.RawMessage(bytes.TrimSpace(buf.Bytes())), nil
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
	case "cherry_in_progress":
		return state.CherryInProgress
	case "has_staged":
		return len(state.StagingArea) > 0
	case "has_working_changes":
		return len(state.WorkingTree) > 0
	case "has_commits":
		return state.CommitCount > 0
	case "has_stash":
		return len(state.StashStack) > 0
	case "is_github_remote":
		return detectWorkflowPlatform(state) == platform.PlatformGitHub
	case "is_gitlab_remote":
		return detectWorkflowPlatform(state) == platform.PlatformGitLab
	case "is_bitbucket_remote":
		return detectWorkflowPlatform(state) == platform.PlatformBitbucket
	default:
		return false
	}
}

func detectWorkflowPlatform(state *status.GitState) platform.Platform {
	if state == nil {
		return platform.PlatformUnknown
	}
	for _, remote := range state.RemoteInfos {
		if strings.EqualFold(remote.Name, "origin") {
			if url := strings.TrimSpace(remote.PushURL); url != "" {
				return platform.DetectPlatform(url)
			}
			if url := strings.TrimSpace(remote.FetchURL); url != "" {
				return platform.DetectPlatform(url)
			}
		}
	}
	for _, remote := range state.RemoteInfos {
		if url := strings.TrimSpace(remote.PushURL); url != "" {
			return platform.DetectPlatform(url)
		}
		if url := strings.TrimSpace(remote.FetchURL); url != "" {
			return platform.DetectPlatform(url)
		}
	}
	return platform.PlatformUnknown
}

type workflowTokenContext struct {
	CurrentBranch string
	DefaultBranch string
	RepoOwner     string
	RepoName      string
}

func buildWorkflowOrchestration(wf workflowDefinition, state *status.GitState) *prompt.WorkflowOrchestration {
	platformID := detectWorkflowPlatform(state)
	if platformID == platform.PlatformUnknown {
		return nil
	}

	prefill := wf.Prefill
	if len(prefill) == 0 {
		prefill = fallbackWorkflowPrefill(platformID, wf)
	}
	if len(prefill) == 0 {
		return nil
	}

	ctx := workflowTokensFromState(platformID, state)
	seen := map[string]struct{}{}
	steps := make([]prompt.WorkflowOrchestrationStep, 0, len(prefill))
	for _, item := range prefill {
		step, ok := workflowPrefillStep(platformID, wf, item, ctx)
		if !ok {
			continue
		}
		identity := workflowStepIdentity(step)
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		steps = append(steps, step)
	}
	if len(steps) == 0 {
		return nil
	}
	return &prompt.WorkflowOrchestration{
		WorkflowID:    strings.TrimSpace(wf.ID),
		WorkflowLabel: strings.TrimSpace(wf.Label),
		Goal:          strings.TrimSpace(wf.Goal),
		Capabilities:  append([]string(nil), wf.Capabilities...),
		Steps:         steps,
	}
}

func fallbackWorkflowPrefill(platformID platform.Platform, wf workflowDefinition) []workflowPrefillDefinition {
	hints := platform.RecommendedExecutorSchemas(platformID, wf.Goal, wf.Capabilities, 3)
	if len(hints) == 0 {
		return nil
	}
	out := make([]workflowPrefillDefinition, 0, len(hints)*2)
	for _, hint := range hints {
		views := defaultInspectViews(hint)
		if len(views) == 0 {
			views = []string{""}
		}
		for _, view := range views {
			item := workflowPrefillDefinition{
				CapabilityID: hint.CapabilityID,
				Flow:         "inspect",
			}
			if strings.TrimSpace(view) != "" {
				item.Query = map[string]string{"view": view}
			}
			out = append(out, item)
		}
	}
	return out
}

func defaultInspectViews(hint platform.ExecutorSchemaHint) []string {
	if len(hint.InspectViews) == 0 {
		return nil
	}
	out := make([]string, 0, 2)
	add := func(view string) {
		view = strings.TrimSpace(view)
		if view == "" {
			return
		}
		for _, existing := range out {
			if strings.EqualFold(existing, view) {
				return
			}
		}
		out = append(out, view)
	}

	for _, view := range hint.InspectViews {
		normalized := strings.TrimSpace(view)
		if normalized == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(normalized), "default ") {
			add("")
			continue
		}
		add(normalized)
		if len(out) >= 2 {
			break
		}
	}
	if len(out) == 0 {
		out = append(out, "")
	}
	return out
}

func workflowPrefillStep(platformID platform.Platform, wf workflowDefinition, item workflowPrefillDefinition, ctx workflowTokenContext) (prompt.WorkflowOrchestrationStep, bool) {
	hint, ok := platform.ExecutorSchemaFor(platformID, strings.TrimSpace(item.CapabilityID))
	if !ok {
		return prompt.WorkflowOrchestrationStep{}, false
	}

	flow := strings.ToLower(strings.TrimSpace(item.Flow))
	if flow == "" {
		flow = "inspect"
	}
	op := workflowStepOperation(item, flow)
	op = applyWorkflowTokenContext(op, ctx)
	if flow == "inspect" && op.Query == nil {
		op.Query = map[string]string{}
	}

	title := workflowPrefillAction(hint, op)
	rationale := workflowPrefillReason(wf, hint, op)
	if strings.TrimSpace(title) == "" {
		return prompt.WorkflowOrchestrationStep{}, false
	}
	return prompt.WorkflowOrchestrationStep{
		Title:      title,
		Rationale:  rationale,
		Capability: strings.TrimSpace(op.CapabilityID),
		Flow:       strings.TrimSpace(op.Flow),
		Operation:  strings.TrimSpace(op.Operation),
		ResourceID: strings.TrimSpace(op.ResourceID),
		Scope:      cloneStringMap(op.Scope),
		Query:      cloneStringMap(op.Query),
		Payload:    cloneRaw(op.Payload),
		Validate:   cloneRaw(op.ValidatePayload),
		Rollback:   cloneRaw(op.RollbackPayload),
	}, true
}

func workflowStepOperation(item workflowPrefillDefinition, flow string) *git.PlatformExecInfo {
	return &git.PlatformExecInfo{
		CapabilityID:    strings.TrimSpace(item.CapabilityID),
		Flow:            flow,
		Operation:       strings.TrimSpace(item.Operation),
		ResourceID:      strings.TrimSpace(item.ResourceID),
		Scope:           cloneStringMap(item.Scope),
		Query:           cloneStringMap(item.Query),
		Payload:         cloneRaw(item.Payload),
		ValidatePayload: cloneRaw(item.Validate),
		RollbackPayload: cloneRaw(item.Rollback),
	}
}

func workflowPrefillAction(hint platform.ExecutorSchemaHint, op *git.PlatformExecInfo) string {
	if op == nil {
		return ""
	}
	label := strings.TrimSpace(hint.Label)
	if label == "" {
		label = strings.TrimSpace(op.CapabilityID)
	}
	switch strings.ToLower(strings.TrimSpace(op.Flow)) {
	case "mutate":
		verb := strings.TrimSpace(op.Operation)
		if verb == "" {
			verb = "update"
		}
		return fmt.Sprintf("%s: %s", label, verb)
	case "validate":
		return fmt.Sprintf("%s: validate", label)
	case "rollback":
		return fmt.Sprintf("%s: rollback", label)
	default:
		if view := strings.TrimSpace(op.Query["view"]); view != "" {
			return fmt.Sprintf("%s: inspect %s", label, view)
		}
		return fmt.Sprintf("%s: inspect", label)
	}
}

func workflowPrefillReason(wf workflowDefinition, hint platform.ExecutorSchemaHint, op *git.PlatformExecInfo) string {
	parts := []string{
		fmt.Sprintf("Workflow %q loaded a schema-backed %s request.", strings.TrimSpace(wf.Label), strings.TrimSpace(op.Flow)),
	}
	if summary := strings.TrimSpace(hint.Summary); summary != "" {
		parts = append(parts, summary)
	}
	if len(hint.Notes) > 0 {
		parts = append(parts, strings.TrimSpace(hint.Notes[0]))
	}
	return strings.Join(parts, " ")
}

func workflowTokensFromState(platformID platform.Platform, state *status.GitState) workflowTokenContext {
	ctx := workflowTokenContext{}
	if state == nil {
		return ctx
	}
	ctx.CurrentBranch = strings.TrimSpace(state.LocalBranch.Name)
	ctx.DefaultBranch = strings.TrimSpace(state.RepoConfig.DefaultBranch)
	if ctx.DefaultBranch == "" {
		ctx.DefaultBranch = ctx.CurrentBranch
	}

	switch platformID {
	case platform.PlatformGitHub:
		remoteURL := platform.PreferredRemoteURL(state.RemoteInfos)
		owner, repo, err := platform.GitHubOwnerRepoFromRemote(remoteURL)
		if err == nil {
			ctx.RepoOwner = strings.TrimSpace(owner)
			ctx.RepoName = strings.TrimSpace(repo)
		}
	}
	return ctx
}

func applyWorkflowTokenContext(op *git.PlatformExecInfo, ctx workflowTokenContext) *git.PlatformExecInfo {
	next := clonePlatformExecInfo(op)
	if next == nil {
		return nil
	}
	replacements := map[string]string{
		"<current_branch>": ctx.CurrentBranch,
		"<default_branch>": ctx.DefaultBranch,
		"<repo_owner>":     ctx.RepoOwner,
		"<repo_name>":      ctx.RepoName,
	}
	for token, value := range replacements {
		if strings.TrimSpace(value) == "" {
			continue
		}
		next.ResourceID = strings.ReplaceAll(next.ResourceID, token, value)
		next.Scope = replaceMapTokens(next.Scope, token, value)
		next.Query = replaceMapTokens(next.Query, token, value)
		next.Payload = replaceJSONTokens(next.Payload, token, value)
		next.ValidatePayload = replaceJSONTokens(next.ValidatePayload, token, value)
		next.RollbackPayload = replaceJSONTokens(next.RollbackPayload, token, value)
	}
	return next
}
