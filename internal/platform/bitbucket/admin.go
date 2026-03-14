package bitbucket

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
		"pull_requests":        bitbucketPullRequestExecutor{client: c},
		"pipelines":            bitbucketPipelinesExecutor{client: c},
		"deployments":          bitbucketDeploymentsExecutor{client: c},
		"branch_restrictions":  bitbucketBranchRestrictionExecutor{client: c},
		"webhooks":             bitbucketWebhookExecutor{client: c},
		"repository_variables": bitbucketRepositoryVariableExecutor{client: c},
	}
}

func (c *Client) doRaw(ctx context.Context, method, path string, body any, statuses ...int) (json.RawMessage, error) {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if !bbMatchesStatus(resp.StatusCode, statuses...) {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bitbucket api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func (c *Client) repoPath(segment string) string {
	segment = "/" + strings.TrimPrefix(strings.TrimSpace(segment), "/")
	return fmt.Sprintf("/repositories/%s/%s%s", url.PathEscape(c.workspace), url.PathEscape(c.repo), segment)
}

func bbMatchesStatus(actual int, allowed ...int) bool {
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

func bbSnapshot(capabilityID, resourceID string, raw json.RawMessage) *platform.AdminSnapshot {
	if raw == nil {
		return nil
	}
	return &platform.AdminSnapshot{
		CapabilityID: capabilityID,
		ResourceID:   strings.TrimSpace(resourceID),
		State:        platform.CloneRaw(raw),
	}
}

func bbValidateSubset(ctx context.Context, inspect func(context.Context, platform.AdminInspectRequest) (*platform.AdminSnapshot, error), req platform.AdminValidationRequest, fallbackResourceID string) (*platform.AdminValidationResult, error) {
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

func bbManualRollback(capabilityID, summary, resourceID string, mutation *platform.AdminMutationResult) *platform.AdminRollbackResult {
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
			OperatorRef: "bitbucket:" + capabilityID,
			LedgerChain: compactRefs(resourceID, ledgerRef),
			Scope: map[string]string{
				"platform":    "bitbucket",
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

func bbQuery(req platform.AdminInspectRequest, keys ...string) string {
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

type bitbucketPullRequestExecutor struct{ client *Client }

func (e bitbucketPullRequestExecutor) CapabilityID() string { return "pull_requests" }

func (e bitbucketPullRequestExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	view := strings.ToLower(strings.TrimSpace(req.Query["view"]))
	pullID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["pull_request_id"], req.Query["pull_request_id"]))
	path := e.client.repoPath("/pullrequests")
	resourceID := platform.FirstNonEmpty(pullID, "pull_requests")
	switch {
	case pullID != "" && view == "activity":
		path = e.client.repoPath("/pullrequests/" + url.PathEscape(pullID) + "/activity")
	case pullID != "" && view == "diffstat":
		path = e.client.repoPath("/pullrequests/" + url.PathEscape(pullID) + "/diffstat")
	case pullID != "":
		path = e.client.repoPath("/pullrequests/" + url.PathEscape(pullID))
	default:
		path += bbQuery(req, "state", "page", "pagelen")
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return bbSnapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e bitbucketPullRequestExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	pullID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["pull_request_id"]))
	body, err := platform.RawObject(req.Payload)
	if err != nil {
		return nil, err
	}
	path := e.client.repoPath("/pullrequests")
	method := http.MethodPost
	var before *platform.AdminSnapshot
	if pullID != "" {
		before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: pullID})
	}
	switch op {
	case "create":
	case "update":
		if pullID == "" {
			return nil, fmt.Errorf("pull request id is required")
		}
		method = http.MethodPut
		path = e.client.repoPath("/pullrequests/" + url.PathEscape(pullID))
	case "decline":
		if pullID == "" {
			return nil, fmt.Errorf("pull request id is required")
		}
		path = e.client.repoPath("/pullrequests/" + url.PathEscape(pullID) + "/decline")
		body = nil
	case "reopen":
		if pullID == "" {
			return nil, fmt.Errorf("pull request id is required")
		}
		path = e.client.repoPath("/pullrequests/" + url.PathEscape(pullID) + "/open")
		body = nil
	default:
		return nil, fmt.Errorf("unsupported pull request operation: %s", op)
	}
	raw, err := e.client.doRaw(ctx, method, path, body, http.StatusOK, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(pullID, platform.ExtractResourceID(raw, "")))
	return &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   resourceID,
		Before:       before,
		After:        bbSnapshot(e.CapabilityID(), resourceID, raw),
	}, nil
}

func (e bitbucketPullRequestExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return bbValidateSubset(ctx, e.Inspect, req, "")
}

func (e bitbucketPullRequestExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	pullID := strings.TrimSpace(req.Mutation.ResourceID)
	switch req.Mutation.Operation {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/pullrequests/"+url.PathEscape(pullID)+"/decline"), nil, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "pull request declined as rollback", Snapshot: bbSnapshot(e.CapabilityID(), pullID, raw)}, nil
	case "update", "decline", "reopen":
		if req.Mutation.Before == nil {
			return bbManualRollback(e.CapabilityID(), "rollback requires previous pull request snapshot", pullID, req.Mutation), nil
		}
		before, err := platform.RawObject(req.Mutation.Before.State)
		if err != nil {
			return nil, err
		}
		path := e.client.repoPath("/pullrequests/" + url.PathEscape(pullID))
		payload := map[string]any{}
		for _, key := range []string{"title", "description"} {
			if value, ok := before[key]; ok {
				payload[key] = value
			}
		}
		state := strings.ToUpper(strings.TrimSpace(platform.StringValue(before["state"])))
		if state == "DECLINED" {
			raw, err := e.client.doRaw(ctx, http.MethodPost, path+"/decline", nil, http.StatusOK)
			if err != nil {
				return nil, err
			}
			return &platform.AdminRollbackResult{OK: true, Summary: "pull request restored to declined state", Snapshot: bbSnapshot(e.CapabilityID(), pullID, raw)}, nil
		}
		if state == "OPEN" {
			raw, err := e.client.doRaw(ctx, http.MethodPost, path+"/open", nil, http.StatusOK)
			if err != nil {
				return nil, err
			}
			return &platform.AdminRollbackResult{OK: true, Summary: "pull request reopened", Snapshot: bbSnapshot(e.CapabilityID(), pullID, raw)}, nil
		}
		raw, err := e.client.doRaw(ctx, http.MethodPut, path, payload, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "pull request restored", Snapshot: bbSnapshot(e.CapabilityID(), pullID, raw)}, nil
	default:
		return bbManualRollback(e.CapabilityID(), "unsupported rollback operation for pull requests", pullID, req.Mutation), nil
	}
}

type bitbucketPipelinesExecutor struct{ client *Client }

func (e bitbucketPipelinesExecutor) CapabilityID() string { return "pipelines" }

func (e bitbucketPipelinesExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	pipelineID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["pipeline_uuid"], req.Query["pipeline_uuid"]))
	path := e.client.repoPath("/pipelines/")
	resourceID := platform.FirstNonEmpty(pipelineID, "pipelines")
	if pipelineID != "" {
		path = e.client.repoPath("/pipelines/" + url.PathEscape(pipelineID))
	} else {
		path += bbQuery(req, "page", "pagelen", "target.ref_name")
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return bbSnapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e bitbucketPipelinesExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	if strings.ToLower(strings.TrimSpace(req.Operation)) != "create" {
		return nil, fmt.Errorf("unsupported pipeline operation: %s", req.Operation)
	}
	body, err := platform.RawObject(req.Payload)
	if err != nil {
		return nil, err
	}
	raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/pipelines/"), body, http.StatusOK, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	resourceID := strings.TrimSpace(platform.ExtractResourceID(raw, "pipeline"))
	return &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    "create",
		ResourceID:   resourceID,
		After:        bbSnapshot(e.CapabilityID(), resourceID, raw),
	}, nil
}

