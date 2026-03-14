package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type codespacesExecutor struct{ client *Client }

func (e codespacesExecutor) CapabilityID() string { return "codespaces" }

func (e codespacesExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
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

func (e codespacesExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   strings.TrimSpace(req.ResourceID),
	}

	switch op {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/codespaces"), json.RawMessage(req.Payload), http.StatusCreated, http.StatusAccepted)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractResourceID(raw, result.ResourceID)
		if result.ResourceID == "" {
			if name, ok := jsonFieldString(raw, "name"); ok {
				result.ResourceID = name
			}
		}
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
		return result, nil
	case "start", "stop", "delete":
		name := strings.TrimSpace(firstNonEmpty(req.ResourceID, normalizeScopeValue(req.Scope, "codespace_name", "")))
		if name == "" {
			return nil, fmt.Errorf("codespace name is required")
		}
		result.ResourceID = name
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: name, Query: map[string]string{"view": "single"}})
		switch op {
		case "start":
			raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.userPath("/codespaces/"+trimResourceID(name)+"/start"), nil, http.StatusOK, http.StatusAccepted)
			if err != nil {
				return nil, err
			}
			result.After = snapshot(e.CapabilityID(), name, raw)
		case "stop":
			raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.userPath("/codespaces/"+trimResourceID(name)+"/stop"), nil, http.StatusOK, http.StatusAccepted)
			if err != nil {
				return nil, err
			}
			result.After = snapshot(e.CapabilityID(), name, raw)
		default:
			if err := e.client.doJSON(ctx, http.MethodDelete, e.client.userPath("/codespaces/"+trimResourceID(name)), nil, nil, http.StatusAccepted, http.StatusNoContent); err != nil {
				return nil, err
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported codespaces operation: %s", op)
	}
}

func (e codespacesExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	name := strings.TrimSpace(firstNonEmpty(req.ResourceID, req.Mutation.ResourceID))
	switch req.Mutation.Operation {
	case "delete":
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: name, Query: map[string]string{"view": "single"}})
		if inspectMissingOK(err) {
			return &platform.AdminValidationResult{OK: true, Summary: "codespace deleted", ResourceID: name}, nil
		}
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: name}, nil
		}
		return &platform.AdminValidationResult{OK: false, Summary: "codespace still exists", ResourceID: name, Snapshot: snap}, nil
	default:
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: name, Query: map[string]string{"view": "single"}})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: name}, nil
		}
		expected := cloneRaw(req.Payload)
		if len(expected) == 0 && req.Mutation.After != nil {
			expected = sanitizeCodespaceState(req.Mutation.After.State)
		}
		if len(expected) == 0 {
			return &platform.AdminValidationResult{OK: true, Summary: "codespace acknowledged", ResourceID: name, Snapshot: snap}, nil
		}
		matched, reason, matchErr := subsetMatches(snap.State, expected)
		if matchErr != nil {
			return nil, matchErr
		}
		return &platform.AdminValidationResult{OK: matched, Summary: summaryFromMatch(matched, reason, "codespace validated"), ResourceID: name, Snapshot: snap}, nil
	}
}

func (e codespacesExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	name := strings.TrimSpace(req.Mutation.ResourceID)
	switch req.Mutation.Operation {
	case "create":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "delete", ResourceID: name}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "codespace deleted as rollback"}, nil
	case "start":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "stop", ResourceID: name}); err != nil {
			return nil, err
		}
		snap, _ := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: name, Query: map[string]string{"view": "single"}})
		return &platform.AdminRollbackResult{OK: true, Summary: "codespace stopped as rollback", Snapshot: snap}, nil
	case "stop":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "start", ResourceID: name}); err != nil {
			return nil, err
		}
		snap, _ := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: name, Query: map[string]string{"view": "single"}})
		return &platform.AdminRollbackResult{OK: true, Summary: "codespace started as rollback", Snapshot: snap}, nil
	case "delete":
		return &platform.AdminRollbackResult{OK: false, Summary: "deleted codespaces cannot be recreated automatically via rollback"}, nil
	default:
		return nil, fmt.Errorf("unsupported codespaces rollback operation: %s", req.Mutation.Operation)
	}
}

func (e codespacesExecutor) inspectPath(resourceID string, scope, query map[string]string) (string, string, error) {
	view := strings.ToLower(normalizeScopeValue(query, "view", normalizeScopeValue(scope, "view", "")))
	switch view {
	case "", "list":
		return appendQuery(e.client.repoPath("/codespaces"), query, "view", "codespace_name"), "", nil
	case "devcontainers":
		return appendQuery(e.client.repoPath("/codespaces/devcontainers"), query, "view", "codespace_name"), "devcontainers", nil
	case "policy", "repo_policy", "prebuild", "prebuilds":
		return appendQuery(e.client.repoPath("/codespaces/machines"), query, "view", "codespace_name"), view, nil
	case "permissions_check":
		return appendQuery(e.client.repoPath("/codespaces/permissions_check"), query, "view", "codespace_name"), "permissions_check", nil
	case "single":
		name := strings.TrimSpace(firstNonEmpty(resourceID, normalizeScopeValue(scope, "codespace_name", normalizeScopeValue(query, "codespace_name", ""))))
		if name == "" {
			return "", "", fmt.Errorf("codespace name is required")
		}
		return e.client.userPath("/codespaces/" + trimResourceID(name)), name, nil
	default:
		return "", "", fmt.Errorf("unsupported codespaces view: %s", view)
	}
}

func sanitizeCodespaceState(raw json.RawMessage) json.RawMessage {
	obj, err := rawObject(raw)
	if err != nil {
		return cloneRaw(raw)
	}
	filtered := keepOnlyKeys(obj, "name", "display_name", "state", "repository", "git_status", "machine")
	out, _ := marshalRaw(filtered)
	return out
}
