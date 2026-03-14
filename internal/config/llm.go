package config

import (
	"os"
	"strings"
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
	if key := strings.TrimSpace(role.APIKey); key != "" {
		return key
	}
	if envName := strings.TrimSpace(role.APIKeyEnv); envName != "" {
		return strings.TrimSpace(os.Getenv(envName))
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
