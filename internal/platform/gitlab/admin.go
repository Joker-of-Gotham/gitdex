package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func (c *Client) AdminExecutors() map[string]platform.AdminExecutor {
	return map[string]platform.AdminExecutor{
		"merge_requests": gitlabMergeRequestExecutor{client: c},
		"pipelines":      gitlabPipelinesExecutor{client: c},
		"environments":   gitlabEnvironmentsExecutor{client: c},
		"pages":          gitlabPagesExecutor{client: c},
		"security":       gitlabSecurityExecutor{client: c},
	}
}

func (c *Client) doRaw(ctx context.Context, method, path string, body any, statuses ...int) (json.RawMessage, error) {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if !matchesStatus(resp.StatusCode, statuses...) {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitlab api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func (c *Client) projectPath(segment string) string {
	segment = "/" + strings.TrimPrefix(strings.TrimSpace(segment), "/")
	return fmt.Sprintf("/projects/%s%s", url.PathEscape(c.projectID), segment)
}

func matchesStatus(actual int, allowed ...int) bool {
	if len(allowed) == 0 {
		return actual >= 200 && actual < 300
	}
	for _, status := range allowed {
		if actual == status {
			return true
		}
	}
	return false
}

func glSnapshot(capabilityID, resourceID string, raw json.RawMessage) *platform.AdminSnapshot {
	if raw == nil {
		return nil
	}
	return &platform.AdminSnapshot{
		CapabilityID: capabilityID,
		ResourceID:   strings.TrimSpace(resourceID),
		State:        platform.CloneRaw(raw),
	}
}

func glInspectMissingOK(err error) bool {
	return err != nil && strings.Contains(err.Error(), fmt.Sprintf("status %d", http.StatusNotFound))
}

func glValidateSubset(ctx context.Context, inspect func(context.Context, platform.AdminInspectRequest) (*platform.AdminSnapshot, error), req platform.AdminValidationRequest, fallbackResourceID string) (*platform.AdminValidationResult, error) {
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, fallbackResourceID))
	if req.Mutation != nil && strings.TrimSpace(req.Mutation.ResourceID) != "" {
		resourceID = strings.TrimSpace(req.Mutation.ResourceID)
	}
	snap, err := inspect(ctx, platform.AdminInspectRequest{
		ResourceID: resourceID,
		Scope:      platform.CloneStringMap(req.Scope),
	})
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: resourceID}, nil
	}
	expected := platform.CloneRaw(req.Payload)
	if len(expected) == 0 && req.Mutation != nil && req.Mutation.After != nil {
		expected = platform.CloneRaw(req.Mutation.After.State)
	}
	matched, reason, matchErr := platform.SubsetMatches(snap.State, expected)
	if matchErr != nil {
		return nil, matchErr
	}
	summary := "state validated"
	if !matched {
		summary = reason
	}
	return &platform.AdminValidationResult{
		OK:         matched,
		Summary:    summary,
		ResourceID: resourceID,
		Snapshot:   snap,
	}, nil
}

func glManualRollback(capabilityID, summary string, resourceID string, mutation *platform.AdminMutationResult) *platform.AdminRollbackResult {
	ledgerRef := ""
	if mutation != nil {
		ledgerRef = strings.TrimSpace(mutation.LedgerID)
	}
	return &platform.AdminRollbackResult{
		OK:      false,
		Summary: summary,
		Compensation: &platform.CompensationAction{
			Kind:        "manual_restore_required",
			Summary:     summary,
			OperatorRef: "gitlab:" + capabilityID,
			LedgerChain: compactRefs(resourceID, ledgerRef),
			Scope: map[string]string{
				"platform":    "gitlab",
				"capability":  capabilityID,
				"resource_id": strings.TrimSpace(resourceID),
			},
		},
	}
}

