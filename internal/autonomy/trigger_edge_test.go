package autonomy

import "testing"

func TestMemoryTriggerStore_SaveNilFails(t *testing.T) {
	s := NewMemoryTriggerStore()
	if err := s.SaveTrigger(nil); err == nil {
		t.Fatal("expected error when saving nil config")
	}
}

func TestMemoryTriggerStore_GetNotFound(t *testing.T) {
	s := NewMemoryTriggerStore()
	_, err := s.GetTrigger("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown trigger ID")
	}
}

func TestMemoryTriggerStore_EnableNotFound(t *testing.T) {
	s := NewMemoryTriggerStore()
	if err := s.EnableTrigger("nonexistent"); err == nil {
		t.Fatal("expected error for enabling unknown trigger")
	}
}

func TestMemoryTriggerStore_DisableNotFound(t *testing.T) {
	s := NewMemoryTriggerStore()
	if err := s.DisableTrigger("nonexistent"); err == nil {
		t.Fatal("expected error for disabling unknown trigger")
	}
}

func TestMemoryTriggerStore_AppendNilFails(t *testing.T) {
	s := NewMemoryTriggerStore()
	if err := s.AppendTriggerEvent(nil); err == nil {
		t.Fatal("expected error when appending nil event")
	}
}

func TestMemoryTriggerStore_ListTriggerEventsDefaultLimit(t *testing.T) {
	s := NewMemoryTriggerStore()
	for i := 0; i < 5; i++ {
		_ = s.AppendTriggerEvent(&TriggerEvent{TriggerID: "tr1", TriggerType: TriggerSchedule, SourceEvent: "cron"})
	}

	events, err := s.ListTriggerEvents("", -1)
	if err != nil {
		t.Fatalf("ListTriggerEvents: %v", err)
	}
	if len(events) != 5 {
		t.Errorf("expected 5 events (default limit=50), got %d", len(events))
	}
}

func TestMemoryTriggerStore_ListTriggerEventsFilterByID(t *testing.T) {
	s := NewMemoryTriggerStore()
	_ = s.AppendTriggerEvent(&TriggerEvent{TriggerID: "tr1", TriggerType: TriggerSchedule, SourceEvent: "cron"})
	_ = s.AppendTriggerEvent(&TriggerEvent{TriggerID: "tr2", TriggerType: TriggerAPI, SourceEvent: "api"})

	events, err := s.ListTriggerEvents("tr1", 10)
	if err != nil {
		t.Fatalf("ListTriggerEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event filtered by trigger ID, got %d", len(events))
	}
}

func TestMemoryTriggerStore_AutoAssignFields(t *testing.T) {
	s := NewMemoryTriggerStore()
	cfg := &TriggerConfig{TriggerType: TriggerSchedule, Name: "t1"}
	_ = s.SaveTrigger(cfg)

	if cfg.TriggerID == "" {
		t.Error("TriggerID should be auto-assigned")
	}
	if cfg.CreatedAt.IsZero() {
		t.Error("CreatedAt should be auto-assigned")
	}

	ev := &TriggerEvent{TriggerID: "tr1", TriggerType: TriggerSchedule, SourceEvent: "cron"}
	_ = s.AppendTriggerEvent(ev)
	if ev.EventID == "" {
		t.Error("EventID should be auto-assigned")
	}
	if ev.Timestamp.IsZero() {
		t.Error("Timestamp should be auto-assigned")
	}
}
