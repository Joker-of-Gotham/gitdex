package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DiagnoseResult is machine-readable diagnostics output.
type DiagnoseResult struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
}

// Lint validates config and returns security/compatibility warnings.
func Lint(c *Config) DiagnoseResult {
	res := DiagnoseResult{Valid: true}
	if err := Validate(c); err != nil {
		res.Valid = false
		res.Warnings = append(res.Warnings, err.Error())
		return res
	}
	res.Warnings = append(res.Warnings, SensitiveFieldWarnings(c)...)
	sort.Strings(res.Warnings)
	return res
}

// Explain returns a human-readable source explanation report.
func Explain(c *Config, trace *LoadTrace) string {
	var b strings.Builder
	b.WriteString("GitDex Config Explain\n")
	b.WriteString("====================\n")
	if c != nil {
		b.WriteString(fmt.Sprintf("version: v%d\n", c.Version))
		b.WriteString(fmt.Sprintf("automation.mode: %s\n", c.Automation.Mode))
		b.WriteString(fmt.Sprintf("planner(primary): %s/%s\n", c.LLM.Primary.Provider, c.LLM.Primary.Model))
		b.WriteString(fmt.Sprintf("helper(secondary): %s/%s\n", c.LLM.Secondary.Provider, c.LLM.Secondary.Model))
	}
	if trace != nil {
		b.WriteString("\nsource precedence: defaults -> global -> project -> env -> cli\n")
		if len(trace.MergedFiles) == 0 {
			b.WriteString("merged files: (none)\n")
		} else {
			b.WriteString("merged files:\n")
			for _, f := range trace.MergedFiles {
				b.WriteString("  - " + f + "\n")
			}
		}
		if len(trace.EnvOverrides) > 0 {
			b.WriteString("env overrides:\n")
			for _, e := range trace.EnvOverrides {
				b.WriteString("  - " + e + "\n")
			}
		}
		if len(trace.Migration.Steps) > 0 {
			b.WriteString("migration steps:\n")
			for _, s := range trace.Migration.Steps {
				b.WriteString("  - " + s + "\n")
			}
		}
	}
	warns := SensitiveFieldWarnings(c)
	if len(warns) > 0 {
		b.WriteString("warnings:\n")
		for _, w := range warns {
			b.WriteString("  - " + w + "\n")
		}
	}
	return b.String()
}

// SourceTraceJSON returns source trace as JSON.
func SourceTraceJSON(trace *LoadTrace) (string, error) {
	if trace == nil {
		trace = &LoadTrace{}
	}
	buf, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// SensitiveFieldWarnings returns warnings for plaintext key usage patterns.
func SensitiveFieldWarnings(c *Config) []string {
	if c == nil {
		return nil
	}
	var warnings []string
	if strings.TrimSpace(c.LLM.APIKey) != "" {
		warnings = append(warnings, "llm.api_key is plaintext; prefer api_key_env")
	}
	if strings.TrimSpace(c.LLM.Primary.APIKey) != "" {
		warnings = append(warnings, "llm.primary.api_key is plaintext; prefer api_key_env")
	}
	if strings.TrimSpace(c.LLM.Secondary.APIKey) != "" {
		warnings = append(warnings, "llm.secondary.api_key is plaintext; prefer api_key_env")
	}
	if LooksLikeLiteralAPIKey(c.LLM.Primary.APIKeyEnv) {
		warnings = append(warnings, "llm.primary.api_key_env looks like literal key; migrate to env var name")
	}
	if LooksLikeLiteralAPIKey(c.LLM.Secondary.APIKeyEnv) {
		warnings = append(warnings, "llm.secondary.api_key_env looks like literal key; migrate to env var name")
	}
	return warnings
}

