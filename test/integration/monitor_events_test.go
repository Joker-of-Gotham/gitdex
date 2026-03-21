package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/cli/command"
)

func TestMonitorEventsEmpty(t *testing.T) {
	store := autonomy.NewMemoryMonitorStore()
	restore := command.SetMonitorStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "events"})

	if err := root.Execute(); err != nil {
		t.Fatalf("monitor events failed: %v", err)
	}
	if !strings.Contains(out.String(), "No events") {
		t.Errorf("expected 'No events' message, got: %s", out.String())
	}
}

func TestMonitorRemoveHappyPath(t *testing.T) {
	store := autonomy.NewMemoryMonitorStore()
	cfg := &autonomy.MonitorConfig{MonitorID: "mon_test123", RepoOwner: "o", RepoName: "r", Interval: "5m", Enabled: true}
	_ = store.SaveMonitorConfig(cfg)
	restore := command.SetMonitorStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "remove", "mon_test123"})

	if err := root.Execute(); err != nil {
		t.Fatalf("monitor remove failed: %v", err)
	}
	if !strings.Contains(out.String(), "removed") {
		t.Errorf("expected 'removed' in output, got: %s", out.String())
	}

	_, err := store.GetMonitorConfig("mon_test123")
	if err == nil {
		t.Error("monitor should have been removed from store")
	}
}

func TestMonitorRemoveNotFound(t *testing.T) {
	store := autonomy.NewMemoryMonitorStore()
	restore := command.SetMonitorStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	root.SetArgs([]string{"monitor", "remove", "nonexistent"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when removing nonexistent monitor")
	}
}

func TestMonitorEventsWithData(t *testing.T) {
	store := autonomy.NewMemoryMonitorStore()
	cfg := &autonomy.MonitorConfig{MonitorID: "mon_ev1", RepoOwner: "org", RepoName: "repo", Interval: "5m"}
	_ = store.SaveMonitorConfig(cfg)
	_ = store.AppendEvent(&autonomy.MonitorEvent{
		MonitorID: "mon_ev1", RepoOwner: "org", RepoName: "repo",
		CheckName: "health", Status: "ok", Message: "all good",
	})
	restore := command.SetMonitorStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "events", "--repo", "org/repo"})

	if err := root.Execute(); err != nil {
		t.Fatalf("monitor events failed: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "health") {
		t.Errorf("expected event check name 'health' in output, got: %s", output)
	}
	if !strings.Contains(output, "ok") {
		t.Errorf("expected event status 'ok' in output, got: %s", output)
	}
}

func TestMonitorEventsJSON(t *testing.T) {
	store := autonomy.NewMemoryMonitorStore()
	_ = store.AppendEvent(&autonomy.MonitorEvent{
		MonitorID: "mon_j1", RepoOwner: "x", RepoName: "y",
		CheckName: "ci", Status: "warning", Message: "flaky",
	})
	restore := command.SetMonitorStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "events", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("monitor events --output json failed: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, `"events"`) {
		t.Errorf("JSON output should contain 'events' key, got: %s", output)
	}
}

func TestMonitorAddAndRemoveFlow(t *testing.T) {
	store := autonomy.NewMemoryMonitorStore()
	restore := command.SetMonitorStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "add", "--repo", "test/repo", "--interval", "10m"})
	if err := root.Execute(); err != nil {
		t.Fatalf("monitor add failed: %v", err)
	}

	configs, _ := store.ListMonitorConfigs()
	if len(configs) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(configs))
	}
	monID := configs[0].MonitorID

	out.Reset()
	root.SetArgs([]string{"monitor", "remove", monID})
	if err := root.Execute(); err != nil {
		t.Fatalf("monitor remove failed: %v", err)
	}

	configs, _ = store.ListMonitorConfigs()
	if len(configs) != 0 {
		t.Errorf("expected 0 monitors after removal, got %d", len(configs))
	}
}
