package chat

import (
	"testing"
)

func TestNewSession(t *testing.T) {
	s := NewSession("you are helpful")
	if s.MessageCount() != 0 {
		t.Errorf("expected 0 messages, got %d", s.MessageCount())
	}
	ctx := s.GetContext()
	if len(ctx) != 1 {
		t.Fatalf("expected 1 system message in context, got %d", len(ctx))
	}
	if ctx[0].Role != "system" || ctx[0].Content != "you are helpful" {
		t.Errorf("unexpected system message: %+v", ctx[0])
	}
}

func TestSession_AddMessage(t *testing.T) {
	s := NewSession("")
	s.AddMessage("user", "hello")
	s.AddMessage("assistant", "hi")
	if s.MessageCount() != 2 {
		t.Fatalf("expected 2 messages, got %d", s.MessageCount())
	}
	ctx := s.GetContext()
	if len(ctx) != 2 {
		t.Fatalf("expected 2 context messages, got %d", len(ctx))
	}
	if ctx[0].Content != "hello" || ctx[1].Content != "hi" {
		t.Errorf("unexpected context: %+v", ctx)
	}
}

func TestSession_SlidingWindow(t *testing.T) {
	s := NewSession("")
	s.SetMaxMessages(3)
	for i := 0; i < 10; i++ {
		s.AddMessage("user", "msg")
	}
	if s.MessageCount() != 3 {
		t.Errorf("expected 3 messages after trim, got %d", s.MessageCount())
	}
}

func TestSession_Clear(t *testing.T) {
	s := NewSession("sys")
	s.AddMessage("user", "hello")
	s.AddMessage("assistant", "hi")
	s.Clear()
	if s.MessageCount() != 0 {
		t.Errorf("expected 0 after clear, got %d", s.MessageCount())
	}
	ctx := s.GetContext()
	if len(ctx) != 1 {
		t.Fatalf("expected only system in context after clear, got %d", len(ctx))
	}
}

func TestSession_TokenTrim(t *testing.T) {
	s := NewSession("")
	s.SetMaxTokens(20)
	s.SetMaxMessages(100)
	for i := 0; i < 50; i++ {
		s.AddMessage("user", "this is a long message that takes tokens")
	}
	if s.MessageCount() >= 50 {
		t.Errorf("expected messages to be trimmed by token limit, got %d", s.MessageCount())
	}
}

func TestSession_SystemPromptInContext(t *testing.T) {
	s := NewSession("be concise")
	s.AddMessage("user", "question")
	ctx := s.GetContext()
	if len(ctx) != 2 {
		t.Fatalf("expected 2 (system+user), got %d", len(ctx))
	}
	if ctx[0].Role != "system" {
		t.Errorf("first should be system, got %s", ctx[0].Role)
	}
	if ctx[1].Role != "user" {
		t.Errorf("second should be user, got %s", ctx[1].Role)
	}
}

func TestSession_EmptySystemPrompt(t *testing.T) {
	s := NewSession("")
	s.AddMessage("user", "hello")
	ctx := s.GetContext()
	if len(ctx) != 1 {
		t.Fatalf("expected 1 (no system), got %d", len(ctx))
	}
}
