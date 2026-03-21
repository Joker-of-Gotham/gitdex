package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/audit"
)

func TestAuditEntry_JSONContract(t *testing.T) {
	entry := &audit.AuditEntry{
		EntryID:       "audit_abc123",
		CorrelationID: "corr_xyz",
		TaskID:        "task_001",
		PlanID:        "plan_001",
		EventType:     audit.EventPlanApproved,
		Actor:         "admin",
		Action:        "approve",
		Target:        "plan",
		PolicyResult:  "allowed",
		EvidenceRefs:  []string{"ev_1"},
		Timestamp:     time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"entry_id"`,
		`"correlation_id"`,
		`"task_id"`,
		`"plan_id"`,
		`"event_type"`,
		`"actor"`,
		`"action"`,
		`"target"`,
		`"evidence_refs"`,
		`"timestamp"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestAuditEntry_RoundTrip(t *testing.T) {
	original := &audit.AuditEntry{
		EntryID:       "audit_rt",
		CorrelationID: "corr_rt",
		TaskID:        "task_rt",
		PlanID:        "plan_rt",
		EventType:     audit.EventTaskSucceeded,
		Actor:         "system",
		Action:        "complete",
		Target:        "repo",
		EvidenceRefs:  []string{"e1", "e2"},
		Timestamp:     time.Date(2026, 3, 19, 14, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded audit.AuditEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.EntryID != original.EntryID {
		t.Errorf("EntryID: got %q, want %q", decoded.EntryID, original.EntryID)
	}
	if decoded.EventType != original.EventType {
		t.Errorf("EventType: got %q, want %q", decoded.EventType, original.EventType)
	}
	if len(decoded.EvidenceRefs) != len(original.EvidenceRefs) {
		t.Errorf("EvidenceRefs: got %d, want %d", len(decoded.EvidenceRefs), len(original.EvidenceRefs))
	}
}

func TestEventType_AllValues(t *testing.T) {
	types := []audit.EventType{
		audit.EventPlanCreated,
		audit.EventPlanApproved,
		audit.EventPlanRejected,
		audit.EventTaskStarted,
		audit.EventTaskSucceeded,
		audit.EventTaskFailed,
		audit.EventPolicyEvaluated,
		audit.EventEmergencyControl,
		audit.EventIdentityRegistered,
	}

	seen := make(map[audit.EventType]bool)
	for _, et := range types {
		if et == "" {
			t.Error("event type should not be empty")
		}
		if seen[et] {
			t.Errorf("duplicate event type: %s", et)
		}
		seen[et] = true
	}
}
