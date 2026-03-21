package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/autonomy"
)

func TestMonitorConfig_JSONContract(t *testing.T) {
	cfg := &autonomy.MonitorConfig{
		MonitorID: "mon_abc123",
		RepoOwner: "owner",
		RepoName:  "repo",
		Interval:  "5m",
		Checks:    []string{"health", "ci"},
		Enabled:   true,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{
		`"monitor_id"`, `"repo_owner"`, `"repo_name"`,
		`"interval"`, `"checks"`, `"enabled"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}

	var decoded autonomy.MonitorConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.MonitorID != cfg.MonitorID {
		t.Errorf("MonitorID = %q, want %q", decoded.MonitorID, cfg.MonitorID)
	}
	if len(decoded.Checks) != 2 {
		t.Errorf("Checks length = %d, want 2", len(decoded.Checks))
	}
}

func TestMonitorEvent_JSONContract(t *testing.T) {
	ev := &autonomy.MonitorEvent{
		EventID:   "ev_abc123",
		MonitorID: "mon_abc123",
		RepoOwner: "owner",
		RepoName:  "repo",
		CheckName: "health",
		Status:    "ok",
		Message:   "all good",
		Timestamp: time.Now().UTC(),
	}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	fields := []string{
		`"event_id"`, `"monitor_id"`, `"repo_owner"`, `"repo_name"`,
		`"check_name"`, `"status"`, `"message"`, `"timestamp"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}

	var decoded autonomy.MonitorEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.EventID != ev.EventID {
		t.Errorf("EventID = %q, want %q", decoded.EventID, ev.EventID)
	}
}
