package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/emergency"
)

func TestControlRequest_JSONContract(t *testing.T) {
	req := emergency.ControlRequest{
		Action:    emergency.ControlPauseTask,
		Scope:     "task_001",
		Reason:    "test",
		Actor:     "admin",
		Timestamp: time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"action"`,
		`"scope"`,
		`"reason"`,
		`"actor"`,
		`"timestamp"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestControlResult_JSONContract(t *testing.T) {
	result := &emergency.ControlResult{
		Request: emergency.ControlRequest{
			Action: emergency.ControlKillSwitch,
			Scope:  "*",
			Actor:  "ops",
		},
		Success:        true,
		AffectedTasks:  []string{"*"},
		AffectedScopes: []string{"*"},
		Message:        "kill switch activated",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"request"`,
		`"success"`,
		`"affected_tasks"`,
		`"affected_scopes"`,
		`"message"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestControlAction_AllValues(t *testing.T) {
	actions := []emergency.ControlAction{
		emergency.ControlPauseTask,
		emergency.ControlPauseScope,
		emergency.ControlSuspendCapability,
		emergency.ControlKillSwitch,
	}

	seen := make(map[emergency.ControlAction]bool)
	for _, a := range actions {
		if a == "" {
			t.Error("control action should not be empty")
		}
		if seen[a] {
			t.Errorf("duplicate action: %s", a)
		}
		seen[a] = true
	}
}

func TestEmergencyContract_ControlRequest_JSONRoundTrip(t *testing.T) {
	orig := emergency.ControlRequest{
		Action:    emergency.ControlPauseTask,
		Scope:     "task_001",
		Reason:    "test reason",
		Actor:     "admin",
		Timestamp: time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded emergency.ControlRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Action != orig.Action {
		t.Errorf("Action: got %q, want %q", decoded.Action, orig.Action)
	}
	if decoded.Scope != orig.Scope {
		t.Errorf("Scope: got %q, want %q", decoded.Scope, orig.Scope)
	}
	if decoded.Reason != orig.Reason {
		t.Errorf("Reason: got %q, want %q", decoded.Reason, orig.Reason)
	}
	if decoded.Actor != orig.Actor {
		t.Errorf("Actor: got %q, want %q", decoded.Actor, orig.Actor)
	}
	if !decoded.Timestamp.Equal(orig.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", decoded.Timestamp, orig.Timestamp)
	}
}

func TestEmergencyContract_ControlResult_JSONRoundTrip(t *testing.T) {
	orig := &emergency.ControlResult{
		Request: emergency.ControlRequest{
			Action: emergency.ControlKillSwitch,
			Scope:  "*",
			Actor:  "ops",
		},
		Success:        true,
		AffectedTasks:  []string{"task_1", "task_2"},
		AffectedScopes: []string{"scope_a"},
		Message:        "kill switch activated",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded emergency.ControlResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Success != orig.Success {
		t.Errorf("Success: got %v, want %v", decoded.Success, orig.Success)
	}
	if decoded.Request.Action != orig.Request.Action {
		t.Errorf("Request.Action: got %q, want %q", decoded.Request.Action, orig.Request.Action)
	}
	if len(decoded.AffectedTasks) != len(orig.AffectedTasks) {
		t.Errorf("AffectedTasks len: got %d, want %d", len(decoded.AffectedTasks), len(orig.AffectedTasks))
	}
	if len(decoded.AffectedScopes) != len(orig.AffectedScopes) {
		t.Errorf("AffectedScopes len: got %d, want %d", len(decoded.AffectedScopes), len(orig.AffectedScopes))
	}
	if decoded.Message != orig.Message {
		t.Errorf("Message: got %q, want %q", decoded.Message, orig.Message)
	}
}
