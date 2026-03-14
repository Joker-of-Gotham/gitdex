package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type actionsExecutor struct{ client *Client }

func (e actionsExecutor) CapabilityID() string { return "actions" }

func (e actionsExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path, resourceID, err := e.inspectPath(req.ResourceID, req.Scope, req.Query)
	if err != nil {
		return nil, err
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e actionsExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
	}

	switch op {
	case "permissions_update":
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "permissions"}})
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/actions/permissions"), sanitizeActionsPayload(req.Payload, "permissions"), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), "permissions", raw)
		result.ResourceID = "permissions"
		return result, nil
	case "allowed_actions_update":
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "allowed_actions"}})
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/actions/permissions/selected-actions"), sanitizeActionsPayload(req.Payload, "allowed_actions"), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), "allowed_actions", raw)
		result.ResourceID = "allowed_actions"
		return result, nil
	case "enable_workflow", "disable_workflow", "dispatch":
		workflowID := strings.TrimSpace(firstNonEmpty(req.ResourceID, normalizeScopeValue(req.Scope, "workflow_id", "")))
		if workflowID == "" {
			return nil, fmt.Errorf("workflow id is required")
		}
		result.ResourceID = workflowID
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: workflowID, Query: map[string]string{"view": "workflow"}})
		switch op {
		case "enable_workflow":
			if err := e.client.doJSON(ctx, http.MethodPut, e.client.repoPath("/actions/workflows/"+trimResourceID(workflowID)+"/enable"), nil, nil, http.StatusNoContent); err != nil {
				return nil, err
			}
		case "disable_workflow":
			if err := e.client.doJSON(ctx, http.MethodPut, e.client.repoPath("/actions/workflows/"+trimResourceID(workflowID)+"/disable"), nil, nil, http.StatusNoContent); err != nil {
				return nil, err
			}
		default:
			if err := e.client.doJSON(ctx, http.MethodPost, e.client.repoPath("/actions/workflows/"+trimResourceID(workflowID)+"/dispatches"), json.RawMessage(req.Payload), nil, http.StatusNoContent, http.StatusCreated, http.StatusAccepted); err != nil {
				return nil, err
			}
		}
		after, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: workflowID, Query: map[string]string{"view": "workflow"}})
		if err == nil {
			result.After = after
		}
		return result, nil
	case "rerun", "cancel_run":
		runID := strings.TrimSpace(firstNonEmpty(req.ResourceID, normalizeScopeValue(req.Scope, "run_id", "")))
		if runID == "" {
			return nil, fmt.Errorf("run id is required")
		}
		result.ResourceID = runID
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: runID, Query: map[string]string{"view": "run"}})
		suffix := "/rerun"
		if op == "cancel_run" {
			suffix = "/cancel"
		}
		if err := e.client.doJSON(ctx, http.MethodPost, e.client.repoPath("/actions/runs/"+trimResourceID(runID)+suffix), json.RawMessage(req.Payload), nil, http.StatusAccepted, http.StatusCreated, http.StatusOK); err != nil {
			return nil, err
		}
		after, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: runID, Query: map[string]string{"view": "run"}})
		if err == nil {
			result.After = after
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported actions operation: %s", op)
	}
}

func (e actionsExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}

	switch req.Mutation.Operation {
	case "permissions_update":
		return e.validateByView(ctx, "permissions", "actions permissions validated", req)
	case "allowed_actions_update":
		return e.validateByView(ctx, "allowed_actions", "selected actions policy validated", req)
	case "enable_workflow", "disable_workflow":
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.Mutation.ResourceID, Query: map[string]string{"view": "workflow"}})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: req.Mutation.ResourceID}, nil
		}
		state, ok := jsonFieldString(snap.State, "state")
		if !ok {
			return &platform.AdminValidationResult{OK: true, Summary: "workflow status acknowledged", ResourceID: req.Mutation.ResourceID, Snapshot: snap}, nil
		}
		expectedDisabled := req.Mutation.Operation == "disable_workflow"
		actualDisabled := strings.Contains(strings.ToLower(strings.TrimSpace(state)), "disable")
		okState := expectedDisabled == actualDisabled
		summary := "workflow state validated"
		if !okState {
			summary = fmt.Sprintf("workflow state mismatch: %s", state)
		}
		return &platform.AdminValidationResult{OK: okState, Summary: summary, ResourceID: req.Mutation.ResourceID, Snapshot: snap}, nil
	case "dispatch", "rerun", "cancel_run":
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.Mutation.ResourceID, Query: map[string]string{"view": "run"}})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: req.Mutation.ResourceID}, nil
		}
		return &platform.AdminValidationResult{OK: true, Summary: "actions run acknowledged", ResourceID: req.Mutation.ResourceID, Snapshot: snap}, nil
	default:
		return nil, fmt.Errorf("unsupported actions validate operation: %s", req.Mutation.Operation)
	}
}

