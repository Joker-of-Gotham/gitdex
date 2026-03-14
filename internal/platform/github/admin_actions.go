package github

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/nacl/box"
)

type actionsConfigExecutor struct {
	client *Client
}

type actionsConfigPayload struct {
	Kind                  string  `json:"kind"`
	Scope                 string  `json:"scope,omitempty"`
	Environment           string  `json:"environment,omitempty"`
	Name                  string  `json:"name"`
	Value                 string  `json:"value,omitempty"`
	SelectedRepositoryIDs []int64 `json:"selected_repository_ids,omitempty"`
	Visibility            string  `json:"visibility,omitempty"`
}

type githubPublicKey struct {
	Key   string `json:"key"`
	KeyID string `json:"key_id"`
}

func (e actionsConfigExecutor) CapabilityID() string {
	return "actions_secrets_variables"
}

func (e actionsConfigExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	kind := strings.ToLower(normalizeScopeValue(req.Scope, "kind", "variable"))
	path, err := e.configPath(kind, req.Scope, req.ResourceID)
	if err != nil {
		return nil, err
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), req.ResourceID, raw), nil
}

func (e actionsConfigExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	payload, err := parseActionsConfigPayload(req)
	if err != nil {
		return nil, err
	}
	before, _ := e.Inspect(ctx, platform.AdminInspectRequest{
		ResourceID: payload.Name,
		Scope: map[string]string{
			"kind":        payload.Kind,
			"scope":       payload.Scope,
			"environment": payload.Environment,
		},
	})

	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   payload.Name,
		Before:       before,
		Metadata: map[string]string{
			"kind":  payload.Kind,
			"scope": payload.Scope,
		},
	}
	if payload.Environment != "" {
		result.Metadata["environment"] = payload.Environment
	}

	switch payload.Kind {
	case "secret":
		if err := e.mutateSecret(ctx, op, payload); err != nil {
			return nil, err
		}
	default:
		if err := e.mutateVariable(ctx, op, payload); err != nil {
			return nil, err
		}
	}

	after, inspectErr := e.Inspect(ctx, platform.AdminInspectRequest{
		ResourceID: payload.Name,
		Scope: map[string]string{
			"kind":        payload.Kind,
			"scope":       payload.Scope,
			"environment": payload.Environment,
		},
	})
	if inspectErr == nil {
		result.After = after
	} else if inspectMissingOK(inspectErr) {
		result.After = nil
	} else {
		return nil, inspectErr
	}
	return result, nil
}

func (e actionsConfigExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	kind := normalizeScopeValue(req.Scope, "kind", normalizeScopeValue(req.Mutation.Metadata, "kind", "variable"))
	scope := normalizeScopeValue(req.Scope, "scope", normalizeScopeValue(req.Mutation.Metadata, "scope", "repository"))
	environment := normalizeScopeValue(req.Scope, "environment", normalizeScopeValue(req.Mutation.Metadata, "environment", ""))
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		resourceID = req.Mutation.ResourceID
	}

	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{
		ResourceID: resourceID,
		Scope: map[string]string{
			"kind":        kind,
			"scope":       scope,
			"environment": environment,
		},
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
	if kind == "secret" {
		return &platform.AdminValidationResult{OK: true, Summary: "secret metadata is present", ResourceID: resourceID, Snapshot: snap}, nil
	}
	matched, reason, matchErr := subsetMatches(snap.State, req.Payload)
	if matchErr != nil {
		return nil, matchErr
	}
	return &platform.AdminValidationResult{
		OK:         matched,
		Summary:    summaryFromMatch(matched, reason, "variable validated"),
		ResourceID: resourceID,
		Snapshot:   snap,
	}, nil
}

