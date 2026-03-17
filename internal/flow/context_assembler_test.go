package flow

import (
	"strings"
	"testing"
)

func TestContextAssemblerGoalProfile(t *testing.T) {
	a := NewContextAssembler("goal", 10000)
	if a.OutputRounds() != 2 {
		t.Fatalf("goal profile should read 2 output rounds, got %d", a.OutputRounds())
	}
	if a.TokensBudget() != 6000 { // 40%% reserved for output
		t.Fatalf("unexpected budget: got %d want %d", a.TokensBudget(), 6000)
	}
}

func TestContextAssemblerSectionUsage(t *testing.T) {
	a := NewContextAssembler("maintain", 4096)
	long := strings.Repeat("git status output\n", 400)
	_ = a.AddGit(long)
	_ = a.AddOutput(long)
	_ = a.AddIndex(long)
	_ = a.AddKnowledge(long)

	usage := a.SectionUsage()
	if len(usage) == 0 {
		t.Fatal("expected non-empty section usage")
	}
	if usage["git_context"] == 0 {
		t.Fatal("expected git_context tokens to be tracked")
	}
	if usage["output_log"] == 0 {
		t.Fatal("expected output_log tokens to be tracked")
	}
}
