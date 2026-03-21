package integration_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestChatSingleMessage_MockProvider(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())
	t.Setenv("GITDEX_LLM_API_KEY", "")

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"chat", "hello"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "mock response") {
		t.Errorf("expected mock response, got %q", output)
	}
}

func TestChatSingleMessage_JSONOutput(t *testing.T) {
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

	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("json decode failed: %v\nraw output: %s", err, out.String())
	}

	if _, ok := result["content"]; !ok {
		t.Error("JSON output missing 'content' field")
	}
	if _, ok := result["role"]; !ok {
		t.Error("JSON output missing 'role' field")
	}
}

func TestChatNoArgs_ReturnsError(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"chat"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no message and no --interactive")
	}
}

func TestChatInteractive_ExitCommand(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())
	t.Setenv("GITDEX_LLM_API_KEY", "")

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader("exit\n"))
	root.SetArgs([]string{"chat", "--interactive"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Goodbye") {
		t.Errorf("expected Goodbye message, got %q", output)
	}
}

func TestChatInteractive_SendMessageThenExit(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())
	t.Setenv("GITDEX_LLM_API_KEY", "")

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader("what can you do?\nexit\n"))
	root.SetArgs([]string{"chat", "--interactive"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "gitdex>") {
		t.Errorf("expected gitdex> response prefix, got %q", output)
	}
}

func TestChatInteractive_CommandDetection(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())
	t.Setenv("GITDEX_LLM_API_KEY", "")

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader("doctor\nexit\n"))
	root.SetArgs([]string{"chat", "--interactive"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "command detected") {
		t.Errorf("expected command detection message, got %q", output)
	}
}

func TestChatHelp(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"chat", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "natural language conversation") {
		t.Errorf("help should describe chat purpose, got %q", output)
	}
	if !strings.Contains(output, "--interactive") {
		t.Errorf("help should mention --interactive flag")
	}
}

func TestChatSingleMessage_OutputFormatFromEnv(t *testing.T) {
	t.Setenv("GITDEX_USER_CONFIG_DIR", t.TempDir())
	t.Setenv("GITDEX_LLM_API_KEY", "")
	t.Setenv("GITDEX_OUTPUT", "json")

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"chat", "hello"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("GITDEX_OUTPUT=json should produce valid JSON: %v\nraw: %s", err, out.String())
	}
	if _, ok := result["content"]; !ok {
		t.Error("JSON output missing 'content' field")
	}
	if _, ok := result["role"]; !ok {
		t.Error("JSON output missing 'role' field")
	}
}
