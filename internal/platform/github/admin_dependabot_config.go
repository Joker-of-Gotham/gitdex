package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

const dependabotConfigPath = ".github/dependabot.yml"

type dependabotConfigExecutor struct {
	client *Client
}

type repoContentResponse struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Encoding    string `json:"encoding"`
	Content     string `json:"content"`
	DownloadURL string `json:"download_url"`
}

func (e dependabotConfigExecutor) CapabilityID() string { return "dependabot_config" }

func (e dependabotConfigExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	raw, err := e.client.doRaw(ctx, http.MethodGet, e.contentsPath(), nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	state, err := normalizeDependabotSnapshot(raw)
	if err != nil {
		return nil, err
	}
	return snapshot(e.CapabilityID(), dependabotConfigPath, state), nil
}

func (e dependabotConfigExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	before, _ := e.Inspect(ctx, platform.AdminInspectRequest{})
	switch op {
	case "create", "update":
		plan, err := parseDependabotMutationPlan(req.Payload, before)
		if err != nil {
			return nil, err
		}
		if plan.NoOp {
			return &platform.AdminMutationResult{
				CapabilityID: e.CapabilityID(),
				Operation:    op,
				ResourceID:   dependabotConfigPath,
				Before:       before,
				After:        before,
				Metadata: map[string]string{
					"no_op": "true",
				},
			}, nil
		}
		payload := map[string]any{
			"message": plan.Message,
			"content": base64.StdEncoding.EncodeToString([]byte(plan.Content)),
		}
		if before != nil {
			if beforeObj, parseErr := rawObject(before.State); parseErr == nil {
				if sha := strings.TrimSpace(stringValue(beforeObj["sha"])); sha != "" {
					payload["sha"] = sha
				}
			}
		}
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.contentsPath(), payload, http.StatusOK, http.StatusCreated)
		if err != nil {
			return nil, err
		}
		after, err := normalizeDependabotCommitSnapshot(raw, plan.Content, plan.Message)
		if err != nil {
			return nil, err
		}
		return &platform.AdminMutationResult{
			CapabilityID: e.CapabilityID(),
			Operation:    op,
			ResourceID:   dependabotConfigPath,
			Before:       before,
			After:        snapshot(e.CapabilityID(), dependabotConfigPath, after),
		}, nil
	case "delete":
		if before == nil {
			return nil, fmt.Errorf("dependabot config does not exist")
		}
		payload, err := parseDependabotDeletePayload(req.Payload, before)
		if err != nil {
			return nil, err
		}
		if err := e.client.doJSON(ctx, http.MethodDelete, e.contentsPath(), payload, nil, http.StatusOK); err != nil {
			return nil, err
		}
		return &platform.AdminMutationResult{
			CapabilityID: e.CapabilityID(),
			Operation:    op,
			ResourceID:   dependabotConfigPath,
			Before:       before,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported dependabot config operation: %s", op)
	}
}

func (e dependabotConfigExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	snap, err := e.Inspect(ctx, platform.AdminInspectRequest{})
	if req.Mutation.Operation == "delete" {
		if inspectMissingOK(err) {
			return &platform.AdminValidationResult{OK: true, Summary: "dependabot config deleted", ResourceID: dependabotConfigPath}, nil
		}
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: dependabotConfigPath}, nil
		}
		return &platform.AdminValidationResult{OK: false, Summary: "dependabot config still exists", ResourceID: dependabotConfigPath, Snapshot: snap}, nil
	}
	if err != nil {
		return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: dependabotConfigPath}, nil
	}
	expected, err := expectedDependabotContentState(req.Payload, req.Mutation)
	if err != nil {
		return nil, err
	}
	matched, reason, matchErr := subsetMatches(snap.State, expected)
	if matchErr != nil {
		return nil, matchErr
	}
	return &platform.AdminValidationResult{
		OK:         matched,
		Summary:    summaryFromMatch(matched, reason, "dependabot config validated"),
		ResourceID: dependabotConfigPath,
		Snapshot:   snap,
	}, nil
}

