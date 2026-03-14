package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type packagesExecutor struct{ client *Client }

type notificationsExecutor struct {
	client       *Client
	capabilityID string
}

func (e packagesExecutor) CapabilityID() string      { return "packages" }
func (e notificationsExecutor) CapabilityID() string { return e.capabilityID }

func (e packagesExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	ownerType := packageOwnerType(req.Scope, req.Query)
	path, resourceID, err := e.packagePath(req.Scope, req.ResourceID, req.Query)
	if err != nil {
		return nil, err
	}
	raw, err := e.doPackageRaw(ctx, ownerType, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	view := packageView(req.Scope, req.Query)
	identity := packageIdentity(e.client.owner, e.client.repo, req.Scope, req.Query, req.ResourceID)
	state, err := normalizePackageSnapshot(raw, identity, view)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), resourceID, state), nil
}

func (e packagesExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	ownerType := packageOwnerType(req.Scope, req.Scope)
	path, resourceID, err := e.packagePath(req.Scope, req.ResourceID, req.Scope)
	if err != nil {
		return nil, err
	}
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   resourceID,
	}
	result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{Scope: req.Scope, ResourceID: req.ResourceID})
	switch op {
	case "delete":
		if err := e.doPackageJSON(ctx, ownerType, http.MethodDelete, path, nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	case "restore":
		if err := e.doPackageJSON(ctx, ownerType, http.MethodPost, path+"/restore", json.RawMessage(req.Payload), nil, http.StatusCreated, http.StatusOK); err != nil {
			return nil, err
		}
		after, err := e.Inspect(ctx, platform.AdminInspectRequest{Scope: req.Scope, ResourceID: req.ResourceID})
		if err == nil {
			result.After = after
		}
	default:
		return nil, fmt.Errorf("unsupported packages operation: %s", op)
	}
	return result, nil
}

func (e packagesExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{Scope: req.Scope, ResourceID: req.ResourceID})
	if strings.EqualFold(req.Mutation.Operation, "delete") {
		if inspectMissingOK(err) {
			return &platform.AdminValidationResult{OK: true, Summary: "package resource deleted", ResourceID: req.ResourceID}, nil
		}
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: req.ResourceID}, nil
		}
		return &platform.AdminValidationResult{OK: false, Summary: "package resource still exists", ResourceID: req.ResourceID, Snapshot: snap}, nil
	}
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: req.ResourceID}, nil
	}
	return &platform.AdminValidationResult{OK: true, Summary: "package resource validated", ResourceID: req.ResourceID, Snapshot: snap}, nil
}

func (e packagesExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	switch req.Mutation.Operation {
	case "delete":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "restore",
			ResourceID: req.Mutation.ResourceID,
			Scope:      req.Scope,
			Payload:    cloneRaw(req.Payload),
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "package resource restored"}, nil
	case "restore":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "delete",
			ResourceID: req.Mutation.ResourceID,
			Scope:      req.Scope,
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "package resource deleted after restore rollback"}, nil
	default:
		return nil, fmt.Errorf("unsupported package rollback operation: %s", req.Mutation.Operation)
	}
}

func (e notificationsExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path, resourceID, err := e.notificationPath(req.ResourceID, req.Scope, req.Query)
	if err != nil {
		return nil, err
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e notificationsExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	path, resourceID, err := e.notificationPath(req.ResourceID, req.Scope, req.Scope)
	if err != nil {
		return nil, err
	}
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   resourceID,
	}
	result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.ResourceID, Scope: req.Scope, Query: req.Scope})
	switch op {
	case "watch", "update":
		raw, err := e.client.doRaw(ctx, http.MethodPut, path, json.RawMessage(req.Payload), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), resourceID, raw)
	case "mark_read":
		if err := e.client.doJSON(ctx, http.MethodPut, path, json.RawMessage(req.Payload), nil, http.StatusResetContent, http.StatusNoContent); err != nil {
			return nil, err
		}
	case "delete", "unwatch":
		if err := e.client.doJSON(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported notifications operation: %s", op)
	}
	return result, nil
}

func (e notificationsExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	if strings.EqualFold(req.Mutation.Operation, "mark_read") {
		return &platform.AdminValidationResult{OK: true, Summary: "notifications marked as read"}, nil
	}
	return validateByInspect(ctx, notificationValidator{executor: e}, req, "notification subscription validated")
}

func (e notificationsExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	if req.Mutation.Operation == "mark_read" {
		return &platform.AdminRollbackResult{OK: false, Summary: "mark_read cannot be rolled back via GitHub API"}, nil
	}
	if req.Mutation.Before == nil {
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "delete",
			ResourceID: req.Mutation.ResourceID,
			Scope:      req.Scope,
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "subscription removed as rollback"}, nil
	}
	restore, err := sanitizeNotificationPayload(req.Mutation.Before.State)
	if err != nil {
		return nil, err
	}
	restoreRaw, err := marshalRaw(restore)
	if err != nil {
		return nil, err
	}
	path, resourceID, err := e.notificationPath(req.Mutation.ResourceID, req.Scope, req.Scope)
	if err != nil {
		return nil, err
	}
	raw, err := e.client.doRaw(ctx, http.MethodPut, path, restoreRaw, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return &platform.AdminRollbackResult{OK: true, Summary: "notification subscription restored", Snapshot: snapshot(e.CapabilityID(), resourceID, raw)}, nil
}

