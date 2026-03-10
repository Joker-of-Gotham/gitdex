package llm

import "context"

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
	TokenCount int
}

type StreamChunk struct {
	Text string
	Done bool
}

type ModelInfo struct {
	Name      string
	Size      int64
	Family    string
	ParamSize string
	Quant     string
}

type LLMProvider interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
	IsAvailable(ctx context.Context) bool
	ModelInfo(ctx context.Context) (*ModelInfo, error)
	ListModels(ctx context.Context) ([]ModelInfo, error)
	SetModel(name string)
	SetModelForRole(role ModelRole, name string)
}
