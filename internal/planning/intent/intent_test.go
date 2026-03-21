package intent_test

import (
	"testing"

	"github.com/your-org/gitdex/internal/planning/intent"
)

func TestSource_Constants(t *testing.T) {
	tests := []struct {
		name     string
		source   intent.Source
		expected string
	}{
		{"SourceCommand", intent.SourceCommand, "command"},
		{"SourceChat", intent.SourceChat, "chat"},
		{"SourceTUI", intent.SourceTUI, "tui"},
		{"SourceAPI", intent.SourceAPI, "api"},
		{"SourceWebhook", intent.SourceWebhook, "webhook"},
		{"SourceSchedule", intent.SourceSchedule, "schedule"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.source) != tt.expected {
				t.Errorf("Source %s = %q, want %q", tt.name, string(tt.source), tt.expected)
			}
		})
	}
}

func TestIntent_StructConstruction(t *testing.T) {
	i := intent.Intent{
		Source:      intent.SourceAPI,
		RawInput:    "raw input",
		ActionType:  "search",
		Parameters:  map[string]string{"q": "test"},
		ContextRefs: []string{"ref1", "ref2"},
	}

	if i.Source != intent.SourceAPI {
		t.Errorf("Source = %q, want %q", i.Source, intent.SourceAPI)
	}
	if i.RawInput != "raw input" {
		t.Errorf("RawInput = %q, want %q", i.RawInput, "raw input")
	}
	if i.ActionType != "search" {
		t.Errorf("ActionType = %q, want %q", i.ActionType, "search")
	}
	if i.Parameters["q"] != "test" {
		t.Errorf("Parameters[q] = %q, want %q", i.Parameters["q"], "test")
	}
	if len(i.ContextRefs) != 2 || i.ContextRefs[0] != "ref1" || i.ContextRefs[1] != "ref2" {
		t.Errorf("ContextRefs = %v, want [ref1 ref2]", i.ContextRefs)
	}
}

func TestIntent_OptionalFields(t *testing.T) {
	i := intent.Intent{
		Source:     intent.SourceCommand,
		RawInput:   "foo",
		ActionType: "bar",
	}

	if i.Parameters != nil {
		t.Errorf("Parameters should be nil when not set, got %v", i.Parameters)
	}
	if i.ContextRefs != nil {
		t.Errorf("ContextRefs should be nil when not set, got %v", i.ContextRefs)
	}
}

func TestNewCommandIntent(t *testing.T) {
	params := map[string]string{"repo": "my-repo", "branch": "main"}
	i := intent.NewCommandIntent("git clone repo", "clone", params)

	if i.Source != intent.SourceCommand {
		t.Errorf("Source = %q, want %q", i.Source, intent.SourceCommand)
	}
	if i.RawInput != "git clone repo" {
		t.Errorf("RawInput = %q, want %q", i.RawInput, "git clone repo")
	}
	if i.ActionType != "clone" {
		t.Errorf("ActionType = %q, want %q", i.ActionType, "clone")
	}
	if len(i.Parameters) != 2 || i.Parameters["repo"] != "my-repo" || i.Parameters["branch"] != "main" {
		t.Errorf("Parameters = %v, want map[repo:my-repo branch:main]", i.Parameters)
	}
	if i.ContextRefs != nil {
		t.Errorf("ContextRefs = %v, want nil", i.ContextRefs)
	}
}

func TestNewCommandIntent_NilParams(t *testing.T) {
	i := intent.NewCommandIntent("raw", "action", nil)

	if i.Source != intent.SourceCommand {
		t.Errorf("Source = %q, want %q", i.Source, intent.SourceCommand)
	}
	if i.RawInput != "raw" {
		t.Errorf("RawInput = %q, want %q", i.RawInput, "raw")
	}
	if i.ActionType != "action" {
		t.Errorf("ActionType = %q, want %q", i.ActionType, "action")
	}
	if i.Parameters != nil {
		t.Errorf("Parameters = %v, want nil", i.Parameters)
	}
}

func TestNewCommandIntent_EmptyParams(t *testing.T) {
	i := intent.NewCommandIntent("raw", "action", map[string]string{})

	if i.Parameters == nil {
		t.Errorf("Parameters should not be nil when empty map passed")
	}
	if len(i.Parameters) != 0 {
		t.Errorf("Parameters should be empty, got %v", i.Parameters)
	}
}

func TestNewChatIntent(t *testing.T) {
	i := intent.NewChatIntent("help me with git workflow")

	if i.Source != intent.SourceChat {
		t.Errorf("Source = %q, want %q", i.Source, intent.SourceChat)
	}
	if i.RawInput != "help me with git workflow" {
		t.Errorf("RawInput = %q, want %q", i.RawInput, "help me with git workflow")
	}
	if i.ActionType != "chat_derived" {
		t.Errorf("ActionType = %q, want %q", i.ActionType, "chat_derived")
	}
	if i.Parameters != nil {
		t.Errorf("Parameters = %v, want nil", i.Parameters)
	}
	if i.ContextRefs != nil {
		t.Errorf("ContextRefs = %v, want nil", i.ContextRefs)
	}
}

func TestNewChatIntent_EmptyInput(t *testing.T) {
	i := intent.NewChatIntent("")

	if i.Source != intent.SourceChat {
		t.Errorf("Source = %q, want %q", i.Source, intent.SourceChat)
	}
	if i.RawInput != "" {
		t.Errorf("RawInput = %q, want empty string", i.RawInput)
	}
	if i.ActionType != "chat_derived" {
		t.Errorf("ActionType = %q, want chat_derived", i.ActionType)
	}
}

func TestSource_StringConversion(t *testing.T) {
	s := intent.SourceCommand
	str := string(s)
	if str != "command" {
		t.Errorf("string(SourceCommand) = %q, want %q", str, "command")
	}
}
