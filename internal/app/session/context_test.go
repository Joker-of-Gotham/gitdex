package session

import (
	"testing"
	"time"
)

func TestNewTaskContext(t *testing.T) {
	tc := NewTaskContext("/repo", "local")
	if tc.RepoPath != "/repo" {
		t.Errorf("RepoPath = %q, want /repo", tc.RepoPath)
	}
	if tc.Profile != "local" {
		t.Errorf("Profile = %q, want local", tc.Profile)
	}
	if tc.SessionStart.IsZero() {
		t.Error("SessionStart should not be zero")
	}
	if len(tc.CommandHistory) != 0 {
		t.Errorf("CommandHistory should be empty, got %d", len(tc.CommandHistory))
	}
	if len(tc.ChatHistory) != 0 {
		t.Errorf("ChatHistory should be empty, got %d", len(tc.ChatHistory))
	}
}

func TestAddCommandRecord(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.AddCommandRecord(CommandRecord{Command: "doctor", Args: []string{}, Output: "pass"})
	tc.AddCommandRecord(CommandRecord{Command: "config", Args: []string{"show"}, Output: "text"})

	if len(tc.CommandHistory) != 2 {
		t.Fatalf("CommandHistory len = %d, want 2", len(tc.CommandHistory))
	}
	if tc.CommandHistory[0].Command != "doctor" {
		t.Errorf("first command = %q, want doctor", tc.CommandHistory[0].Command)
	}
	if tc.CommandHistory[0].Timestamp.IsZero() {
		t.Error("auto-filled timestamp should not be zero")
	}
}

func TestAddCommandRecord_PreservesTimestamp(t *testing.T) {
	tc := NewTaskContext("", "")
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tc.AddCommandRecord(CommandRecord{Command: "test", Timestamp: ts})
	if !tc.CommandHistory[0].Timestamp.Equal(ts) {
		t.Error("provided timestamp should be preserved")
	}
}

func TestAddChatMessage(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.AddChatMessage(ChatMessage{Role: "user", Content: "hello"})
	tc.AddChatMessage(ChatMessage{Role: "assistant", Content: "hi"})

	history := tc.GetChatHistory()
	if len(history) != 2 {
		t.Fatalf("ChatHistory len = %d, want 2", len(history))
	}
	if history[0].Role != "user" {
		t.Errorf("first message role = %q, want user", history[0].Role)
	}
}

func TestGetChatHistory_ReturnsCopy(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.AddChatMessage(ChatMessage{Role: "user", Content: "hello"})

	history := tc.GetChatHistory()
	history[0].Content = "modified"

	original := tc.GetChatHistory()
	if original[0].Content == "modified" {
		t.Error("GetChatHistory should return a copy, not a reference")
	}
}

func TestDiagnosticData(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.SetDiagnosticData("doctor", map[string]string{"status": "pass"})

	val, ok := tc.GetDiagnosticData("doctor")
	if !ok {
		t.Fatal("expected diagnostic data for key 'doctor'")
	}
	m, _ := val.(map[string]string)
	if m["status"] != "pass" {
		t.Errorf("diagnostic status = %q, want pass", m["status"])
	}

	_, ok = tc.GetDiagnosticData("missing")
	if ok {
		t.Error("expected no diagnostic data for key 'missing'")
	}
}

func TestMetadata(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.SetMetadata("key", "value")
	got, ok := tc.GetMetadata("key")
	if !ok {
		t.Fatal("expected metadata for key 'key'")
	}
	if got != "value" {
		t.Errorf("GetMetadata(key) = %q, want value", got)
	}

	_, ok = tc.GetMetadata("missing")
	if ok {
		t.Error("expected no metadata for key 'missing'")
	}
}

func TestRecentCommands(t *testing.T) {
	tc := NewTaskContext("", "")
	for i := range 10 {
		tc.AddCommandRecord(CommandRecord{Command: "cmd", Args: []string{string(rune('0' + i))}})
	}

	recent := tc.RecentCommands(3)
	if len(recent) != 3 {
		t.Fatalf("RecentCommands(3) len = %d, want 3", len(recent))
	}
	if recent[0].Args[0] != "7" {
		t.Errorf("first of recent 3 should be cmd 7, got %q", recent[0].Args[0])
	}
}

func TestRecentCommands_FewerThanN(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.AddCommandRecord(CommandRecord{Command: "only"})
	recent := tc.RecentCommands(5)
	if len(recent) != 1 {
		t.Fatalf("RecentCommands(5) with 1 entry = %d, want 1", len(recent))
	}
}

func TestRecentCommands_Zero(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.AddCommandRecord(CommandRecord{Command: "x"})
	if got := tc.RecentCommands(0); got != nil {
		t.Errorf("RecentCommands(0) should be nil, got %v", got)
	}
}

func TestRecentCommands_ReturnsCopy(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.AddCommandRecord(CommandRecord{Command: "original"})
	recent := tc.RecentCommands(1)
	recent[0].Command = "modified"
	if tc.CommandHistory[0].Command == "modified" {
		t.Error("RecentCommands should return a copy")
	}
}

func TestTruncateChatHistory_NoTruncation(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.AddChatMessage(ChatMessage{Role: "user", Content: "a"})
	tc.AddChatMessage(ChatMessage{Role: "assistant", Content: "b"})
	tc.TruncateChatHistory(10)
	if len(tc.ChatHistory) != 2 {
		t.Errorf("should not truncate when under limit, got %d", len(tc.ChatHistory))
	}
}

func TestTruncateChatHistory_WithSystemMessage(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.AddChatMessage(ChatMessage{Role: "system", Content: "sys"})
	for i := range 10 {
		tc.AddChatMessage(ChatMessage{Role: "user", Content: string(rune('a' + i))})
	}

	tc.TruncateChatHistory(5)

	if len(tc.ChatHistory) != 5 {
		t.Fatalf("after truncation len = %d, want 5", len(tc.ChatHistory))
	}
	if tc.ChatHistory[0].Role != "system" {
		t.Error("system message should be preserved at index 0")
	}
	if tc.ChatHistory[0].Content != "sys" {
		t.Error("system message content should be preserved")
	}
}

func TestTruncateChatHistory_WithoutSystemMessage(t *testing.T) {
	tc := NewTaskContext("", "")
	for i := range 10 {
		tc.AddChatMessage(ChatMessage{Role: "user", Content: string(rune('a' + i))})
	}

	tc.TruncateChatHistory(3)

	if len(tc.ChatHistory) != 3 {
		t.Fatalf("after truncation len = %d, want 3", len(tc.ChatHistory))
	}
	if tc.ChatHistory[0].Content != string(rune('a'+7)) {
		t.Errorf("first kept message = %q, want %q", tc.ChatHistory[0].Content, string(rune('a'+7)))
	}
}

func TestInjectCommandResult(t *testing.T) {
	tc := NewTaskContext("", "")
	tc.InjectCommandResult("doctor", nil, "all checks pass")

	if len(tc.CommandHistory) != 1 {
		t.Fatalf("CommandHistory len = %d, want 1", len(tc.CommandHistory))
	}
	if tc.CommandHistory[0].Command != "doctor" {
		t.Errorf("command = %q, want doctor", tc.CommandHistory[0].Command)
	}

	if len(tc.ChatHistory) != 1 {
		t.Fatalf("ChatHistory len = %d, want 1", len(tc.ChatHistory))
	}
	if tc.ChatHistory[0].Role != "system" {
		t.Errorf("injected chat role = %q, want system", tc.ChatHistory[0].Role)
	}
}