func compactRefs(values ...string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func glKeep(raw json.RawMessage, keys ...string) json.RawMessage {
	obj, err := platform.RawObject(raw)
	if err != nil {
		return nil
	}
	out := map[string]any{}
	allowed := map[string]struct{}{}
	for _, key := range keys {
		allowed[key] = struct{}{}
	}
	for key, value := range obj {
		if _, ok := allowed[key]; ok {
			out[key] = value
		}
	}
	data, _ := platform.MarshalRaw(out)
	return data
}

func glQuery(req platform.AdminInspectRequest, keys ...string) string {
	values := url.Values{}
	for _, key := range keys {
		value := strings.TrimSpace(platform.FirstNonEmpty(req.Query[key], req.Scope[key]))
		if value != "" {
			values.Set(key, value)
		}
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}

type gitlabMergeRequestExecutor struct{ client *Client }

func (e gitlabMergeRequestExecutor) CapabilityID() string { return "merge_requests" }

func (e gitlabMergeRequestExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	view := strings.ToLower(strings.TrimSpace(req.Query["view"]))
	iid := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["merge_request_iid"], req.Query["merge_request_iid"]))
	path := e.client.projectPath("/merge_requests")
	resourceID := iid
	switch {
	case iid != "" && view == "changes":
		path = e.client.projectPath("/merge_requests/" + url.PathEscape(iid) + "/changes")
	case iid != "" && view == "approvals":
		path = e.client.projectPath("/merge_requests/" + url.PathEscape(iid) + "/approvals")
	case iid != "":
		path = e.client.projectPath("/merge_requests/" + url.PathEscape(iid))
	default:
		path += glQuery(req, "state", "source_branch", "target_branch", "labels", "page", "per_page")
		resourceID = "merge_requests"
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return glSnapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e gitlabMergeRequestExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	iid := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["merge_request_iid"]))
	var before *platform.AdminSnapshot
	if iid != "" {
		before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: iid})
	}
	body, err := platform.RawObject(req.Payload)
	if err != nil {
		return nil, err
	}
	path := e.client.projectPath("/merge_requests")
	switch op {
	case "create":
	case "update":
		if iid == "" {
			return nil, fmt.Errorf("merge request iid is required")
		}
		path = e.client.projectPath("/merge_requests/" + url.PathEscape(iid))
	case "close":
		if iid == "" {
			return nil, fmt.Errorf("merge request iid is required")
		}
		path = e.client.projectPath("/merge_requests/" + url.PathEscape(iid))
		body = map[string]any{"state_event": "close"}
	case "reopen":
		if iid == "" {
			return nil, fmt.Errorf("merge request iid is required")
		}
		path = e.client.projectPath("/merge_requests/" + url.PathEscape(iid))
		body = map[string]any{"state_event": "reopen"}
	default:
		return nil, fmt.Errorf("unsupported merge request operation: %s", op)
	}
	method := http.MethodPost
	if op != "create" {
		method = http.MethodPut
	}
	raw, err := e.client.doRaw(ctx, method, path, body, http.StatusOK, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(iid, platform.ExtractResourceID(raw, "")))
	return &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   resourceID,
		Before:       before,
		After:        glSnapshot(e.CapabilityID(), resourceID, raw),
	}, nil
}

func (e gitlabMergeRequestExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return glValidateSubset(ctx, e.Inspect, req, "")
}

