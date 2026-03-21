package guardrails

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/app/session"
)

func TestBaseSystemPrompt_ContainsBoundaries(t *testing.T) {
	prompt := BaseSystemPrompt()

	required := []string{
		"MUST NOT directly execute",
		"MUST NOT bypass structured plan",
		"Responsibilities",
		"Boundaries",
		"Gitdex",
	}

	for _, phrase := range required {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("base system prompt missing required phrase: %q", phrase)
		}
	}
}

func TestBuildSystemPrompt_NilContext(t *testing.T) {
	prompt := BuildSystemPrompt(nil)
	if !strings.Contains(prompt, "You are Gitdex") {
		t.Error("prompt with nil context should still contain identity")
	}
}

func TestBuildSystemPrompt_WithRepoPath(t *testing.T) {
	tc := session.NewTaskContext("/home/user/project", "staging")
	prompt := BuildSystemPrompt(tc)

	if !strings.Contains(prompt, "/home/user/project") {
		t.Error("prompt should contain repo path")
	}
	if !strings.Contains(prompt, "staging") {
		t.Error("prompt should contain profile")
	}
}

func TestBuildSystemPrompt_WithRecentCommands(t *testing.T) {
	tc := session.NewTaskContext("", "")
	tc.AddCommandRecord(session.CommandRecord{
		Command: "doctor",
		Output:  "all pass",
	})

	prompt := BuildSystemPrompt(tc)
	if !strings.Contains(prompt, "doctor") {
		t.Error("prompt should include recent command")
	}
}

func TestBuildSystemPrompt_TruncatesLongOutput(t *testing.T) {
	tc := session.NewTaskContext("", "")
	longOutput := strings.Repeat("x", 1000)
	tc.AddCommandRecord(session.CommandRecord{
		Command: "test",
		Output:  longOutput,
	})

	prompt := BuildSystemPrompt(tc)
	if strings.Contains(prompt, longOutput) {
		t.Error("long output should be truncated in prompt")
	}
	if !strings.Contains(prompt, "truncated") {
		t.Error("truncated output should indicate truncation")
	}
}
