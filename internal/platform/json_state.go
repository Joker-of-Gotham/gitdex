package platform

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func RawObject(raw json.RawMessage) (map[string]any, error) {
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

func MarshalRaw(v any) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func CloneRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func CloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func StringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func ExtractResourceID(raw json.RawMessage, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	for _, key := range []string{"id", "iid", "name", "slug", "uuid", "node_id"} {
		value := strings.TrimSpace(StringValue(obj[key]))
		if value != "" {
			return value
		}
	}
	return ""
}

func SubsetMatches(actualRaw, expectedRaw json.RawMessage) (bool, string, error) {
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
