package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func TestDetectPlatform(t *testing.T) {
	client := New("token", "group%2Frepo")
	got, err := client.DetectPlatform(context.Background(), "https://gitlab.com/group/repo.git")
	if err != nil || got != platform.PlatformGitLab {
		t.Fatalf("unexpected detect result: %v %v", got, err)
	}
}

func TestMergeRequestExecutorCreateValidateRollback(t *testing.T) {
	current := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/projects/group/repo/merge_requests":
			current = map[string]any{
				"iid":           7,
				"title":         "Ship pages",
				"description":   "Prepare rollout",
				"source_branch": "feature/pages",
				"target_branch": "main",
				"state":         "opened",
			}
			_ = json.NewEncoder(w).Encode(current)
		case r.Method == http.MethodGet && r.URL.Path == "/projects/group/repo/merge_requests/7":
			_ = json.NewEncoder(w).Encode(current)
		case r.Method == http.MethodPut && r.URL.Path == "/projects/group/repo/merge_requests/7":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if stateEvent, ok := body["state_event"].(string); ok && stateEvent == "close" {
				current["state"] = "closed"
			}
			_ = json.NewEncoder(w).Encode(current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "group/repo")
	client.baseURL = server.URL
	exec := client.AdminExecutors()["merge_requests"]
	payload := mustJSONRaw(t, map[string]any{
		"title":         "Ship pages",
		"description":   "Prepare rollout",
		"source_branch": "feature/pages",
		"target_branch": "main",
	})
	result, err := exec.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceID != "7" {
		t.Fatalf("unexpected resource id %q", result.ResourceID)
	}
	validation, err := exec.Validate(context.Background(), platform.AdminValidationRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %+v", validation)
	}
	rollback, err := exec.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func mustJSONRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
