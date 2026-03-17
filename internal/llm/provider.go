package llm

import (
	"context"
	"fmt"
)

type ModelRole string

const (
	RolePrimary   ModelRole = "primary"
	RoleSecondary ModelRole = "secondary"
)

type GenerateRequest struct {
	Model       string
	Role        ModelRole
	System      string
	Prompt      string
	Temperature float64
}

type GenerateResponse struct {
	Text       string
	Thinking   string
	Raw        string
	TokenCount int
}

type StreamChunk struct {
	Text     string
	Thinking string
	Done     bool
}

type ModelInfo struct {
	Name      string
	Provider  string
	Size      int64
	Family    string
	ParamSize string
	Quant     string
}

type LLMProvider interface {
	Name() string
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
	IsAvailable(ctx context.Context) bool
	ModelInfo(ctx context.Context) (*ModelInfo, error)
	ListModels(ctx context.Context) ([]ModelInfo, error)
	SetModel(name string)
	SetModelForRole(role ModelRole, name string)
}

var ErrNoProvider = fmt.Errorf("no LLM provider configured — please configure one via /config or config.yaml")

// NopProvider is a nil-safe fallback that returns ErrNoProvider for every call.
type NopProvider struct{}

func (NopProvider) Name() string                                                       { return "none" }
func (NopProvider) Generate(_ context.Context, _ GenerateRequest) (*GenerateResponse, error) {
	return nil, ErrNoProvider
}
func (NopProvider) GenerateStream(_ context.Context, _ GenerateRequest) (<-chan StreamChunk, error) {
	return nil, ErrNoProvider
}
func (NopProvider) IsAvailable(_ context.Context) bool                   { return false }
func (NopProvider) ModelInfo(_ context.Context) (*ModelInfo, error)      { return nil, ErrNoProvider }
func (NopProvider) ListModels(_ context.Context) ([]ModelInfo, error)    { return nil, ErrNoProvider }
func (NopProvider) SetModel(_ string)                                    {}
func (NopProvider) SetModelForRole(_ ModelRole, _ string)                {}
