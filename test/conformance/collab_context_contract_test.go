package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/collaboration"
)

func TestObjectLink_JSONContract(t *testing.T) {
	original := &collaboration.ObjectLink{
		SourceRef: "owner/repo#1",
		TargetRef: "owner/repo#2",
		LinkType:  collaboration.LinkBlocks,
		CreatedAt: time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(data)
	required := []string{"source_ref", "target_ref", "link_type", "created_at"}
	for _, f := range required {
		if !strings.Contains(jsonStr, "\""+f+"\"") {
			t.Errorf("JSON missing snake_case field %q", f)
		}
	}

	var decoded collaboration.ObjectLink
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.SourceRef != original.SourceRef {
		t.Errorf("SourceRef: got %q, want %q", decoded.SourceRef, original.SourceRef)
	}
	if decoded.TargetRef != original.TargetRef {
		t.Errorf("TargetRef: got %q, want %q", decoded.TargetRef, original.TargetRef)
	}
	if decoded.LinkType != original.LinkType {
		t.Errorf("LinkType: got %q, want %q", decoded.LinkType, original.LinkType)
	}
	if !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", decoded.CreatedAt, original.CreatedAt)
	}
}

func TestTaskContext_JSONContract(t *testing.T) {
	original := &collaboration.TaskContext{
		ContextID:        "ctx-123",
		PrimaryObjectRef: "owner/repo#1",
		LinkedObjects: []collaboration.ObjectLink{
			{
				SourceRef: "owner/repo#1",
				TargetRef: "owner/repo#2",
				LinkType:  collaboration.LinkRelatesTo,
				CreatedAt: time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
			},
		},
		RelatedTasks: []string{"task-a"},
		Notes:        "test notes",
		CreatedAt:    time.Date(2026, 3, 19, 13, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(data)
	required := []string{"context_id", "primary_object_ref", "linked_objects", "related_tasks", "notes", "created_at"}
	for _, f := range required {
		if !strings.Contains(jsonStr, "\""+f+"\"") {
			t.Errorf("JSON missing snake_case field %q", f)
		}
	}

	var decoded collaboration.TaskContext
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ContextID != original.ContextID {
		t.Errorf("ContextID: got %q, want %q", decoded.ContextID, original.ContextID)
	}
	if decoded.PrimaryObjectRef != original.PrimaryObjectRef {
		t.Errorf("PrimaryObjectRef: got %q, want %q", decoded.PrimaryObjectRef, original.PrimaryObjectRef)
	}
	if len(decoded.LinkedObjects) != len(original.LinkedObjects) {
		t.Errorf("LinkedObjects len: got %d, want %d", len(decoded.LinkedObjects), len(original.LinkedObjects))
	}
	if len(decoded.RelatedTasks) != len(original.RelatedTasks) {
		t.Errorf("RelatedTasks len: got %d, want %d", len(decoded.RelatedTasks), len(original.RelatedTasks))
	}
	if decoded.Notes != original.Notes {
		t.Errorf("Notes: got %q, want %q", decoded.Notes, original.Notes)
	}
	if !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", decoded.CreatedAt, original.CreatedAt)
	}
}
