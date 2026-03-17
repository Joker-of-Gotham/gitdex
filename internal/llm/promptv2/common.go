package promptv2

import (
	"runtime"
	"strings"
)

func languageName(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh", "zh-cn", "zh-hans":
		return "Simplified Chinese"
	case "ja":
		return "Japanese"
	default:
		return "English"
	}
}

// platformGuidance returns a one-line OS identifier for prompt injection.
// All detailed platform rules (allowed commands, tool restrictions) are
// enforced by the executor code layer, not the prompt.
func platformGuidance() string {
	switch runtime.GOOS {
	case "windows":
		return "PLATFORM: Windows. Use file_read/file_write for all file operations."
	default:
		return "PLATFORM: Unix/Linux/macOS. Prefer file_read/file_write for file modifications."
	}
}

// outputSchema returns the shared JSON output schema for planner prompts (B/D).
func outputSchema() string {
	return `OUTPUT FORMAT (strict JSON, no markdown fences):
{
  "analysis": "brief situational summary",
  "suggestions": [
    {
      "name": "human-readable action title",
      "action": {
        "type": "<tool type>",
        "command": "complete command string (for git_command/shell_command/github_op)",
        "file_path": "relative/path (for file_write/file_read)",
        "file_content": "full file content (for file_write create/update/append)",
        "file_operation": "create|update|delete|append|mkdir (for file_write)"
      },
      "reason": "why this action is needed"
    }
  ]
}`
}
