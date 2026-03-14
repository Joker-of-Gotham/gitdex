package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type deployKeyExecutor struct{ client *Client }

type repoSecretExecutor struct {
	client       *Client
	capabilityID string
	baseSegment  string
}

type repoSecretPayload struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

func (e deployKeyExecutor) CapabilityID() string  { return "deploy_keys" }
func (e repoSecretExecutor) CapabilityID() string { return e.capabilityID }

func (e deployKeyExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path := e.client.repoPath("/keys")
	if strings.TrimSpace(req.ResourceID) != "" {
		path += "/" + trimResourceID(req.ResourceID)
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
}

func (e deployKeyExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{CapabilityID: e.CapabilityID(), Operation: op, ResourceID: strings.TrimSpace(req.ResourceID)}
	if result.ResourceID != "" {
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: result.ResourceID})
	}
	switch op {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/keys"), json.RawMessage(req.Payload), http.StatusCreated)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractResourceID(raw, result.ResourceID)
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "delete":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("deploy key id is required")
		}
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/keys/"+trimResourceID(result.ResourceID)), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported deploy key operation: %s", op)
	}
	return result, nil
}

func (e deployKeyExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return validateByInspect(ctx, e, req, "deploy key validated")
}

func (e deployKeyExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return rollbackBySnapshot(ctx, e, req, sanitizeDeployKeyPayload, "deploy key")
}

func (e repoSecretExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path, err := e.secretPath(req.ResourceID, normalizeScopeValue(req.Query, "view", ""))
	if err != nil {
		return nil, err
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
}

func (e repoSecretExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	payload, err := parseRepoSecretPayload(req)
	if err != nil {
		return nil, err
	}
	before, _ := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: payload.Name})
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   payload.Name,
		Before:       before,
		Metadata:     map[string]string{"kind": "secret"},
	}
	switch op {
	case "create", "update":
		if strings.TrimSpace(payload.Value) == "" {
			return nil, fmt.Errorf("secret value is required")
		}
		keyPath, err := e.secretPath("", "public_key")
		if err != nil {
			return nil, err
		}
		var publicKey githubPublicKey
		if err := e.client.doJSON(ctx, http.MethodGet, keyPath, nil, &publicKey, http.StatusOK); err != nil {
			return nil, err
		}
		encrypted, err := encryptSecretValue(publicKey.Key, payload.Value)
		if err != nil {
			return nil, err
		}
		path, err := e.secretPath(payload.Name, "")
		if err != nil {
			return nil, err
		}
		if err := e.client.doJSON(ctx, http.MethodPut, path, map[string]any{
			"encrypted_value": encrypted,
			"key_id":          publicKey.KeyID,
		}, nil, http.StatusCreated, http.StatusNoContent); err != nil {
			return nil, err
		}
	case "delete":
		path, err := e.secretPath(payload.Name, "")
		if err != nil {
			return nil, err
		}
		if err := e.client.doJSON(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported secret operation: %s", op)
	}

	after, inspectErr := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: payload.Name})
	if inspectErr == nil {
		result.After = after
	} else if !inspectMissingOK(inspectErr) {
		return nil, inspectErr
	}
	return result, nil
}

func (e repoSecretExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		resourceID = req.Mutation.ResourceID
	}
	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: resourceID})
	if strings.EqualFold(req.Mutation.Operation, "delete") {
		if inspectMissingOK(err) {
			return &platform.AdminValidationResult{OK: true, Summary: "secret deleted", ResourceID: resourceID}, nil
		}
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: resourceID}, nil
		}
		return &platform.AdminValidationResult{OK: false, Summary: "secret still exists", ResourceID: resourceID, Snapshot: snap}, nil
	}
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: resourceID}, nil
	}
	return &platform.AdminValidationResult{OK: true, Summary: "secret metadata is present", ResourceID: resourceID, Snapshot: snap}, nil
}

func (e repoSecretExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	if req.Mutation.Before == nil {
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "delete",
			ResourceID: req.Mutation.ResourceID,
			Payload:    mustMarshalRaw(repoSecretPayload{Name: req.Mutation.ResourceID}),
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "secret deleted as rollback"}, nil
	}
	if len(req.Payload) == 0 {
		return &platform.AdminRollbackResult{
			OK:       false,
			Summary:  "secret rollback requires rollback payload with previous plaintext value",
			Snapshot: req.Mutation.Before,
		}, nil
	}
	var payload repoSecretPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return nil, err
	}
	payload.Name = req.Mutation.ResourceID
	if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
		Operation: "update",
		Payload:   mustMarshalRaw(payload),
	}); err != nil {
		return nil, err
	}
	return &platform.AdminRollbackResult{OK: true, Summary: "secret restored from rollback payload", Snapshot: req.Mutation.Before}, nil
}

func (e repoSecretExecutor) secretPath(name, view string) (string, error) {
	base := e.client.repoPath("/" + strings.TrimSpace(e.baseSegment) + "/secrets")
	if strings.EqualFold(strings.TrimSpace(view), "public_key") {
		return base + "/public-key", nil
	}
	if strings.TrimSpace(name) == "" {
		return base, nil
	}
	return base + "/" + trimResourceID(name), nil
}

func parseRepoSecretPayload(req platform.AdminMutationRequest) (repoSecretPayload, error) {
	var payload repoSecretPayload
	if len(req.Payload) > 0 {
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return payload, err
		}
	}
	if strings.TrimSpace(payload.Name) == "" {
		payload.Name = strings.TrimSpace(req.ResourceID)
	}
	if strings.TrimSpace(payload.Name) == "" {
		return payload, fmt.Errorf("name is required")
	}
	return payload, nil
}

func sanitizeDeployKeyPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	deleteKeys(obj, "id", "node_id", "url", "verified", "created_at", "enabled")
	return keepOnlyKeys(obj, "title", "key", "read_only"), nil
}
