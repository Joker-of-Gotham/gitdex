package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func (c *Client) AdminExecutors() map[string]platform.AdminExecutor {
	return map[string]platform.AdminExecutor{
		"actions":                         actionsExecutor{client: c},
		"rulesets":                        rulesetExecutor{client: c, capabilityID: "rulesets"},
		"branch_rulesets":                 rulesetExecutor{client: c, capabilityID: "branch_rulesets"},
		"check_runs_failure_threshold":    rulesetExecutor{client: c, capabilityID: "check_runs_failure_threshold"},
		"actions_secrets_variables":       actionsConfigExecutor{client: c},
		"codespaces":                      codespacesExecutor{client: c},
		"codespaces_secrets":              repoSecretExecutor{client: c, capabilityID: "codespaces_secrets", baseSegment: "codespaces"},
		"dependabot_secrets":              repoSecretExecutor{client: c, capabilityID: "dependabot_secrets", baseSegment: "dependabot"},
		"dependabot_config":               dependabotConfigExecutor{client: c},
		"webhooks":                        webhookExecutor{client: c},
		"pages":                           pagesExecutor{client: c},
		"deployment":                      deploymentExecutor{client: c},
		"environments":                    environmentExecutor{client: c},
		"release":                         releaseExecutor{client: c},
		"pull_request":                    pullRequestExecutor{client: c},
		"pr_review":                       prReviewExecutor{client: c},
		"deploy_keys":                     deployKeyExecutor{client: c},
		"packages":                        packagesExecutor{client: c},
		"notifications":                   notificationsExecutor{client: c, capabilityID: "notifications"},
		"email_notifications":             notificationsExecutor{client: c, capabilityID: "email_notifications"},
		"security":                        repoSecurityExecutor{client: c, capabilityID: "security"},
		"advanced_security":               repoSecurityExecutor{client: c, capabilityID: "advanced_security"},
		"dependency_graph":                repoSecurityExecutor{client: c, capabilityID: "dependency_graph"},
		"dependabot":                      repoSecurityExecutor{client: c, capabilityID: "dependabot"},
		"dependabot_posture":              repoSecurityExecutor{client: c, capabilityID: "dependabot_posture"},
		"dependabot_security_updates":     repoSecurityExecutor{client: c, capabilityID: "dependabot_security_updates"},
		"grouped_security_updates":        repoSecurityExecutor{client: c, capabilityID: "grouped_security_updates"},
		"dependabot_version_updates":      repoSecurityExecutor{client: c, capabilityID: "dependabot_version_updates"},
		"private_vulnerability_reporting": repoSecurityExecutor{client: c, capabilityID: "private_vulnerability_reporting"},
		"secret_scanning_settings":        repoSecurityExecutor{client: c, capabilityID: "secret_scanning_settings"},
		"code_scanning_tool_settings":     repoSecurityExecutor{client: c, capabilityID: "code_scanning_tool_settings"},
		"protection_rules":                repoSecurityExecutor{client: c, capabilityID: "protection_rules"},
		"push_protection":                 repoSecurityExecutor{client: c, capabilityID: "push_protection"},
		"dependabot_alerts":               alertExecutor{client: c, capabilityID: "dependabot_alerts", baseSegment: "dependabot/alerts"},
		"secret_scanning_alerts":          alertExecutor{client: c, capabilityID: "secret_scanning_alerts", baseSegment: "secret-scanning/alerts"},
		"code_scanning":                   codeScanningExecutor{client: c, capabilityID: "code_scanning"},
		"code_scanning_default_setup":     codeScanningExecutor{client: c, capabilityID: "code_scanning_default_setup"},
		"codeql_setup":                    codeScanningExecutor{client: c, capabilityID: "codeql_setup"},
		"codeql_analysis":                 codeScanningExecutor{client: c, capabilityID: "codeql_analysis"},
		"copilot_autofix":                 codeScanningExecutor{client: c, capabilityID: "copilot_autofix"},
		"secret_protection":               alertExecutor{client: c, capabilityID: "secret_protection", baseSegment: "secret-scanning/alerts"},
		"copilot_code_review":             copilotExecutor{client: c, capabilityID: "copilot_code_review"},
		"copilot_coding_agent":            copilotExecutor{client: c, capabilityID: "copilot_coding_agent"},
		"copilot_seat_management":         copilotExecutor{client: c, capabilityID: "copilot_seat_management"},
	}
}

