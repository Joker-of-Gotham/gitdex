package git

import (
	"sort"
	"strings"
)

func PlatformExecIdentity(op *PlatformExecInfo) string {
	if op == nil {
		return ""
	}
	parts := []string{
		strings.ToLower(strings.TrimSpace(op.CapabilityID)),
		strings.ToLower(strings.TrimSpace(op.Flow)),
		strings.ToLower(strings.TrimSpace(op.Operation)),
		strings.ToLower(strings.TrimSpace(op.ResourceID)),
	}
	if len(op.Scope) > 0 {
		keys := make([]string, 0, len(op.Scope))
		for key := range op.Scope {
			keys = append(keys, strings.ToLower(strings.TrimSpace(key))+"="+strings.ToLower(strings.TrimSpace(op.Scope[key])))
		}
		sort.Strings(keys)
		parts = append(parts, keys...)
	}
	return strings.Join(parts, ":")
}