func (e gitlabMergeRequestExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	iid := strings.TrimSpace(req.Mutation.ResourceID)
	switch req.Mutation.Operation {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.projectPath("/merge_requests/"+url.PathEscape(iid)), map[string]any{"state_event": "close"}, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "merge request closed as rollback", Snapshot: glSnapshot(e.CapabilityID(), iid, raw)}, nil
	case "update", "close", "reopen":
		if req.Mutation.Before == nil {
			return glManualRollback(e.CapabilityID(), "rollback requires previous merge request snapshot", iid, req.Mutation), nil
		}
		before, err := platform.RawObject(req.Mutation.Before.State)
		if err != nil {
			return nil, err
		}
		payload := map[string]any{}
		for _, key := range []string{"title", "description", "target_branch"} {
			if value, ok := before[key]; ok {
				payload[key] = value
			}
		}
		switch strings.ToLower(strings.TrimSpace(platform.StringValue(before["state"]))) {
		case "closed", "merged":
			payload["state_event"] = "close"
		default:
			payload["state_event"] = "reopen"
		}
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.projectPath("/merge_requests/"+url.PathEscape(iid)), payload, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "merge request restored", Snapshot: glSnapshot(e.CapabilityID(), iid, raw)}, nil
	default:
		return glManualRollback(e.CapabilityID(), "unsupported rollback operation for merge requests", iid, req.Mutation), nil
	}
}

type gitlabPipelinesExecutor struct{ client *Client }

func (e gitlabPipelinesExecutor) CapabilityID() string { return "pipelines" }

func (e gitlabPipelinesExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	view := strings.ToLower(strings.TrimSpace(req.Query["view"]))
	pipelineID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["pipeline_id"], req.Query["pipeline_id"]))
	path := e.client.projectPath("/pipelines")
	resourceID := platform.FirstNonEmpty(pipelineID, "pipelines")
	switch {
	case pipelineID != "" && view == "jobs":
		path = e.client.projectPath("/pipelines/" + url.PathEscape(pipelineID) + "/jobs")
	case pipelineID != "":
		path = e.client.projectPath("/pipelines/" + url.PathEscape(pipelineID))
	default:
		path += glQuery(req, "ref", "status", "source", "page", "per_page")
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return glSnapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e gitlabPipelinesExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	pipelineID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["pipeline_id"]))
	body, err := platform.RawObject(req.Payload)
	if err != nil {
		return nil, err
	}
	path := e.client.projectPath("/pipelines")
	method := http.MethodPost
	switch op {
	case "create":
		ref := strings.TrimSpace(platform.StringValue(body["ref"]))
		if ref == "" {
			return nil, fmt.Errorf("pipeline create requires payload.ref")
		}
		delete(body, "ref")
		path = e.client.projectPath("/pipeline?ref=" + url.QueryEscape(ref))
	case "retry":
		if pipelineID == "" {
			return nil, fmt.Errorf("pipeline id is required")
		}
		path = e.client.projectPath("/pipelines/" + url.PathEscape(pipelineID) + "/retry")
		body = nil
	case "cancel":
		if pipelineID == "" {
			return nil, fmt.Errorf("pipeline id is required")
		}
		path = e.client.projectPath("/pipelines/" + url.PathEscape(pipelineID) + "/cancel")
		body = nil
	default:
		return nil, fmt.Errorf("unsupported pipeline operation: %s", op)
	}
	raw, err := e.client.doRaw(ctx, method, path, body, http.StatusOK, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(pipelineID, platform.ExtractResourceID(raw, "")))
	return &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   resourceID,
		After:        glSnapshot(e.CapabilityID(), resourceID, raw),
	}, nil
}

func (e gitlabPipelinesExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return glValidateSubset(ctx, e.Inspect, req, "")
}

func (e gitlabPipelinesExecutor) Rollback(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return glManualRollback("pipelines", "pipeline run-control requires compensation or manual restore", "", nil), nil
}

type gitlabEnvironmentsExecutor struct{ client *Client }

func (e gitlabEnvironmentsExecutor) CapabilityID() string { return "environments" }

