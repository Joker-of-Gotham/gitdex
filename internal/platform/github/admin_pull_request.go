package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type pullRequestExecutor struct{ client *Client }

func (e pullRequestExecutor) CapabilityID() string { return "pull_request" }

func (e pullRequestExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	pullNumber := pullNumberFromValues(req.ResourceID, req.Scope)
	view := strings.ToLower(normalizeScopeValue(req.Query, "view", ""))
	path := e.client.repoPath("/pulls")

	switch view {
	case "files":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		path = e.client.repoPath("/pulls/" + trimResourceID(pullNumber) + "/files")
	case "commits":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		path = e.client.repoPath("/pulls/" + trimResourceID(pullNumber) + "/commits")
	default:
		if pullNumber != "" {
			path = e.client.repoPath("/pulls/" + trimResourceID(pullNumber))
		} else {
			path = appendQuery(path, req.Query, "view", "pull_number")
		}
	}

	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), pullNumber, raw), nil
}

func (e pullRequestExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	pullNumber := pullNumberFromValues(req.ResourceID, req.Scope)
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   pullNumber,
		Metadata:     cloneStringMap(req.Scope),
	}
	if pullNumber != "" {
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullNumber})
	}

	switch op {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/pulls"), json.RawMessage(req.Payload), http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractPullNumber(raw, result.ResourceID)
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "update":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)), json.RawMessage(req.Payload), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), pullNumber, raw)
	case "close":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)), map[string]any{"state": "closed"}, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), pullNumber, raw)
	case "reopen":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)), map[string]any{"state": "open"}, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), pullNumber, raw)
	case "merge":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)+"/merge"), sanitizePullRequestMergePayload(req.Payload), http.StatusOK, http.StatusCreated)
		if err != nil {
			return nil, err
		}
		result.After, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullNumber})
		if result.After == nil {
			result.After = snapshot(e.CapabilityID(), pullNumber, raw)
		}
	case "enable_auto_merge":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		nodeID, err := pullRequestNodeID(result.Before)
		if err != nil {
			return nil, err
		}
		if err := e.enableAutoMerge(ctx, nodeID, req.Payload); err != nil {
			return nil, err
		}
		result.After, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullNumber})
	case "disable_auto_merge":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		nodeID, err := pullRequestNodeID(result.Before)
		if err != nil {
			return nil, err
		}
		if err := e.disableAutoMerge(ctx, nodeID); err != nil {
			return nil, err
		}
		result.After, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullNumber})
	default:
		return nil, fmt.Errorf("unsupported pull request operation: %s", op)
	}

	if result.After != nil {
		result.ResourceID = firstNonEmpty(result.ResourceID, result.After.ResourceID)
	}
	return result, nil
}

func (e pullRequestExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	pullNumber := pullNumberFromValues(req.ResourceID, req.Scope)
	if pullNumber == "" {
		pullNumber = strings.TrimSpace(req.Mutation.ResourceID)
	}
	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullNumber})
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: pullNumber}, nil
	}

	switch req.Mutation.Operation {
	case "close":
		return validatePullRequestState(snap, pullNumber, "closed")
	case "reopen":
		return validatePullRequestState(snap, pullNumber, "open")
	case "merge":
		return validatePullRequestMerged(snap, pullNumber, true)
	case "enable_auto_merge":
		return validatePullRequestAutoMerge(snap, pullNumber, true)
	case "disable_auto_merge":
		return validatePullRequestAutoMerge(snap, pullNumber, false)
	default:
		ok, summary, err := validatePullRequestSnapshot(snap.State, req.Payload)
		if err != nil {
			return nil, err
		}
		return &platform.AdminValidationResult{
			OK:         ok,
			Summary:    summary,
			ResourceID: pullNumber,
			Snapshot:   snap,
		}, nil
	}
}

