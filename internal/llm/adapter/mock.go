package adapter

import "context"

type MockProvider struct {
	ChatCompletionFn       func(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	StreamChatCompletionFn func(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)
}

var _ Provider = (*MockProvider)(nil)

func (m *MockProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if m.ChatCompletionFn != nil {
		return m.ChatCompletionFn(ctx, req)
	}
	return &ChatResponse{
		Content:      "This is a mock response.",
		FinishReason: "stop",
		Usage:        TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}, nil
}

func (m *MockProvider) StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	if m.StreamChatCompletionFn != nil {
		return m.StreamChatCompletionFn(ctx, req)
	}
	ch := make(chan ChatResponse, 1)
	ch <- ChatResponse{Content: "This is a mock streamed response.", FinishReason: "stop"}
	close(ch)
	return ch, nil
}
