package promptv2

import (
	"strings"
	"testing"
)

func TestLanguageName(t *testing.T) {
	tests := []struct {
		lang string
		want string
	}{
		{"zh", "Simplified Chinese"},
		{"zh-cn", "Simplified Chinese"},
		{"zh-hans", "Simplified Chinese"},
		{"ZH", "Simplified Chinese"},
		{"ja", "Japanese"},
		{"JA", "Japanese"},
		{"en", "English"},
		{"", "English"},
		{"fr", "English"},
		{"  zh  ", "Simplified Chinese"},
	}
	for _, tt := range tests {
		sys, _ := BuildPromptA("a", "b", "c", tt.lang)
		if !strings.Contains(sys, tt.want) {
			t.Errorf("languageName(%q) should produce prompt containing %q, got:\n%s", tt.lang, tt.want, sys)
		}
	}
}

func TestBuildPromptA_ContainsGitContext(t *testing.T) {
	gitContent := "branch main, 3 commits ahead"
	sys, user := BuildPromptA(gitContent, "output", "index", "en")
	if sys == "" || user == "" {
		t.Fatalf("system and user prompts must be non-empty")
	}
	if !strings.Contains(user, gitContent) {
		t.Errorf("user prompt must contain git content %q", gitContent)
	}
}

func TestBuildPromptB_ContainsKnowledge(t *testing.T) {
	knowledgeCtx := "sync.yaml: how to sync remotes"
	sys, user := BuildPromptB("git", "output", knowledgeCtx, "en")
	if sys == "" || user == "" {
		t.Fatalf("system and user prompts must be non-empty")
	}
	if !strings.Contains(user, knowledgeCtx) {
		t.Errorf("user prompt must contain knowledge context %q", knowledgeCtx)
	}
}

func TestBuildPromptC_ContainsGoal(t *testing.T) {
	goal := "Create a PR for feature X"
	sys, user := BuildPromptC("git", "output", "index", goal, "todo", "en")
	if sys == "" || user == "" {
		t.Fatalf("system and user prompts must be non-empty")
	}
	if !strings.Contains(user, goal) {
		t.Errorf("user prompt must contain goal %q", goal)
	}
}

func TestBuildPromptD_ContainsGoal(t *testing.T) {
	goal := "Deploy to staging"
	sys, user := BuildPromptD("git", "output", "knowledge", goal, "todo", "en")
	if sys == "" || user == "" {
		t.Fatalf("system and user prompts must be non-empty")
	}
	if !strings.Contains(user, goal) {
		t.Errorf("user prompt must contain goal %q", goal)
	}
}

func TestBuildPromptE_ContainsGitHub(t *testing.T) {
	githubCtx := "3 open PRs, 2 issues"
	sys, user := BuildPromptE("git", "output", "index", "goals", "todo", githubCtx, "en")
	if sys == "" || user == "" {
		t.Fatalf("system and user prompts must be non-empty")
	}
	if !strings.Contains(user, githubCtx) {
		t.Errorf("user prompt must contain github context %q", githubCtx)
	}
}

// estimateTokens provides a rough word-based token estimate (~0.75 tokens/word).
func estimateTokens(s string) int {
	words := len(strings.Fields(s))
	return int(float64(words) * 1.33)
}

func TestPromptTokenBudgets(t *testing.T) {
	tests := []struct {
		name     string
		buildSys func() string
		maxTok   int
	}{
		{"prompt_a", func() string { s, _ := BuildPromptA("g", "o", "i", "en"); return s }, 200},
		{"prompt_b", func() string { s, _ := BuildPromptB("g", "o", "k", "en"); return s }, 500},
		{"prompt_c", func() string { s, _ := BuildPromptC("g", "o", "i", "gl", "td", "en"); return s }, 200},
		{"prompt_d", func() string { s, _ := BuildPromptD("g", "o", "k", "gl", "td", "en"); return s }, 500},
		{"prompt_e", func() string { s, _ := BuildPromptE("g", "o", "i", "gl", "td", "gh", "en"); return s }, 350},
	}
	for _, tt := range tests {
		sys := tt.buildSys()
		tok := estimateTokens(sys)
		if tok > tt.maxTok {
			t.Errorf("%s system prompt ~%d tokens exceeds budget %d", tt.name, tok, tt.maxTok)
		}
	}
}

func TestPlannerPromptsContainToolDefs(t *testing.T) {
	toolNames := []string{"git_command", "shell_command", "file_write", "file_read", "github_op"}

	sysB, _ := BuildPromptB("g", "o", "k", "en")
	sysD, _ := BuildPromptD("g", "o", "k", "gl", "td", "en")

	for _, name := range toolNames {
		if !strings.Contains(sysB, name) {
			t.Errorf("prompt_b system prompt missing tool definition %q", name)
		}
		if !strings.Contains(sysD, name) {
			t.Errorf("prompt_d system prompt missing tool definition %q", name)
		}
	}
}

func TestPlannerPromptsContainOutputSchema(t *testing.T) {
	sysB, _ := BuildPromptB("g", "o", "k", "en")
	sysD, _ := BuildPromptD("g", "o", "k", "gl", "td", "en")

	for _, marker := range []string{"analysis", "suggestions", "action", "type"} {
		if !strings.Contains(sysB, marker) {
			t.Errorf("prompt_b missing output schema field %q", marker)
		}
		if !strings.Contains(sysD, marker) {
			t.Errorf("prompt_d missing output schema field %q", marker)
		}
	}
}

func TestNoNegativeRulesInPrompts(t *testing.T) {
	banned := []string{"NEVER", "FORBIDDEN", "DO NOT"}

	prompts := map[string]string{}
	s, _ := BuildPromptA("g", "o", "i", "en")
	prompts["prompt_a"] = s
	s, _ = BuildPromptB("g", "o", "k", "en")
	prompts["prompt_b"] = s
	s, _ = BuildPromptC("g", "o", "i", "gl", "td", "en")
	prompts["prompt_c"] = s
	s, _ = BuildPromptD("g", "o", "k", "gl", "td", "en")
	prompts["prompt_d"] = s
	s, _ = BuildPromptE("g", "o", "i", "gl", "td", "gh", "en")
	prompts["prompt_e"] = s

	for name, sys := range prompts {
		upper := strings.ToUpper(sys)
		for _, word := range banned {
			if strings.Contains(upper, word) {
				t.Errorf("%s system prompt contains banned negative rule word %q — these should be enforced by code, not prompts", name, word)
			}
		}
	}
}

func TestRenderToolDefs(t *testing.T) {
	rendered := RenderToolDefs()
	if rendered == "" {
		t.Fatal("RenderToolDefs returned empty string")
	}
	for _, tool := range Tools {
		if !strings.Contains(rendered, tool.Name) {
			t.Errorf("RenderToolDefs missing tool %q", tool.Name)
		}
	}
}

func TestValidToolTypes(t *testing.T) {
	vt := ValidToolTypes()
	expected := []string{"git_command", "shell_command", "file_write", "file_read", "github_op"}
	for _, name := range expected {
		if !vt[name] {
			t.Errorf("ValidToolTypes missing %q", name)
		}
	}
	if vt["invalid_type"] {
		t.Error("ValidToolTypes should not contain invalid_type")
	}
}
