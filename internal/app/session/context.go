package session

import (
	"sync"
	"time"
)

type CommandRecord struct {
	Command   string    `json:"command"`
	Args      []string  `json:"args,omitempty"`
	Output    string    `json:"output,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type TaskContext struct {
	mu             sync.RWMutex
	RepoPath       string            `json:"repo_path"`
	Profile        string            `json:"profile"`
	SessionStart   time.Time         `json:"session_start"`
	CommandHistory []CommandRecord   `json:"command_history"`
	ChatHistory    []ChatMessage     `json:"chat_history"`
	DiagnosticData map[string]any    `json:"diagnostic_data,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewTaskContext(repoPath, profile string) *TaskContext {
	return &TaskContext{
		RepoPath:       repoPath,
		Profile:        profile,
		SessionStart:   time.Now(),
		CommandHistory: make([]CommandRecord, 0),
		ChatHistory:    make([]ChatMessage, 0),
		DiagnosticData: make(map[string]any),
		Metadata:       make(map[string]string),
	}
}

func (tc *TaskContext) AddCommandRecord(rec CommandRecord) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	tc.CommandHistory = append(tc.CommandHistory, rec)
}

func (tc *TaskContext) AddChatMessage(msg ChatMessage) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.ChatHistory = append(tc.ChatHistory, msg)
}

func (tc *TaskContext) SetDiagnosticData(key string, value any) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.DiagnosticData[key] = value
}

func (tc *TaskContext) GetDiagnosticData(key string) (any, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	v, ok := tc.DiagnosticData[key]
	return v, ok
}

func (tc *TaskContext) SetMetadata(key, value string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.Metadata[key] = value
}

func (tc *TaskContext) GetMetadata(key string) (string, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	v, ok := tc.Metadata[key]
	return v, ok
}

func (tc *TaskContext) GetRepoPath() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.RepoPath
}

func (tc *TaskContext) GetProfile() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.Profile
}

func (tc *TaskContext) GetChatHistory() []ChatMessage {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	out := make([]ChatMessage, len(tc.ChatHistory))
	copy(out, tc.ChatHistory)
	return out
}

func (tc *TaskContext) RecentCommands(n int) []CommandRecord {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if n <= 0 || len(tc.CommandHistory) == 0 {
		return nil
	}
	start := len(tc.CommandHistory) - n
	if start < 0 {
		start = 0
	}
	out := make([]CommandRecord, len(tc.CommandHistory[start:]))
	copy(out, tc.CommandHistory[start:])
	return out
}

// TruncateChatHistory keeps the system message (if any) at index 0
// and retains only the most recent maxMessages non-system entries.
func (tc *TaskContext) TruncateChatHistory(maxMessages int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if len(tc.ChatHistory) <= maxMessages {
		return
	}

	var sysMsg *ChatMessage
	start := 0
	if len(tc.ChatHistory) > 0 && tc.ChatHistory[0].Role == "system" {
		sysMsg = &tc.ChatHistory[0]
		start = 1
	}

	tail := tc.ChatHistory[start:]
	keep := maxMessages
	if sysMsg != nil {
		keep = maxMessages - 1
	}
	if keep < 0 {
		keep = 0
	}
	if len(tail) > keep {
		tail = tail[len(tail)-keep:]
	}

	if sysMsg != nil {
		tc.ChatHistory = append([]ChatMessage{*sysMsg}, tail...)
	} else {
		tc.ChatHistory = tail
	}
}

func (tc *TaskContext) InjectCommandResult(cmd string, args []string, output string) {
	tc.AddCommandRecord(CommandRecord{
		Command:   cmd,
		Args:      args,
		Output:    output,
		Timestamp: time.Now(),
	})
	tc.AddChatMessage(ChatMessage{
		Role:    "system",
		Content: "Command `" + cmd + "` was executed. Result:\n" + output,
	})
}