func (e bitbucketPipelinesExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return bbValidateSubset(ctx, e.Inspect, req, "")
}

func (e bitbucketPipelinesExecutor) Rollback(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return bbManualRollback("pipelines", "bitbucket pipeline execution requires compensation or manual restore", "", nil), nil
}

type bitbucketDeploymentsExecutor struct{ client *Client }

func (e bitbucketDeploymentsExecutor) CapabilityID() string { return "deployments" }

func (e bitbucketDeploymentsExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path := e.client.repoPath("/deployments_config/environments")
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return bbSnapshot(e.CapabilityID(), "deployments", raw), nil
}

func (e bitbucketDeploymentsExecutor) Mutate(context.Context, platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	return nil, fmt.Errorf("bitbucket deployments surface is inspect-only in the current executor")
}

func (e bitbucketDeploymentsExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return bbValidateSubset(ctx, e.Inspect, req, "deployments")
}

func (e bitbucketDeploymentsExecutor) Rollback(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return &platform.AdminRollbackResult{OK: false, Summary: "bitbucket deployments surface is inspect-only"}, nil
}

type bitbucketBranchRestrictionExecutor struct{ client *Client }

func (e bitbucketBranchRestrictionExecutor) CapabilityID() string { return "branch_restrictions" }

func (e bitbucketBranchRestrictionExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["restriction_id"]))
	path := e.client.repoPath("/branch-restrictions")
	if resourceID != "" {
		path += "/" + url.PathEscape(resourceID)
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return bbSnapshot(e.CapabilityID(), platform.FirstNonEmpty(resourceID, "branch_restrictions"), raw), nil
}

func (e bitbucketBranchRestrictionExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["restriction_id"]))
	body, err := platform.RawObject(req.Payload)
	if err != nil {
		return nil, err
	}
	path := e.client.repoPath("/branch-restrictions")
	method := http.MethodPost
	var before *platform.AdminSnapshot
	if resourceID != "" {
		before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: resourceID})
	}
	switch op {
	case "create":
	case "update":
		if resourceID == "" {
			return nil, fmt.Errorf("branch restriction id is required")
		}
		method = http.MethodPut
		path += "/" + url.PathEscape(resourceID)
	case "delete":
		if resourceID == "" {
			return nil, fmt.Errorf("branch restriction id is required")
		}
		method = http.MethodDelete
		path += "/" + url.PathEscape(resourceID)
		body = nil
	default:
		return nil, fmt.Errorf("unsupported branch restriction operation: %s", op)
	}
	raw, err := e.client.doRaw(ctx, method, path, body, http.StatusOK, http.StatusCreated, http.StatusNoContent)
	if err != nil {
		return nil, err
	}
	nextID := strings.TrimSpace(platform.FirstNonEmpty(resourceID, platform.ExtractResourceID(raw, "")))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   nextID,
		Before:       before,
	}
	if op != "delete" {
		result.After = bbSnapshot(e.CapabilityID(), nextID, raw)
	}
	return result, nil
}

func (e bitbucketBranchRestrictionExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return bbValidateSubset(ctx, e.Inspect, req, "")
}

func (e bitbucketBranchRestrictionExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return bbRollbackCRUD(ctx, e.client, e.CapabilityID(), "/branch-restrictions", req)
}

type bitbucketWebhookExecutor struct{ client *Client }

func (e bitbucketWebhookExecutor) CapabilityID() string { return "webhooks" }

func (e bitbucketWebhookExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["webhook_uuid"]))
	path := e.client.repoPath("/hooks")
	if resourceID != "" {
		path += "/" + url.PathEscape(resourceID)
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return bbSnapshot(e.CapabilityID(), platform.FirstNonEmpty(resourceID, "webhooks"), raw), nil
}

func (e bitbucketWebhookExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	return bbMutateCRUD(ctx, e.client, e.CapabilityID(), "/hooks", "webhook_uuid", req)
}

func (e bitbucketWebhookExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return bbValidateSubset(ctx, e.Inspect, req, "")
}

func (e bitbucketWebhookExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return bbRollbackCRUD(ctx, e.client, e.CapabilityID(), "/hooks", req)
}

type bitbucketRepositoryVariableExecutor struct{ client *Client }

func (e bitbucketRepositoryVariableExecutor) CapabilityID() string { return "repository_variables" }

func (e bitbucketRepositoryVariableExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope["variable_uuid"]))
	path := e.client.repoPath("/pipelines_config/variables/")
	if resourceID != "" {
		path += url.PathEscape(resourceID)
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return bbSnapshot(e.CapabilityID(), platform.FirstNonEmpty(resourceID, "repository_variables"), raw), nil
}

func (e bitbucketRepositoryVariableExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	return bbMutateCRUD(ctx, e.client, e.CapabilityID(), "/pipelines_config/variables/", "variable_uuid", req)
}

func (e bitbucketRepositoryVariableExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return bbValidateSubset(ctx, e.Inspect, req, "")
}

func (e bitbucketRepositoryVariableExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return bbRollbackCRUD(ctx, e.client, e.CapabilityID(), "/pipelines_config/variables/", req)
}

func bbMutateCRUD(ctx context.Context, client *Client, capabilityID, basePath, scopeKey string, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	resourceID := strings.TrimSpace(platform.FirstNonEmpty(req.ResourceID, req.Scope[scopeKey]))
	body, err := platform.RawObject(req.Payload)
	if err != nil {
		return nil, err
	}
	path := client.repoPath(basePath)
	method := http.MethodPost
	var before *platform.AdminSnapshot
	if resourceID != "" {
		before = bbSnapshot(capabilityID, resourceID, nil)
	}
	switch op {
	case "create":
	case "update":
		if op == "update" {
			if resourceID == "" {
				return nil, fmt.Errorf("%s id is required", scopeKey)
			}
			before, _ = bbInspectByPath(ctx, client, capabilityID, basePath, resourceID)
			method = http.MethodPut
			path = client.repoPath(strings.TrimRight(basePath, "/") + "/" + url.PathEscape(resourceID))
		}
	case "delete":
		if resourceID == "" {
			return nil, fmt.Errorf("%s id is required", scopeKey)
		}
		before, _ = bbInspectByPath(ctx, client, capabilityID, basePath, resourceID)
		method = http.MethodDelete
		path = client.repoPath(strings.TrimRight(basePath, "/") + "/" + url.PathEscape(resourceID))
		body = nil
	default:
		return nil, fmt.Errorf("unsupported operation: %s", op)
	}
	raw, err := client.doRaw(ctx, method, path, body, http.StatusOK, http.StatusCreated, http.StatusNoContent)
	if err != nil {
		return nil, err
	}
	nextID := strings.TrimSpace(platform.FirstNonEmpty(resourceID, platform.ExtractResourceID(raw, "")))
	result := &platform.AdminMutationResult{
		CapabilityID: capabilityID,
		Operation:    op,
		ResourceID:   nextID,
		Before:       before,
	}
	if op != "delete" {
		result.After = bbSnapshot(capabilityID, nextID, raw)
	}
	return result, nil
}

func bbRollbackCRUD(ctx context.Context, client *Client, capabilityID, basePath string, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	resourceID := strings.TrimSpace(req.Mutation.ResourceID)
	switch req.Mutation.Operation {
	case "create":
		if _, err := client.doRaw(ctx, http.MethodDelete, client.repoPath(strings.TrimRight(basePath, "/")+"/"+url.PathEscape(resourceID)), nil, http.StatusNoContent, http.StatusOK); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "created resource deleted as rollback"}, nil
	case "update", "delete":
		if req.Mutation.Before == nil {
			return bbManualRollback(capabilityID, "rollback requires previous snapshot", resourceID, req.Mutation), nil
		}
		before, err := platform.RawObject(req.Mutation.Before.State)
		if err != nil {
			return nil, err
		}
		path := client.repoPath(strings.TrimRight(basePath, "/"))
		method := http.MethodPost
		if req.Mutation.Operation == "update" {
			method = http.MethodPut
			path += "/" + url.PathEscape(resourceID)
		}
		if req.Mutation.Operation == "delete" {
			method = http.MethodPost
		}
		raw, err := client.doRaw(ctx, method, path, before, http.StatusOK, http.StatusCreated)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "resource restored", Snapshot: bbSnapshot(capabilityID, platform.FirstNonEmpty(resourceID, platform.ExtractResourceID(raw, "")), raw)}, nil
	default:
		return bbManualRollback(capabilityID, "unsupported rollback operation", resourceID, req.Mutation), nil
	}
}

func bbInspectByPath(ctx context.Context, client *Client, capabilityID, basePath, resourceID string) (*platform.AdminSnapshot, error) {
	raw, err := client.doRaw(ctx, http.MethodGet, client.repoPath(strings.TrimRight(basePath, "/")+"/"+url.PathEscape(resourceID)), nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return bbSnapshot(capabilityID, resourceID, raw), nil
}
