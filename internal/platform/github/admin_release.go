package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type releaseExecutor struct{ client *Client }

func (e releaseExecutor) CapabilityID() string { return "release" }

func (e releaseExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	path := e.client.repoPath("/releases")
	view := strings.ToLower(normalizeScopeValue(req.Query, "view", ""))
	switch view {
	case "latest":
		path += "/latest"
	case "by_tag":
		tag := normalizeScopeValue(req.Query, "tag", strings.TrimSpace(req.ResourceID))
		if tag == "" {
			return nil, fmt.Errorf("tag is required")
		}
		path += "/tags/" + trimResourceID(tag)
	case "assets":
		releaseID := normalizeScopeValue(req.Query, "release_id", strings.TrimSpace(req.ResourceID))
		if releaseID == "" {
			return nil, fmt.Errorf("release id is required")
		}
		path += "/" + trimResourceID(releaseID) + "/assets"
	case "asset":
		assetID := normalizeScopeValue(req.Query, "asset_id", strings.TrimSpace(req.ResourceID))
		if assetID == "" {
			return nil, fmt.Errorf("asset id is required")
		}
		path += "/assets/" + trimResourceID(assetID)
	case "asset_download":
		assetID := normalizeScopeValue(req.Query, "asset_id", strings.TrimSpace(req.ResourceID))
		if assetID == "" {
			return nil, fmt.Errorf("asset id is required")
		}
		raw, err := e.inspectReleaseAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		return snapshot(e.CapabilityID(), assetID, raw), nil
	default:
		if strings.TrimSpace(req.ResourceID) != "" {
			path += "/" + trimResourceID(req.ResourceID)
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

func (e releaseExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   strings.TrimSpace(req.ResourceID),
	}
	if result.ResourceID != "" && op != "asset_upload" && op != "asset_delete" {
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: result.ResourceID})
	}

	switch op {
	case "create":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/releases"), json.RawMessage(req.Payload), http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractResourceID(raw, result.ResourceID)
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "update":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("release id is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath("/releases/"+trimResourceID(result.ResourceID)), json.RawMessage(req.Payload), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "delete":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("release id is required")
		}
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/releases/"+trimResourceID(result.ResourceID)), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
	case "publish_draft":
		if result.ResourceID == "" {
			return nil, fmt.Errorf("release id is required")
		}
		raw, err := e.client.doRaw(ctx, http.MethodPatch, e.client.repoPath("/releases/"+trimResourceID(result.ResourceID)), map[string]any{"draft": false}, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "generate_notes":
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/releases/generate-notes"), json.RawMessage(req.Payload), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "asset_upload":
		asset, releaseID, rollbackMeta, err := e.uploadReleaseAsset(ctx, req, result.ResourceID)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractResourceID(asset, result.ResourceID)
		result.After = snapshot(e.CapabilityID(), result.ResourceID, asset)
		if result.Metadata == nil {
			result.Metadata = map[string]string{}
		}
		result.Metadata["release_id"] = releaseID
		for key, value := range rollbackMeta {
			result.Metadata[key] = value
		}
		result.Metadata["rollback_grade"] = "reversible"
		result.Metadata["recoverability"] = "reversible"
		result.Metadata["partial_restore_required"] = "false"
	case "asset_delete":
		assetID := normalizeScopeValue(req.Scope, "asset_id", result.ResourceID)
		if assetID == "" {
			return nil, fmt.Errorf("asset id is required")
		}
		result.ResourceID = assetID
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{
			ResourceID: assetID,
			Query:      map[string]string{"view": "asset"},
		})
		if err := e.client.doJSON(ctx, http.MethodDelete, e.client.repoPath("/releases/assets/"+trimResourceID(assetID)), nil, nil, http.StatusNoContent); err != nil {
			return nil, err
		}
		if result.Metadata == nil {
			result.Metadata = map[string]string{}
		}
		if releaseID := normalizeScopeValue(req.Scope, "release_id", ""); releaseID != "" {
			result.Metadata["release_id"] = releaseID
		}
		if result.Before != nil {
			if meta, err := assetRestoreMetadata(result.Before.State); err == nil {
				for key, value := range meta {
					result.Metadata[key] = value
				}
			}
			if ref, err := cacheReleaseAssetRollbackBytes(ctx, e.client, result.Before.State); err == nil && strings.TrimSpace(ref) != "" {
				result.Metadata["stored_bytes_ref"] = ref
				result.Metadata["rollback_grade"] = "reversible"
				result.Metadata["recoverability"] = "reversible"
				result.Metadata["partial_restore_required"] = "false"
			}
		}
		if strings.TrimSpace(result.Metadata["rollback_grade"]) == "" {
			result.Metadata["rollback_grade"] = "manual restore required"
			result.Metadata["recoverability"] = "manual restore required"
			result.Metadata["partial_restore_required"] = "true"
		}
	default:
		return nil, fmt.Errorf("unsupported release operation: %s", op)
	}
	return result, nil
}

