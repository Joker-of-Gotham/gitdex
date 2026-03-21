package server_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/daemon/server"
	"github.com/your-org/gitdex/internal/storage"
)

func TestGitHubWebhookValidatesAndAppendsTriggerEvent(t *testing.T) {
	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()

	if err := provider.TriggerStore().SaveTrigger(&autonomy.TriggerConfig{
		TriggerID:   "tr_webhook",
		TriggerType: autonomy.TriggerTypeEvent,
		Name:        "pull-request-trigger",
		Source:      "owner/repo:pull_request",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("save trigger: %v", err)
	}

	srv := server.New(server.DefaultConfig(), provider)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const secret = "webhook-secret"
	t.Setenv("GITDEX_GITHUB_WEBHOOK_SECRET", secret)

	body := []byte(`{"action":"opened","repository":{"full_name":"owner/repo"}}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/webhooks/github", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-GitHub-Delivery", "delivery-1")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if validated, _ := payload["validated"].(bool); !validated {
		t.Fatalf("validated = %v, want true", payload["validated"])
	}
	if matched, _ := payload["matched_triggers"].(float64); matched != 1 {
		t.Fatalf("matched_triggers = %v, want 1", payload["matched_triggers"])
	}

	events, err := provider.TriggerStore().ListTriggerEvents("tr_webhook", 10)
	if err != nil {
		t.Fatalf("list trigger events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0].SourceEvent != "opened" {
		t.Fatalf("source event = %q, want %q", events[0].SourceEvent, "opened")
	}
}

func TestGitHubWebhookCallbackCanPopulateResultingTaskID(t *testing.T) {
	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()

	if err := provider.TriggerStore().SaveTrigger(&autonomy.TriggerConfig{
		TriggerID:   "tr_exec",
		TriggerType: autonomy.TriggerTypeEvent,
		Name:        "push-trigger",
		Source:      "owner/repo:push",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("save trigger: %v", err)
	}

	cfg := server.DefaultConfig()
	cfg.OnGitHubWebhook = func(ctx context.Context, trigger *autonomy.TriggerConfig, repoFullName string, ev *autonomy.TriggerEvent) error {
		ev.ResultingTaskID = "cycle-123"
		return nil
	}
	srv := server.New(cfg, provider)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte(`{"action":"synchronize","repository":{"full_name":"owner/repo"}}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/webhooks/github", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	events, err := provider.TriggerStore().ListTriggerEvents("tr_exec", 10)
	if err != nil {
		t.Fatalf("list trigger events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0].ResultingTaskID != "cycle-123" {
		t.Fatalf("resulting task id = %q, want %q", events[0].ResultingTaskID, "cycle-123")
	}
}

func TestGitHubWebhookRejectsInvalidSignature(t *testing.T) {
	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()

	srv := server.New(server.DefaultConfig(), provider)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	t.Setenv("GITDEX_GITHUB_WEBHOOK_SECRET", "webhook-secret")

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/webhooks/github", bytes.NewBufferString(`{"repository":{"full_name":"owner/repo"}}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", "sha256=bad")

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}