type notificationValidator struct{ executor notificationsExecutor }

func (n notificationValidator) CapabilityID() string { return n.executor.CapabilityID() }
func (n notificationValidator) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	return n.executor.Inspect(ctx, req)
}
func (n notificationValidator) Mutate(context.Context, platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	return nil, fmt.Errorf("unsupported")
}
func (n notificationValidator) Validate(context.Context, platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return nil, fmt.Errorf("unsupported")
}
func (n notificationValidator) Rollback(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return nil, fmt.Errorf("unsupported")
}

func (e packagesExecutor) packagePath(scope map[string]string, resourceID string, query map[string]string) (string, string, error) {
	packageType := strings.TrimSpace(normalizeScopeValue(scope, "package_type", normalizeScopeValue(query, "package_type", "")))
	if packageType == "" {
		return "", "", fmt.Errorf("package_type is required")
	}
	packageName := strings.TrimSpace(normalizeScopeValue(scope, "package_name", normalizeScopeValue(query, "package_name", normalizeScopeValue(scope, "name", normalizeScopeValue(query, "name", "")))))
	versionID := strings.TrimSpace(resourceID)
	if versionID == "" {
		versionID = strings.TrimSpace(normalizeScopeValue(scope, "version_id", normalizeScopeValue(query, "version_id", normalizeScopeValue(scope, "version", normalizeScopeValue(query, "version", "")))))
	}
	view := strings.ToLower(normalizeScopeValue(query, "view", ""))
	ownerType := strings.ToLower(strings.TrimSpace(normalizeScopeValue(scope, "owner_type", normalizeScopeValue(query, "owner_type", ""))))

	base, err := e.ownerPackagesBase(ownerType)
	if err != nil {
		return "", "", err
	}
	switch {
	case packageName == "":
		return appendQuery(base, query, "view", "package_name", "package_type", "owner_type", "version_id"), base, nil
	case view == "latest_version":
		return appendQuery(base+"/"+trimResourceID(packageType)+"/"+trimResourceID(packageName)+"/versions", map[string]string{
			"per_page": "1",
		}, "view", "package_name", "package_type", "owner_type", "version_id"), packageName, nil
	case view == "versions":
		return appendQuery(base+"/"+trimResourceID(packageType)+"/"+trimResourceID(packageName)+"/versions", query, "view", "package_name", "package_type", "owner_type", "version_id"), packageName, nil
	case view == "assets" && versionID != "":
		return base + "/" + trimResourceID(packageType) + "/" + trimResourceID(packageName) + "/versions/" + trimResourceID(versionID), versionID, nil
	case versionID != "":
		return base + "/" + trimResourceID(packageType) + "/" + trimResourceID(packageName) + "/versions/" + trimResourceID(versionID), versionID, nil
	default:
		return base + "/" + trimResourceID(packageType) + "/" + trimResourceID(packageName), packageName, nil
	}
}

func (e packagesExecutor) ownerPackagesBase(ownerType string) (string, error) {
	switch ownerType {
	case "org", "organization":
		return "/orgs/" + trimResourceID(e.client.owner) + "/packages", nil
	case "user", "":
		return "/users/" + trimResourceID(e.client.owner) + "/packages", nil
	default:
		return "", fmt.Errorf("unsupported owner_type %q", ownerType)
	}
}

func (e packagesExecutor) doPackageRaw(ctx context.Context, ownerType, method, path string, body interface{}, expected ...int) (json.RawMessage, error) {
	raw, err := e.client.doRaw(ctx, method, path, body, expected...)
	if err == nil {
		return raw, nil
	}
	if strings.TrimSpace(ownerType) != "" {
		return nil, err
	}
	if strings.Contains(err.Error(), "status 404") && strings.HasPrefix(path, "/users/") {
		return e.client.doRaw(ctx, method, strings.Replace(path, "/users/", "/orgs/", 1), body, expected...)
	}
	if strings.Contains(err.Error(), "status 404") && strings.HasPrefix(path, "/orgs/") {
		return e.client.doRaw(ctx, method, strings.Replace(path, "/orgs/", "/users/", 1), body, expected...)
	}
	return nil, err
}

func (e packagesExecutor) doPackageJSON(ctx context.Context, ownerType, method, path string, body interface{}, out interface{}, expected ...int) error {
	raw, err := e.doPackageRaw(ctx, ownerType, method, path, body, expected...)
	if err != nil {
		return err
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func packageOwnerType(scope map[string]string, query map[string]string) string {
	value := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		normalizeScopeValue(scope, "owner_type", normalizeScopeValue(query, "owner_type", "")),
		normalizeScopeValue(scope, "scope", normalizeScopeValue(query, "scope", "")),
	)))
	switch value {
	case "organization":
		return "org"
	case "repository", "repo":
		// Repository-scoped package permission models still resolve through the owner namespace REST endpoints.
		return ""
	default:
		return value
	}
}