func (e releaseExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	switch req.Mutation.Operation {
	case "generate_notes":
		if req.Mutation.After == nil {
			return &platform.AdminValidationResult{OK: false, Summary: "release notes were not generated"}, nil
		}
		return &platform.AdminValidationResult{
			OK:       true,
			Summary:  "release notes generated",
			Snapshot: req.Mutation.After,
		}, nil
	case "asset_upload":
		assetID := strings.TrimSpace(req.Mutation.ResourceID)
		if assetID == "" {
			return &platform.AdminValidationResult{OK: false, Summary: "uploaded asset id missing"}, nil
		}
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{
			ResourceID: assetID,
			Query:      map[string]string{"view": "asset"},
		})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: assetID}, nil
		}
		return &platform.AdminValidationResult{
			OK:         true,
			Summary:    "release asset uploaded",
			ResourceID: assetID,
			Snapshot:   snap,
		}, nil
	case "publish_draft":
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: req.Mutation.ResourceID})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: req.Mutation.ResourceID}, nil
		}
		ok, summary, err := validateReleaseDraftState(snap.State, false)
		if err != nil {
			return nil, err
		}
		return &platform.AdminValidationResult{
			OK:         ok,
			Summary:    summary,
			ResourceID: req.Mutation.ResourceID,
			Snapshot:   snap,
		}, nil
	default:
		return validateByInspect(ctx, e, req, "release validated")
	}
}

func (e releaseExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	switch req.Mutation.Operation {
	case "generate_notes":
		return &platform.AdminRollbackResult{OK: true, Summary: "release notes preview does not require rollback"}, nil
	case "publish_draft":
		if req.Mutation.Before == nil {
			return &platform.AdminRollbackResult{
				OK:      false,
				Summary: "publish draft rollback requires previous release snapshot",
			}, nil
		}
		return rollbackBySnapshot(ctx, e, req, sanitizeReleasePayload, "release")
	case "asset_upload":
		assetID := strings.TrimSpace(req.Mutation.ResourceID)
		if assetID == "" {
			return &platform.AdminRollbackResult{OK: false, Summary: "uploaded asset id missing"}, nil
		}
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation:  "asset_delete",
			ResourceID: assetID,
			Scope:      map[string]string{"asset_id": assetID},
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "uploaded release asset deleted as rollback"}, nil
	case "asset_delete":
		restorePayload, err := buildReleaseAssetRestorePayload(ctx, e.client, req.Mutation)
		if err != nil {
			return &platform.AdminRollbackResult{
				OK:       false,
				Summary:  err.Error(),
				Snapshot: req.Mutation.Before,
				Compensation: &platform.CompensationAction{
					Kind:        "partial_restore_required",
					Summary:     "re-upload the deleted release asset from a trusted local artifact or backup",
					OperatorRef: "release_asset_restore",
					LedgerChain: []string{strings.TrimSpace(req.Mutation.LedgerID), strings.TrimSpace(req.Mutation.ResourceID)},
				},
			}, nil
		}
		result, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation: "asset_upload",
			Scope:     map[string]string{"release_id": restorePayload.ReleaseID},
			Payload:   restorePayload.Payload,
		})
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{
			OK:       true,
			Summary:  "deleted release asset restored",
			Snapshot: result.After,
		}, nil
	default:
		return rollbackBySnapshot(ctx, e, req, sanitizeReleasePayload, "release")
	}
}

func sanitizeReleasePayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	deleteKeys(obj, "id", "node_id", "url", "html_url", "assets", "assets_url", "upload_url", "author", "created_at", "published_at", "tarball_url", "zipball_url", "immutable")
	return keepOnlyKeys(obj, "tag_name", "target_commitish", "name", "body", "draft", "prerelease", "make_latest", "discussion_category_name", "generate_release_notes"), nil
}

func (e releaseExecutor) uploadReleaseAsset(ctx context.Context, req platform.AdminMutationRequest, fallbackReleaseID string) (json.RawMessage, string, map[string]string, error) {
	payload, err := rawObject(req.Payload)
	if err != nil {
		return nil, "", nil, err
	}
	releaseID := firstNonEmpty(
		normalizeScopeValue(req.Scope, "release_id", ""),
		strings.TrimSpace(fallbackReleaseID),
		strings.TrimSpace(stringValue(payload["release_id"])),
	)
	if releaseID == "" {
		return nil, "", nil, fmt.Errorf("release id is required for asset_upload")
	}
	releaseSnap, err := e.Inspect(ctx, platform.AdminInspectRequest{ResourceID: releaseID})
	if err != nil {
		return nil, "", nil, err
	}
	uploadURL, err := releaseUploadURL(releaseSnap.State)
	if err != nil {
		return nil, "", nil, err
	}
	data, contentType, name, label, rollbackMeta, err := releaseAssetBytes(payload)
	if err != nil {
		return nil, "", nil, err
	}
	rollbackMeta["release_id"] = releaseID
	uploadTarget, err := releaseAssetUploadTarget(uploadURL, name, label)
	if err != nil {
		return nil, "", nil, err
	}
	raw, err := e.client.doBinaryUpload(ctx, uploadTarget, contentType, data, http.StatusCreated, http.StatusOK)
	if err != nil {
		return nil, "", nil, err
	}
	return raw, releaseID, rollbackMeta, nil
}

func releaseUploadURL(raw json.RawMessage) (string, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return "", err
	}
	uploadURL := strings.TrimSpace(stringValue(obj["upload_url"]))
	if uploadURL == "" {
		return "", fmt.Errorf("release upload_url is unavailable")
	}
	return uploadURL, nil
}

func releaseAssetUploadTarget(uploadURL, name, label string) (string, error) {
	base := strings.TrimSpace(uploadURL)
	if idx := strings.Index(base, "{"); idx >= 0 {
		base = base[:idx]
	}
	if strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("release upload target is unavailable")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("name", strings.TrimSpace(name))
	if strings.TrimSpace(label) != "" {
		query.Set("label", strings.TrimSpace(label))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func releaseAssetBytes(payload map[string]any) ([]byte, string, string, string, map[string]string, error) {
	name := strings.TrimSpace(stringValue(payload["name"]))
	if name == "" {
		return nil, "", "", "", nil, fmt.Errorf("asset name is required")
	}
	label := strings.TrimSpace(stringValue(payload["label"]))
	contentType := strings.TrimSpace(stringValue(payload["content_type"]))
	if filePath := strings.TrimSpace(stringValue(payload["file_path"])); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", "", "", nil, err
		}
		rollbackMeta := releaseAssetMetadata(name, label, contentType, data)
		rollbackMeta["file_path"] = filePath
		rollbackMeta["source_kind"] = "local"
		rollbackMeta["recoverable"] = "true"
		if entry, err := platform.StoreReleaseAssetBytes(name, data, rollbackMeta); err == nil {
			rollbackMeta["stored_bytes_ref"] = entry.Ref
		}
		return data, firstNonEmpty(contentType, "application/octet-stream"), name, label, rollbackMeta, nil
	}
	if encoded := strings.TrimSpace(stringValue(payload["content_base64"])); encoded != "" {
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, "", "", "", nil, err
		}
		rollbackMeta := releaseAssetMetadata(name, label, contentType, data)
		rollbackMeta["content_base64"] = encoded
		rollbackMeta["source_kind"] = "inline_base64"
		rollbackMeta["recoverable"] = "true"
		if entry, cacheErr := platform.StoreReleaseAssetBytes(name, data, rollbackMeta); cacheErr == nil {
			rollbackMeta["stored_bytes_ref"] = entry.Ref
		}
		return data, firstNonEmpty(contentType, "application/octet-stream"), name, label, rollbackMeta, nil
	}
	if inline := strings.TrimSpace(stringValue(payload["content"])); inline != "" {
		data := []byte(inline)
		rollbackMeta := releaseAssetMetadata(name, label, contentType, data)
		rollbackMeta["content_base64"] = base64.StdEncoding.EncodeToString(data)
		rollbackMeta["source_kind"] = "inline_text"
		rollbackMeta["recoverable"] = "true"
		if entry, err := platform.StoreReleaseAssetBytes(name, data, rollbackMeta); err == nil {
			rollbackMeta["stored_bytes_ref"] = entry.Ref
		}
		return data, firstNonEmpty(contentType, "text/plain; charset=utf-8"), name, label, rollbackMeta, nil
	}
	return nil, "", "", "", nil, fmt.Errorf("asset_upload requires file_path, content_base64, or content")
}