func snapshot(capabilityID, resourceID string, raw json.RawMessage) *platform.AdminSnapshot {
	if raw == nil {
		return nil
	}
	return &platform.AdminSnapshot{
		CapabilityID: capabilityID,
		ResourceID:   strings.TrimSpace(resourceID),
		State:        append(json.RawMessage(nil), raw...),
	}
}

func trimResourceID(id string) string {
	return url.PathEscape(strings.TrimSpace(id))
}

func rawObject(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func marshalRaw(v any) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func extractResourceID(raw json.RawMessage, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	for _, key := range []string{"id", "name", "node_id"} {
		switch value := obj[key].(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		case float64:
			return fmt.Sprintf("%.0f", value)
		}
	}
	return ""
}

func subsetMatches(actualRaw, expectedRaw json.RawMessage) (bool, string, error) {
	if len(expectedRaw) == 0 {
		return true, "", nil
	}
	var actual any
	if err := json.Unmarshal(actualRaw, &actual); err != nil {
		return false, "", err
	}
	var expected any
	if err := json.Unmarshal(expectedRaw, &expected); err != nil {
		return false, "", err
	}
	ok, reason := subsetValueMatches(actual, expected, "")
	return ok, reason, nil
}

func subsetValueMatches(actual, expected any, path string) (bool, string) {
	switch exp := expected.(type) {
	case map[string]any:
		actMap, ok := actual.(map[string]any)
		if !ok {
			return false, path + ": type mismatch"
		}
		keys := make([]string, 0, len(exp))
		for key := range exp {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPath := key
			if path != "" {
				nextPath = path + "." + key
			}
			value, ok := actMap[key]
			if !ok {
				return false, nextPath + ": missing"
			}
			if matched, reason := subsetValueMatches(value, exp[key], nextPath); !matched {
				return false, reason
			}
		}
		return true, ""
	case []any:
		actSlice, ok := actual.([]any)
		if !ok {
			return false, path + ": type mismatch"
		}
		if len(exp) > len(actSlice) {
			return false, path + ": array shorter than expected"
		}
		for idx, item := range exp {
			nextPath := fmt.Sprintf("%s[%d]", path, idx)
			if matched, reason := subsetValueMatches(actSlice[idx], item, nextPath); !matched {
				return false, reason
			}
		}
		return true, ""
	default:
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return false, path + ": value mismatch"
		}
		return true, ""
	}
}

func deleteKeys(obj map[string]any, keys ...string) {
	for _, key := range keys {
		delete(obj, key)
	}
}

func keepOnlyKeys(obj map[string]any, keys ...string) map[string]any {
	if obj == nil {
		return map[string]any{}
	}
	allowed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		allowed[key] = struct{}{}
	}
	out := make(map[string]any, len(allowed))
	for key, value := range obj {
		if _, ok := allowed[key]; ok {
			out[key] = value
		}
	}
	return out
}

func inspectMissingOK(err error) bool {
	return err != nil && strings.Contains(err.Error(), fmt.Sprintf("status %d", http.StatusNotFound))
}

func normalizeScopeValue(scope map[string]string, key, fallback string) string {
	if scope == nil {
		return fallback
	}
	value := strings.TrimSpace(scope[key])
	if value == "" {
		return fallback
	}
	return value
}

func appendQuery(path string, query map[string]string, skipKeys ...string) string {
	if len(query) == 0 {
		return path
	}
	skip := make(map[string]struct{}, len(skipKeys))
	for _, key := range skipKeys {
		skip[strings.ToLower(strings.TrimSpace(key))] = struct{}{}
	}
	values := url.Values{}
	for key, value := range query {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if _, ok := skip[strings.ToLower(key)]; ok {
			continue
		}
		values.Set(key, value)
	}
	if len(values) == 0 {
		return path
	}
	if strings.Contains(path, "?") {
		return path + "&" + values.Encode()
	}
	return path + "?" + values.Encode()
}
