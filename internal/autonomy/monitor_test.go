package autonomy

import (
	"testing"
	"time"
)

func TestMemoryMonitorStore_SaveAndGet(t *testing.T) {
	s := NewMemoryMonitorStore()
	cfg := &MonitorConfig{
		RepoOwner: "owner",
		RepoName:  "repo",
		Interval:  "5m",
		Enabled:   true,
	}
	if err := s.SaveMonitorConfig(cfg); err != nil {
		t.Fatalf("SaveMonitorConfig: %v", err)
	}
	if cfg.MonitorID == "" {
		t.Error("MonitorID should be auto-assigned")
	}

	got, err := s.GetMonitorConfig(cfg.MonitorID)
	if err != nil {
		t.Fatalf("GetMonitorConfig: %v", err)
	}
	if got.RepoOwner != "owner" || got.RepoName != "repo" {
		t.Errorf("unexpected config: %+v", got)
	}
}

func TestMemoryMonitorStore_ListMonitorConfigs(t *testing.T) {
	s := NewMemoryMonitorStore()
	_ = s.SaveMonitorConfig(&MonitorConfig{RepoOwner: "a", RepoName: "b", Interval: "5m"})
	_ = s.SaveMonitorConfig(&MonitorConfig{RepoOwner: "c", RepoName: "d", Interval: "1h"})

	list, err := s.ListMonitorConfigs()
	if err != nil {
		t.Fatalf("ListMonitorConfigs: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 configs, got %d", len(list))
	}
}

func TestMemoryMonitorStore_AppendAndListEvents(t *testing.T) {
	s := NewMemoryMonitorStore()
	cfg := &MonitorConfig{MonitorID: "m1", RepoOwner: "o", RepoName: "r", Interval: "5m"}
	_ = s.SaveMonitorConfig(cfg)

	ev := &MonitorEvent{
		MonitorID: "m1",
		RepoOwner: "o",
		RepoName:  "r",
		CheckName: "health",
		Status:    "ok",
		Message:   "all good",
		Timestamp: time.Now().UTC(),
	}
	if err := s.AppendEvent(ev); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}

	events, err := s.ListEvents(MonitorEventFilter{MonitorID: "m1", Limit: 10})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].Status != "ok" {
		t.Errorf("unexpected event status: %s", events[0].Status)
	}
}

func TestMemoryMonitorStore_RemoveMonitorConfig(t *testing.T) {
	s := NewMemoryMonitorStore()
	cfg := &MonitorConfig{RepoOwner: "x", RepoName: "y", Interval: "5m"}
	_ = s.SaveMonitorConfig(cfg)
	mid := cfg.MonitorID

	if err := s.RemoveMonitorConfig(mid); err != nil {
		t.Fatalf("RemoveMonitorConfig: %v", err)
	}
	_, err := s.GetMonitorConfig(mid)
	if err == nil {
		t.Error("expected error when getting removed monitor")
	}
}