func (e actionsExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}

	switch req.Mutation.Operation {
	case "permissions_update":
		return e.rollbackByView(ctx, "permissions", req)
	case "allowed_actions_update":
		return e.rollbackByView(ctx, "allowed_actions", req)
	case "enable_workflow":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "disable_workflow", ResourceID: req.Mutation.ResourceID}); err != nil {
			return nil, err
		}
		snap, _ := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.Mutation.ResourceID, Query: map[string]string{"view": "workflow"}})
		return &platform.AdminRollbackResult{OK: true, Summary: "workflow disabled as rollback", Snapshot: snap}, nil
	case "disable_workflow":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "enable_workflow", ResourceID: req.Mutation.ResourceID}); err != nil {
			return nil, err
		}
		snap, _ := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.Mutation.ResourceID, Query: map[string]string{"view": "workflow"}})
		return &platform.AdminRollbackResult{OK: true, Summary: "workflow enabled as rollback", Snapshot: snap}, nil
	case "dispatch", "rerun", "cancel_run":
		return &platform.AdminRollbackResult{OK: false, Summary: "workflow dispatch and run control cannot be rolled back automatically"}, nil
	default:
		return nil, fmt.Errorf("unsupported actions rollback operation: %s", req.Mutation.Operation)
	}
}

func (e actionsExecutor) inspectPath(resourceID string, scope, query map[string]string) (string, string, error) {
	view := strings.ToLower(normalizeScopeValue(query, "view", normalizeScopeValue(scope, "view", "")))
	switch view {
	case "", "workflows":
		return appendQuery(e.client.repoPath("/actions/workflows"), query, "view", "workflow_id", "run_id"), "", nil
	case "workflow":
		workflowID := strings.TrimSpace(firstNonEmpty(resourceID, normalizeScopeValue(scope, "workflow_id", normalizeScopeValue(query, "workflow_id", ""))))
		if workflowID == "" {
			return "", "", fmt.Errorf("workflow id is required")
		}
		return e.client.repoPath("/actions/workflows/" + trimResourceID(workflowID)), workflowID, nil
	case "runs", "workflow_runs":
		return appendQuery(e.client.repoPath("/actions/runs"), query, "view", "workflow_id", "run_id"), "", nil
	case "run":
		runID := strings.TrimSpace(firstNonEmpty(resourceID, normalizeScopeValue(scope, "run_id", normalizeScopeValue(query, "run_id", ""))))
		if runID == "" {
			return "", "", fmt.Errorf("run id is required")
		}
		return e.client.repoPath("/actions/runs/" + trimResourceID(runID)), runID, nil
	case "artifacts":
		return appendQuery(e.client.repoPath("/actions/artifacts"), query, "view", "workflow_id", "run_id"), "artifacts", nil
	case "caches":
		return appendQuery(e.client.repoPath("/actions/caches"), query, "view", "workflow_id", "run_id"), "caches", nil
	case "workflow_usage":
		workflowID := strings.TrimSpace(firstNonEmpty(resourceID, normalizeScopeValue(scope, "workflow_id", normalizeScopeValue(query, "workflow_id", ""))))
		if workflowID == "" {
			return "", "", fmt.Errorf("workflow id is required")
		}
		return e.client.repoPath("/actions/workflows/" + trimResourceID(workflowID) + "/timing"), workflowID, nil
	case "repo_policy", "permissions":
		return e.client.repoPath("/actions/permissions"), "permissions", nil
	case "allowed_actions":
		return e.client.repoPath("/actions/permissions/selected-actions"), "allowed_actions", nil
	default:
		return "", "", fmt.Errorf("unsupported actions view: %s", view)
	}
}

func (e actionsExecutor) validateByView(ctx context.Context, view, okSummary string, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.Mutation.ResourceID, Query: map[string]string{"view": view}})
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: req.Mutation.ResourceID}, nil
	}
	expected := cloneRaw(req.Payload)
	if len(expected) == 0 && req.Mutation.After != nil {
		expected = sanitizeActionsPayload(req.Mutation.After.State, view)
	}
	matched, reason, matchErr := subsetMatches(snap.State, expected)
	if matchErr != nil {
		return nil, matchErr
	}
	return &platform.AdminValidationResult{
		OK:         matched,
		Summary:    summaryFromMatch(matched, reason, okSummary),
		ResourceID: req.Mutation.ResourceID,
		Snapshot:   snap,
	}, nil
}

func (e actionsExecutor) rollbackByView(ctx context.Context, view string, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation.Before == nil {
		return &platform.AdminRollbackResult{OK: false, Summary: "rollback requires previous snapshot"}, nil
	}
	restore := sanitizeActionsPayload(req.Mutation.Before.State, view)
	path, resourceID, err := e.inspectPath(req.Mutation.ResourceID, nil, map[string]string{"view": view})
	if err != nil {
		return nil, err
	}
	raw, err := e.client.doRaw(ctx, http.MethodPut, path, restore, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return &platform.AdminRollbackResult{
		OK:       true,
		Summary:  "actions setting restored",
		Snapshot: snapshot(e.CapabilityID(), resourceID, raw),
	}, nil
}

func sanitizeActionsPayload(raw json.RawMessage, view string) json.RawMessage {
	obj, err := rawObject(raw)
	if err != nil {
		return cloneRaw(raw)
	}
	switch view {
	case "permissions":
		filtered := keepOnlyKeys(obj, "enabled", "allowed_actions", "sha_pinning_required")
		out, _ := marshalRaw(filtered)
		return out
	case "allowed_actions":
		filtered := keepOnlyKeys(obj, "github_owned_allowed", "verified_allowed", "patterns")
		out, _ := marshalRaw(filtered)
		return out
	default:
		return cloneRaw(raw)
	}
}

func jsonFieldString(raw json.RawMessage, key string) (string, bool) {
	obj, err := rawObject(raw)
	if err != nil {
		return "", false
	}
	value, ok := obj[key].(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}
