package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type repoSecurityExecutor struct {
	client       *Client
	capabilityID string
}

type alertExecutor struct {
	client       *Client
	capabilityID string
	baseSegment  string
}

type codeScanningExecutor struct {
	client       *Client
	capabilityID string
}

func (e repoSecurityExecutor) CapabilityID() string { return e.capabilityID }
func (e alertExecutor) CapabilityID() string        { return e.capabilityID }
func (e codeScanningExecutor) CapabilityID() string { return e.capabilityID }

func (e repoSecurityExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	view := strings.ToLower(normalizeScopeValue(req.Query, "view", ""))
	switch {
	case e.capabilityID == "dependency_graph" || view == "sbom":
		raw, err := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath("/dependency-graph/sbom"), nil, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return snapshot(e.CapabilityID(), "", raw), nil
	case e.capabilityID == "dependabot_security_updates" || e.capabilityID == "dependabot_posture" || view == "automated_security_fixes":
		raw, err := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath("/automated-security-fixes"), nil, http.StatusOK)
		if err != nil {
			return nil, err
		}
		if e.capabilityID == "dependabot_posture" {
			state, summarizeErr := e.inspectDependabotPosture(ctx, raw)
			if summarizeErr != nil {
				return nil, summarizeErr
			}
			return snapshot(e.CapabilityID(), "", state), nil
		}
		return snapshot(e.CapabilityID(), "", raw), nil
	case e.capabilityID == "code_scanning_tool_settings" || view == "tool_settings":
		raw, err := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath("/code-security-configuration"), nil, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return snapshot(e.CapabilityID(), "tool_settings", raw), nil
	case view == "configuration":
		raw, err := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath("/code-security-configuration"), nil, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return snapshot(e.CapabilityID(), "", raw), nil
	case view == "summary":
		state, err := e.inspectAdvancedSecuritySummary(ctx)
		if err != nil {
			return nil, err
		}
		return snapshot(e.CapabilityID(), "", state), nil
	default:
		raw, err := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath(""), nil, http.StatusOK)
		if err != nil {
			return nil, err
		}
		state, err := sanitizeRepoSecuritySnapshot(raw, e.capabilityID)
		if err != nil {
			return nil, err
		}
		return snapshot(e.CapabilityID(), "", state), nil
	}
}

func (e repoSecurityExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	before, _ := e.Inspect(ctx, platform.AdminInspectRequest{})
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		Before:       before,
	}

	switch e.capabilityID {
	case "dependency_graph":
		return nil, fmt.Errorf("dependency graph does not expose repository-level mutation through this executor")
	case "code_scanning_tool_settings":
		return nil, fmt.Errorf("code scanning tool settings are inspect-only through the public GitHub API")
	case "dependabot_security_updates":
		path := e.client.repoPath("/automated-security-fixes")
		switch op {
		case "enable", "create", "update":
			if err := e.client.doJSON(ctx, http.MethodPut, path, nil, nil, http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
				return nil, err
			}
		case "disable", "delete":
			if err := e.client.doJSON(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent, http.StatusAccepted); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported dependabot security updates operation: %s", op)
		}
	case "dependabot_posture":
		path := e.client.repoPath("/automated-security-fixes")
		switch op {
		case "enable", "create", "update":
			if err := e.client.doJSON(ctx, http.MethodPut, path, nil, nil, http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
				return nil, err
			}
		case "disable", "delete":
			if err := e.client.doJSON(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent, http.StatusAccepted); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported dependabot posture operation: %s", op)
		}
	default:
		payload, err := normalizeRepoSecurityMutationPayload(e.capabilityID, op, req.Payload)
		if err != nil {
			return nil, err
		}
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath(""), payload, http.StatusOK)
		if err != nil {
			return nil, err
		}
		state, err := sanitizeRepoSecuritySnapshot(raw, e.capabilityID)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), "", state)
		return result, nil
	}

	after, err := e.Inspect(ctx, platform.AdminInspectRequest{})
	if err != nil {
		return nil, err
	}
	result.After = after
	return result, nil
}

