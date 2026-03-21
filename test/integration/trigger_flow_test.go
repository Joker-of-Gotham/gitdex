package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/cli/command"
)

func TestTriggerAddRequiresType(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"trigger", "add", "--name", "test"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --type is missing")
	}
}

func TestTriggerAddRequiresName(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"trigger", "add", "--type", "schedule"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --name is missing")
	}
}

func TestTriggerAddInvalidType(t *testing.T) {
	store := autonomy.NewMemoryTriggerStore()
	restore := command.SetTriggerStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	root.SetArgs([]string{"trigger", "add", "--type", "invalid_type", "--name", "bad"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid trigger type")
	}
}

func TestTriggerAddThenList(t *testing.T) {
	store := autonomy.NewMemoryTriggerStore()
	restore := command.SetTriggerStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"trigger", "add", "--type", "schedule", "--name", "daily-sync", "--pattern", "0 0 * * *", "--action", "repo sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("trigger add failed: %v", err)
	}
	if !strings.Contains(out.String(), "daily-sync") {
		t.Errorf("output should contain trigger name, got: %s", out.String())
	}

	out.Reset()
	root.SetArgs([]string{"trigger", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("trigger list failed: %v", err)
	}
	if !strings.Contains(out.String(), "daily-sync") {
		t.Errorf("list should show added trigger, got: %s", out.String())
	}
}

func TestTriggerEnableDisableFlow(t *testing.T) {
	store := autonomy.NewMemoryTriggerStore()
	cfg := &autonomy.TriggerConfig{TriggerID: "tr_test1", TriggerType: autonomy.TriggerSchedule, Name: "test-trigger", Enabled: false}
	_ = store.SaveTrigger(cfg)
	restore := command.SetTriggerStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"trigger", "enable", "tr_test1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("trigger enable failed: %v", err)
	}
	if !strings.Contains(out.String(), "enabled") {
		t.Errorf("expected 'enabled' in output, got: %s", out.String())
	}

	got, _ := store.GetTrigger("tr_test1")
	if !got.Enabled {
		t.Error("trigger should be enabled after enable command")
	}

	out.Reset()
	root.SetArgs([]string{"trigger", "disable", "tr_test1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("trigger disable failed: %v", err)
	}
	if !strings.Contains(out.String(), "disabled") {
		t.Errorf("expected 'disabled' in output, got: %s", out.String())
	}

	got, _ = store.GetTrigger("tr_test1")
	if got.Enabled {
		t.Error("trigger should be disabled after disable command")
	}
}

func TestTriggerEnableNotFound(t *testing.T) {
	store := autonomy.NewMemoryTriggerStore()
	restore := command.SetTriggerStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	root.SetArgs([]string{"trigger", "enable", "nonexistent"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when enabling nonexistent trigger")
	}
}

func TestTriggerDisableNotFound(t *testing.T) {
	store := autonomy.NewMemoryTriggerStore()
	restore := command.SetTriggerStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	root.SetArgs([]string{"trigger", "disable", "nonexistent"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when disabling nonexistent trigger")
	}
}

func TestTriggerEventsEmpty(t *testing.T) {
	store := autonomy.NewMemoryTriggerStore()
	restore := command.SetTriggerStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"trigger", "events"})
	if err := root.Execute(); err != nil {
		t.Fatalf("trigger events failed: %v", err)
	}
	if !strings.Contains(out.String(), "No trigger events") {
		t.Errorf("expected 'No trigger events' message, got: %s", out.String())
	}
}

func TestTriggerEventsJSON(t *testing.T) {
	store := autonomy.NewMemoryTriggerStore()
	_ = store.AppendTriggerEvent(&autonomy.TriggerEvent{
		TriggerID: "tr1", TriggerType: autonomy.TriggerSchedule,
		SourceEvent: "cron", ResultingTaskID: "task_001",
	})
	restore := command.SetTriggerStoreForTest(store)
	defer restore()

	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"trigger", "events", "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("trigger events --output json failed: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, `"events"`) {
		t.Errorf("JSON output should contain 'events' key, got: %s", output)
	}
}
