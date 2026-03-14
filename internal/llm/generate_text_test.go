package llm

import (
	"context"
	"testing"
)

type stubProvider struct {
	stream      chan StreamChunk
	resp        *GenerateResponse
	streamErr   error
	generateErr error
}

func (s stubProvider) Name() string { return "stub" }
func (s stubProvider) Generate(context.Context, GenerateRequest) (*GenerateResponse, error) {
	return s.resp, s.generateErr
}
func (s stubProvider) GenerateStream(context.Context, GenerateRequest) (<-chan StreamChunk, error) {
	if s.stream == nil {
		return nil, s.streamErr
	}
	return s.stream, nil
}
func (s stubProvider) IsAvailable(context.Context) bool { return true }
func (s stubProvider) ModelInfo(context.Context) (*ModelInfo, error) {
	return &ModelInfo{Name: "stub"}, nil
}
func (s stubProvider) ListModels(context.Context) ([]ModelInfo, error) { return nil, nil }
func (s stubProvider) SetModel(string)                                 {}
func (s stubProvider) SetModelForRole(ModelRole, string)               {}

func TestGenerateTextFallsBackToGenerate(t *testing.T) {
	resp, err := GenerateText(context.Background(), stubProvider{
		resp:      &GenerateResponse{Text: "ok"},
		streamErr: context.Canceled,
	}, GenerateRequest{})
	if err != nil || resp.Text != "ok" {
		t.Fatalf("unexpected response: %+v err=%v", resp, err)
	}
}

func TestGenerateTextAggregatesStream(t *testing.T) {
	stream := make(chan StreamChunk, 2)
	stream <- StreamChunk{Text: "hello ", Thinking: "step1\n"}
	stream <- StreamChunk{Text: "world", Thinking: "step2", Done: true}
	close(stream)

	resp, err := GenerateText(context.Background(), stubProvider{stream: stream}, GenerateRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "hello world" {
		t.Fatalf("unexpected text: %q", resp.Text)
	}
}