func (e pullRequestExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	pullNumber := pullNumberFromValues(req.Mutation.ResourceID, req.Scope)

	switch req.Mutation.Operation {
	case "create":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "close",
			ResourceID: pullNumber,
			Scope:      cloneStringMap(req.Scope),
		}); err != nil {
			return nil, err
		}
		current, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullNumber})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "pull request closed as rollback", Snapshot: current}, nil
	case "update", "close", "reopen":
		return rollbackBySnapshot(ctx, e, req, sanitizePullRequestPayload, "pull request")
	case "merge":
		return &platform.AdminRollbackResult{
			OK:       false,
			Summary:  "merged pull requests cannot be automatically rolled back through GitHub API",
			Snapshot: req.Mutation.After,
		}, nil
	case "enable_auto_merge":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "disable_auto_merge",
			ResourceID: pullNumber,
			Scope:      cloneStringMap(req.Scope),
		}); err != nil {
			return nil, err
		}
		current, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullNumber})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "auto-merge disabled as rollback", Snapshot: current}, nil
	case "disable_auto_merge":
		if req.Mutation.Before == nil || !pullRequestHasAutoMerge(req.Mutation.Before.State) {
			return &platform.AdminRollbackResult{
				OK:       false,
				Summary:  "disable auto-merge rollback requires a previous auto-merge state",
				Snapshot: req.Mutation.Before,
			}, nil
		}
		nodeID, err := pullRequestNodeID(req.Mutation.Before)
		if err != nil {
			return nil, err
		}
		restorePayload, err := restoreAutoMergePayload(req.Mutation.Before.State)
		if err != nil {
			return nil, err
		}
		if err := e.enableAutoMerge(ctx, nodeID, restorePayload); err != nil {
			return nil, err
		}
		current, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullNumber})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "auto-merge restored", Snapshot: current}, nil
	default:
		return &platform.AdminRollbackResult{OK: false, Summary: "unsupported rollback operation for pull request"}, nil
	}
}

func (e pullRequestExecutor) enableAutoMerge(ctx context.Context, nodeID string, payload json.RawMessage) error {
	vars, err := autoMergeVariables(payload)
	if err != nil {
		return err
	}
	vars["pullRequestId"] = nodeID
	query := `
mutation EnableAutoMerge($pullRequestId: ID!, $mergeMethod: PullRequestMergeMethod, $commitHeadline: String, $commitBody: String, $authorEmail: String) {
  enablePullRequestAutoMerge(input: {
    pullRequestId: $pullRequestId,
    mergeMethod: $mergeMethod,
    commitHeadline: $commitHeadline,
    commitBody: $commitBody,
    authorEmail: $authorEmail
  }) {
    pullRequest {
      id
      number
    }
  }
}`
	return e.client.doGraphQL(ctx, query, vars, nil)
}

func (e pullRequestExecutor) disableAutoMerge(ctx context.Context, nodeID string) error {
	query := `
mutation DisableAutoMerge($pullRequestId: ID!) {
  disablePullRequestAutoMerge(input: { pullRequestId: $pullRequestId }) {
    pullRequest {
      id
      number
    }
  }
}`
	return e.client.doGraphQL(ctx, query, map[string]any{"pullRequestId": nodeID}, nil)
}

func pullNumberFromValues(resourceID string, scope map[string]string) string {
	return firstNonEmpty(strings.TrimSpace(resourceID), normalizeScopeValue(scope, "pull_number", ""))
}

func extractPullNumber(raw json.RawMessage, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	obj, err := rawObject(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stringValue(obj["number"]))
}

func sanitizePullRequestMergePayload(raw json.RawMessage) map[string]any {
	obj, err := rawObject(raw)
	if err != nil {
		return map[string]any{}
	}
	return keepOnlyKeys(obj, "commit_title", "commit_message", "sha", "merge_method")
}

func sanitizePullRequestPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	deleteKeys(obj, "id", "node_id", "number", "html_url", "url", "state_reason", "merged", "merged_at", "closed_at", "created_at", "updated_at", "auto_merge", "head", "base", "user")
	out := keepOnlyKeys(obj, "title", "body", "state", "maintainer_can_modify")
	if root, err := rawObject(raw); err == nil {
		if baseObj, ok := root["base"].(map[string]any); ok {
			if ref := strings.TrimSpace(stringValue(baseObj["ref"])); ref != "" {
				out["base"] = ref
			}
		}
	}
	return out, nil
}

func validatePullRequestState(snap *platform.AdminSnapshot, resourceID, want string) (*platform.AdminValidationResult, error) {
	obj, err := rawObject(snap.State)
	if err != nil {
		return nil, err
	}
	actual := strings.TrimSpace(strings.ToLower(stringValue(obj["state"])))
	ok := actual == strings.ToLower(strings.TrimSpace(want))
	summary := "pull request state mismatch"
	if ok {
		summary = "pull request state validated"
	}
	return &platform.AdminValidationResult{OK: ok, Summary: summary, ResourceID: resourceID, Snapshot: snap}, nil
}

func validatePullRequestMerged(snap *platform.AdminSnapshot, resourceID string, want bool) (*platform.AdminValidationResult, error) {
	obj, err := rawObject(snap.State)
	if err != nil {
		return nil, err
	}
	actual := boolFromAny(obj["merged"])
	ok := actual == want
	summary := "pull request merge state mismatch"
	if ok {
		summary = "pull request merge validated"
	}
	return &platform.AdminValidationResult{OK: ok, Summary: summary, ResourceID: resourceID, Snapshot: snap}, nil
}

