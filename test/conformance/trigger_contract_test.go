package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/autonomy"
)

func TestTriggerConfig_JSONContract(t *testing.T) {
	cfg := &autonomy.TriggerConfig{
		TriggerID:      "tr_abc123",
		TriggerType:    autonomy.TriggerSchedule,
		Name:           "nightly-sync",
		Source:         "github",
		Pattern:        "0 0 * * *",
		ActionTemplate: "repo sync",
		Enabled:        true,
		CreatedAt:      time.Now().UTC(),
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{
		`"trigger_id"`, `"trigger_type"`, `"name"`,
		`"source"`, `"pattern"`, `"action_template"`,
		`"enabled"`, `"created_at"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}

	var decoded autonomy.TriggerConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.TriggerID != cfg.TriggerID {
		t.Errorf("TriggerID = %q, want %q", decoded.TriggerID, cfg.TriggerID)
	}
	if decoded.TriggerType != autonomy.TriggerSchedule {
		t.Errorf("TriggerType = %q, want schedule", decoded.TriggerType)
	}
}

func TestTriggerEvent_JSONContract(t *testing.T) {
	ev := &autonomy.TriggerEvent{
		EventID:         "tev_abc123",
		TriggerID:       "tr_abc123",
		TriggerType:     autonomy.TriggerSchedule,
		SourceEvent:     "cron",
		ResultingTaskID: "task_001",
		Timestamp:       time.Now().UTC(),
	}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{
		`"event_id"`, `"trigger_id"`, `"trigger_type"`,
		`"source_event"`, `"resulting_task_id"`, `"timestamp"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}

	var decoded autonomy.TriggerEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.ResultingTaskID != "task_001" {
		t.Errorf("ResultingTaskID = %q, want task_001", decoded.ResultingTaskID)
	}
}

func TestTriggerType_AllValues(t *testing.T) {
	types := []autonomy.TriggerType{
		autonomy.TriggerTypeEvent, autonomy.TriggerSchedule,
		autonomy.TriggerAPI, autonomy.TriggerOperator,
	}
	seen := make(map[autonomy.TriggerType]bool)
	for _, tt := range types {
		if tt == "" {
			t.Error("trigger type should not be empty")
		}
		if seen[tt] {
			t.Errorf("duplicate trigger type: %s", tt)
		}
		seen[tt] = true
	}
	if len(types) != 4 {
		t.Errorf("expected 4 trigger types, got %d", len(types))
	}
}
