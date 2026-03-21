package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/collaboration"
)

func TestCollaborationObject_JSONSnakeCase(t *testing.T) {
	obj := &collaboration.CollaborationObject{
		ObjectID:      "id1",
		ObjectType:    collaboration.ObjectTypeIssue,
		RepoOwner:     "owner",
		RepoName:      "repo",
		Number:        1,
		Title:         "title",
		State:         "open",
		Author:        "alice",
		CommentsCount: 5,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(data)
	required := []string{"object_id", "object_type", "repo_owner", "repo_name", "comments_count", "created_at", "updated_at"}
	for _, f := range required {
		if !strings.Contains(jsonStr, "\""+f+"\"") {
			t.Errorf("JSON missing snake_case field %q", f)
		}
	}
}

func TestCollaborationObject_RoundTrip(t *testing.T) {
	original := &collaboration.CollaborationObject{
		ObjectID:      "id1",
		ObjectType:    collaboration.ObjectTypePullRequest,
		RepoOwner:     "o",
		RepoName:      "r",
		Number:        42,
		Title:         "PR title",
		State:         "open",
		Author:        "bob",
		Labels:        []string{"bug"},
		Assignees:     []string{"alice"},
		CommentsCount: 3,
		CreatedAt:     time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 3, 19, 11, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded collaboration.CollaborationObject
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Number != original.Number {
		t.Errorf("Number: got %d, want %d", decoded.Number, original.Number)
	}
	if decoded.ObjectType != original.ObjectType {
		t.Errorf("ObjectType: got %s, want %s", decoded.ObjectType, original.ObjectType)
	}
}

func TestObjectFilter_JSONSnakeCase(t *testing.T) {
	f := &collaboration.ObjectFilter{
		ObjectType:  collaboration.ObjectTypeIssue,
		State:       "open",
		RepoOwner:   "o",
		RepoName:    "r",
		SearchQuery: "q",
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "object_type") {
		t.Error("missing object_type")
	}
	if !strings.Contains(jsonStr, "search_query") {
		t.Error("missing search_query")
	}
}