func releaseAssetMetadata(name, label, contentType string, data []byte) map[string]string {
	return map[string]string{
		"asset_name":   name,
		"asset_label":  label,
		"content_type": contentType,
		"asset_size":   strconv.Itoa(len(data)),
		"asset_digest": fmt.Sprintf("%x", sha256.Sum256(data)),
	}
}

func cacheReleaseAssetRollbackBytes(ctx context.Context, client *Client, raw json.RawMessage) (string, error) {
	meta, err := assetRestoreMetadata(raw)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(meta["asset_name"])
	if name == "" {
		name = "asset.bin"
	}
	switch {
	case strings.TrimSpace(meta["asset_api_url"]) != "":
		data, err := client.downloadBytes(ctx, strings.TrimSpace(meta["asset_api_url"]), "application/octet-stream", http.StatusOK, http.StatusFound, http.StatusTemporaryRedirect)
		if err != nil {
			return "", err
		}
		entry, err := platform.StoreReleaseAssetBytes(name, data, meta)
		if err != nil {
			return "", err
		}
		return entry.Ref, nil
	case strings.TrimSpace(meta["download_url"]) != "":
		data, err := client.downloadBytes(ctx, strings.TrimSpace(meta["download_url"]), "application/octet-stream", http.StatusOK)
		if err != nil {
			return "", err
		}
		entry, err := platform.StoreReleaseAssetBytes(name, data, meta)
		if err != nil {
			return "", err
		}
		return entry.Ref, nil
	default:
		return "", fmt.Errorf("release asset bytes are unavailable for stable rollback")
	}
}

func assetRestoreMetadata(raw json.RawMessage) (map[string]string, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	meta := map[string]string{
		"asset_name":   strings.TrimSpace(stringValue(obj["name"])),
		"asset_label":  strings.TrimSpace(stringValue(obj["label"])),
		"content_type": strings.TrimSpace(stringValue(obj["content_type"])),
	}
	if size := strings.TrimSpace(stringValue(obj["size"])); size != "" {
		meta["asset_size"] = size
	}
	if url := strings.TrimSpace(stringValue(obj["browser_download_url"])); url != "" {
		meta["download_url"] = url
		meta["source_kind"] = "downloaded"
		meta["recoverable"] = "true"
	}
	if apiURL := strings.TrimSpace(stringValue(obj["url"])); apiURL != "" {
		meta["asset_api_url"] = apiURL
		if meta["source_kind"] == "" {
			meta["source_kind"] = "downloaded"
			meta["recoverable"] = "true"
		}
	}
	return meta, nil
}