func (e repoSecurityExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{})
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error()}, nil
	}
	expected := cloneRaw(req.Payload)
	if len(expected) == 0 && req.Mutation.After != nil {
		expected = cloneRaw(req.Mutation.After.State)
	}
	if len(expected) == 0 {
		return &platform.AdminValidationResult{OK: true, Summary: "security surface validated", Snapshot: snap}, nil
	}
	if e.capabilityID != "dependabot_security_updates" && e.capabilityID != "dependabot_posture" && !looksLikeRepoSecurityState(expected) {
		wrapped, wrapErr := normalizeRepoSecurityMutationPayload(e.capabilityID, req.Mutation.Operation, expected)
		if wrapErr == nil {
			expected = wrapped
		}
	}
	matched, reason, matchErr := subsetMatches(snap.State, expected)
	if matchErr != nil {
		return nil, matchErr
	}
	return &platform.AdminValidationResult{
		OK:       matched,
		Summary:  summaryFromMatch(matched, reason, "security surface validated"),
		Snapshot: snap,
	}, nil
}

func (e repoSecurityExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	if req.Mutation.Before == nil {
		switch e.capabilityID {
		case "dependabot_security_updates":
			if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "disable"}); err != nil {
				return nil, err
			}
			return &platform.AdminRollbackResult{OK: true, Summary: "dependabot security updates disabled as rollback"}, nil
		default:
			return &platform.AdminRollbackResult{OK: false, Summary: "rollback requires previous security snapshot"}, nil
		}
	}

	switch e.capabilityID {
	case "dependabot_security_updates":
		if repoSecurityEnabled(req.Mutation.Before.State) {
			if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "enable"}); err != nil {
				return nil, err
			}
			return &platform.AdminRollbackResult{OK: true, Summary: "dependabot security updates restored", Snapshot: req.Mutation.Before}, nil
		}
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "disable"}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "dependabot security updates disabled", Snapshot: req.Mutation.Before}, nil
	case "dependabot_posture":
		if repoSecurityEnabled(req.Mutation.Before.State) {
			if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "enable"}); err != nil {
				return nil, err
			}
		} else {
			if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "disable"}); err != nil {
				return nil, err
			}
		}
		current, err := e.Inspect(ctx, platform.AdminInspectRequest{})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "dependabot posture restored", Snapshot: current}, nil
	case "code_scanning_tool_settings":
		return &platform.AdminRollbackResult{OK: false, Summary: "code scanning tool settings are inspect-only through the public GitHub API"}, nil
	default:
		restore, err := sanitizeRepoSecurityRollback(req.Mutation.Before.State)
		if err != nil {
			return nil, err
		}
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation: "update",
			Payload:   restore,
		}); err != nil {
			return nil, err
		}
		current, err := e.Inspect(ctx, platform.AdminInspectRequest{})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "security surface restored", Snapshot: current}, nil
	}
}

func (e repoSecurityExecutor) inspectDependabotPosture(ctx context.Context, fixesRaw json.RawMessage) (json.RawMessage, error) {
	state := map[string]any{}
	if len(fixesRaw) > 0 {
		var fixes any
		if json.Unmarshal(fixesRaw, &fixes) == nil {
			state["automated_security_fixes"] = fixes
		}
	}
	repoRaw, err := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath(""), nil, http.StatusOK)
	if err == nil {
		if sanitized, sanitizeErr := sanitizeRepoSecuritySnapshot(repoRaw, "dependabot_posture"); sanitizeErr == nil {
			var parsed any
			if json.Unmarshal(sanitized, &parsed) == nil {
				state["repo_security"] = parsed
			}
		}
	}
	return marshalRaw(state)
}

func (e repoSecurityExecutor) inspectAdvancedSecuritySummary(ctx context.Context) (json.RawMessage, error) {
	summary := map[string]any{
		"capability_id": e.capabilityID,
	}

	repoRaw, err := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath(""), nil, http.StatusOK)
	if err == nil {
		if sanitized, sanitizeErr := sanitizeRepoSecuritySnapshot(repoRaw, "advanced_security"); sanitizeErr == nil {
			var parsed any
			if json.Unmarshal(sanitized, &parsed) == nil {
				summary["repo_security"] = parsed
			}
		}
	}
	if raw, fixErr := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath("/automated-security-fixes"), nil, http.StatusOK); fixErr == nil && len(raw) > 0 {
		var parsed any
		if json.Unmarshal(raw, &parsed) == nil {
			summary["automated_security_fixes"] = parsed
		}
	}
	if raw, cfgErr := e.client.doRaw(ctx, http.MethodGet, e.client.repoPath("/code-security-configuration"), nil, http.StatusOK); cfgErr == nil && len(raw) > 0 {
		var parsed any
		if json.Unmarshal(raw, &parsed) == nil {
			summary["code_security_configuration"] = parsed
		}
	}
	return marshalRaw(summary)
}