func (e dependabotConfigExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	if req.Mutation.Before == nil {
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{Operation: "delete"}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "dependabot config deleted as rollback"}, nil
	}
	beforeState, err := rawObject(req.Mutation.Before.State)
	if err != nil {
		return nil, err
	}
	content, _ := beforeState["content"].(string)
	sha, _ := beforeState["sha"].(string)
	message := "Restore dependabot configuration via gitdex rollback"
	payload := map[string]any{
		"message": message,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
	}
	if strings.TrimSpace(sha) != "" {
		payload["sha"] = sha
	}
	raw, err := e.client.doRaw(ctx, http.MethodPut, e.contentsPath(), payload, http.StatusOK, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	state, err := normalizeDependabotCommitSnapshot(raw, content, message)
	if err != nil {
		return nil, err
	}
	return &platform.AdminRollbackResult{
		OK:       true,
		Summary:  "dependabot config restored",
		Snapshot: snapshot(e.CapabilityID(), dependabotConfigPath, state),
	}, nil
}

func (e dependabotConfigExecutor) contentsPath() string {
	return e.client.repoPath("/contents/" + dependabotConfigPath)
}

type dependabotMutationPlan struct {
	Content string
	Message string
	NoOp    bool
}

func parseDependabotMutationPlan(raw json.RawMessage, before *platform.AdminSnapshot) (dependabotMutationPlan, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return dependabotMutationPlan{}, err
	}
	content, err := dependabotStructuredContent(obj)
	if err != nil {
		return dependabotMutationPlan{}, err
	}
	if content == "" {
		return dependabotMutationPlan{}, fmt.Errorf("payload.content or payload.config is required")
	}
	message := strings.TrimSpace(stringValue(obj["message"]))
	if message == "" {
		message = "Update dependabot configuration via gitdex"
	}
	plan := dependabotMutationPlan{
		Content: content,
		Message: message,
	}
	if before != nil {
		if beforeObj, parseErr := rawObject(before.State); parseErr == nil {
			current := strings.TrimSpace(stringValue(beforeObj["deterministic_content"]))
			if current == "" {
				current = strings.TrimSpace(stringValue(beforeObj["content"]))
			}
			if current != "" && current == strings.TrimSpace(content) {
				plan.NoOp = true
			}
		}
	}
	return plan, nil
}

func parseDependabotDeletePayload(raw json.RawMessage, before *platform.AdminSnapshot) (map[string]any, error) {
	if before == nil {
		return nil, fmt.Errorf("existing dependabot config snapshot is required")
	}
	beforeObj, err := rawObject(before.State)
	if err != nil {
		return nil, err
	}
	sha := strings.TrimSpace(stringValue(beforeObj["sha"]))
	if sha == "" {
		return nil, fmt.Errorf("dependabot config sha is required for deletion")
	}
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	message := strings.TrimSpace(stringValue(obj["message"]))
	if message == "" {
		message = "Delete dependabot configuration via gitdex"
	}
	return map[string]any{
		"message": message,
		"sha":     sha,
	}, nil
}

func expectedDependabotContentState(raw json.RawMessage, mutation *platform.AdminMutationResult) (json.RawMessage, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	_, hasStructuredConfig := obj["config"]
	content, err := dependabotStructuredContent(obj)
	if err != nil {
		return nil, err
	}
	if content == "" && mutation != nil && mutation.After != nil {
		afterObj, parseErr := rawObject(mutation.After.State)
		if parseErr == nil {
			content = strings.TrimSpace(stringValue(afterObj["content"]))
		}
	}
	state := map[string]any{}
	if content != "" {
		if cfg, parseErr := platform.ParseDependabotConfigYAML(content); parseErr == nil {
			cfg = platform.NormalizeDependabotConfig(cfg)
			state["config"] = cfg
			if canonical, renderErr := platform.RenderDependabotConfigYAML(cfg); renderErr == nil {
				state["deterministic_content"] = canonical
			}
		}
		if _, ok := state["deterministic_content"]; !ok || !hasStructuredConfig {
			state["content"] = content
		}
	}
	return marshalRaw(state)
}

