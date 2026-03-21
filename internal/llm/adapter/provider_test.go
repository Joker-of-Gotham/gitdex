package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockProvider_DefaultResponse(t *testing.T) {
	mock := &MockProvider{}
	resp, err := mock.ChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "This is a mock response." {
		t.Errorf("Content = %q, want default mock response", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", resp.FinishReason)
	}
}

func TestMockProvider_CustomFunction(t *testing.T) {
	mock := &MockProvider{
		ChatCompletionFn: func(_ context.Context, req ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{
				Content:      "custom: " + req.Messages[0].Content,
				FinishReason: "stop",
			}, nil
		},
	}

	resp, err := mock.ChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "custom: test" {
		t.Errorf("Content = %q, want 'custom: test'", resp.Content)
	}
}

func TestMockProvider_StreamDefault(t *testing.T) {
	mock := &MockProvider{}
	ch, err := mock.StreamChatCompletion(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, ok := <-ch
	if !ok {
		t.Fatal("channel closed before receiving response")
	}
	if resp.Content != "This is a mock streamed response." {
		t.Errorf("Content = %q", resp.Content)
	}

	if _, ok := <-ch; ok {
		t.Error("channel should be closed after single response")
	}
}

func TestMockProvider_Error(t *testing.T) {
	mock := &MockProvider{
		ChatCompletionFn: func(_ context.Context, _ ChatRequest) (*ChatResponse, error) {
			return nil, fmt.Errorf("provider error")
		},
	}

	_, err := mock.ChatCompletion(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "provider error" {
		t.Errorf("error = %q, want 'provider error'", err)
	}
}

func TestOpenAIProvider_ChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s, want /chat/completions", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("auth = %q, want 'Bearer test-key'", auth)
		}

		resp := openAIResponse{
			Choices: []openAIChoice{
				{
					Message:      ChatMessage{Role: "assistant", Content: "Hello from OpenAI!"},
					FinishReason: "stop",
				},
			},
			Usage: openAIUsage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{
		APIKey:   "test-key",
		Endpoint: server.URL,
		Model:    "test-model",
	})

	resp, err := provider.ChatCompletion(context.Background(), ChatRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "Hi"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from OpenAI!" {
		t.Errorf("Content = %q, want 'Hello from OpenAI!'", resp.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestOpenAIProvider_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{
		APIKey:   "bad-key",
		Endpoint: server.URL,
	})

	_, err := provider.ChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestOpenAIProvider_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIResponse{Choices: []openAIChoice{}})
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{Endpoint: server.URL})
	_, err := provider.ChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error for no choices")
	}
}

func TestOpenAIProvider_DefaultEndpointAndModel(t *testing.T) {
	p := NewOpenAIProvider(OpenAIConfig{})
	if p.config.Endpoint != "https://api.openai.com/v1" {
		t.Errorf("default endpoint = %q", p.config.Endpoint)
	}
	if p.config.Model != "gpt-4o-mini" {
		t.Errorf("default model = %q", p.config.Model)
	}
}

func TestOpenAIProvider_StreamFallsBackToNonStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := openAIResponse{
			Choices: []openAIChoice{
				{Message: ChatMessage{Role: "assistant", Content: "streamed"}, FinishReason: "stop"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{Endpoint: server.URL})
	ch, err := provider.StreamChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := <-ch
	if resp.Content != "streamed" {
		t.Errorf("Content = %q, want 'streamed'", resp.Content)
	}
}
