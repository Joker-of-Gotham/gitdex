package context

import (
	"strings"
	"testing"
)

func TestBudgetManagerAssembleDetailed(t *testing.T) {
	mgr := NewBudgetManager(200, 50)
	_, user, usage := mgr.AssembleDetailed("system", []Partition{
		{Name: "critical", Priority: PrioCriticalState, Content: strings.Repeat("a", 200), Required: true},
		{Name: "optional", Priority: PrioKnowledge, Content: strings.Repeat("b", 400)},
	})
	if user == "" {
		t.Fatal("expected assembled user prompt")
	}
	if len(usage) != 2 {
		t.Fatalf("unexpected usage count: %d", len(usage))
	}
}

func TestCompressOperationLog(t *testing.T) {
	out := CompressOperationLog([]string{"executed one", "skipped two", "cancelled three"}, 1)
	if !strings.Contains(out, "older operations") {
		t.Fatalf("unexpected compressed log: %s", out)
	}
}
