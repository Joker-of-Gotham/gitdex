package autonomy

import (
	"testing"
	"time"
)

func TestMemoryAutonomyStore_SaveGetConfig(t *testing.T) {
	store := NewMemoryAutonomyStore()

	cfg := &AutonomyConfig{
		Name:         "default",
		DefaultLevel: LevelManual,
		CapabilityAutonomies: []CapabilityAutonomy{
			{
				Capability:       "repo_sync",
				Level:            LevelSupervised,
				RequiresApproval: true,
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := store.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if cfg.ConfigID == "" {
		t.Error("expected ConfigID to be set")
	}

	got, err := store.GetConfig(cfg.ConfigID)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if got.Name != cfg.Name || got.DefaultLevel != cfg.DefaultLevel {
		t.Errorf("config mismatch: got %+v", got)
	}
}

func TestMemoryAutonomyStore_GetActiveConfig(t *testing.T) {
	store := NewMemoryAutonomyStore()

	cfg := &AutonomyConfig{
		Name:         "active",
		DefaultLevel: LevelAutonomous,
		CreatedAt:    time.Now().UTC(),
	}

	_ = store.SaveConfig(cfg)
	_ = store.SetActiveConfig(cfg.ConfigID)

	got, err := store.GetActiveConfig()
	if err != nil {
		t.Fatalf("GetActiveConfig: %v", err)
	}
	if got == nil || got.ConfigID != cfg.ConfigID {
		t.Errorf("GetActiveConfig: got %+v, want config %s", got, cfg.ConfigID)
	}
}

func TestMemoryAutonomyStore_ListConfigs(t *testing.T) {
	store := NewMemoryAutonomyStore()

	c1 := &AutonomyConfig{Name: "a", DefaultLevel: LevelManual, CreatedAt: time.Now().UTC()}
	c2 := &AutonomyConfig{Name: "b", DefaultLevel: LevelSupervised, CreatedAt: time.Now().UTC()}

	_ = store.SaveConfig(c1)
	_ = store.SaveConfig(c2)

	list, err := store.ListConfigs()
	if err != nil {
		t.Fatalf("ListConfigs: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListConfigs: want 2, got %d", len(list))
	}
}

func TestMemoryAutonomyStore_GetConfigNotFound(t *testing.T) {
	store := NewMemoryAutonomyStore()

	_, err := store.GetConfig("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent config")
	}
}
