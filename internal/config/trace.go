package config

import (
	"os"
	"sort"
	"strings"
)

// LoadTrace explains where the effective config came from.
type LoadTrace struct {
	DefaultsApplied bool          `json:"defaults_applied"`
	MergedFiles     []string      `json:"merged_files"`
	EnvOverrides    []string      `json:"env_overrides"`
	Migration       MigrationInfo `json:"migration"`
}

var lastLoadTrace *LoadTrace

// LastLoadTrace returns a snapshot of the latest load trace.
func LastLoadTrace() *LoadTrace {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	if lastLoadTrace == nil {
		return nil
	}
	cp := *lastLoadTrace
	cp.MergedFiles = append([]string(nil), lastLoadTrace.MergedFiles...)
	cp.EnvOverrides = append([]string(nil), lastLoadTrace.EnvOverrides...)
	return &cp
}

func NewLoadTrace() *LoadTrace {
	return &LoadTrace{
		MergedFiles:  make([]string, 0, 4),
		EnvOverrides: make([]string, 0, 8),
	}
}

func detectConfigEnvVars() []string {
	envs := os.Environ()
	out := make([]string, 0, 8)
	for _, kv := range envs {
		key := kv
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			key = kv[:idx]
		}
		if strings.HasPrefix(key, "GITDEX_") || strings.HasPrefix(key, "GITMANUAL_") {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