func (e actionsConfigExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}

	kind := normalizeScopeValue(req.Scope, "kind", normalizeScopeValue(req.Mutation.Metadata, "kind", "variable"))
	scope := normalizeScopeValue(req.Scope, "scope", normalizeScopeValue(req.Mutation.Metadata, "scope", "repository"))
	environment := normalizeScopeValue(req.Scope, "environment", normalizeScopeValue(req.Mutation.Metadata, "environment", ""))
	resourceID := req.Mutation.ResourceID

	if req.Mutation.Before == nil {
		deleteReq := platform.AdminMutationRequest{
			Operation:  "delete",
			ResourceID: resourceID,
			Scope: map[string]string{
				"kind":        kind,
				"scope":       scope,
				"environment": environment,
			},
			Payload: mustMarshalRaw(actionsConfigPayload{
				Kind:        kind,
				Scope:       scope,
				Environment: environment,
				Name:        resourceID,
			}),
		}
		if _, err := e.Mutate(ctx, deleteReq); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "resource deleted as rollback"}, nil
	}

	if kind == "secret" {
		if len(req.Payload) == 0 {
			return &platform.AdminRollbackResult{
				OK:       false,
				Summary:  "secret rollback requires rollback payload with previous plaintext value",
				Snapshot: req.Mutation.Before,
			}, nil
		}
		var payload actionsConfigPayload
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return nil, err
		}
		payload.Kind = "secret"
		payload.Scope = scope
		payload.Environment = environment
		payload.Name = resourceID
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation: "update",
			Scope: map[string]string{
				"kind":        kind,
				"scope":       scope,
				"environment": environment,
			},
			Payload: mustMarshalRaw(payload),
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "secret restored from rollback payload", Snapshot: req.Mutation.Before}, nil
	}

	restore, err := sanitizeVariablePayload(req.Mutation.Before.State)
	if err != nil {
		return nil, err
	}
	restoreRaw, err := marshalRaw(restore)
	if err != nil {
		return nil, err
	}
	if _, err := e.mutateVariableRaw(ctx, "update", scope, environment, restoreRaw); err != nil {
		return nil, err
	}
	current, err := e.Inspect(ctx, platform.AdminInspectRequest{
		ResourceID: resourceID,
		Scope: map[string]string{
			"kind":        kind,
			"scope":       scope,
			"environment": environment,
		},
	})
	if err != nil {
		return nil, err
	}
	return &platform.AdminRollbackResult{OK: true, Summary: "variable restored", Snapshot: current}, nil
}

func (e actionsConfigExecutor) configPath(kind string, scope map[string]string, resourceID string) (string, error) {
	scopeName := strings.ToLower(normalizeScopeValue(scope, "scope", "repository"))
	environment := strings.TrimSpace(normalizeScopeValue(scope, "environment", ""))
	name := strings.TrimSpace(resourceID)
	base := e.client.repoPath("")

	switch kind {
	case "secret":
		switch scopeName {
		case "environment":
			if environment == "" {
				return "", fmt.Errorf("environment scope requires environment name")
			}
			if name == "" {
				return fmt.Sprintf("%s/environments/%s/secrets", base, trimResourceID(environment)), nil
			}
			return fmt.Sprintf("%s/environments/%s/secrets/%s", base, trimResourceID(environment), trimResourceID(name)), nil
		default:
			if name == "" {
				return base + "/actions/secrets", nil
			}
			return fmt.Sprintf("%s/actions/secrets/%s", base, trimResourceID(name)), nil
		}
	default:
		switch scopeName {
		case "environment":
			if environment == "" {
				return "", fmt.Errorf("environment scope requires environment name")
			}
			if name == "" {
				return fmt.Sprintf("%s/environments/%s/variables", base, trimResourceID(environment)), nil
			}
			return fmt.Sprintf("%s/environments/%s/variables/%s", base, trimResourceID(environment), trimResourceID(name)), nil
		default:
			if name == "" {
				return base + "/actions/variables", nil
			}
			return fmt.Sprintf("%s/actions/variables/%s", base, trimResourceID(name)), nil
		}
	}
}

func (e actionsConfigExecutor) mutateVariable(ctx context.Context, operation string, payload actionsConfigPayload) error {
	raw, err := marshalRaw(payload)
	if err != nil {
		return err
	}
	_, err = e.mutateVariableRaw(ctx, operation, payload.Scope, payload.Environment, raw)
	return err
}

func (e actionsConfigExecutor) mutateVariableRaw(ctx context.Context, operation, scope, environment string, raw json.RawMessage) (json.RawMessage, error) {
	payloadMap, err := sanitizeVariablePayload(raw)
	if err != nil {
		return nil, err
	}
	name, _ := payloadMap["name"].(string)
	path, err := e.configPath("variable", map[string]string{"scope": scope, "environment": environment}, name)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case "create":
		listPath, err := e.configPath("variable", map[string]string{"scope": scope, "environment": environment}, "")
		if err != nil {
			return nil, err
		}
		return nil, e.client.doJSON(ctx, http.MethodPost, listPath, payloadMap, nil, http.StatusCreated, http.StatusNoContent)
	case "update":
		return nil, e.client.doJSON(ctx, http.MethodPatch, path, payloadMap, nil, http.StatusNoContent)
	case "delete":
		return nil, e.client.doJSON(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent)
	default:
		return nil, fmt.Errorf("unsupported variable operation: %s", operation)
	}
}