func (e alertExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
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

func (e alertExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		resourceID = normalizeScopeValue(req.Scope, "alert_number", "")
	}
	if resourceID == "" {
		return nil, fmt.Errorf("alert number is required")
	}
	before, _ := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: resourceID})
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	payload, err := e.defaultAlertPayload(op, req.Payload)
	if err != nil {
		return nil, err
	}
	raw, err := e.client.doRaw(ctx, http.MethodPatch, e.alertPath(resourceID), payload, http.StatusOK, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	return &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   resourceID,
		Before:       before,
		After:        snapshot(e.CapabilityID(), resourceID, raw),
	}, nil
}

func (e alertExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	resourceID := strings.TrimSpace(firstNonEmpty(req.ResourceID, req.Mutation.ResourceID))
	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: resourceID})
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: resourceID}, nil
	}
	expected := cloneRaw(req.Payload)
	if len(expected) == 0 && req.Mutation.After != nil {
		expected = sanitizeAlertStateForMatch(req.Mutation.After.State, e.capabilityID)
	}
	matched, reason, matchErr := subsetMatches(snap.State, expected)
	if matchErr != nil {
		return nil, matchErr
	}
	return &platform.AdminValidationResult{
		OK:         matched,
		Summary:    summaryFromMatch(matched, reason, "alert validated"),
		ResourceID: resourceID,
		Snapshot:   snap,
	}, nil
}

func (e alertExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	if req.Mutation.Before == nil {
		return &platform.AdminRollbackResult{OK: false, Summary: "alert rollback requires previous alert state"}, nil
	}
	payload := sanitizeAlertStateForMatch(req.Mutation.Before.State, e.capabilityID)
	raw, err := e.client.doRaw(ctx, http.MethodPatch, e.alertPath(req.Mutation.ResourceID), payload, http.StatusOK, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	return &platform.AdminRollbackResult{
		OK:       true,
		Summary:  "alert restored",
		Snapshot: snapshot(e.CapabilityID(), req.Mutation.ResourceID, raw),
	}, nil
}

func (e alertExecutor) inspectPath(resourceID string, scope, query map[string]string) (string, string, error) {
	view := strings.ToLower(normalizeScopeValue(query, "view", normalizeScopeValue(scope, "view", "")))
	alertID := strings.TrimSpace(resourceID)
	if alertID == "" {
		alertID = normalizeScopeValue(scope, "alert_number", normalizeScopeValue(query, "alert_number", ""))
	}
	switch {
	case (e.capabilityID == "secret_protection" || e.capabilityID == "secret_scanning_alerts") && view == "push_protection_bypasses":
		return appendQuery(e.client.repoPath("/secret-scanning/push-protection-bypasses"), query), "push_protection_bypasses", nil
	case (e.capabilityID == "secret_protection" || e.capabilityID == "secret_scanning_alerts") && view == "locations":
		if alertID == "" {
			return "", "", fmt.Errorf("alert number is required")
		}
		return e.alertPath(alertID) + "/locations", alertID, nil
	case alertID != "":
		return e.alertPath(alertID), alertID, nil
	default:
		return appendQuery(e.client.repoPath("/"+strings.TrimPrefix(e.baseSegment, "/")), query, "view", "alert_number"), "", nil
	}
}

func (e alertExecutor) alertPath(resourceID string) string {
	return e.client.repoPath("/" + strings.TrimPrefix(e.baseSegment, "/") + "/" + trimResourceID(resourceID))
}

func (e alertExecutor) defaultAlertPayload(op string, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) > 0 {
		return raw, nil
	}
	var payload map[string]any
	switch e.capabilityID {
	case "dependabot_alerts":
		switch op {
		case "dismiss", "update":
			payload = map[string]any{
				"state":             "dismissed",
				"dismissed_reason":  "tolerable_risk",
				"dismissed_comment": "dismissed by gitdex",
			}
		case "reopen":
			payload = map[string]any{"state": "open"}
		}
	default:
		switch op {
		case "resolve", "dismiss", "update":
			payload = map[string]any{
				"state":              "resolved",
				"resolution":         "used_in_tests",
				"resolution_comment": "resolved by gitdex",
			}
		case "reopen":
			payload = map[string]any{"state": "open"}
		}
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("unsupported alert operation: %s", op)
	}
	return marshalRaw(payload)
}

