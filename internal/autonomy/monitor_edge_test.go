package autonomy

import "testing"

func TestMemoryMonitorStore_SaveNilFails(t *testing.T) {
	s := NewMemoryMonitorStore()
	if err := s.SaveMonitorConfig(nil); err == nil {
		t.Fatal("expected error when saving nil config")
	}
}

func TestMemoryMonitorStore_GetNotFound(t *testing.T) {
	s := NewMemoryMonitorStore()
	_, err := s.GetMonitorConfig("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown monitor ID")
	}
}

func TestMemoryMonitorStore_RemoveNotFound(t *testing.T) {
	s := NewMemoryMonitorStore()
	if err := s.RemoveMonitorConfig("nonexistent"); err == nil {
		t.Fatal("expected error for removing unknown monitor")
	}
}

func TestMemoryMonitorStore_AppendEventNilFails(t *testing.T) {
	s := NewMemoryMonitorStore()
	if err := s.AppendEvent(nil); err == nil {
		t.Fatal("expected error when appending nil event")
	}
}

func TestMemoryMonitorStore_ListEventsFilterByRepo(t *testing.T) {
	s := NewMemoryMonitorStore()
	_ = s.AppendEvent(&MonitorEvent{MonitorID: "m1", RepoOwner: "a", RepoName: "b", CheckName: "c1", Status: "ok"})
	_ = s.AppendEvent(&MonitorEvent{MonitorID: "m2", RepoOwner: "x", RepoName: "y", CheckName: "c2", Status: "warning"})

	events, err := s.ListEvents(MonitorEventFilter{RepoOwner: "a", RepoName: "b", Limit: 10})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event filtered by repo, got %d", len(events))
	}
	if events[0].RepoOwner != "a" {
		t.Errorf("expected repo owner 'a', got %s", events[0].RepoOwner)
	}
}

func TestMemoryMonitorStore_ListEventsDefaultLimit(t *testing.T) {
	s := NewMemoryMonitorStore()
	for i := 0; i < 5; i++ {
		_ = s.AppendEvent(&MonitorEvent{MonitorID: "m1", RepoOwner: "o", RepoName: "r", Status: "ok"})
	}

	events, err := s.ListEvents(MonitorEventFilter{Limit: -1})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 5 {
		t.Errorf("expected 5 events (default limit=100), got %d", len(events))
	}
}

func TestMemoryMonitorStore_DeepCopy(t *testing.T) {
	s := NewMemoryMonitorStore()
	cfg := &MonitorConfig{RepoOwner: "o", RepoName: "r", Interval: "5m", Checks: []string{"health"}}
	_ = s.SaveMonitorConfig(cfg)

	got, _ := s.GetMonitorConfig(cfg.MonitorID)
	got.Checks[0] = "MUTATED"

	got2, _ := s.GetMonitorConfig(cfg.MonitorID)
	if got2.Checks[0] == "MUTATED" {
		t.Error("store returned shallow copy; mutation leaked to internal state")
	}
}
