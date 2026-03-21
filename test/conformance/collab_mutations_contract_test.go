package conformance

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/collaboration"
)

func TestMutationRequest_JSONSnakeCase(t *testing.T) {
	num := 1
	req := &collaboration.MutationRequest{
		MutationType: collaboration.MutationCreate,
		ObjectType:   collaboration.ObjectTypeIssue,
		RepoOwner:    "o",
		RepoName:     "r",
		Number:       &num,
		Title:        "t",
		Body:         "b",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonStr := string(data)
	required := []string{"mutation_type", "object_type", "repo_owner", "repo_name"}
	for _, f := range required {
		if !strings.Contains(jsonStr, "\""+f+"\"") {
			t.Errorf("JSON missing snake_case field %q", f)
		}
	}
}

func TestMutationResult_JSONSnakeCase(t *testing.T) {
	result := &collaboration.MutationResult{
		Request: collaboration.MutationRequest{
			MutationType: collaboration.MutationCreate,
		},
		Success: true,
		Message: "ok",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "mutation_type") {
		t.Error("missing mutation_type in request")
	}
	if !strings.Contains(jsonStr, "success") {
		t.Error("missing success")
	}
	if !strings.Contains(jsonStr, "message") {
		t.Error("missing message")
	}
}

func TestMutationResult_RoundTrip(t *testing.T) {
	num := 42
	original := &collaboration.MutationResult{
		Request: collaboration.MutationRequest{
			MutationType: collaboration.MutationComment,
			RepoOwner:    "owner",
			RepoName:     "repo",
			Number:       &num,
			Body:         "comment body",
		},
		Success: true,
		Message: "comment added",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded collaboration.MutationResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Success != original.Success {
		t.Errorf("Success: got %v, want %v", decoded.Success, original.Success)
	}
	if decoded.Request.MutationType != original.Request.MutationType {
		t.Errorf("MutationType: got %s, want %s", decoded.Request.MutationType, original.Request.MutationType)
	}
}
