package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/nacl/box"
)

func TestAdminExecutorsExposeAdminBatches(t *testing.T) {
	client := New("token", "owner", "repo")
	executors := client.AdminExecutors()
	for _, capability := range []string{
		"actions",
		"rulesets",
		"branch_rulesets",
		"check_runs_failure_threshold",
		"actions_secrets_variables",
		"codespaces",
		"codespaces_secrets",
		"dependabot_secrets",
		"dependabot_config",
		"webhooks",
		"pages",
		"deployment",
		"environments",
		"release",
		"pull_request",
		"pr_review",
		"deploy_keys",
		"packages",
		"notifications",
		"email_notifications",
		"security",
		"advanced_security",
		"dependabot_posture",
		"dependency_graph",
		"dependabot",
		"dependabot_security_updates",
		"grouped_security_updates",
		"dependabot_version_updates",
		"secret_scanning_settings",
		"dependabot_alerts",
		"secret_scanning_alerts",
		"code_scanning",
		"code_scanning_tool_settings",
		"code_scanning_default_setup",
		"codeql_setup",
		"codeql_analysis",
		"copilot_autofix",
		"secret_protection",
		"private_vulnerability_reporting",
		"protection_rules",
		"push_protection",
		"copilot_code_review",
		"copilot_coding_agent",
		"copilot_seat_management",
	} {
		if _, ok := executors[capability]; !ok {
			t.Fatalf("missing executor for %s", capability)
		}
	}
}

