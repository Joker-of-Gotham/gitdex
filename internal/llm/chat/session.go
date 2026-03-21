package chat

import (
	"sync"

	"github.com/your-org/gitdex/internal/llm/adapter"
)

const (
	DefaultMaxMessages = 20
	DefaultMaxTokens   = 4096
)

type Session struct {
	mu           sync.Mutex
	messages     []adapter.ChatMessage
	maxMessages  int
	maxTokens    int
	systemPrompt string
}

func NewSession(systemPrompt string) *Session {
	return &Session{
		maxMessages:  DefaultMaxMessages,
		maxTokens:    DefaultMaxTokens,
		systemPrompt: systemPrompt,
	}
}

func (s *Session) SetMaxMessages(n int) { s.mu.Lock(); s.maxMessages = n; s.mu.Unlock() }
func (s *Session) SetMaxTokens(n int)   { s.mu.Lock(); s.maxTokens = n; s.mu.Unlock() }

func (s *Session) AddMessage(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, adapter.ChatMessage{Role: role, Content: content})
	s.trim()
}

func (s *Session) GetContext() []adapter.ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []adapter.ChatMessage
	if s.systemPrompt != "" {
		out = append(out, adapter.ChatMessage{Role: "system", Content: s.systemPrompt})
	}
	out = append(out, s.messages...)
	return out
}

func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
}

func (s *Session) MessageCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

func (s *Session) trim() {
	if s.maxMessages > 0 && len(s.messages) > s.maxMessages {
		s.messages = s.messages[len(s.messages)-s.maxMessages:]
	}

	for s.estimateTokens() > s.maxTokens && len(s.messages) > 1 {
		s.messages = s.messages[1:]
	}
}

func (s *Session) estimateTokens() int {
	total := 0
	if s.systemPrompt != "" {
		total += len(s.systemPrompt) / 4
	}
	for _, m := range s.messages {
		total += len(m.Content) / 4
	}
	return total
}
