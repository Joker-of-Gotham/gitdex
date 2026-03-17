package config

import (
	"os"
	"strings"
	"unicode"
)

const (
	DefaultOllamaEndpoint   = "http://localhost:11434"
	DefaultOpenAIEndpoint   = "https://api.openai.com/v1"
	DefaultDeepSeekEndpoint = "https://api.deepseek.com"
	DefaultProvider         = "ollama"
	DefaultModel            = "qwen2.5:3b"
)

func (c LLMConfig) PrimaryRole() ModelConfig {
	role := c.Primary
	if strings.TrimSpace(role.Provider) == "" {
		role.Provider = c.Provider
	}
	if strings.TrimSpace(role.Endpoint) == "" {
		role.Endpoint = c.Endpoint
	}
	if strings.TrimSpace(role.APIKey) == "" {
		role.APIKey = c.APIKey
	}
	if strings.TrimSpace(role.APIKeyEnv) == "" {
		role.APIKeyEnv = c.APIKeyEnv
	}
	return role
}

func (c LLMConfig) SecondaryRole() ModelConfig {
	role := c.Secondary
	if strings.TrimSpace(role.Provider) == "" {
		role.Provider = c.Provider
	}
	if strings.TrimSpace(role.Endpoint) == "" {
		role.Endpoint = c.Endpoint
	}
	if strings.TrimSpace(role.APIKey) == "" {
		role.APIKey = c.APIKey
	}
	if strings.TrimSpace(role.APIKeyEnv) == "" {
		role.APIKeyEnv = c.APIKeyEnv
	}
	return role
}

func ResolveRoleAPIKey(role ModelConfig) string {
	if key := stripWrappingQuotes(strings.TrimSpace(role.APIKey)); key != "" {
		return key
	}
	if keyOrEnv := stripWrappingQuotes(strings.TrimSpace(role.APIKeyEnv)); keyOrEnv != "" {
		// Prefer env-var lookup first when APIKeyEnv is a variable name.
		if fromEnv := strings.TrimSpace(os.Getenv(keyOrEnv)); fromEnv != "" {
			return fromEnv
		}
		// Backward-compatible fallback: some historical configs stored a literal
		// API key in api_key_env by mistake. Accept it to avoid hard failure.
		if looksLikeLiteralAPIKey(keyOrEnv) {
			return keyOrEnv
		}
	}
	switch strings.ToLower(strings.TrimSpace(role.Provider)) {
	case "openai":
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	case "deepseek":
		return strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	default:
		return ""
	}
}

func stripWrappingQuotes(v string) string {
	v = strings.TrimSpace(v)
	if len(v) < 2 {
		return v
	}
	if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) ||
		(strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
		return strings.TrimSpace(v[1 : len(v)-1])
	}
	return v
}

func looksLikeLiteralAPIKey(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	if strings.ContainsAny(v, " \t\r\n") {
		return false
	}
	// Common API key prefixes used by cloud providers.
	if strings.HasPrefix(v, "sk-") || strings.HasPrefix(v, "sk_") ||
		strings.HasPrefix(v, "dsk_") || strings.HasPrefix(v, "dsk-") {
		return true
	}
	// Likely env var name: uppercase + underscores, no separators used in keys.
	if strings.EqualFold(v, strings.ToUpper(v)) && strings.Contains(v, "_") && !strings.Contains(v, "-") {
		return false
	}
	// Heuristic fallback for key-like tokens.
	if len(v) < 24 {
		return false
	}
	hasLower := false
	hasDigit := false
	hasSep := false
	for _, r := range v {
		if unicode.IsLower(r) {
			hasLower = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if r == '-' || r == '_' {
			hasSep = true
		}
	}
	return hasLower && hasDigit && hasSep
}

// LooksLikeLiteralAPIKey exposes key-shape detection for diagnostics/migration.
func LooksLikeLiteralAPIKey(v string) bool {
	return looksLikeLiteralAPIKey(v)
}

func RoleEndpoint(role ModelConfig) string {
	if endpoint := strings.TrimSpace(role.Endpoint); endpoint != "" {
		return endpoint
	}
	return defaultEndpointForProvider(role.Provider)
}

func RoleProvider(role ModelConfig) string {
	provider := strings.ToLower(strings.TrimSpace(role.Provider))
	if provider == "" {
		return "ollama"
	}
	return provider
}
