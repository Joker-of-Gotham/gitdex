package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type copilotExecutor struct {
	client       *Client
	capabilityID string
}

func (e copilotExecutor) CapabilityID() string { return e.capabilityID }

func (e copilotExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
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

func (e copilotExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	switch op {
	case "update", "update_content_exclusions":
		before, _ := e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "content_exclusions"}})
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.orgPath("/copilot/content_exclusions"), json.RawMessage(req.Payload), http.StatusOK, http.StatusAccepted, http.StatusCreated, http.StatusNoContent)
		if err != nil {
			return nil, err
		}
		return &platform.AdminMutationResult{
			CapabilityID: e.CapabilityID(),
			Operation:    op,
			ResourceID:   "content_exclusions",
			Before:       before,
			After:        snapshot(e.CapabilityID(), "content_exclusions", raw),
		}, nil
	case "delete", "delete_content_exclusions":
		before, _ := e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "content_exclusions"}})
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.orgPath("/copilot/content_exclusions"), nil, nil, http.StatusNoContent, http.StatusAccepted, http.StatusOK); err != nil {
			return nil, err
		}
		return &platform.AdminMutationResult{
			CapabilityID: e.CapabilityID(),
			Operation:    op,
			ResourceID:   "content_exclusions",
			Before:       before,
		}, nil
	case "add_users", "remove_users", "add_teams", "remove_teams":
		before, _ := e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "seats"}})
		method, path, body, err := e.seatMutation(op, req.Payload)
		if err != nil {
			return nil, err
		}
		raw, err := e.client.doRaw(ctx, method, path, body, http.StatusCreated, http.StatusOK, http.StatusAccepted)
		if err != nil {
			return nil, err
		}
		return &platform.AdminMutationResult{
			CapabilityID: e.CapabilityID(),
			Operation:    op,
			ResourceID:   "seats",
			Before:       before,
			After:        snapshot(e.CapabilityID(), "seats", raw),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported Copilot admin operation: %s", op)
	}
}

func (e copilotExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	switch req.Mutation.Operation {
	case "add_users", "remove_users", "add_teams", "remove_teams":
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "seats"}})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: "seats"}, nil
		}
		ok, summary, validateErr := validateCopilotSeatMutation(snap.State, req.Payload, req.Mutation.Operation)
		if validateErr != nil {
			return nil, validateErr
		}
		return &platform.AdminValidationResult{OK: ok, Summary: summary, ResourceID: "seats", Snapshot: snap}, nil
	}
	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "content_exclusions"}})
	if strings.Contains(strings.ToLower(req.Mutation.Operation), "delete") {
		if inspectMissingOK(err) || (err == nil && len(strings.TrimSpace(string(snap.State))) == 0) {
			return &platform.AdminValidationResult{OK: true, Summary: "Copilot content exclusions cleared", ResourceID: "content_exclusions"}, nil
		}
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: "content_exclusions"}, nil
		}
		return &platform.AdminValidationResult{OK: false, Summary: "Copilot content exclusions still present", ResourceID: "content_exclusions", Snapshot: snap}, nil
	}
	expected := cloneRaw(req.Payload)
	if len(expected) == 0 && req.Mutation.After != nil {
		expected = cloneRaw(req.Mutation.After.State)
	}
	matched, reason, matchErr := subsetMatches(snap.State, expected)
	if matchErr != nil {
		return nil, matchErr
	}
	return &platform.AdminValidationResult{
		OK:         matched,
		Summary:    summaryFromMatch(matched, reason, "Copilot content exclusions validated"),
		ResourceID: "content_exclusions",
		Snapshot:   snap,
	}, nil
}

func (e copilotExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	switch req.Mutation.Operation {
	case "add_users":
		_, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "remove_users", Payload: cloneRaw(req.Payload)})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "Copilot users removed as rollback"}, nil
	case "remove_users":
		if req.Mutation.Before == nil {
			return &platform.AdminRollbackResult{OK: false, Summary: "remove_users rollback requires previous seat snapshot"}, nil
		}
		payload, err := rebuildCopilotSeatPayload(req.Mutation.Before.State, "selected_usernames")
		if err != nil {
			return nil, err
		}
		_, err = e.Mutate(ctx, platform.AdminMutationRequest{Operation: "add_users", Payload: payload})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "Copilot users restored"}, nil
	case "add_teams":
		_, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "remove_teams", Payload: cloneRaw(req.Payload)})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "Copilot teams removed as rollback"}, nil
	case "remove_teams":
		if req.Mutation.Before == nil {
			return &platform.AdminRollbackResult{OK: false, Summary: "remove_teams rollback requires previous seat snapshot"}, nil
		}
		payload, err := rebuildCopilotSeatPayload(req.Mutation.Before.State, "selected_team_slugs")
		if err != nil {
			return nil, err
		}
		_, err = e.Mutate(ctx, platform.AdminMutationRequest{Operation: "add_teams", Payload: payload})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "Copilot teams restored"}, nil
	}
	if req.Mutation.Before == nil {
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "delete_content_exclusions"}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "Copilot content exclusions removed as rollback"}, nil
	}
	raw := cloneRaw(req.Mutation.Before.State)
	if len(raw) == 0 {
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "delete_content_exclusions"}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "Copilot content exclusions removed as rollback"}, nil
	}
	result, err := e.Mutate(ctx, platform.AdminMutationRequest{
		Operation: "update_content_exclusions",
		Payload:   raw,
	})
	if err != nil {
		return nil, err
	}
	return &platform.AdminRollbackResult{OK: true, Summary: "Copilot content exclusions restored", Snapshot: result.After}, nil
}

