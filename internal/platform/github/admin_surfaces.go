package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type rulesetExecutor struct {
	client       *Client
	capabilityID string
}
type webhookExecutor struct{ client *Client }
type pagesExecutor struct{ client *Client }
type deploymentExecutor struct{ client *Client }
type environmentExecutor struct{ client *Client }

func (e rulesetExecutor) CapabilityID() string {
	if strings.TrimSpace(e.capabilityID) != "" {
		return strings.TrimSpace(e.capabilityID)
	}
	return "rulesets"
}
func (e webhookExecutor) CapabilityID() string { return "webhooks" }
func (e pagesExecutor) CapabilityID() string   { return "pages" }
func (e deploymentExecutor) CapabilityID() string {
	return "deployment"
}
func (e environmentExecutor) CapabilityID() string { return "environments" }

func (e rulesetExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path := e.client.repoPath("/rulesets")
	view := strings.ToLower(normalizeScopeValue(req.Query, "view", ""))
	switch view {
	case "branch", "branch_rules":
		branch := normalizeScopeValue(req.Query, "branch", strings.TrimSpace(req.ResourceID))
		if branch == "" {
			return nil, fmt.Errorf("branch name is required")
		}
		path = e.client.repoPath("/rules/branches/" + trimResourceID(branch))
	case "rule_suites":
		path = appendQuery(e.client.repoPath("/rulesets/rule-suites"), req.Query, "view")
	case "rule_suite":
		ruleSuiteID := normalizeScopeValue(req.Query, "rule_suite_id", strings.TrimSpace(req.ResourceID))
		if ruleSuiteID == "" {
			return nil, fmt.Errorf("rule suite id is required")
		}
		path = e.client.repoPath("/rulesets/rule-suites/" + trimResourceID(ruleSuiteID))
	default:
		if strings.TrimSpace(req.ResourceID) != "" {
			path = e.client.repoPath("/rulesets/" + trimResourceID(req.ResourceID))
		} else {
			path = appendQuery(path, req.Query, "view")
		}
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
}

func (e rulesetExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{CapabilityID: e.CapabilityID(), Operation: op, ResourceID: strings.TrimSpace(req.ResourceID)}
	if result.ResourceID != "" {
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: result.ResourceID})
	}

	switch op {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/rulesets"), json.RawMessage(req.Payload), http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractResourceID(raw, result.ResourceID)
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "update":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("resource id is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/rulesets/"+trimResourceID(result.ResourceID)), json.RawMessage(req.Payload), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "delete":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("resource id is required")
		}
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/rulesets/"+trimResourceID(result.ResourceID)), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported ruleset operation: %s", op)
	}
	return result, nil
}

func (e rulesetExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return validateByInspect(ctx, e, req, "ruleset validated")
}

func (e rulesetExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return rollbackBySnapshot(ctx, e, req, sanitizeRulesetPayload, "ruleset")
}

func (e webhookExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path := e.client.repoPath("/hooks")
	if strings.TrimSpace(req.ResourceID) != "" {
		path = e.client.repoPath("/hooks/" + trimResourceID(req.ResourceID))
		if strings.EqualFold(normalizeScopeValue(req.Query, "include_deliveries", ""), "true") {
			path += "/deliveries"
		}
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
}

func (e webhookExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{CapabilityID: e.CapabilityID(), Operation: op, ResourceID: strings.TrimSpace(req.ResourceID)}
	if result.ResourceID != "" {
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: result.ResourceID})
	}
	switch op {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/hooks"), json.RawMessage(req.Payload), http.StatusCreated)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractResourceID(raw, result.ResourceID)
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "update":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("resource id is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath("/hooks/"+trimResourceID(result.ResourceID)), json.RawMessage(req.Payload), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "ping":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("resource id is required")
		}
		if err := e.client.doJSON(ctx, http.MethodPost, e.client.repoPath("/hooks/"+trimResourceID(result.ResourceID)+"/pings"), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
		after, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: result.ResourceID})
		if err == nil {
			result.After = after
		}
	case "delete":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("resource id is required")
		}
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/hooks/"+trimResourceID(result.ResourceID)), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported webhook operation: %s", op)
	}
	return result, nil
}