func (e codeScanningExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
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

func (e codeScanningExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   strings.TrimSpace(req.ResourceID),
	}
	switch op {
	case "default_setup_update":
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "default_setup"}})
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/code-scanning/default-setup"), json.RawMessage(req.Payload), http.StatusOK, http.StatusAccepted)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), "", raw)
	case "delete_analysis":
		analysisID := strings.TrimSpace(firstNonEmpty(req.ResourceID, normalizeScopeValue(req.Scope, "analysis_id", "")))
		if analysisID == "" {
			return nil, fmt.Errorf("analysis id is required")
		}
		result.ResourceID = analysisID
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: analysisID, Query: map[string]string{"view": "analysis"}})
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/code-scanning/analyses/"+trimResourceID(analysisID)), nil, nil, http.StatusAccepted, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		alertID := strings.TrimSpace(firstNonEmpty(req.ResourceID, normalizeScopeValue(req.Scope, "alert_number", "")))
		if alertID == "" {
			return nil, fmt.Errorf("alert number is required")
		}
		result.ResourceID = alertID
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: alertID})
		payload, err := defaultCodeScanningPayload(op, req.Payload)
		if err != nil {
			return nil, err
		}
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath("/code-scanning/alerts/"+trimResourceID(alertID)), payload, http.StatusOK, http.StatusAccepted)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), alertID, raw)
	}
	return result, nil
}

func (e codeScanningExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	switch req.Mutation.Operation {
	case "delete_analysis":
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.Mutation.ResourceID, Query: map[string]string{"view": "analysis"}})
		if inspectMissingOK(err) {
			return &platform.AdminValidationResult{OK: true, Summary: "analysis deleted", ResourceID: req.Mutation.ResourceID}, nil
		}
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: req.Mutation.ResourceID}, nil
		}
		return &platform.AdminValidationResult{OK: false, Summary: "analysis still exists", ResourceID: req.Mutation.ResourceID, Snapshot: snap}, nil
	case "default_setup_update":
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "default_setup"}})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error()}, nil
		}
		expected := cloneRaw(req.Payload)
		if len(expected) == 0 && req.Mutation.After != nil {
			expected = sanitizeCodeScanningStateForMatch(req.Mutation.After.State, "default_setup")
		}
		matched, reason, matchErr := subsetMatches(snap.State, expected)
		if matchErr != nil {
			return nil, matchErr
		}
		return &platform.AdminValidationResult{OK: matched, Summary: summaryFromMatch(matched, reason, "code scanning default setup validated"), Snapshot: snap}, nil
	default:
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.Mutation.ResourceID})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: req.Mutation.ResourceID}, nil
		}
		expected := cloneRaw(req.Payload)
		if len(expected) == 0 && req.Mutation.After != nil {
			expected = sanitizeCodeScanningStateForMatch(req.Mutation.After.State, "alert")
		}
		matched, reason, matchErr := subsetMatches(snap.State, expected)
		if matchErr != nil {
			return nil, matchErr
		}
		return &platform.AdminValidationResult{OK: matched, Summary: summaryFromMatch(matched, reason, "code scanning alert validated"), ResourceID: req.Mutation.ResourceID, Snapshot: snap}, nil
	}
}

func (e codeScanningExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	switch req.Mutation.Operation {
	case "delete_analysis":
		return &platform.AdminRollbackResult{OK: false, Summary: "deleted analyses cannot be recreated automatically"}, nil
	case "default_setup_update":
		if req.Mutation.Before == nil {
			return &platform.AdminRollbackResult{OK: false, Summary: "default setup rollback requires previous snapshot"}, nil
		}
		restore := sanitizeCodeScanningStateForMatch(req.Mutation.Before.State, "default_setup")
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/code-scanning/default-setup"), restore, http.StatusOK, http.StatusAccepted)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "default setup restored", Snapshot: snapshot(e.CapabilityID(), "", raw)}, nil
	default:
		if req.Mutation.Before == nil {
			return &platform.AdminRollbackResult{OK: false, Summary: "alert rollback requires previous alert state"}, nil
		}
		restore := sanitizeCodeScanningStateForMatch(req.Mutation.Before.State, "alert")
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath("/code-scanning/alerts/"+trimResourceID(req.Mutation.ResourceID)), restore, http.StatusOK, http.StatusAccepted)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "code scanning alert restored", Snapshot: snapshot(e.CapabilityID(), req.Mutation.ResourceID, raw)}, nil
	}
}

