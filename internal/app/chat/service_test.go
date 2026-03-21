package chat

import (
	"context"
	"fmt"
	"testing"

	"github.com/your-org/gitdex/internal/app/session"
	"github.com/your-org/gitdex/internal/llm/adapter"
)

func TestService_Chat_SingleMessage(t *testing.T) {
	mock := &adapter.MockProvider{
		ChatCompletionFn: func(_ context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			if len(req.Messages) < 2 {
				t.Errorf("expected at least 2 messages (system + user), got %d", len(req.Messages))
			}
			if req.Messages[0].Role != "system" {
				t.Errorf("first message role = %q, want system", req.Messages[0].Role)
			}
			return &adapter.ChatResponse{
				Content:      "I can help with repository operations.",
				FinishReason: "stop",
				Usage:        adapter.TokenUsage{TotalTokens: 20},
			}, nil
		},
	}

	svc := NewService(mock)
	tc := session.NewTaskContext("/test/repo", "local")

	result, err := svc.Chat(context.Background(), tc, "What can you do?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "I can help with repository operations." {
		t.Errorf("Content = %q", result.Content)
	}
	if result.Role != "assistant" {
		t.Errorf("Role = %q, want assistant", result.Role)
	}
	if result.Context.RepoPath != "/test/repo" {
		t.Errorf("Context.RepoPath = %q", result.Context.RepoPath)
	}
}

func TestService_Chat_AccumulatesHistory(t *testing.T) {
	callCount := 0
	mock := &adapter.MockProvider{
		ChatCompletionFn: func(_ context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			callCount++
			userMsgCount := 0
			for _, m := range req.Messages {
				if m.Role == "user" {
					userMsgCount++
				}
			}
			if callCount == 2 && userMsgCount != 2 {
				t.Errorf("second call should have 2 user messages, got %d", userMsgCount)
			}
			return &adapter.ChatResponse{
				Content:      fmt.Sprintf("response %d", callCount),
				FinishReason: "stop",
			}, nil
		},
	}

	svc := NewService(mock)
	tc := session.NewTaskContext("", "")

	if _, err := svc.Chat(context.Background(), tc, "first"); err != nil {
		t.Fatalf("first chat error: %v", err)
	}
	if _, err := svc.Chat(context.Background(), tc, "second"); err != nil {
		t.Fatalf("second chat error: %v", err)
	}

	history := tc.GetChatHistory()
	if len(history) != 4 {
		t.Errorf("expected 4 chat messages (2 user + 2 assistant), got %d", len(history))
	}
}

func TestService_Chat_NilProvider(t *testing.T) {
	svc := NewService(nil)
	tc := session.NewTaskContext("", "")

	_, err := svc.Chat(context.Background(), tc, "hello")
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
}

func TestService_Chat_ProviderError(t *testing.T) {
	mock := &adapter.MockProvider{
		ChatCompletionFn: func(_ context.Context, _ adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return nil, fmt.Errorf("api error")
		},
	}

	svc := NewService(mock)
	tc := session.NewTaskContext("", "")

	_, err := svc.Chat(context.Background(), tc, "hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestService_Chat_SystemPromptContainsRepoContext(t *testing.T) {
	var receivedSystem string
	mock := &adapter.MockProvider{
		ChatCompletionFn: func(_ context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			for _, m := range req.Messages {
				if m.Role == "system" {
					receivedSystem = m.Content
					break
				}
			}
			return &adapter.ChatResponse{Content: "ok", FinishReason: "stop"}, nil
		},
	}

	svc := NewService(mock)
	tc := session.NewTaskContext("/my/repo", "production")

	if _, err := svc.Chat(context.Background(), tc, "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedSystem == "" {
		t.Fatal("no system message received")
	}
	if len(receivedSystem) < 100 {
		t.Error("system prompt seems too short")
	}
}

func TestService_Chat_CommandResultInContext(t *testing.T) {
	var receivedMessages []adapter.ChatMessage
	mock := &adapter.MockProvider{
		ChatCompletionFn: func(_ context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			receivedMessages = req.Messages
			return &adapter.ChatResponse{Content: "noted", FinishReason: "stop"}, nil
		},
	}

	svc := NewService(mock)
	tc := session.NewTaskContext("", "")

	tc.InjectCommandResult("doctor", nil, "all checks pass")

	if _, err := svc.Chat(context.Background(), tc, "what did doctor say?"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	systemCount := 0
	for _, m := range receivedMessages {
		if m.Role == "system" {
			systemCount++
		}
	}
	if systemCount < 1 {
		t.Error("expected at least one system message with command context")
	}
}

func TestBuildMessages_FiltersSystemFromHistory(t *testing.T) {
	tc := session.NewTaskContext("", "")
	tc.AddChatMessage(session.ChatMessage{Role: "system", Content: "injected"})
	tc.AddChatMessage(session.ChatMessage{Role: "user", Content: "hello"})
	tc.AddChatMessage(session.ChatMessage{Role: "assistant", Content: "hi"})

	messages := buildMessages(tc)

	systemCount := 0
	for _, m := range messages {
		if m.Role == "system" {
			systemCount++
		}
	}
	if systemCount != 1 {
		t.Errorf("expected exactly 1 system message (the generated prompt), got %d", systemCount)
	}
}