func (e webhookExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return validateByInspect(ctx, e, req, "webhook validated")
}

func (e webhookExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return rollbackBySnapshot(ctx, e, req, sanitizeWebhookPayload, "webhook")
}

func (e pagesExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path := e.client.repoPath("/pages")
	switch strings.ToLower(normalizeScopeValue(req.Query, "view", "")) {
	case "builds", "build_history":
		path = appendQuery(path+"/builds", req.Query, "view")
	case "build", "build_detail":
		buildID := normalizeScopeValue(req.Query, "build_id", strings.TrimSpace(req.ResourceID))
		if buildID == "" {
			return nil, fmt.Errorf("build id is required")
		}
		path = path + "/builds/" + trimResourceID(buildID)
	case "latest_build":
		path += "/builds/latest"
	case "health":
		path += "/health"
	case "domain", "dns":
		raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
	default:
		path = appendQuery(path, req.Query, "view")
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
}

func (e pagesExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{CapabilityID: e.CapabilityID(), Operation: op}
	result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{})

	switch op {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/pages"), json.RawMessage(req.Payload), http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), "", raw)
	case "update":
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/pages"), json.RawMessage(req.Payload), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), "", raw)
	case "build":
		fallthrough
	case "rebuild":
		if err := e.client.doJSON(ctx, http.MethodPost, e.client.repoPath("/pages/builds"), nil, nil, http.StatusCreated, http.StatusAccepted); err != nil {
			return nil, err
		}
		after, err := e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "latest_build"}})
		if err == nil {
			result.After = after
		}
	case "delete":
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/pages"), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported pages operation: %s", op)
	}
	return result, nil
}

func (e pagesExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	if req.Mutation.Operation == "build" || req.Mutation.Operation == "rebuild" {
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{Query: map[string]string{"view": "latest_build"}})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error()}, nil
		}
		return &platform.AdminValidationResult{
			OK:       true,
			Summary:  "pages build triggered",
			Snapshot: snap,
		}, nil
	}
	validation, err := validateByInspect(ctx, e, req, "pages validated")
	if err != nil || validation == nil || !validation.OK {
		return validation, err
	}
	cname := desiredPagesCNAME(req)
	if cname == "" {
		return validation, nil
	}
	if ok, summary := probePagesDNS(ctx, cname); !ok {
		validation.OK = false
		validation.Summary = summary
		return validation, nil
	}
	validation.Summary = validation.Summary + " | DNS validated"
	if validation.Snapshot != nil {
		if ok, summary := validatePagesReadiness(validation.Snapshot.State); !ok {
			validation.OK = false
			validation.Summary = summary
			return validation, nil
		} else if strings.TrimSpace(summary) != "" {
			validation.Summary = validation.Summary + " | " + summary
		}
	}
	return validation, nil
}

func (e pagesExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return rollbackBySnapshot(ctx, e, req, sanitizePagesPayload, "pages")
}

func (e deploymentExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path := e.client.repoPath("/deployments")
	if strings.TrimSpace(req.ResourceID) != "" {
		path += "/" + trimResourceID(req.ResourceID)
		if strings.EqualFold(normalizeScopeValue(req.Query, "view", ""), "statuses") {
			path += "/statuses"
		}
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
}

func (e deploymentExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{CapabilityID: e.CapabilityID(), Operation: op, ResourceID: strings.TrimSpace(req.ResourceID)}
	if result.ResourceID != "" {
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: result.ResourceID})
	}
	switch op {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/deployments"), json.RawMessage(req.Payload), http.StatusCreated, http.StatusAccepted)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractResourceID(raw, result.ResourceID)
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "status":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("deployment id is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/deployments/"+trimResourceID(result.ResourceID)+"/statuses"), json.RawMessage(req.Payload), http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "delete":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("deployment id is required")
		}
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/deployments/"+trimResourceID(result.ResourceID)), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported deployment operation: %s", op)
	}
	return result, nil
}