func (e codeScanningExecutor) inspectPath(resourceID string, scope, query map[string]string) (string, string, error) {
	view := strings.ToLower(normalizeScopeValue(query, "view", normalizeScopeValue(scope, "view", "")))
	if view == "" {
		switch e.capabilityID {
		case "codeql_analysis", "codeql_setup", "code_scanning_default_setup":
			view = "default_setup"
		default:
			view = "alerts"
		}
	}
	alertID := strings.TrimSpace(firstNonEmpty(resourceID, normalizeScopeValue(scope, "alert_number", normalizeScopeValue(query, "alert_number", ""))))
	analysisID := strings.TrimSpace(firstNonEmpty(resourceID, normalizeScopeValue(scope, "analysis_id", normalizeScopeValue(query, "analysis_id", ""))))
	switch view {
	case "default_setup":
		return e.client.repoPath("/code-scanning/default-setup"), "default_setup", nil
	case "analyses":
		return appendQuery(e.client.repoPath("/code-scanning/analyses"), query, "view", "analysis_id", "alert_number"), "analyses", nil
	case "analysis":
		if analysisID == "" {
			return "", "", fmt.Errorf("analysis id is required")
		}
		return e.client.repoPath("/code-scanning/analyses/" + trimResourceID(analysisID)), analysisID, nil
	case "instances":
		if alertID == "" {
			return "", "", fmt.Errorf("alert number is required")
		}
		return e.client.repoPath("/code-scanning/alerts/" + trimResourceID(alertID) + "/instances"), alertID, nil
	case "autofix":
		if alertID == "" {
			return "", "", fmt.Errorf("alert number is required")
		}
		return e.client.repoPath("/code-scanning/alerts/" + trimResourceID(alertID) + "/autofix"), alertID, nil
	case "autofix_commits":
		if alertID == "" {
			return "", "", fmt.Errorf("alert number is required")
		}
		return e.client.repoPath("/code-scanning/alerts/" + trimResourceID(alertID) + "/autofix/commits"), alertID, nil
	case "alert":
		if alertID == "" {
			return "", "", fmt.Errorf("alert number is required")
		}
		return e.client.repoPath("/code-scanning/alerts/" + trimResourceID(alertID)), alertID, nil
	default:
		if alertID != "" {
			return e.client.repoPath("/code-scanning/alerts/" + trimResourceID(alertID)), alertID, nil
		}
		return appendQuery(e.client.repoPath("/code-scanning/alerts"), query, "view", "alert_number", "analysis_id"), "alerts", nil
	}
}

func defaultCodeScanningPayload(op string, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) > 0 {
		return raw, nil
	}
	switch op {
	case "dismiss", "update":
		return marshalRaw(map[string]any{
			"state":             "dismissed",
			"dismissed_reason":  "used in tests",
			"dismissed_comment": "dismissed by gitdex",
		})
	case "reopen":
		return marshalRaw(map[string]any{
			"state": "open",
		})
	default:
		return nil, fmt.Errorf("unsupported code scanning operation: %s", op)
	}
}

func sanitizeRepoSecuritySnapshot(raw json.RawMessage, capabilityID string) (json.RawMessage, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	securityState, _ := obj["security_and_analysis"].(map[string]any)
	if securityState == nil {
		securityState = map[string]any{}
	}
	if pvr, ok := obj["private_vulnerability_reporting"]; ok {
		if _, exists := securityState["private_vulnerability_reporting"]; !exists {
			securityState["private_vulnerability_reporting"] = pvr
		}
	}
	if capabilityID == "advanced_security" {
		return marshalRaw(map[string]any{"security_and_analysis": securityState})
	}
	if capabilityID == "dependabot_posture" {
		if entry, ok := securityState["dependabot_security_updates"]; ok {
			return marshalRaw(map[string]any{"dependabot_security_updates": entry})
		}
		return marshalRaw(map[string]any{})
	}
	if capabilityID == "dependabot_security_updates" {
		if entry, ok := securityState["dependabot_security_updates"]; ok {
			return marshalRaw(map[string]any{"dependabot_security_updates": entry})
		}
		return marshalRaw(map[string]any{})
	}
	if key := repoSecuritySettingKey(capabilityID); key != "" {
		if entry, ok := securityState[key]; ok {
			return marshalRaw(map[string]any{"security_and_analysis": map[string]any{key: entry}})
		}
		return marshalRaw(map[string]any{"security_and_analysis": map[string]any{}})
	}
	return marshalRaw(map[string]any{"security_and_analysis": securityState})
}