func packageView(scope, query map[string]string) string {
	return strings.ToLower(strings.TrimSpace(normalizeScopeValue(query, "view", normalizeScopeValue(scope, "view", ""))))
}

func packageIdentity(owner, repo string, scope map[string]string, query map[string]string, resourceID string) platform.PackageIdentity {
	scopeValue := strings.TrimSpace(normalizeScopeValue(scope, "scope",
		normalizeScopeValue(query, "scope",
			normalizeScopeValue(scope, "owner_type", normalizeScopeValue(query, "owner_type", "repo")),
		),
	))
	switch strings.ToLower(scopeValue) {
	case "organization":
		scopeValue = "org"
	case "repository":
		scopeValue = "repo"
	}
	namespace := strings.TrimSpace(normalizeScopeValue(scope, "namespace", normalizeScopeValue(query, "namespace", "")))
	if namespace == "" {
		namespace = strings.TrimSpace(owner)
		if strings.EqualFold(scopeValue, "repo") && strings.TrimSpace(repo) != "" {
			namespace = strings.TrimSpace(owner) + "/" + strings.TrimSpace(repo)
		}
	}
	identity := platform.PackageIdentity{
		Registry:    strings.TrimSpace(normalizeScopeValue(scope, "registry", normalizeScopeValue(query, "registry", "github-packages"))),
		PackageType: strings.TrimSpace(normalizeScopeValue(scope, "package_type", normalizeScopeValue(query, "package_type", ""))),
		Namespace:   namespace,
		Name:        strings.TrimSpace(normalizeScopeValue(scope, "package_name", normalizeScopeValue(query, "package_name", normalizeScopeValue(scope, "name", normalizeScopeValue(query, "name", ""))))),
		Version:     strings.TrimSpace(firstNonEmpty(resourceID, normalizeScopeValue(scope, "version_id", normalizeScopeValue(query, "version_id", normalizeScopeValue(scope, "version", normalizeScopeValue(query, "version", "")))))),
		Scope:       scopeValue,
	}
	return platform.NormalizePackageIdentity(identity)
}

func normalizePackageSnapshot(raw json.RawMessage, identity platform.PackageIdentity, view string) (json.RawMessage, error) {
	state := map[string]any{
		"identity": identity,
		"view":     strings.TrimSpace(view),
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	switch value := decoded.(type) {
	case []any:
		switch view {
		case "latest_version":
			state["versions"] = value
			if len(value) > 0 {
				state["latest_version"] = value[0]
			}
		default:
			state["versions"] = value
		}
	case map[string]any:
		if metadata, ok := value["metadata"]; ok {
			state["registry_metadata"] = metadata
		}
		switch view {
		case "assets":
			if files, ok := value["files"]; ok {
				state["assets"] = files
			} else if assets, ok := value["assets"]; ok {
				state["assets"] = assets
			}
			state["version"] = value
		default:
			if identity.Version != "" {
				state["version"] = value
			} else {
				state["package"] = value
			}
		}
	default:
		state["package"] = decoded
	}
	return marshalRaw(state)
}

func (e notificationsExecutor) notificationPath(resourceID string, scope map[string]string, query map[string]string) (string, string, error) {
	view := strings.ToLower(normalizeScopeValue(query, "view", normalizeScopeValue(scope, "view", "repo_subscription")))
	threadID := normalizeScopeValue(scope, "thread_id", strings.TrimSpace(resourceID))
	switch view {
	case "inbox", "global_inbox":
		return appendQuery("/notifications", query, "view", "thread_id"), "global_inbox", nil
	case "participating", "participating_inbox":
		merged := cloneStringMap(query)
		if merged == nil {
			merged = map[string]string{}
		}
		merged["participating"] = "true"
		return appendQuery("/notifications", merged, "view", "thread_id"), "participating_inbox", nil
	case "repo_notifications":
		return appendQuery(e.client.repoPath("/notifications"), query, "view", "thread_id"), "repo_notifications", nil
	case "thread":
		if threadID == "" {
			return "", "", fmt.Errorf("thread id is required")
		}
		return "/notifications/threads/" + trimResourceID(threadID), threadID, nil
	case "thread_subscription":
		if threadID == "" {
			return "", "", fmt.Errorf("thread id is required")
		}
		return "/notifications/threads/" + trimResourceID(threadID) + "/subscription", threadID, nil
	case "repo_subscription", "":
		return e.client.repoPath("/subscription"), "repo_subscription", nil
	default:
		return "", "", fmt.Errorf("unsupported notifications view: %s", view)
	}
}

func sanitizeNotificationPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	return keepOnlyKeys(obj, "subscribed", "ignored"), nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprintf("%v", typed)
	}
}