type releaseAssetRestorePayload struct {
	ReleaseID string
	Payload   json.RawMessage
}

func buildReleaseAssetRestorePayload(ctx context.Context, client *Client, mutation *platform.AdminMutationResult) (releaseAssetRestorePayload, error) {
	if mutation == nil {
		return releaseAssetRestorePayload{}, fmt.Errorf("release asset restore requires mutation result")
	}
	releaseID := strings.TrimSpace(mutation.Metadata["release_id"])
	name := strings.TrimSpace(mutation.Metadata["asset_name"])
	if releaseID == "" || name == "" {
		return releaseAssetRestorePayload{}, fmt.Errorf("release asset restore requires release id and asset name")
	}
	contentType := strings.TrimSpace(mutation.Metadata["content_type"])
	label := strings.TrimSpace(mutation.Metadata["asset_label"])
	payload := map[string]any{
		"release_id":   releaseID,
		"name":         name,
		"label":        label,
		"content_type": firstNonEmpty(contentType, "application/octet-stream"),
	}
	switch {
	case strings.TrimSpace(mutation.Metadata["stored_bytes_ref"]) != "":
		ref := strings.TrimSpace(mutation.Metadata["stored_bytes_ref"])
		if strings.HasPrefix(ref, "release-asset:") {
			entry, err := platform.ResolveReleaseAssetRef(ref)
			if err != nil {
				return releaseAssetRestorePayload{}, err
			}
			data, err := os.ReadFile(strings.TrimSpace(entry.BytesPath))
			if err != nil {
				return releaseAssetRestorePayload{}, err
			}
			payload["content_base64"] = base64.StdEncoding.EncodeToString(data)
		} else {
			data, err := os.ReadFile(ref)
			if err != nil {
				return releaseAssetRestorePayload{}, err
			}
			payload["content_base64"] = base64.StdEncoding.EncodeToString(data)
		}
	case strings.TrimSpace(mutation.Metadata["file_path"]) != "":
		payload["file_path"] = strings.TrimSpace(mutation.Metadata["file_path"])
	case strings.TrimSpace(mutation.Metadata["content_base64"]) != "":
		payload["content_base64"] = strings.TrimSpace(mutation.Metadata["content_base64"])
	case strings.TrimSpace(mutation.Metadata["asset_api_url"]) != "":
		data, err := client.downloadBytes(ctx, strings.TrimSpace(mutation.Metadata["asset_api_url"]), "application/octet-stream", http.StatusOK, http.StatusFound, http.StatusTemporaryRedirect)
		if err != nil {
			return releaseAssetRestorePayload{}, err
		}
		payload["content_base64"] = base64.StdEncoding.EncodeToString(data)
	case strings.TrimSpace(mutation.Metadata["download_url"]) != "":
		data, err := client.downloadBytes(ctx, strings.TrimSpace(mutation.Metadata["download_url"]), "application/octet-stream", http.StatusOK)
		if err != nil {
			return releaseAssetRestorePayload{}, err
		}
		payload["content_base64"] = base64.StdEncoding.EncodeToString(data)
	default:
		return releaseAssetRestorePayload{}, fmt.Errorf("release asset restore requires file_path, content_base64, or downloadable asset URL")
	}
	raw, err := marshalRaw(payload)
	if err != nil {
		return releaseAssetRestorePayload{}, err
	}
	return releaseAssetRestorePayload{ReleaseID: releaseID, Payload: raw}, nil
}

func validateReleaseDraftState(raw json.RawMessage, wantDraft bool) (bool, string, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return false, "", err
	}
	actual := boolFromAny(obj["draft"])
	ok := actual == wantDraft
	summary := "release draft state mismatch"
	if ok {
		summary = "release draft state validated"
	}
	return ok, summary, nil
}

func (e releaseExecutor) inspectReleaseAsset(ctx context.Context, assetID string) (json.RawMessage, error) {
	return e.client.doRaw(ctx, http.MethodGet, e.client.repoPath("/releases/assets/"+trimResourceID(assetID)), nil, http.StatusOK)
}
