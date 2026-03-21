package audit

import (
	"testing"
	"time"
)

func TestMemoryAuditLedger_Append(t *testing.T) {
	l := NewMemoryAuditLedger()

	e := &AuditEntry{
		CorrelationID: "corr_1",
		TaskID:        "task_1",
		PlanID:        "plan_1",
		EventType:     EventTaskStarted,
		Actor:         "system",
		Action:        "start",
		Target:        "repo",
	}

	err := l.Append(e)
	if err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if e.EntryID == "" {
		t.Error("EntryID should be set after append")
	}

	entries, err := l.Query(AuditFilter{})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestMemoryAuditLedger_AppendNil(t *testing.T) {
	l := NewMemoryAuditLedger()
	err := l.Append(nil)
	if err == nil {
		t.Fatal("expected error for nil entry")
	}
}

func TestMemoryAuditLedger_GetByCorrelation(t *testing.T) {
	l := NewMemoryAuditLedger()

	_ = l.Append(&AuditEntry{
		CorrelationID: "corr_a",
		TaskID:        "task_1",
		EventType:     EventPlanCreated,
	})
	_ = l.Append(&AuditEntry{
		CorrelationID: "corr_a",
		TaskID:        "task_1",
		EventType:     EventTaskStarted,
	})
	_ = l.Append(&AuditEntry{
		CorrelationID: "corr_b",
		TaskID:        "task_2",
		EventType:     EventPlanCreated,
	})

	entries, err := l.GetByCorrelation("corr_a")
	if err != nil {
		t.Fatalf("GetByCorrelation error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries for corr_a, got %d", len(entries))
	}
}

func TestMemoryAuditLedger_GetByTask(t *testing.T) {
	l := NewMemoryAuditLedger()

	_ = l.Append(&AuditEntry{
		CorrelationID: "corr_1",
		TaskID:        "task_x",
		EventType:     EventTaskStarted,
	})
	_ = l.Append(&AuditEntry{
		CorrelationID: "corr_1",
		TaskID:        "task_y",
		EventType:     EventTaskStarted,
	})

	entries, err := l.GetByTask("task_x")
	if err != nil {
		t.Fatalf("GetByTask error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry for task_x, got %d", len(entries))
	}
}

func TestMemoryAuditLedger_QueryTimeFilter(t *testing.T) {
	l := NewMemoryAuditLedger()

	t1 := time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 19, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)

	e1 := &AuditEntry{CorrelationID: "c1", EventType: EventPlanCreated, Timestamp: t1}
	e2 := &AuditEntry{CorrelationID: "c2", EventType: EventPlanCreated, Timestamp: t2}
	e3 := &AuditEntry{CorrelationID: "c3", EventType: EventPlanCreated, Timestamp: t3}

	_ = l.Append(e1)
	_ = l.Append(e2)
	_ = l.Append(e3)

	from := time.Date(2026, 3, 19, 10, 30, 0, 0, time.UTC)
	to := time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)

	entries, err := l.Query(AuditFilter{FromTime: &from, ToTime: &to})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry in time range, got %d", len(entries))
	}
}

func TestMemoryAuditLedger_GetByEntryID(t *testing.T) {
	l := NewMemoryAuditLedger()

	e := &AuditEntry{
		EntryID:       "audit_explicit",
		CorrelationID: "corr_1",
		EventType:     EventPlanApproved,
	}
	_ = l.Append(e)

	found, ok := l.GetByEntryID("audit_explicit")
	if !ok {
		t.Fatal("GetByEntryID should find entry")
	}
	if found.EntryID != "audit_explicit" {
		t.Errorf("EntryID: got %q", found.EntryID)
	}
}