func normalizeDependabotSnapshot(raw json.RawMessage) (json.RawMessage, error) {
	var item repoContentResponse
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	content := decodeRepoContent(item.Content, item.Encoding)
	state := map[string]any{
		"path":         strings.TrimSpace(item.Path),
		"sha":          strings.TrimSpace(item.SHA),
		"content":      content,
		"download_url": strings.TrimSpace(item.DownloadURL),
	}
	if cfg, err := platform.ParseDependabotConfigYAML(content); err == nil {
		cfg = platform.NormalizeDependabotConfig(cfg)
		state["config"] = cfg
		state["ecosystems"] = dependabotEcosystems(cfg)
		state["directories"] = dependabotDirectories(cfg)
		canonical, renderErr := platform.RenderDependabotConfigYAML(cfg)
		if renderErr == nil {
			state["deterministic_content"] = canonical
		}
	} else if strings.TrimSpace(content) != "" {
		state["parse_error"] = err.Error()
	}
	return marshalRaw(state)
}

func normalizeDependabotCommitSnapshot(raw json.RawMessage, content, fallbackMessage string) (json.RawMessage, error) {
	var response struct {
		Content *struct {
			Path string `json:"path"`
			SHA  string `json:"sha"`
		} `json:"content"`
		Commit *struct {
			Message string `json:"message"`
			SHA     string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	state := map[string]any{
		"path":    dependabotConfigPath,
		"content": content,
	}
	if response.Content != nil {
		state["path"] = strings.TrimSpace(response.Content.Path)
		state["sha"] = strings.TrimSpace(response.Content.SHA)
	}
	if response.Commit != nil {
		state["commit_message"] = strings.TrimSpace(firstNonEmpty(response.Commit.Message, fallbackMessage))
		state["commit_sha"] = strings.TrimSpace(response.Commit.SHA)
	}
	if cfg, err := platform.ParseDependabotConfigYAML(content); err == nil {
		cfg = platform.NormalizeDependabotConfig(cfg)
		state["config"] = cfg
		state["ecosystems"] = dependabotEcosystems(cfg)
		state["directories"] = dependabotDirectories(cfg)
	}
	return marshalRaw(state)
}

func dependabotStructuredContent(obj map[string]any) (string, error) {
	if content := strings.TrimSpace(stringValue(obj["content"])); content != "" {
		cfg, err := platform.ParseDependabotConfigYAML(content)
		if err != nil {
			return "", err
		}
		return platform.RenderDependabotConfigYAML(cfg)
	}
	rawConfig, ok := obj["config"]
	if !ok {
		return "", nil
	}
	data, err := json.Marshal(rawConfig)
	if err != nil {
		return "", err
	}
	var cfg platform.DependabotConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	return platform.RenderDependabotConfigYAML(cfg)
}

func dependabotEcosystems(cfg platform.DependabotConfig) []string {
	items := make([]string, 0, len(cfg.Updates))
	for _, update := range cfg.Updates {
		items = append(items, update.Ecosystem)
	}
	return compactDependabotValues(items)
}

func dependabotDirectories(cfg platform.DependabotConfig) []string {
	items := make([]string, 0, len(cfg.Updates)*2)
	for _, update := range cfg.Updates {
		items = append(items, update.Directories...)
	}
	return compactDependabotValues(items)
}

func compactDependabotValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
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
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func decodeRepoContent(content, encoding string) string {
	trimmed := strings.ReplaceAll(content, "\n", "")
	if strings.EqualFold(strings.TrimSpace(encoding), "base64") {
		if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
			return string(decoded)
		}
	}
	return content
}