func validatePullRequestAutoMerge(snap *platform.AdminSnapshot, resourceID string, want bool) (*platform.AdminValidationResult, error) {
	actual := pullRequestHasAutoMerge(snap.State)
	ok := actual == want
	summary := "pull request auto-merge state mismatch"
	if ok {
		summary = "pull request auto-merge validated"
	}
	return &platform.AdminValidationResult{OK: ok, Summary: summary, ResourceID: resourceID, Snapshot: snap}, nil
}

func validatePullRequestSnapshot(raw json.RawMessage, payload json.RawMessage) (bool, string, error) {
	if len(payload) == 0 {
		return true, "pull request validated", nil
	}
	actual, err := rawObject(raw)
	if err != nil {
		return false, "", err
	}
	expected, err := rawObject(payload)
	if err != nil {
		return false, "", err
	}

	for _, key := range []string{"title", "body", "state"} {
		if _, ok := expected[key]; ok {
			if strings.TrimSpace(stringValue(actual[key])) != strings.TrimSpace(stringValue(expected[key])) {
				return false, key + ": value mismatch", nil
			}
		}
	}
	for _, key := range []string{"draft", "maintainer_can_modify"} {
		if _, ok := expected[key]; ok {
			if boolFromAny(actual[key]) != boolFromAny(expected[key]) {
				return false, key + ": value mismatch", nil
			}
		}
	}
	if value, ok := expected["base"]; ok {
		baseObj, _ := actual["base"].(map[string]any)
		if strings.TrimSpace(stringValue(baseObj["ref"])) != strings.TrimSpace(stringValue(value)) {
			return false, "base: value mismatch", nil
		}
	}
	if value, ok := expected["head"]; ok {
		headObj, _ := actual["head"].(map[string]any)
		if strings.TrimSpace(stringValue(headObj["ref"])) != strings.TrimSpace(stringValue(value)) {
			return false, "head: value mismatch", nil
		}
	}
	return true, "pull request validated", nil
}

func pullRequestNodeID(snap *platform.AdminSnapshot) (string, error) {
	if snap == nil {
		return "", fmt.Errorf("pull request snapshot is required")
	}
	obj, err := rawObject(snap.State)
	if err != nil {
		return "", err
	}
	nodeID := strings.TrimSpace(stringValue(obj["node_id"]))
	if nodeID == "" {
		return "", fmt.Errorf("pull request node id is required for auto-merge")
	}
	return nodeID, nil
}

func pullRequestHasAutoMerge(raw json.RawMessage) bool {
	obj, err := rawObject(raw)
	if err != nil {
		return false
	}
	value, exists := obj["auto_merge"]
	return exists && value != nil
}

func restoreAutoMergePayload(raw json.RawMessage) (json.RawMessage, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	autoMerge, _ := obj["auto_merge"].(map[string]any)
	if autoMerge == nil {
		return nil, fmt.Errorf("auto_merge data is required")
	}
	payload := map[string]any{}
	if method := strings.TrimSpace(stringValue(autoMerge["merge_method"])); method != "" {
		payload["merge_method"] = method
	}
	if title := strings.TrimSpace(stringValue(autoMerge["commit_title"])); title != "" {
		payload["commit_title"] = title
	}
	if body := strings.TrimSpace(stringValue(autoMerge["commit_message"])); body != "" {
		payload["commit_message"] = body
	}
	if email := strings.TrimSpace(stringValue(autoMerge["author_email"])); email != "" {
		payload["author_email"] = email
	}
	return marshalRaw(payload)
}

func autoMergeVariables(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	vars := map[string]any{}
	if method := strings.ToUpper(strings.TrimSpace(stringValue(obj["merge_method"]))); method != "" {
		switch method {
		case "MERGE", "SQUASH", "REBASE":
			vars["mergeMethod"] = method
		default:
			switch strings.ToLower(method) {
			case "merge":
				vars["mergeMethod"] = "MERGE"
			case "squash":
				vars["mergeMethod"] = "SQUASH"
			case "rebase":
				vars["mergeMethod"] = "REBASE"
			default:
				return nil, fmt.Errorf("unsupported auto-merge method: %s", method)
			}
		}
	}
	if title := strings.TrimSpace(stringValue(obj["commit_title"])); title != "" {
		vars["commitHeadline"] = title
	}
	if body := strings.TrimSpace(stringValue(obj["commit_message"])); body != "" {
		vars["commitBody"] = body
	}
	if email := strings.TrimSpace(stringValue(obj["author_email"])); email != "" {
		vars["authorEmail"] = email
	}
	return vars, nil
}

func boolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	case float64:
		return typed != 0
	default:
		return false
	}
}