func TestRulesetExecutorCreateValidateRollback(t *testing.T) {
	var current map[string]any
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/rulesets":
			body := readBody(t, r)
			current = body
			current["id"] = float64(42)
			writeJSON(t, w, http.StatusCreated, current)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/rulesets/42":
			if current == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/rulesets/42":
			current = nil
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["rulesets"]

	payload := rawJSON(t, map[string]any{
		"name":        "Protect main",
		"target":      "branch",
		"enforcement": "active",
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceID != "42" {
		t.Fatalf("expected resource id 42, got %s", result.ResourceID)
	}

	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %s", validation.Summary)
	}

	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestActionsVariableExecutorCreateValidateRollback(t *testing.T) {
	var current map[string]any
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/actions/variables/APP_MODE":
			if current == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/actions/variables":
			current = readBody(t, r)
			writeJSON(t, w, http.StatusCreated, map[string]any{})
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/actions/variables/APP_MODE":
			current = nil
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["actions_secrets_variables"]

	payload := rawJSON(t, map[string]any{
		"kind":  "variable",
		"name":  "APP_MODE",
		"value": "prod",
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Scope:     map[string]string{"kind": "variable"},
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}

	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Scope:    map[string]string{"kind": "variable"},
		Payload: rawJSON(t, map[string]any{
			"name":  "APP_MODE",
			"value": "prod",
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %s", validation.Summary)
	}

	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestActionsSecretExecutorEncryptsAndValidatesMetadata(t *testing.T) {
	publicKey, privateKey, err := box.GenerateKey(strings.NewReader(strings.Repeat("a", 64)))
	if err != nil {
		t.Fatal(err)
	}
	var encryptedValue string
	var keyID string
	secretMeta := map[string]any{"name": "DEPLOY_TOKEN", "updated_at": "2026-03-11T10:00:00Z"}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/actions/secrets/DEPLOY_TOKEN":
			if encryptedValue == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, secretMeta)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/actions/secrets/public-key":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"key":    base64.StdEncoding.EncodeToString(publicKey[:]),
				"key_id": "key-1",
			})
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/actions/secrets/DEPLOY_TOKEN":
			body := readBody(t, r)
			encryptedValue, _ = body["encrypted_value"].(string)
			keyID, _ = body["key_id"].(string)
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["actions_secrets_variables"]

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Scope:     map[string]string{"kind": "secret"},
		Payload: rawJSON(t, map[string]any{
			"kind":  "secret",
			"name":  "DEPLOY_TOKEN",
			"value": "super-secret",
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if keyID != "key-1" || encryptedValue == "" {
		t.Fatalf("expected encrypted secret request, got key=%s value=%q", keyID, encryptedValue)
	}
	if got := decryptSecretForTest(t, encryptedValue, privateKey, publicKey); got != "super-secret" {
		t.Fatalf("unexpected decrypted value %q", got)
	}

	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Scope:    map[string]string{"kind": "secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %s", validation.Summary)
	}
}

func TestWebhookExecutorUpdateRollback(t *testing.T) {
	current := map[string]any{
		"id":     float64(7),
		"name":   "web",
		"active": true,
		"events": []any{"push"},
		"config": map[string]any{"url": "https://old.example/webhook"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/hooks/7":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/hooks/7":
			body := readBody(t, r)
			for key, value := range body {
				current[key] = value
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["webhooks"]
	payload := rawJSON(t, map[string]any{
		"active": false,
		"config": map[string]any{"url": "https://new.example/webhook"},
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "update",
		ResourceID: "7",
		Payload:    payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation:   result,
		ResourceID: "7",
		Payload:    payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestPagesExecutorUpdateRollback(t *testing.T) {
	current := map[string]any{
		"cname":      "old.example.com",
		"build_type": "workflow",
		"source":     map[string]any{"branch": "main", "path": "/docs"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pages":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/pages":
			body := readBody(t, r)
			for key, value := range body {
				current[key] = value
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["pages"]
	payload := rawJSON(t, map[string]any{
		"cname": "new.example.com",
		"source": map[string]any{
			"branch": "gh-pages",
			"path":   "/",
		},
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "update",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestPagesExecutorBuildAndDNSValidation(t *testing.T) {
	current := map[string]any{
		"cname":      "localhost",
		"build_type": "workflow",
		"source":     map[string]any{"branch": "main", "path": "/docs"},
	}
	latestBuild := map[string]any{"status": "built", "commit": "abc123"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pages":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/pages/builds":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pages/builds/latest":
			writeJSON(t, w, http.StatusOK, latestBuild)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["pages"]

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "build",
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected pages build validation success, got %s", validation.Summary)
	}

	validation, err = executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: &platform.AdminMutationResult{CapabilityID: "pages", Operation: "update"},
		Payload:  rawJSON(t, map[string]any{"cname": "localhost"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected pages DNS validation success, got %s", validation.Summary)
	}
}

func TestReleaseExecutorAssetUploadValidateRollbackAndPublishDraft(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "release-asset-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	if _, err := tmpFile.WriteString("release asset"); err != nil {
		t.Fatal(err)
	}

	release := map[string]any{
		"id":         float64(11),
		"tag_name":   "v1.0.0",
		"name":       "v1.0.0",
		"draft":      true,
		"upload_url": "https://uploads.example.com/repos/owner/repo/releases/11/assets{?name,label}",
	}
	asset := map[string]any{
		"id":                   float64(99),
		"name":                 "gitdex.txt",
		"content_type":         "text/plain",
		"label":                "notes",
		"browser_download_url": "https://download.example.com/gitdex.txt",
		"url":                  "https://api.example.com/assets/99",
	}
	var uploadQuery url.Values
	var uploadedBytes []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/releases/11":
			writeJSON(t, w, http.StatusOK, release)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/releases/11":
			body := readBody(t, r)
			for key, value := range body {
				release[key] = value
			}
			writeJSON(t, w, http.StatusOK, release)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/repos/owner/repo/releases/generate-notes"):
			writeJSON(t, w, http.StatusOK, map[string]any{"body": "generated"})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/releases/assets/99":
			writeJSON(t, w, http.StatusOK, asset)
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/releases/assets/99":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected api request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected upload request %s", r.Method)
		}
		uploadQuery = r.URL.Query()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		uploadedBytes = data
		writeJSON(t, w, http.StatusCreated, asset)
	}))
	defer uploadServer.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["release"]

	release["upload_url"] = uploadServer.URL + "/repos/owner/repo/releases/11/assets{?name,label}"

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "asset_upload",
		Scope:     map[string]string{"release_id": "11"},
		Payload: rawJSON(t, map[string]any{
			"name":         "gitdex.txt",
			"label":        "notes",
			"content_type": "text/plain",
			"file_path":    tmpFile.Name(),
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(uploadedBytes) != "release asset" {
		t.Fatalf("unexpected uploaded asset body %q", string(uploadedBytes))
	}
	if uploadQuery.Get("name") != "gitdex.txt" || uploadQuery.Get("label") != "notes" {
		t.Fatalf("unexpected upload query %v", uploadQuery)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected release asset validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected release asset rollback success, got %+v", rollback)
	}

	publishResult, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "publish_draft",
		ResourceID: "11",
	})
	if err != nil {
		t.Fatal(err)
	}
	publishValidation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{Mutation: publishResult})
	if err != nil {
		t.Fatal(err)
	}
	if !publishValidation.OK {
		t.Fatalf("expected publish draft validation success, got %s", publishValidation.Summary)
	}
}

func TestDeploymentExecutorCreateAndRollback(t *testing.T) {
	var deployment map[string]any
	var inactivePosted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/deployments":
			deployment = readBody(t, r)
			deployment["id"] = float64(9)
			writeJSON(t, w, http.StatusCreated, deployment)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/deployments/9":
			if deployment == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, deployment)
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/deployments/9/statuses":
			body := readBody(t, r)
			if body["state"] == "inactive" {
				inactivePosted = true
			}
			writeJSON(t, w, http.StatusCreated, body)
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/deployments/9":
			deployment = nil
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["deployment"]

	payload := rawJSON(t, map[string]any{"ref": "main", "environment": "prod"})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK || !inactivePosted {
		t.Fatalf("expected inactive status and rollback success, got %+v", rollback)
	}
}

func TestEnvironmentExecutorUpdateRollback(t *testing.T) {
	current := map[string]any{
		"name":       "prod",
		"wait_timer": float64(0),
		"deployment_branch_policy": map[string]any{
			"protected_branches": true,
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/environments/prod":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/environments/prod":
			body := readBody(t, r)
			for key, value := range body {
				current[key] = value
			}
			current["name"] = "prod"
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["environments"]

	payload := rawJSON(t, map[string]any{
		"wait_timer": 30,
		"deployment_branch_policy": map[string]any{
			"protected_branches": false,
		},
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "update",
		ResourceID: "prod",
		Payload:    payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation:   result,
		ResourceID: "prod",
		Payload:    payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestReleaseExecutorCreateValidateRollback(t *testing.T) {
	var current map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/releases":
			current = readBody(t, r)
			current["id"] = float64(77)
			writeJSON(t, w, http.StatusCreated, current)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/releases/77":
			if current == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/releases/77":
			current = nil
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["release"]
	payload := rawJSON(t, map[string]any{
		"tag_name":   "v1.2.3",
		"name":       "v1.2.3",
		"draft":      false,
		"prerelease": false,
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestPullRequestExecutorCreateValidateRollback(t *testing.T) {
	var current map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/pulls":
			body := readBody(t, r)
			current = map[string]any{
				"number":                float64(42),
				"node_id":               "PR_node_42",
				"title":                 stringValue(body["title"]),
				"body":                  stringValue(body["body"]),
				"state":                 "open",
				"draft":                 body["draft"],
				"maintainer_can_modify": true,
				"merged":                false,
				"auto_merge":            nil,
				"head":                  map[string]any{"ref": stringValue(body["head"])},
				"base":                  map[string]any{"ref": stringValue(body["base"])},
			}
			writeJSON(t, w, http.StatusCreated, current)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pulls/42":
			if current == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/pulls/42":
			body := readBody(t, r)
			if value, ok := body["title"]; ok {
				current["title"] = value
			}
			if value, ok := body["body"]; ok {
				current["body"] = value
			}
			if value, ok := body["state"]; ok {
				current["state"] = value
			}
			if value, ok := body["maintainer_can_modify"]; ok {
				current["maintainer_can_modify"] = value
			}
			if value, ok := body["base"]; ok {
				current["base"] = map[string]any{"ref": stringValue(value)}
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["pull_request"]

	payload := rawJSON(t, map[string]any{
		"title": "Release v1.2.3",
		"body":  "Prepare ship PR",
		"head":  "release/v1.2.3",
		"base":  "main",
		"draft": false,
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceID != "42" {
		t.Fatalf("expected pull number 42, got %s", result.ResourceID)
	}

	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation:   result,
		ResourceID: "42",
		Payload:    payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected pull request validation success, got %s", validation.Summary)
	}

	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestPullRequestExecutorUpdateCloseAndAutoMerge(t *testing.T) {
	current := map[string]any{
		"number":                float64(42),
		"node_id":               "PR_node_42",
		"title":                 "Initial title",
		"body":                  "Initial body",
		"state":                 "open",
		"draft":                 false,
		"maintainer_can_modify": true,
		"merged":                false,
		"auto_merge":            nil,
		"head":                  map[string]any{"ref": "feature/refactor"},
		"base":                  map[string]any{"ref": "main"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pulls/42":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/pulls/42":
			body := readBody(t, r)
			if value, ok := body["title"]; ok {
				current["title"] = value
			}
			if value, ok := body["body"]; ok {
				current["body"] = value
			}
			if value, ok := body["state"]; ok {
				current["state"] = value
			}
			if value, ok := body["maintainer_can_modify"]; ok {
				current["maintainer_can_modify"] = value
			}
			if value, ok := body["base"]; ok {
				current["base"] = map[string]any{"ref": stringValue(value)}
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			body := readBody(t, r)
			query := stringValue(body["query"])
			if strings.Contains(query, "EnableAutoMerge") {
				current["auto_merge"] = map[string]any{
					"merge_method":   "SQUASH",
					"commit_title":   "Ship it",
					"commit_message": "Ready",
				}
			}
			if strings.Contains(query, "DisableAutoMerge") {
				current["auto_merge"] = nil
			}
			writeJSON(t, w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true}})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["pull_request"]

	updatePayload := rawJSON(t, map[string]any{
		"title":                 "Updated title",
		"body":                  "Updated body",
		"base":                  "main",
		"maintainer_can_modify": false,
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "update",
		ResourceID: "42",
		Payload:    updatePayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation:   result,
		ResourceID: "42",
		Payload:    updatePayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected update validation success, got %s", validation.Summary)
	}

	closeResult, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "close",
		ResourceID: "42",
	})
	if err != nil {
		t.Fatal(err)
	}
	closeValidation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation:   closeResult,
		ResourceID: "42",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !closeValidation.OK {
		t.Fatalf("expected close validation success, got %s", closeValidation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: closeResult})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected close rollback success, got %+v", rollback)
	}

	autoMergePayload := rawJSON(t, map[string]any{
		"merge_method":   "squash",
		"commit_title":   "Ship it",
		"commit_message": "Ready",
	})
	autoMergeResult, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "enable_auto_merge",
		ResourceID: "42",
		Payload:    autoMergePayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	autoMergeValidation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation:   autoMergeResult,
		ResourceID: "42",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !autoMergeValidation.OK {
		t.Fatalf("expected auto-merge validation success, got %s", autoMergeValidation.Summary)
	}
	autoMergeRollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: autoMergeResult})
	if err != nil {
		t.Fatal(err)
	}
	if !autoMergeRollback.OK {
		t.Fatalf("expected auto-merge rollback success, got %+v", autoMergeRollback)
	}
}

func TestPullRequestExecutorMergeValidateRollbackBoundary(t *testing.T) {
	current := map[string]any{
		"number":                float64(42),
		"node_id":               "PR_node_42",
		"title":                 "Ship release",
		"state":                 "open",
		"maintainer_can_modify": true,
		"merged":                false,
		"head":                  map[string]any{"ref": "release/v1.2.3"},
		"base":                  map[string]any{"ref": "main"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pulls/42":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/pulls/42/merge":
			current["state"] = "closed"
			current["merged"] = true
			writeJSON(t, w, http.StatusOK, map[string]any{"merged": true, "message": "Pull Request successfully merged"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["pull_request"]
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "merge",
		ResourceID: "42",
		Payload:    rawJSON(t, map[string]any{"merge_method": "squash"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation:   result,
		ResourceID: "42",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected merge validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if rollback.OK {
		t.Fatalf("expected merge rollback to be unavailable, got %+v", rollback)
	}
}

func TestPRReviewExecutorApproveValidateRollback(t *testing.T) {
	var review map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/pulls/12/reviews":
			review = readBody(t, r)
			review["id"] = float64(501)
			review["state"] = "APPROVED"
			writeJSON(t, w, http.StatusCreated, review)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pulls/12/reviews/501":
			writeJSON(t, w, http.StatusOK, review)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/pulls/12/reviews/501/dismissals":
			review["state"] = "DISMISSED"
			writeJSON(t, w, http.StatusOK, review)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["pr_review"]
	payload := rawJSON(t, map[string]any{"body": "ship it"})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "approve",
		Scope:     map[string]string{"pull_number": "12"},
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Scope:    map[string]string{"pull_number": "12"},
		Payload:  rawJSON(t, map[string]any{"body": "ship it", "state": "APPROVED"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected review validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{
		Mutation: result,
		Scope:    map[string]string{"pull_number": "12"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestDeployKeyExecutorCreateValidateRollback(t *testing.T) {
	var current map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/keys":
			current = readBody(t, r)
			current["id"] = float64(66)
			writeJSON(t, w, http.StatusCreated, current)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/keys/66":
			if current == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/keys/66":
			current = nil
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["deploy_keys"]
	payload := rawJSON(t, map[string]any{
		"title":     "ci-key",
		"key":       "ssh-rsa AAAAB3Nza...",
		"read_only": true,
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected deploy key validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestRepoSecretExecutorsEncryptAndValidate(t *testing.T) {
	for _, capability := range []string{"codespaces_secrets", "dependabot_secrets"} {
		t.Run(capability, func(t *testing.T) {
			publicKey, privateKey, err := box.GenerateKey(strings.NewReader(strings.Repeat("b", 64)))
			if err != nil {
				t.Fatal(err)
			}
			var encryptedValue string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/secrets/API_KEY"):
					if encryptedValue == "" {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					writeJSON(t, w, http.StatusOK, map[string]any{"name": "API_KEY"})
				case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/secrets/public-key"):
					writeJSON(t, w, http.StatusOK, map[string]any{
						"key":    base64.StdEncoding.EncodeToString(publicKey[:]),
						"key_id": "key-2",
					})
				case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/secrets/API_KEY"):
					body := readBody(t, r)
					encryptedValue, _ = body["encrypted_value"].(string)
					w.WriteHeader(http.StatusCreated)
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			client := New("token", "owner", "repo")
			client.baseURL = server.URL
			executor := client.AdminExecutors()[capability]
			result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
				Operation: "create",
				Payload: rawJSON(t, map[string]any{
					"name":  "API_KEY",
					"value": "super-secret",
				}),
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := decryptSecretForTest(t, encryptedValue, privateKey, publicKey); got != "super-secret" {
				t.Fatalf("unexpected decrypted value %q", got)
			}
			validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
				Mutation: result,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !validation.OK {
				t.Fatalf("expected validation success, got %s", validation.Summary)
			}
		})
	}
}

func TestPackagesExecutorDeleteRestoreVersion(t *testing.T) {
	current := map[string]any{
		"id":   float64(9),
		"name": "1.0.0",
	}
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/users/owner/packages/container/app/versions/9"):
			if deleted {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/users/owner/packages/container/app/versions/9"):
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/users/owner/packages/container/app/versions/9/restore"):
			deleted = false
			writeJSON(t, w, http.StatusCreated, map[string]any{})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["packages"]
	scope := map[string]string{
		"owner_type":   "user",
		"package_type": "container",
		"package_name": "app",
	}
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "delete",
		ResourceID: "9",
		Scope:      scope,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation:   result,
		ResourceID: "9",
		Scope:      scope,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected delete validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{
		Mutation: result,
		Scope:    scope,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestPackagesExecutorInspectAssetsIncludesIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/users/owner/packages/container/app/versions/9"):
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":   float64(9),
				"name": "1.0.0",
				"files": []map[string]any{{
					"name": "app.tar.gz",
					"size": 123,
				}},
				"metadata": map[string]any{
					"container": map[string]any{"tags": []string{"latest"}},
				},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["packages"]
	scope := map[string]string{
		"owner_type":   "user",
		"package_type": "container",
		"package_name": "app",
	}
	snapshot, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{
		ResourceID: "9",
		Scope:      scope,
		Query:      map[string]string{"view": "assets"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(snapshot.State, &raw); err != nil {
		t.Fatal(err)
	}
	identity, _ := raw["identity"].(map[string]any)
	if identity["package_type"] != "container" || identity["version"] != "9" {
		t.Fatalf("unexpected package identity %+v", identity)
	}
	if _, ok := raw["assets"].([]any); !ok {
		t.Fatalf("expected assets list, got %+v", raw)
	}
	if _, ok := raw["registry_metadata"].(map[string]any); !ok {
		t.Fatalf("expected registry metadata, got %+v", raw)
	}
}

func TestPackagesExecutorRepoScopePreservesRepositoryIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/users/owner/packages/container/app/versions/9"):
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id":   float64(9),
				"name": "1.0.0",
				"files": []map[string]any{{
					"name": "app.tar.gz",
					"size": 123,
				}},
				"metadata": map[string]any{
					"container": map[string]any{"tags": []string{"latest"}},
				},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["packages"]
	scope := map[string]string{
		"scope":        "repo",
		"package_type": "container",
		"package_name": "app",
	}
	snapshot, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{
		ResourceID: "9",
		Scope:      scope,
		Query:      map[string]string{"view": "assets"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(snapshot.State, &raw); err != nil {
		t.Fatal(err)
	}
	identity, _ := raw["identity"].(map[string]any)
	if identity["scope"] != "repo" {
		t.Fatalf("expected repo scope identity, got %+v", identity)
	}
	if identity["namespace"] != "owner/repo" {
		t.Fatalf("expected repo namespace identity, got %+v", identity)
	}
}

func TestNotificationsExecutorWatchValidateRollback(t *testing.T) {
	current := map[string]any{"subscribed": true, "ignored": false}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/subscription":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/subscription":
			body := readBody(t, r)
			for key, value := range body {
				current[key] = value
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["email_notifications"]
	payload := rawJSON(t, map[string]any{"subscribed": false, "ignored": true})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "update",
		Scope:     map[string]string{"view": "repo_subscription"},
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
		Scope:    map[string]string{"view": "repo_subscription"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected notification validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{
		Mutation: result,
		Scope:    map[string]string{"view": "repo_subscription"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestBranchRulesetsExecutorInspectBranchRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/rules/branches/main" {
			writeJSON(t, w, http.StatusOK, map[string]any{"branch": "main", "ruleset_ids": []int{1, 2}})
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["branch_rulesets"]
	snap, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{
		Query: map[string]string{"view": "branch", "branch": "main"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(snap.State), `"branch":"main"`) {
		t.Fatalf("expected branch rule snapshot, got %s", string(snap.State))
	}
}

func TestAdvancedSecurityExecutorUpdateValidateRollback(t *testing.T) {
	current := map[string]any{
		"security_and_analysis": map[string]any{
			"advanced_security": map[string]any{"status": "disabled"},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo":
			body := readBody(t, r)
			if nested, ok := body["security_and_analysis"].(map[string]any); ok {
				current["security_and_analysis"] = nested
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/automated-security-fixes":
			writeJSON(t, w, http.StatusOK, map[string]any{"enabled": true})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/code-security-configuration":
			writeJSON(t, w, http.StatusOK, map[string]any{"id": 11, "name": "default"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["advanced_security"]
	payload := rawJSON(t, map[string]any{"status": "enabled"})

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "update",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected rollback success, got %+v", rollback)
	}
}

func TestDependabotPostureExecutorUpdateValidateRollback(t *testing.T) {
	enabled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"security_and_analysis": map[string]any{
					"dependabot_security_updates": map[string]any{
						"status": map[bool]string{true: "enabled", false: "disabled"}[enabled],
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/automated-security-fixes":
			writeJSON(t, w, http.StatusOK, map[string]any{"enabled": enabled})
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/automated-security-fixes":
			enabled = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/automated-security-fixes":
			enabled = false
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["dependabot_posture"]

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{Operation: "enable"})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected dependabot posture validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK || enabled {
		t.Fatalf("expected rollback to disable posture, got %+v enabled=%t", rollback, enabled)
	}
}

func TestSecretScanningSettingsExecutorUpdateValidateRollback(t *testing.T) {
	current := map[string]any{
		"security_and_analysis": map[string]any{
			"secret_scanning": map[string]any{"status": "disabled"},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo":
			body := readBody(t, r)
			if nested, ok := body["security_and_analysis"].(map[string]any); ok {
				current["security_and_analysis"] = nested
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["secret_scanning_settings"]
	payload := rawJSON(t, map[string]any{"status": "enabled"})

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "update",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected secret scanning settings validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected secret scanning settings rollback success, got %+v", rollback)
	}
}

func TestCodeScanningToolSettingsExecutorIsInspectOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/code-security-configuration":
			writeJSON(t, w, http.StatusOK, map[string]any{"id": 11, "name": "default"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["code_scanning_tool_settings"]

	snap, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(snap.State), `"name":"default"`) {
		t.Fatalf("expected tool settings snapshot, got %s", string(snap.State))
	}
	if _, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{Operation: "update"}); err == nil {
		t.Fatal("expected inspect-only mutate error")
	}
}

func TestDependabotAlertsExecutorDismissValidateRollback(t *testing.T) {
	current := map[string]any{
		"number": float64(7),
		"state":  "open",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/dependabot/alerts/7":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/dependabot/alerts/7":
			body := readBody(t, r)
			for key, value := range body {
				current[key] = value
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["dependabot_alerts"]
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "dismiss",
		ResourceID: "7",
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected dependabot alert validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected dependabot alert rollback success, got %+v", rollback)
	}
}

func TestSecretScanningAlertsExecutorResolveValidateRollback(t *testing.T) {
	current := map[string]any{
		"number": float64(17),
		"state":  "open",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/secret-scanning/alerts/17":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/secret-scanning/alerts/17":
			body := readBody(t, r)
			for key, value := range body {
				current[key] = value
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["secret_scanning_alerts"]
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "resolve",
		ResourceID: "17",
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected secret scanning alert validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected secret scanning alert rollback success, got %+v", rollback)
	}
}

func TestCodeScanningExecutorDefaultSetupUpdateRollback(t *testing.T) {
	current := map[string]any{
		"state":       "configured",
		"query_suite": "default",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/code-scanning/default-setup":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/code-scanning/default-setup":
			body := readBody(t, r)
			for key, value := range body {
				current[key] = value
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["codeql_analysis"]
	payload := rawJSON(t, map[string]any{
		"state":       "configured",
		"query_suite": "extended",
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "default_setup_update",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected code scanning default setup validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected code scanning default setup rollback success, got %+v", rollback)
	}
}

func TestCodeQLSetupExecutorDefaultSetupUpdateRollback(t *testing.T) {
	current := map[string]any{
		"state":       "configured",
		"query_suite": "default",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/code-scanning/default-setup":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/code-scanning/default-setup":
			body := readBody(t, r)
			for key, value := range body {
				current[key] = value
			}
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["codeql_setup"]
	payload := rawJSON(t, map[string]any{
		"state":       "configured",
		"query_suite": "extended",
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "default_setup_update",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected codeql setup validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected codeql setup rollback success, got %+v", rollback)
	}
}

func TestCopilotExecutorUpdateContentExclusionsRollback(t *testing.T) {
	current := map[string]any{
		"paths": []any{"/vendor/**"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/owner/copilot/content_exclusions":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/orgs/owner/copilot/content_exclusions":
			current = readBody(t, r)
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodDelete && r.URL.Path == "/orgs/owner/copilot/content_exclusions":
			current = map[string]any{}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/owner/copilot/billing":
			writeJSON(t, w, http.StatusOK, map[string]any{"seat_breakdown": map[string]any{"total": 3}})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["copilot_code_review"]
	payload := rawJSON(t, map[string]any{"paths": []string{"/vendor/**", "/generated/**"}})

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "update_content_exclusions",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected Copilot validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected Copilot rollback success, got %+v", rollback)
	}
}

func TestDependabotConfigExecutorUpdateValidateRollback(t *testing.T) {
	current := repoContentResponse{
		Path:     dependabotConfigPath,
		SHA:      "sha-old",
		Encoding: "base64",
		Content:  base64.StdEncoding.EncodeToString([]byte("version: 2\nupdates: []\n")),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/contents/.github/dependabot.yml":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/contents/.github/dependabot.yml":
			body := readBody(t, r)
			current.SHA = "sha-new"
			current.Content = stringValue(body["content"])
			writeJSON(t, w, http.StatusOK, map[string]any{
				"content": map[string]any{
					"path": dependabotConfigPath,
					"sha":  current.SHA,
				},
				"commit": map[string]any{
					"sha":     "commit-new",
					"message": stringValue(body["message"]),
				},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["dependabot_config"]
	payload := rawJSON(t, map[string]any{
		"content": "version: 2\nupdates:\n  - package-ecosystem: github-actions\n    directory: /\n    schedule:\n      interval: weekly\n",
		"message": "Update dependabot policy",
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "update",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected dependabot config validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected dependabot config rollback success, got %+v", rollback)
	}
}

func TestDependabotConfigExecutorStructuredNoOpAndInspectSnapshot(t *testing.T) {
	currentContent := "version: 2\nupdates:\n  - package-ecosystem: github-actions\n    directory: /\n    schedule:\n      interval: weekly\n"
	current := repoContentResponse{
		Path:     dependabotConfigPath,
		SHA:      "sha-old",
		Encoding: "base64",
		Content:  base64.StdEncoding.EncodeToString([]byte(currentContent)),
	}
	putCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/contents/.github/dependabot.yml":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/contents/.github/dependabot.yml":
			putCalls++
			t.Fatalf("did not expect update request for deterministic no-op")
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["dependabot_config"]
	payload := rawJSON(t, map[string]any{
		"config": map[string]any{
			"version": 2,
			"updates": []map[string]any{{
				"ecosystem":   "github-actions",
				"directories": []string{"/"},
				"schedule": map[string]any{
					"interval": "weekly",
				},
			}},
		},
	})

	snapshot, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err := json.Unmarshal(snapshot.State, &state); err != nil {
		t.Fatal(err)
	}
	if _, ok := state["config"].(map[string]any); !ok {
		t.Fatalf("expected structured config in inspect snapshot, got %+v", state)
	}

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "update",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata["no_op"] != "true" {
		t.Fatalf("expected no-op metadata, got %+v", result.Metadata)
	}
	if putCalls != 0 {
		t.Fatalf("expected no PUT calls, got %d", putCalls)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected no-op validation success, got %s", validation.Summary)
	}
}

func TestCopilotSeatManagementAddUsersValidateRollback(t *testing.T) {
	seats := []map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/owner/copilot/billing/seats":
			writeJSON(t, w, http.StatusOK, seats)
		case r.Method == http.MethodPost && r.URL.Path == "/orgs/owner/copilot/billing/selected_users":
			body := readBody(t, r)
			for _, user := range body["selected_usernames"].([]any) {
				seats = append(seats, map[string]any{
					"assignee": map[string]any{"login": stringValue(user)},
				})
			}
			writeJSON(t, w, http.StatusCreated, seats)
		case r.Method == http.MethodDelete && r.URL.Path == "/orgs/owner/copilot/billing/selected_users":
			body := readBody(t, r)
			remove := map[string]struct{}{}
			for _, user := range body["selected_usernames"].([]any) {
				remove[strings.ToLower(stringValue(user))] = struct{}{}
			}
			next := make([]map[string]any, 0, len(seats))
			for _, seat := range seats {
				login := strings.ToLower(stringValue(seat["assignee"].(map[string]any)["login"]))
				if _, ok := remove[login]; ok {
					continue
				}
				next = append(next, seat)
			}
			seats = next
			writeJSON(t, w, http.StatusOK, seats)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["copilot_seat_management"]
	payload := rawJSON(t, map[string]any{"selected_usernames": []string{"octocat"}})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "add_users",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected Copilot seat validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected Copilot seat rollback success, got %+v", rollback)
	}
}

func TestActionsExecutorPermissionsUpdateValidateRollback(t *testing.T) {
	current := map[string]any{
		"enabled":              true,
		"allowed_actions":      "all",
		"sha_pinning_required": false,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/actions/permissions":
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodPut && r.URL.Path == "/repos/owner/repo/actions/permissions":
			current = readBody(t, r)
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["actions"]

	payload := rawJSON(t, map[string]any{
		"enabled":              true,
		"allowed_actions":      "selected",
		"sha_pinning_required": true,
	})
	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "permissions_update",
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected actions validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected actions rollback success, got %+v", rollback)
	}
}

func TestActionsExecutorInspectUsageArtifactsAndCaches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/actions/workflows/build.yml/timing":
			writeJSON(t, w, http.StatusOK, map[string]any{"billable": map[string]any{"UBUNTU": map[string]any{"total_ms": 1200}}})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/actions/artifacts":
			writeJSON(t, w, http.StatusOK, map[string]any{"total_count": 1})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/actions/caches":
			writeJSON(t, w, http.StatusOK, map[string]any{"total_count": 2})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["actions"]

	usage, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{
		ResourceID: "build.yml",
		Query:      map[string]string{"view": "workflow_usage"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if usage.ResourceID != "build.yml" {
		t.Fatalf("expected workflow usage resource id, got %s", usage.ResourceID)
	}
	artifacts, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{Query: map[string]string{"view": "artifacts"}})
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.ResourceID != "artifacts" {
		t.Fatalf("expected artifacts resource id, got %s", artifacts.ResourceID)
	}
	caches, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{Query: map[string]string{"view": "caches"}})
	if err != nil {
		t.Fatal(err)
	}
	if caches.ResourceID != "caches" {
		t.Fatalf("expected caches resource id, got %s", caches.ResourceID)
	}
}

func TestCodespacesExecutorCreateValidateRollback(t *testing.T) {
	current := map[string]any{
		"name":  "gitdex-main",
		"state": "Available",
		"repository": map[string]any{
			"full_name": "owner/repo",
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/codespaces":
			writeJSON(t, w, http.StatusCreated, current)
		case r.Method == http.MethodGet && r.URL.Path == "/user/codespaces/gitdex-main":
			if current == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, http.StatusOK, current)
		case r.Method == http.MethodDelete && r.URL.Path == "/user/codespaces/gitdex-main":
			current = nil
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["codespaces"]

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation: "create",
		Payload: rawJSON(t, map[string]any{
			"ref": "main",
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceID != "gitdex-main" {
		t.Fatalf("expected codespace name, got %s", result.ResourceID)
	}
	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: result,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected codespaces validation success, got %s", validation.Summary)
	}
	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected codespaces rollback success, got %+v", rollback)
	}
}

func TestCodespacesExecutorInspectPolicyAndPrebuilds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/codespaces/machines":
			writeJSON(t, w, http.StatusOK, map[string]any{"machines": []map[string]any{{"name": "basicLinux32gb", "prebuild_availability": "ready"}}})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/codespaces/permissions_check":
			writeJSON(t, w, http.StatusOK, map[string]any{"can_create_codespace": true})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["codespaces"]

	policy, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{Query: map[string]string{"view": "repo_policy"}})
	if err != nil {
		t.Fatal(err)
	}
	if policy.ResourceID != "repo_policy" {
		t.Fatalf("expected repo policy resource id, got %s", policy.ResourceID)
	}
	prebuilds, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{Query: map[string]string{"view": "prebuilds"}})
	if err != nil {
		t.Fatal(err)
	}
	if prebuilds.ResourceID != "prebuilds" {
		t.Fatalf("expected prebuild resource id, got %s", prebuilds.ResourceID)
	}
	check, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{Query: map[string]string{"view": "permissions_check"}})
	if err != nil {
		t.Fatal(err)
	}
	if check.ResourceID != "permissions_check" {
		t.Fatalf("expected permissions check resource id, got %s", check.ResourceID)
	}
}

func TestBuildReleaseAssetRestorePayloadPrefersStoredBytesRef(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cached-asset.txt")
	if err := os.WriteFile(file, []byte("cached release asset"), 0o600); err != nil {
		t.Fatal(err)
	}
	payload, err := buildReleaseAssetRestorePayload(context.Background(), New("token", "owner", "repo"), &platform.AdminMutationResult{
		Metadata: map[string]string{
			"release_id":       "11",
			"asset_name":       "gitdex.txt",
			"content_type":     "text/plain",
			"stored_bytes_ref": file,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload.Payload, &raw); err != nil {
		t.Fatal(err)
	}
	got, _ := raw["content_base64"].(string)
	if got != base64.StdEncoding.EncodeToString([]byte("cached release asset")) {
		t.Fatalf("expected stored-bytes restore payload, got %q", got)
	}
}

func TestReleaseExecutorAssetDeleteCachesRollbackBytes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	release := map[string]any{
		"id":         float64(11),
		"tag_name":   "v1.0.0",
		"name":       "v1.0.0",
		"draft":      false,
		"upload_url": "https://uploads.example.com/repos/owner/repo/releases/11/assets{?name,label}",
	}
	asset := map[string]any{
		"id":                   float64(99),
		"name":                 "gitdex.txt",
		"content_type":         "text/plain",
		"label":                "notes",
		"browser_download_url": "https://download.example.com/gitdex.txt",
		"url":                  "https://api.example.com/assets/99",
	}
	var uploadedBytes []byte

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/releases/assets/99":
			asset["browser_download_url"] = server.URL + "/asset-download/99"
			asset["url"] = server.URL + "/asset-bytes/99"
			writeJSON(t, w, http.StatusOK, asset)
		case r.Method == http.MethodGet && r.URL.Path == "/asset-bytes/99":
			_, _ = w.Write([]byte("cached release asset"))
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/releases/assets/99":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/releases/11":
			writeJSON(t, w, http.StatusOK, release)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected upload request %s", r.Method)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		uploadedBytes = data
		writeJSON(t, w, http.StatusCreated, asset)
	}))
	defer uploadServer.Close()

	release["upload_url"] = uploadServer.URL + "/repos/owner/repo/releases/11/assets{?name,label}"

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["release"]

	result, err := executor.Mutate(context.Background(), platform.AdminMutationRequest{
		Operation:  "asset_delete",
		ResourceID: "99",
		Scope: map[string]string{
			"asset_id":   "99",
			"release_id": "11",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata["rollback_grade"] != "reversible" {
		t.Fatalf("expected reversible rollback grade, got %q", result.Metadata["rollback_grade"])
	}
	if result.Metadata["stored_bytes_ref"] == "" {
		t.Fatal("expected stored bytes ref to be captured")
	}
	entry, err := platform.ResolveReleaseAssetRef(result.Metadata["stored_bytes_ref"])
	if err != nil {
		t.Fatalf("expected asset ledger entry: %v", err)
	}
	if _, err := os.Stat(entry.BytesPath); err != nil {
		t.Fatalf("expected cached asset bytes: %v", err)
	}

	rollback, err := executor.Rollback(context.Background(), platform.AdminRollbackRequest{Mutation: result})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.OK {
		t.Fatalf("expected cached rollback to succeed, got %+v", rollback)
	}
	if string(uploadedBytes) != "cached release asset" {
		t.Fatalf("expected cached bytes to be re-uploaded, got %q", string(uploadedBytes))
	}
}

func TestPagesExecutorValidateReadinessRequiresCertificate(t *testing.T) {
	current := map[string]any{
		"cname":                  "localhost",
		"build_type":             "workflow",
		"protected_domain_state": "verified",
		"https_enforced":         true,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pages":
			writeJSON(t, w, http.StatusOK, current)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New("token", "owner", "repo")
	client.baseURL = server.URL
	executor := client.AdminExecutors()["pages"]

	validation, err := executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: &platform.AdminMutationResult{CapabilityID: "pages", Operation: "update"},
		Payload:  rawJSON(t, map[string]any{"cname": "localhost", "https_enforced": true}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if validation.OK {
		t.Fatalf("expected readiness validation failure, got %+v", validation)
	}

	current["https_certificate"] = map[string]any{"state": "approved"}
	validation, err = executor.Validate(context.Background(), platform.AdminValidationRequest{
		Mutation: &platform.AdminMutationResult{CapabilityID: "pages", Operation: "update"},
		Payload:  rawJSON(t, map[string]any{"cname": "localhost", "https_enforced": true}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validation.OK {
		t.Fatalf("expected readiness validation success, got %+v", validation)
	}
}

func rawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func writeJSON(t *testing.T, w http.ResponseWriter, code int, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if value == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}

func decryptSecretForTest(t *testing.T, encrypted string, recipientPriv, recipientPub *[32]byte) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 32 {
		t.Fatal("ciphertext too short")
	}
	var ephemeralPub [32]byte
	copy(ephemeralPub[:], raw[:32])
	hash, err := blake2b.New(24, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = hash.Write(ephemeralPub[:])
	_, _ = hash.Write(recipientPub[:])
	sum := hash.Sum(nil)
	var nonce [24]byte
	copy(nonce[:], sum[:24])
	plain, ok := box.Open(nil, raw[32:], &nonce, &ephemeralPub, recipientPriv)
	if !ok {
		t.Fatal("failed to decrypt secret")
	}
	return string(plain)
}
