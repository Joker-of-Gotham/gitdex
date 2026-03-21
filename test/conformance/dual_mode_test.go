package conformance_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/app/session"
	"github.com/your-org/gitdex/internal/cli/command"
	"github.com/your-org/gitdex/internal/cli/input"
	"github.com/your-org/gitdex/internal/llm/guardrails"
)

func TestDualMode_CommandAndNLClassification(t *testing.T) {
	root := command.NewRootCommand()
	parser := input.NewParser(root)

	commands := []string{"doctor", "chat hello", "capabilities", "version", "config show"}
	for _, cmd := range commands {
		result := parser.Classify(cmd)
		if result.Type != input.InputCommand {
			t.Errorf("Classify(%q) = %v, want InputCommand", cmd, result.Type)
		}
	}

	nlInputs := []string{
		"what is the status of this repo?",
		"help me understand diagnostics",
		"how do I sync upstream?",
	}
	for _, nl := range nlInputs {
		result := parser.Classify(nl)
		if result.Type != input.InputNaturalLanguage {
			t.Errorf("Classify(%q) = %v, want InputNaturalLanguage", nl, result.Type)
		}
	}
}

func TestDualMode_TaskContextSharedBetweenModes(t *testing.T) {
	tc := session.NewTaskContext("/test/repo", "local")

	tc.InjectCommandResult("doctor", nil, "all checks pass")

	tc.AddChatMessage(session.ChatMessage{Role: "user", Content: "what did doctor say?"})

	history := tc.GetChatHistory()
	if len(history) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(history))
	}

	commandRecords := tc.RecentCommands(1)
	if len(commandRecords) != 1 || commandRecords[0].Command != "doctor" {
		t.Error("command record should be preserved in TaskContext")
	}
}

func TestDualMode_SystemPromptEnforcesBoundaries(t *testing.T) {
	prompt := guardrails.BaseSystemPrompt()

	boundaries := []string{
		"MUST NOT directly execute",
		"MUST NOT bypass structured plan",
		"MUST NOT generate code",
	}

	for _, boundary := range boundaries {
		if !strings.Contains(prompt, boundary) {
			t.Errorf("system prompt missing boundary: %q", boundary)
		}
	}
}

func TestDualMode_CapabilitiesJSONSchema(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--output", "json", "capabilities"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var result struct {
		Capabilities []struct {
			Command     string `json:"command"`
			Description string `json:"description"`
			Available   bool   `json:"available"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("capabilities JSON does not match expected schema: %v", err)
	}

	for _, cap := range result.Capabilities {
		if cap.Command == "" {
			t.Error("capability command must not be empty")
		}
		if cap.Description == "" {
			t.Error("capability description must not be empty")
		}
	}
}

func TestDualMode_ChatJSONSchema(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())
	t.Setenv("GITDEX_LLM_API_KEY", "")

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--output", "json", "chat", "hello"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var result struct {
		Role    string `json:"role"`
		Content string `json:"content"`
		Context struct {
			RepoPath string `json:"repo_path"`
			Profile  string `json:"profile"`
		} `json:"context"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("chat JSON does not match expected schema: %v", err)
	}

	if result.Role != "assistant" {
		t.Errorf("role = %q, want assistant", result.Role)
	}
	if result.Content == "" {
		t.Error("content must not be empty")
	}
}

func TestDualMode_ShellCompletionCoversNewCommands(t *testing.T) {
	root := command.NewRootCommand()

	newCommands := []string{"chat", "capabilities", "repo"}
	for _, name := range newCommands {
		found := false
		for _, cmd := range root.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not registered in root (shell completion will miss it)", name)
		}
	}
}
