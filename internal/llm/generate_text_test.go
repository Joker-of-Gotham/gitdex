package llm

import (
	"context"
	"fmt"
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

func TestGenerateText_FatalStreamError_DoesNotFallback(t *testing.T) {
	_, err := GenerateText(context.Background(), stubProvider{
		resp:      &GenerateResponse{Text: "should not reach"},
		streamErr: fmt.Errorf("ollama: model \"bad\" not found (404)"),
	}, GenerateRequest{})
	if err == nil {
		t.Fatal("expected error for fatal stream error")
	}
	if !isFatalProviderError(err) {
		t.Fatalf("expected fatal error, got: %v", err)
	}
}

func TestIsFatalProviderError(t *testing.T) {
	cases := []struct {
		msg   string
		fatal bool
	}{
		{"ollama: model not found (404)", true},
		{"ollama: chat failed: status 404", true},
		{"openai: status 401: unauthorized", true},
		{"openai: status 403: forbidden", true},
		{"openai: missing API key", true},
		{"ollama: request failed: connection refused", false},
		{"context deadline exceeded", false},
	}
	for _, tc := range cases {
		got := isFatalProviderError(fmt.Errorf("%s", tc.msg))
		if got != tc.fatal {
			t.Errorf("isFatalProviderError(%q) = %v, want %v", tc.msg, got, tc.fatal)
		}
	}
}
