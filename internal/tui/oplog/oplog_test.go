package oplog

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	l := New(50)
	if l == nil {
		t.Fatal("New(50) must not return nil")
	}
	if l.Len() != 0 {
		t.Errorf("new log should have 0 entries, got %d", l.Len())
	}
}

func TestAdd_And_Entries(t *testing.T) {
	l := New(10)
	e1 := Entry{Type: EntryLLMStart, Summary: "a", Detail: "b"}
	e2 := Entry{Type: EntryCmdExec, Summary: "c", Detail: "d"}
	l.Add(e1)
	l.Add(e2)
	entries := l.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Summary != "a" || entries[0].Type != EntryLLMStart {
		t.Errorf("first entry: want Summary=a Type=llm_start, got Summary=%q Type=%s", entries[0].Summary, entries[0].Type)
	}
	if entries[1].Summary != "c" || entries[1].Type != EntryCmdExec {
		t.Errorf("second entry: want Summary=c Type=cmd_exec, got Summary=%q Type=%s", entries[1].Summary, entries[1].Type)
	}
}

func TestAdd_Eviction(t *testing.T) {
	l := New(3)
	for i := 0; i < 5; i++ {
		l.Add(Entry{Type: EntryLLMStart, Summary: string(rune('a' + i)), Detail: ""})
	}
	entries := l.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries after eviction, got %d", len(entries))
	}
	if entries[0].Summary != "c" {
		t.Errorf("oldest kept entry should be c, got %q", entries[0].Summary)
	}
}

func TestEntryNormalized(t *testing.T) {
	now := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)
	e := Entry{Type: EntryCmdSuccess, Summary: "  x  ", Detail: "  y  "}
	norm := e.Normalized(now)
	if norm.Timestamp != now {
		t.Errorf("zero timestamp should be set to now, got %v", norm.Timestamp)
	}
	if norm.Summary != "x" || norm.Detail != "y" {
		t.Errorf("Normalized should trim Summary and Detail, got Summary=%q Detail=%q", norm.Summary, norm.Detail)
	}
	eWithTime := Entry{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Type: EntryCmdSuccess}
	norm2 := eWithTime.Normalized(now)
	if norm2.Timestamp != eWithTime.Timestamp {
		t.Errorf("non-zero timestamp should be preserved, got %v", norm2.Timestamp)
	}
}

func TestEntryIcon(t *testing.T) {
	tests := []struct {
		typ  EntryType
		want string
	}{
		{EntryLLMStart, "⟳"},
		{EntryLLMOutput, "✦"},
		{EntryLLMError, "✗"},
		{EntryCmdExec, "▸"},
		{EntryCmdSuccess, "✓"},
		{EntryCmdFail, "✗"},
		{EntryStateRefresh, "↻"},
		{EntryUserAction, "▹"},
		{EntryType("unknown"), "·"},
	}
	for _, tt := range tests {
		e := Entry{Type: tt.typ}
		if got := e.Icon(); got != tt.want {
			t.Errorf("Icon() for %s: want %q, got %q", tt.typ, tt.want, got)
		}
	}
}

func TestLen(t *testing.T) {
	l := New(10)
	if l.Len() != 0 {
		t.Errorf("empty log Len() want 0, got %d", l.Len())
	}
	l.Add(Entry{Type: EntryLLMStart})
	l.Add(Entry{Type: EntryLLMOutput})
	if l.Len() != 2 {
		t.Errorf("after 2 adds Len() want 2, got %d", l.Len())
	}
}