func (e deploymentExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return validateByInspect(ctx, e, req, "deployment validated")
}

func (e deploymentExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	if req.Mutation.Before == nil && req.Mutation.After != nil {
		_, _ = e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "status",
			ResourceID: req.Mutation.ResourceID,
			Payload:    json.RawMessage(`{"state":"inactive"}`),
		})
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "delete",
			ResourceID: req.Mutation.ResourceID,
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "deployment marked inactive and deleted"}, nil
	}
	return rollbackBySnapshot(ctx, e, req, sanitizeDeploymentPayload, "deployment")
}

func (e environmentExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path := e.client.repoPath("/environments")
	if strings.TrimSpace(req.ResourceID) != "" {
		path += "/" + trimResourceID(req.ResourceID)
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
}

func (e environmentExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		resourceID = extractEnvironmentName(req.Payload)
	}
	if resourceID == "" {
		return nil, fmt.Errorf("environment name is required")
	}
	result := &platform.AdminMutationResult{CapabilityID: e.CapabilityID(), Operation: op, ResourceID: resourceID}
	result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: resourceID})

	switch op {
	case "create", "update":
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/environments/"+trimResourceID(resourceID)), json.RawMessage(req.Payload), http.StatusOK, http.StatusCreated)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), resourceID, raw)
	case "delete":
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/environments/"+trimResourceID(resourceID)), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported environment operation: %s", op)
	}
	return result, nil
}

func (e environmentExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return validateByInspect(ctx, e, req, "environment validated")
}

func (e environmentExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return rollbackBySnapshot(ctx, e, req, sanitizeEnvironmentPayload, "environment")
}

func validateByInspect(ctx context.Context, executor platform.AdminExecutor, req platform.AdminValidationRequest, success string) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		resourceID = req.Mutation.ResourceID
	}
	snap, err := executor.Inspect(ctx, platform.AdminInspectRequest{
		ResourceID: resourceID,
		Scope:      req.Scope,
	})
	if strings.EqualFold(req.Mutation.Operation, "delete") {
		if inspectMissingOK(err) {
			return &platform.AdminValidationResult{OK: true, Summary: "resource deleted", ResourceID: resourceID}, nil
		}
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: resourceID}, nil
		}
		return &platform.AdminValidationResult{OK: false, Summary: "resource still exists", ResourceID: resourceID, Snapshot: snap}, nil
	}
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: resourceID}, nil
	}
	matched, reason, matchErr := subsetMatches(snap.State, req.Payload)
	if matchErr != nil {
		return nil, matchErr
	}
	return &platform.AdminValidationResult{
		OK:         matched,
		Summary:    summaryFromMatch(matched, reason, success),
		ResourceID: resourceID,
		Snapshot:   snap,
	}, nil
}

func rollbackBySnapshot(
	ctx context.Context,
	executor platform.AdminExecutor,
	req platform.AdminRollbackRequest,
	sanitize func(json.RawMessage) (map[string]any, error),
	label string,
) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	resourceID := req.Mutation.ResourceID

	if req.Mutation.Before == nil {
		if _, err := executor.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "delete",
			ResourceID: resourceID,
			Scope:      req.Scope,
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: label + " deleted as rollback"}, nil
	}

	restore, err := sanitize(req.Mutation.Before.State)
	if err != nil {
		return nil, err
	}
	restoreRaw, err := marshalRaw(restore)
	if err != nil {
		return nil, err
	}

	op := "update"
	if req.Mutation.Operation == "delete" {
		op = "create"
	}
	if _, err := executor.Mutate(ctx, platform.AdminMutationRequest{
		Operation:  op,
		ResourceID: resourceID,
		Scope:      req.Scope,
		Payload:    restoreRaw,
	}); err != nil {
		return nil, err
	}
	current, err := executor.Inspect(ctx, platform.AdminInspectRequest{
		ResourceID: resourceID,
		Scope:      req.Scope,
	})
	if err != nil {
		return nil, err
	}
	return &platform.AdminRollbackResult{OK: true, Summary: label + " restored", Snapshot: current}, nil
}

func sanitizeRulesetPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	deleteKeys(obj, "id", "node_id", "_links", "source", "source_type", "created_at", "updated_at", "current_user_can_bypass", "url")
	return keepOnlyKeys(obj, "name", "target", "enforcement", "bypass_actors", "conditions", "rules"), nil
}

func sanitizeWebhookPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	return keepOnlyKeys(obj, "name", "active", "events", "config"), nil
}

func sanitizePagesPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	deleteKeys(obj, "html_url", "status", "protected_domain_state", "pending_domain_unverified_at", "subdomain", "url", "public")
	return keepOnlyKeys(obj, "source", "build_type", "cname", "https_enforced"), nil
}

func desiredPagesCNAME(req platform.AdminValidationRequest) string {
	payload := req.Payload
	if len(payload) == 0 && req.Mutation != nil && req.Mutation.After != nil {
		payload = req.Mutation.After.State
	}
	obj, err := rawObject(payload)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stringValue(obj["cname"]))
}

func probePagesDNS(ctx context.Context, cname string) (bool, string) {
	cname = strings.TrimSpace(cname)
	if cname == "" {
		return true, "DNS validation skipped"
	}
	if _, err := net.DefaultResolver.LookupCNAME(ctx, cname); err == nil {
		return true, "custom domain DNS resolves"
	}
	ips, err := net.DefaultResolver.LookupHost(ctx, cname)
	if err == nil && len(ips) > 0 {
		return true, "custom domain DNS resolves"
	}
	if err != nil {
		return false, "custom domain DNS unresolved: " + err.Error()
	}
	return false, "custom domain DNS unresolved"
}

func validatePagesReadiness(raw json.RawMessage) (bool, string) {
	obj, err := rawObject(raw)
	if err != nil {
		return false, err.Error()
	}
	cname := strings.TrimSpace(stringValue(obj["cname"]))
	domainState := strings.ToLower(strings.TrimSpace(stringValue(obj["protected_domain_state"])))
	if cname != "" {
		switch domainState {
		case "pending", "pending_verification", "pending_dns":
			return false, "pages custom domain verification is still pending"
		case "unverified", "invalid":
			return false, "pages custom domain is not verified"
		}
	}
	if boolFromAny(obj["https_enforced"]) && obj["https_certificate"] == nil {
		return false, "pages HTTPS is enforced but certificate readiness is unavailable"
	}
	if status := strings.ToLower(strings.TrimSpace(stringValue(obj["status"]))); status == "errored" || status == "error" {
		return false, "pages health reports an error state"
	}
	return true, "readiness validated"
}

func sanitizeDeploymentPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	deleteKeys(obj, "id", "node_id", "created_at", "updated_at", "statuses_url", "repository_url", "creator", "sha", "original_environment")
	return keepOnlyKeys(obj, "ref", "task", "auto_merge", "required_contexts", "payload", "environment", "description", "transient_environment", "production_environment"), nil
}

func sanitizeEnvironmentPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	if branchPolicy, ok := obj["deployment_branch_policy"].(map[string]any); ok {
		obj["deployment_branch_policy"] = keepOnlyKeys(branchPolicy, "protected_branches", "custom_branch_policies")
	}
	deleteKeys(obj, "id", "node_id", "name", "html_url", "url", "created_at", "updated_at", "protection_rules", "custom_branch_policies")
	return keepOnlyKeys(obj, "wait_timer", "reviewers", "deployment_branch_policy", "prevent_self_review", "can_admins_bypass"), nil
}

func extractEnvironmentName(raw json.RawMessage) string {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	name, _ := obj["name"].(string)
	return strings.TrimSpace(name)
}