func (e actionsConfigExecutor) mutateSecret(ctx context.Context, operation string, payload actionsConfigPayload) error {
	path, err := e.configPath("secret", map[string]string{"scope": payload.Scope, "environment": payload.Environment}, payload.Name)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case "create", "update":
		if strings.TrimSpace(payload.Value) == "" {
			return fmt.Errorf("secret value is required")
		}
		keyPath, err := e.secretPublicKeyPath(payload.Scope, payload.Environment)
		if err != nil {
			return err
		}
		var publicKey githubPublicKey
		if err := e.client.doJSON(ctx, http.MethodGet, keyPath, nil, &publicKey, http.StatusOK); err != nil {
			return err
		}
		encrypted, err := encryptSecretValue(publicKey.Key, payload.Value)
		if err != nil {
			return err
		}
		body := map[string]any{
			"encrypted_value": encrypted,
			"key_id":          publicKey.KeyID,
		}
		if payload.Visibility != "" {
			body["visibility"] = payload.Visibility
		}
		if len(payload.SelectedRepositoryIDs) > 0 {
			body["selected_repository_ids"] = payload.SelectedRepositoryIDs
		}
		return e.client.doJSON(ctx, http.MethodPut, path, body, nil, http.StatusCreated, http.StatusNoContent)
	case "delete":
		return e.client.doJSON(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent)
	default:
		return fmt.Errorf("unsupported secret operation: %s", operation)
	}
}

func (e actionsConfigExecutor) secretPublicKeyPath(scope, environment string) (string, error) {
	scopeName := strings.ToLower(strings.TrimSpace(scope))
	switch scopeName {
	case "environment":
		if strings.TrimSpace(environment) == "" {
			return "", fmt.Errorf("environment scope requires environment name")
		}
		return e.client.repoPath(fmt.Sprintf("/environments/%s/secrets/public-key", trimResourceID(environment))), nil
	default:
		return e.client.repoPath("/actions/secrets/public-key"), nil
	}
}

func parseActionsConfigPayload(req platform.AdminMutationRequest) (actionsConfigPayload, error) {
	var payload actionsConfigPayload
	if len(req.Payload) > 0 {
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return payload, err
		}
	}
	if strings.TrimSpace(payload.Kind) == "" {
		payload.Kind = normalizeScopeValue(req.Scope, "kind", "variable")
	}
	if strings.TrimSpace(payload.Scope) == "" {
		payload.Scope = normalizeScopeValue(req.Scope, "scope", "repository")
	}
	if strings.TrimSpace(payload.Environment) == "" {
		payload.Environment = normalizeScopeValue(req.Scope, "environment", "")
	}
	if strings.TrimSpace(payload.Name) == "" {
		payload.Name = strings.TrimSpace(req.ResourceID)
	}
	payload.Kind = strings.ToLower(strings.TrimSpace(payload.Kind))
	payload.Scope = strings.ToLower(strings.TrimSpace(payload.Scope))
	if payload.Kind != "secret" {
		payload.Kind = "variable"
	}
	if payload.Scope == "" {
		payload.Scope = "repository"
	}
	if payload.Name == "" {
		return payload, fmt.Errorf("name is required")
	}
	return payload, nil
}

func sanitizeVariablePayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	deleteKeys(obj, "created_at", "updated_at")
	return keepOnlyKeys(obj, "name", "value", "visibility", "selected_repository_ids"), nil
}

func encryptSecretValue(publicKeyBase64, value string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKeyBase64))
	if err != nil {
		return "", fmt.Errorf("decode public key: %w", err)
	}
	if len(keyBytes) != 32 {
		return "", fmt.Errorf("unexpected public key length: %d", len(keyBytes))
	}
	var recipientPub [32]byte
	copy(recipientPub[:], keyBytes)
	ephemeralPub, ephemeralPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate ephemeral key: %w", err)
	}
	hash, err := blake2b.New(24, nil)
	if err != nil {
		return "", fmt.Errorf("create blake2b: %w", err)
	}
	_, _ = hash.Write(ephemeralPub[:])
	_, _ = hash.Write(recipientPub[:])
	sum := hash.Sum(nil)
	var nonce [24]byte
	copy(nonce[:], sum[:24])
	ciphertext := box.Seal(ephemeralPub[:], []byte(value), &nonce, &recipientPub, ephemeralPriv)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func mustMarshalRaw(v any) json.RawMessage {
	raw, err := marshalRaw(v)
	if err != nil {
		return nil
	}
	return raw
}

func summaryFromMatch(ok bool, reason, success string) string {
	if ok {
		return success
	}
	if strings.TrimSpace(reason) != "" {
		return reason
	}
	return "validation failed"
}
