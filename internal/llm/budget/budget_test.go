package budget

import (
	"strings"
	"testing"
)

func TestEstimateTokens_Empty(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Errorf("EstimateTokens empty: got %d, want 0", got)
	}
}

func TestEstimateTokens_ASCII(t *testing.T) {
	text := "Hello world, this is a test of the token estimation"
	tokens := EstimateTokens(text)
	if tokens < 10 || tokens > 20 {
		t.Errorf("EstimateTokens ASCII: got %d, expected 10-20 for %q", tokens, text)
	}
}

func TestEstimateTokens_CJK(t *testing.T) {
	text := "这是一个中文测试字符串用于令牌估算"
	tokens := EstimateTokens(text)
	if tokens < 5 || tokens > 20 {
		t.Errorf("EstimateTokens CJK: got %d, expected 5-20 for %q", tokens, text)
	}
}

func TestNewBudget_Available(t *testing.T) {
	b := NewBudget(1000, 250)
	if b.Available() != 750 {
		t.Errorf("Available: got %d, want 750", b.Available())
	}
}

func TestBudget_AddFits(t *testing.T) {
	b := NewBudget(10000, 2000)
	text := "short text"
	got := b.Add("test", text)
	if got != text {
		t.Errorf("Add returned truncated text unexpectedly")
	}
	if b.Used() == 0 {
		t.Error("Used() should be > 0 after Add")
	}
}

func TestBudget_AddTruncates(t *testing.T) {
	b := NewBudget(100, 25)
	longText := strings.Repeat("the quick brown fox jumps over the lazy dog ", 100)
	got := b.Add("big", longText)
	if len(got) >= len(longText) {
		t.Errorf("Add should truncate: got len %d, orig len %d", len(got), len(longText))
	}
	if !strings.Contains(got, "truncated") {
		t.Error("truncated text should contain truncation marker")
	}
}

func TestBudget_FormatUsage(t *testing.T) {
	b := NewBudget(10000, 2500)
	b.Add("test", strings.Repeat("x", 400))
	usage := b.FormatUsage()
	if usage == "" {
		t.Error("FormatUsage should not be empty")
	}
}

func TestCompressGitContent_Small(t *testing.T) {
	content := "## Branch: main\nstatus: clean\n"
	got := CompressGitContent(content, 1000)
	if got != content {
		t.Error("small content should not be compressed")
	}
}

func TestCompressGitContent_LargeRemovesReflog(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("## Branch: main\nstatus: clean\n")
	sb.WriteString("## Recent Reflog\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("abcdefg1234567 HEAD@{0}: commit: something\n")
	}
	sb.WriteString("## Remotes\nurl: git@github.com:user/repo.git\n")

	got := CompressGitContent(sb.String(), 100)
	if strings.Contains(got, "HEAD@{0}") {
		t.Error("reflog lines should be compressed out")
	}
	if !strings.Contains(got, "## Branch: main") {
		t.Error("essential content should be preserved")
	}
}

func TestCompressOutputLog_Small(t *testing.T) {
	output := "step 1: ok\nstep 2: ok\n"
	got := CompressOutputLog(output, 1000)
	if got != output {
		t.Error("small output should not be compressed")
	}
}

func TestTruncateToTokens_NoTruncation(t *testing.T) {
	text := "short"
	got := TruncateToTokens(text, 1000)
	if got != text {
		t.Error("short text should not be truncated")
	}
}

func TestTruncateToTokens_Truncated(t *testing.T) {
	text := strings.Repeat("word ", 1000)
	got := TruncateToTokens(text, 10)
	if len(got) >= len(text) {
		t.Error("should be truncated")
	}
}

func TestTruncateToTokens_ZeroMax(t *testing.T) {
	got := TruncateToTokens("something", 0)
	if got != "" {
		t.Errorf("zero max should return empty, got %q", got)
	}
}
