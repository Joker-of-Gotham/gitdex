package llm

import (
	"context"
	"fmt"
	"strings"
)

// Router dispatches requests to role-specific providers so primary and
// secondary models can use different backends.
type Router struct {
	primary   LLMProvider
	secondary LLMProvider
}

func NewRouter(primary, secondary LLMProvider) *Router {
	return &Router{
		primary:   primary,
		secondary: secondary,
	}
}

func (r *Router) Name() string {
	if r == nil {
		return ""
	}
	if r.primary == nil && r.secondary == nil {
		return ""
	}
	if r.secondary == nil || r.secondary == r.primary {
		return providerName(r.primary)
	}
	if providerName(r.secondary) == providerName(r.primary) {
		return providerName(r.primary)
	}
	return providerName(r.primary) + "+" + providerName(r.secondary)
}

func (r *Router) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	provider := r.providerForRole(req.Role)
	if provider == nil {
		return nil, fmt.Errorf("llm provider not configured for role %s", req.Role)
	}
	return provider.Generate(ctx, req)
}

func (r *Router) GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error) {
	provider := r.providerForRole(req.Role)
	if provider == nil {
		return nil, fmt.Errorf("llm provider not configured for role %s", req.Role)
	}
	return provider.GenerateStream(ctx, req)
}

func (r *Router) IsAvailable(ctx context.Context) bool {
	if r == nil || r.primary == nil {
		return false
	}
	return r.primary.IsAvailable(ctx)
}

func (r *Router) ModelInfo(ctx context.Context) (*ModelInfo, error) {
	provider := r.providerForRole(RolePrimary)
	if provider == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}
	return provider.ModelInfo(ctx)
}

func (r *Router) ListModels(ctx context.Context) ([]ModelInfo, error) {
	provider := r.providerForRole(RolePrimary)
	if provider == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}
	return provider.ListModels(ctx)
}

func (r *Router) SetModel(name string) {
	if r == nil || r.primary == nil {
		return
	}
	r.primary.SetModel(name)
}

func (r *Router) SetModelForRole(role ModelRole, name string) {
	if r == nil {
		return
	}
	switch role {
	case RoleSecondary:
		if r.secondary != nil {
			r.secondary.SetModelForRole(role, name)
			return
		}
		if r.primary != nil {
			r.primary.SetModelForRole(role, name)
		}
	default:
		if r.primary != nil {
			r.primary.SetModelForRole(role, name)
		}
	}
}

func (r *Router) providerForRole(role ModelRole) LLMProvider {
	if r == nil {
		return nil
	}
	switch role {
	case RoleSecondary:
		if r.secondary != nil {
			return r.secondary
		}
	}
	return r.primary
}

func providerName(provider LLMProvider) string {
	if provider == nil {
		return ""
	}
	return strings.TrimSpace(provider.Name())
}
