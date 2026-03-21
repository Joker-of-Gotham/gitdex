package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/collaboration"
)

func TestTriageResult_JSONContract(t *testing.T) {
	original := &collaboration.TriageResult{
		ObjectRef:       "owner/repo#1",
		Priority:        collaboration.TriageCritical,
		Reason:          "security label",
		SuggestedAction: "address immediately",
		Tags:            []string{"issue", "open", "security"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(data)
	required := []string{"object_ref", "priority", "reason", "suggested_action", "tags"}
	for _, f := range required {
		if !strings.Contains(jsonStr, "\""+f+"\"") {
			t.Errorf("JSON missing snake_case field %q", f)
		}
	}

	var decoded collaboration.TriageResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ObjectRef != original.ObjectRef {
		t.Errorf("ObjectRef: got %q, want %q", decoded.ObjectRef, original.ObjectRef)
	}
	if decoded.Priority != original.Priority {
		t.Errorf("Priority: got %q, want %q", decoded.Priority, original.Priority)
	}
	if decoded.Reason != original.Reason {
		t.Errorf("Reason: got %q, want %q", decoded.Reason, original.Reason)
	}
	if decoded.SuggestedAction != original.SuggestedAction {
		t.Errorf("SuggestedAction: got %q, want %q", decoded.SuggestedAction, original.SuggestedAction)
	}
	if len(decoded.Tags) != len(original.Tags) {
		t.Errorf("Tags len: got %d, want %d", len(decoded.Tags), len(original.Tags))
	}
}

func TestActivitySummary_JSONContract(t *testing.T) {
	original := &collaboration.ActivitySummary{
		RepoOwner:    "owner",
		RepoName:     "repo",
		Period:       "7d",
		TotalObjects: 5,
		ByType:       map[string]int{"issue": 3, "pull_request": 2},
		ByPriority:   map[string]int{"high": 1, "medium": 4},
		TopItems: []collaboration.TriageResult{
			{ObjectRef: "owner/repo#1", Priority: collaboration.TriageHigh, Reason: "bug label"},
		},
		GeneratedAt: time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(data)
	required := []string{"repo_owner", "repo_name", "period", "total_objects", "by_type", "by_priority", "top_items", "generated_at"}
	for _, f := range required {
		if !strings.Contains(jsonStr, "\""+f+"\"") {
			t.Errorf("JSON missing snake_case field %q", f)
		}
	}

	var decoded collaboration.ActivitySummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.RepoOwner != original.RepoOwner {
		t.Errorf("RepoOwner: got %q, want %q", decoded.RepoOwner, original.RepoOwner)
	}
	if decoded.RepoName != original.RepoName {
		t.Errorf("RepoName: got %q, want %q", decoded.RepoName, original.RepoName)
	}
	if decoded.Period != original.Period {
		t.Errorf("Period: got %q, want %q", decoded.Period, original.Period)
	}
	if decoded.TotalObjects != original.TotalObjects {
		t.Errorf("TotalObjects: got %d, want %d", decoded.TotalObjects, original.TotalObjects)
	}
	if !decoded.GeneratedAt.Equal(original.GeneratedAt) {
		t.Errorf("GeneratedAt: got %v, want %v", decoded.GeneratedAt, original.GeneratedAt)
	}
}
