package chat

import (
	"context"
	"fmt"

	"github.com/your-org/gitdex/internal/app/session"
	"github.com/your-org/gitdex/internal/llm/adapter"
	"github.com/your-org/gitdex/internal/llm/guardrails"
)

const defaultMaxTokens = 1024
const maxChatHistory = 50

type Service struct {
	provider adapter.Provider
}

func NewService(provider adapter.Provider) *Service {
	return &Service{provider: provider}
}

type ChatResult struct {
	Role    string            `json:"role"`
	Content string            `json:"content"`
	Context ChatResultContext `json:"context,omitempty"`
}

type ChatResultContext struct {
	RepoPath string `json:"repo_path,omitempty"`
	Profile  string `json:"profile,omitempty"`
}

func (s *Service) Chat(ctx context.Context, tc *session.TaskContext, userMessage string) (*ChatResult, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("no LLM provider configured; set llm.api_key in config or GITDEX_LLM_API_KEY env var")
	}

	tc.AddChatMessage(session.ChatMessage{Role: "user", Content: userMessage})
	tc.TruncateChatHistory(maxChatHistory)

	messages := buildMessages(tc)

	resp, err := s.provider.ChatCompletion(ctx, adapter.ChatRequest{
		Messages:    messages,
		MaxTokens:   defaultMaxTokens,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("llm chat completion: %w", err)
	}

	tc.AddChatMessage(session.ChatMessage{Role: "assistant", Content: resp.Content})

	return &ChatResult{
		Role:    "assistant",
		Content: resp.Content,
		Context: ChatResultContext{
			RepoPath: tc.GetRepoPath(),
			Profile:  tc.GetProfile(),
		},
	}, nil
}

func buildMessages(tc *session.TaskContext) []adapter.ChatMessage {
	systemPrompt := guardrails.BuildSystemPrompt(tc)

	history := tc.GetChatHistory()

	messages := make([]adapter.ChatMessage, 0, len(history)+1)
	messages = append(messages, adapter.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	for _, msg := range history {
		if msg.Role == "system" {
			continue
		}
		messages = append(messages, adapter.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}