func (e copilotExecutor) inspectPath(resourceID string, scope, query map[string]string) (string, string, error) {
	view := strings.ToLower(normalizeScopeValue(query, "view", normalizeScopeValue(scope, "view", "")))
	switch view {
	case "", "billing":
		return e.client.orgPath("/copilot/billing"), "billing", nil
	case "seats":
		return appendQuery(e.client.orgPath("/copilot/billing/seats"), query, "view"), "seats", nil
	case "seat_assignments":
		return appendQuery(e.client.orgPath("/copilot/billing/selected_users"), query, "view"), "seat_assignments", nil
	case "metrics":
		return appendQuery(e.client.orgPath("/copilot/metrics"), query, "view"), "metrics", nil
	case "content_exclusions":
		return e.client.orgPath("/copilot/content_exclusions"), "content_exclusions", nil
	default:
		return "", "", fmt.Errorf("unsupported Copilot view: %s", view)
	}
}

func (e copilotExecutor) seatMutation(op string, raw json.RawMessage) (string, string, any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return "", "", nil, err
	}
	switch op {
	case "add_users":
		return http.MethodPost, e.client.orgPath("/copilot/billing/selected_users"), obj, nil
	case "remove_users":
		return http.MethodDelete, e.client.orgPath("/copilot/billing/selected_users"), obj, nil
	case "add_teams":
		return http.MethodPost, e.client.orgPath("/copilot/billing/selected_teams"), obj, nil
	case "remove_teams":
		return http.MethodDelete, e.client.orgPath("/copilot/billing/selected_teams"), obj, nil
	default:
		return "", "", nil, fmt.Errorf("unsupported seat mutation operation: %s", op)
	}
}

func validateCopilotSeatMutation(actual json.RawMessage, expected json.RawMessage, operation string) (bool, string, error) {
	var current []map[string]any
	if err := json.Unmarshal(actual, &current); err != nil {
		return false, "", err
	}
	var desired map[string][]string
	if err := json.Unmarshal(expected, &desired); err != nil {
		return false, "", err
	}
	values := map[string]struct{}{}
	switch {
	case strings.Contains(operation, "user"):
		for _, item := range current {
			if assignee, ok := item["assignee"].(map[string]any); ok {
				values[strings.ToLower(strings.TrimSpace(stringValue(assignee["login"])))] = struct{}{}
			}
		}
		for _, user := range desired["selected_usernames"] {
			_, ok := values[strings.ToLower(strings.TrimSpace(user))]
			if strings.HasPrefix(operation, "add_") && !ok {
				return false, "expected Copilot user assignment missing", nil
			}
			if strings.HasPrefix(operation, "remove_") && ok {
				return false, "expected Copilot user removal missing", nil
			}
		}
	default:
		for _, item := range current {
			if team, ok := item["assigning_team"].(map[string]any); ok {
				values[strings.ToLower(strings.TrimSpace(stringValue(team["slug"])))] = struct{}{}
			}
		}
		for _, team := range desired["selected_team_slugs"] {
			_, ok := values[strings.ToLower(strings.TrimSpace(team))]
			if strings.HasPrefix(operation, "add_") && !ok {
				return false, "expected Copilot team assignment missing", nil
			}
			if strings.HasPrefix(operation, "remove_") && ok {
				return false, "expected Copilot team removal missing", nil
			}
		}
	}
	return true, "Copilot seat assignment validated", nil
}

func rebuildCopilotSeatPayload(raw json.RawMessage, key string) (json.RawMessage, error) {
	var current []map[string]any
	if err := json.Unmarshal(raw, &current); err != nil {
		return nil, err
	}
	values := make([]string, 0, len(current))
	for _, item := range current {
		switch key {
		case "selected_usernames":
			if assignee, ok := item["assignee"].(map[string]any); ok {
				if login := strings.TrimSpace(stringValue(assignee["login"])); login != "" {
					values = append(values, login)
				}
			}
		case "selected_team_slugs":
			if team, ok := item["assigning_team"].(map[string]any); ok {
				if slug := strings.TrimSpace(stringValue(team["slug"])); slug != "" {
					values = append(values, slug)
				}
			}
		}
	}
	return marshalRaw(map[string]any{key: values})
}