func (e gitlabEnvironmentsExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	view := strings.ToLower(strings.TrimSpace(req.Query["view"]))
	envID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["environment_id"], req.Query["environment_id"]))
	path := e.client.projectPath("/environments")
	resourceID := platform.FirstNonEmpty(envID, "environments")
	switch {
	case envID != "" && view == "deployments":
		path = e.client.projectPath("/deployments?environment=" + url.QueryEscape(envID))
	case envID != "":
		path = e.client.projectPath("/environments/" + url.PathEscape(envID))
	default:
		path += glQuery(req, "name", "page", "per_page")
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return glSnapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e gitlabEnvironmentsExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	if strings.ToLower(strings.TrimSpace(req.Operation)) != "stop" {
		return nil, fmt.Errorf("unsupported environment operation: %s", req.Operation)
	}
	envID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["environment_id"]))
	if envID == "" {
		return nil, fmt.Errorf("environment id is required")
	}
	before, _ := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: envID})
	raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.projectPath("/environments/"+url.PathEscape(envID)+"/stop"), nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    "stop",
		ResourceID:   envID,
		Before:       before,
		After:        glSnapshot(e.CapabilityID(), envID, raw),
	}, nil
}

func (e gitlabEnvironmentsExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return glValidateSubset(ctx, e.Inspect, req, "")
}

func (e gitlabEnvironmentsExecutor) Rollback(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return glManualRollback("environments", "environment stop requires manual recovery or redeploy", "", nil), nil
}

type gitlabPagesExecutor struct{ client *Client }

func (e gitlabPagesExecutor) CapabilityID() string { return "pages" }

func (e gitlabPagesExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	view := strings.ToLower(strings.TrimSpace(req.Query["view"]))
	domain := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["domain"], req.Query["domain"]))
	path := e.client.projectPath("/pages")
	resourceID := platform.FirstNonEmpty(domain, "pages")
	switch {
	case domain != "" && (view == "domain" || view == ""):
		path = e.client.projectPath("/pages/domains/" + url.PathEscape(domain))
	case view == "domains":
		path = e.client.projectPath("/pages/domains")
		resourceID = "domains"
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return glSnapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e gitlabPagesExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	if strings.ToLower(strings.TrimSpace(req.Operation)) != "delete" {
		return nil, fmt.Errorf("unsupported pages operation: %s", req.Operation)
	}
	before, _ := e.Inspect(ctx, platform.AdminInspectRequest{})
	if _, err := e.client.doRaw(ctx, http.MethodDelete, e.client.projectPath("/pages"), nil, http.StatusNoContent, http.StatusOK); err != nil {
		return nil, err
	}
	return &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    "delete",
		ResourceID:   "pages",
		Before:       before,
	}, nil
}

func (e gitlabPagesExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation != nil && strings.EqualFold(req.Mutation.Operation, "delete") {
		_, err := e.Inspect(ctx, platform.AdminInspectRequest{})
		if glInspectMissingOK(err) {
			return &platform.AdminValidationResult{OK: true, Summary: "pages deployment removed", ResourceID: "pages"}, nil
		}
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: "pages"}, nil
		}
		return &platform.AdminValidationResult{OK: false, Summary: "pages deployment still exists", ResourceID: "pages"}, nil
	}
	return glValidateSubset(ctx, e.Inspect, req, "pages")
}

func (e gitlabPagesExecutor) Rollback(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return glManualRollback("pages", "restoring GitLab Pages requires a new deployment or manual restore", "pages", nil), nil
}

type gitlabSecurityExecutor struct{ client *Client }

func (e gitlabSecurityExecutor) CapabilityID() string { return "security" }

func (e gitlabSecurityExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	raw, err := e.client.doRaw(ctx, http.MethodGet, e.client.projectPath(""), nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return glSnapshot(e.CapabilityID(), "project_security", raw), nil
}

func (e gitlabSecurityExecutor) Mutate(context.Context, platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	return nil, fmt.Errorf("gitlab security surface is inspect-only")
}

func (e gitlabSecurityExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return glValidateSubset(ctx, e.Inspect, req, "project_security")
}

func (e gitlabSecurityExecutor) Rollback(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return &platform.AdminRollbackResult{OK: false, Summary: "gitlab security surface is inspect-only"}, nil
}
