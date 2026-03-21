package autonomy

import (
	"testing"
	"time"
)

func TestMemoryTriggerStore_SaveAndGet(t *testing.T) {
	s := NewMemoryTriggerStore()
	cfg := &TriggerConfig{
		TriggerType:    TriggerSchedule,
		Name:           "nightly-sync",
		Pattern:        "0 0 * * *",
		ActionTemplate: "repo sync",
		Enabled:        true,
	}
	if err := s.SaveTrigger(cfg); err != nil {
		t.Fatalf("SaveTrigger: %v", err)
	}
	if cfg.TriggerID == "" {
		t.Error("TriggerID should be auto-assigned")
	}

	got, err := s.GetTrigger(cfg.TriggerID)
	if err != nil {
		t.Fatalf("GetTrigger: %v", err)
	}
	if got.Name != "nightly-sync" || got.Pattern != "0 0 * * *" {
		t.Errorf("unexpected config: %+v", got)
	}
}

func TestMemoryTriggerStore_EnableDisable(t *testing.T) {
	s := NewMemoryTriggerStore()
	cfg := &TriggerConfig{TriggerType: TriggerSchedule, Name: "t1", Enabled: false}
	_ = s.SaveTrigger(cfg)

	if err := s.EnableTrigger(cfg.TriggerID); err != nil {
		t.Fatalf("EnableTrigger: %v", err)
	}
	got, _ := s.GetTrigger(cfg.TriggerID)
	if !got.Enabled {
		t.Error("expected enabled after EnableTrigger")
	}

	if err := s.DisableTrigger(cfg.TriggerID); err != nil {
		t.Fatalf("DisableTrigger: %v", err)
	}
	got, _ = s.GetTrigger(cfg.TriggerID)
	if got.Enabled {
		t.Error("expected disabled after DisableTrigger")
	}
}

func TestMemoryTriggerStore_AppendAndListEvents(t *testing.T) {
	s := NewMemoryTriggerStore()
	cfg := &TriggerConfig{TriggerID: "tr1", TriggerType: TriggerSchedule, Name: "t1", Enabled: true}
	_ = s.SaveTrigger(cfg)

	ev := &TriggerEvent{
		TriggerID:       "tr1",
		TriggerType:     TriggerSchedule,
		SourceEvent:     "cron",
		ResultingTaskID: "task_001",
		Timestamp:       time.Now().UTC(),
	}
	if err := s.AppendTriggerEvent(ev); err != nil {
		t.Fatalf("AppendTriggerEvent: %v", err)
	}

	events, err := s.ListTriggerEvents("tr1", 10)
	if err != nil {
		t.Fatalf("ListTriggerEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].ResultingTaskID != "task_001" {
		t.Errorf("unexpected resulting_task_id: %s", events[0].ResultingTaskID)
	}
}
