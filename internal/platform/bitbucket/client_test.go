package bitbucket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func TestDetectPlatform(t *testing.T) {
	client := New("token", "team", "repo")
	got, err := client.DetectPlatform(context.Background(), "git@bitbucket.org:team/repo.git")
	if err != nil || got != platform.PlatformBitbucket {
		t.Fatalf("unexpected detect result: %v %v", got, err)
	}
}

func TestRepositoryVariableExecutorCreateValidateRollback(t *testing.T) {
	current := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repositories/team/repo/pipelines_config/variables/":
			current = map[string]any{
				"uuid":    "{var-1}",
				"key":     "DEPLOY_ENV",
				"value":   "prod",
				"secured": false,
			}
			_ = json.NewEncoder(w).Encode(current)
		case r.Method == http.MethodGet && r.URL.Path == "/repositories/team/repo/pipelines_config/variables/{var-1}":
			_ = json.NewEncoder(w).Encode(current)
		case r.Method == http.MethodDelete && r.URL.Path == "/repositories/team/repo/pipelines_config/variables/{var-1}":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "team", "repo")
	client.baseURL = server.URL
	exec := client.AdminExecutors()["repository_variables"]
	payload := mustJSONRaw(t, map[string]any{
		"key":     "DEPLOY_ENV",
		"value":   "prod",
		"secured": false,
	})
	result, err := exec.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceID != "{var-1}" {
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
