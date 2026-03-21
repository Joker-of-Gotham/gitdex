package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/autonomy"
)

func TestHandoffPackage_JSONContract(t *testing.T) {
	pkg := &autonomy.HandoffPackage{
		PackageID:       "pkg_abc123",
		TaskID:          "task_xyz",
		TaskSummary:     "summary",
		CurrentState:    "running",
		CompletedSteps:  []string{"s1"},
		PendingSteps:    []string{"s2"},
		ContextData:     map[string]string{"k": "v"},
		Artifacts:       []string{"a1"},
		Recommendations: []string{"r1"},
		CreatedAt:       time.Now().UTC(),
	}

	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"package_id"`,
		`"task_id"`,
		`"task_summary"`,
		`"current_state"`,
		`"completed_steps"`,
		`"pending_steps"`,
		`"context_data"`,
		`"artifacts"`,
		`"recommendations"`,
		`"created_at"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestRecoveryStrategy_AllValues(t *testing.T) {
	strategies := []autonomy.RecoveryStrategy{
		autonomy.RecoveryRetry,
		autonomy.RecoveryRollback,
		autonomy.RecoveryEscalate,
		autonomy.RecoverySkip,
		autonomy.RecoveryManualIntervention,
	}

	seen := make(map[autonomy.RecoveryStrategy]bool)
	for _, s := range strategies {
		if s == "" {
			t.Error("recovery strategy should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate strategy: %s", s)
		}
		seen[s] = true
	}
}