func sanitizeRepoSecurityRollback(raw json.RawMessage) (json.RawMessage, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	if _, ok := obj["security_and_analysis"]; ok {
		return marshalRaw(obj)
	}
	for _, key := range []string{
		"advanced_security",
		"dependabot_security_updates",
		"private_vulnerability_reporting",
		"secret_scanning",
		"secret_scanning_push_protection",
		"secret_scanning_non_provider_patterns",
	} {
		if value, ok := obj[key]; ok {
			return marshalRaw(map[string]any{"security_and_analysis": map[string]any{key: value}})
		}
	}
	return marshalRaw(obj)
}

func normalizeRepoSecurityMutationPayload(capabilityID, operation string, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) > 0 {
		obj, err := rawObject(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := obj["security_and_analysis"]; ok {
			return marshalRaw(obj)
		}
		if key := repoSecuritySettingKey(capabilityID); key != "" {
			if status, ok := obj["status"].(string); ok {
				obj = map[string]any{"status": strings.TrimSpace(status)}
			}
			return marshalRaw(map[string]any{"security_and_analysis": map[string]any{key: obj}})
		}
		return marshalRaw(obj)
	}
	key := repoSecuritySettingKey(capabilityID)
	if key == "" {
		return nil, fmt.Errorf("%s requires an explicit payload", capabilityID)
	}
	status := "enabled"
	if strings.EqualFold(operation, "disable") || strings.EqualFold(operation, "delete") {
		status = "disabled"
	}
	return marshalRaw(map[string]any{
		"security_and_analysis": map[string]any{
			key: map[string]any{"status": status},
		},
	})
}

func repoSecuritySettingKey(capabilityID string) string {
	switch strings.TrimSpace(capabilityID) {
	case "advanced_security":
		return "advanced_security"
	case "secret_scanning_settings":
		return "secret_scanning"
	case "dependabot", "grouped_security_updates", "dependabot_version_updates":
		return "dependabot_security_updates"
	case "private_vulnerability_reporting":
		return "private_vulnerability_reporting"
	case "protection_rules":
		return "secret_scanning_non_provider_patterns"
	case "push_protection":
		return "secret_scanning_push_protection"
	default:
		return ""
	}
}

func looksLikeRepoSecurityState(raw json.RawMessage) bool {
	obj, err := rawObject(raw)
	if err != nil {
		return false
	}
	_, has := obj["security_and_analysis"]
	return has
}

func repoSecurityEnabled(raw json.RawMessage) bool {
	obj, err := rawObject(raw)
	if err != nil {
		return false
	}
	switch value := obj["enabled"].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true") || strings.EqualFold(strings.TrimSpace(value), "enabled")
	}
	if nested, ok := obj["dependabot_security_updates"].(map[string]any); ok {
		switch status := nested["status"].(type) {
		case string:
			return strings.EqualFold(strings.TrimSpace(status), "enabled")
		}
	}
	if nested, ok := obj["repo_security"].(map[string]any); ok {
		if inner, ok := nested["dependabot_security_updates"].(map[string]any); ok {
			switch status := inner["status"].(type) {
			case string:
				return strings.EqualFold(strings.TrimSpace(status), "enabled")
			}
		}
	}
	if nested, ok := obj["automated_security_fixes"].(map[string]any); ok {
		switch enabled := nested["enabled"].(type) {
		case bool:
			return enabled
		case string:
			return strings.EqualFold(strings.TrimSpace(enabled), "true") || strings.EqualFold(strings.TrimSpace(enabled), "enabled")
		}
	}
	return false
}

func sanitizeAlertStateForMatch(raw json.RawMessage, capabilityID string) json.RawMessage {
	obj, err := rawObject(raw)
	if err != nil {
		return cloneRaw(raw)
	}
	switch capabilityID {
	case "dependabot_alerts":
		filtered := keepOnlyKeys(obj, "state", "dismissed_reason", "dismissed_comment")
		out, _ := marshalRaw(filtered)
		return out
	default:
		filtered := keepOnlyKeys(obj, "state", "resolution", "resolution_comment")
		out, _ := marshalRaw(filtered)
		return out
	}
}

func sanitizeCodeScanningStateForMatch(raw json.RawMessage, view string) json.RawMessage {
	obj, err := rawObject(raw)
	if err != nil {
		return cloneRaw(raw)
	}
	switch view {
	case "default_setup":
		filtered := keepOnlyKeys(obj, "state", "languages", "query_suite", "threat_model", "runner_type", "updated_at")
		out, _ := marshalRaw(filtered)
		return out
	default:
		filtered := keepOnlyKeys(obj, "state", "dismissed_reason", "dismissed_comment")
		out, _ := marshalRaw(filtered)
		return out
	}
}
