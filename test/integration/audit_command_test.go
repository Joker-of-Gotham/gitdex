package integration

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/cli/command"
)

func TestAuditCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	auditCmd, _, err := root.Find([]string{"audit"})
	if err != nil {
		t.Fatalf("audit command not found: %v", err)
	}

	subs := []string{"log", "show", "trace"}
	for _, name := range subs {
		found := false
		for _, c := range auditCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'audit'", name)
		}
	}
}

func TestAuditLogRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"audit", "log"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("audit log failed: %v", err)
	}
}

func TestAuditShowRequiresEntryID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"audit", "show"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when entry_id is missing")
	}
}

func TestAuditTraceRequiresCorrelationID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"audit", "trace"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when correlation_id is missing")
	}
}

func TestAuditShowWithInjectedEntry(t *testing.T) {
	ledger := audit.NewMemoryAuditLedger()
	entry := &audit.AuditEntry{
		EntryID:       "audit_inject_001",
		CorrelationID: "corr_test_abc",
		TaskID:        "task_1",
		PlanID:        "plan_1",
		EventType:     audit.EventTaskStarted,
		Actor:         "test-actor",
		Action:        "start",
		Target:        "repo/main",
		Timestamp:     time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}
	if err := ledger.Append(entry); err != nil {
		t.Fatalf("ledger append failed: %v", err)
	}

	restore := command.SetAuditLedgerForTest(ledger)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"audit", "show", "audit_inject_001"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("audit show audit_inject_001 failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "audit_inject_001") {
		t.Error("output should contain entry_id audit_inject_001")
	}
	if !strings.Contains(output, "corr_test_abc") {
		t.Error("output should contain correlation_id corr_test_abc")
	}
	if !strings.Contains(output, "task_started") {
		t.Error("output should contain event_type task_started")
	}
}

func TestAuditTraceWithInjectedEntries(t *testing.T) {
	ledger := audit.NewMemoryAuditLedger()
	corrID := "corr_trace_xyz"
	ts := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	eventTypes := []audit.EventType{audit.EventPlanCreated, audit.EventTaskStarted}
	for i, eid := range []string{"audit_trace_a", "audit_trace_b"} {
		entry := &audit.AuditEntry{
			EntryID:       eid,
			CorrelationID: corrID,
			TaskID:        "task_trace",
			PlanID:        "plan_trace",
			EventType:     eventTypes[i],
			Actor:         "actor",
			Action:        "action",
			Target:        "target",
			Timestamp:     ts.Add(time.Duration(i) * time.Second),
		}
		if err := ledger.Append(entry); err != nil {
			t.Fatalf("ledger append failed: %v", err)
		}
	}

	restore := command.SetAuditLedgerForTest(ledger)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"audit", "trace", corrID})

	err := root.Execute()
	if err != nil {
		t.Fatalf("audit trace %s failed: %v", corrID, err)
	}

	output := out.String()
	if !strings.Contains(output, corrID) {
		t.Error("output should contain correlation_id")
	}
	if !strings.Contains(output, "audit_trace_a") || !strings.Contains(output, "audit_trace_b") {
		t.Error("output should contain both injected entry IDs")
	}
}
