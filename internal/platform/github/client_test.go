package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, mux *http.ServeMux) (*Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	client, err := NewClientWithBaseURL(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return client, ts
}

func TestGetRepository(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"full_name":      "owner/repo",
			"description":    "test desc",
			"default_branch": "main",
			"private":        true,
		})
	})

	client, _ := newTestServer(t, mux)
	state, err := client.GetRepository(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.FullName != "owner/repo" {
		t.Errorf("full_name = %q, want %q", state.FullName, "owner/repo")
	}
	if state.DefaultBranch != "main" {
		t.Errorf("default_branch = %q, want %q", state.DefaultBranch, "main")
	}
	if !state.IsPrivate {
		t.Error("expected private repo")
	}
}

func TestListOpenPullRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"number": 42,
				"title":  "Fix bug",
				"user":   map[string]interface{}{"login": "alice"},
				"labels": []map[string]interface{}{{"name": "bug"}},
				"draft":  false,
			},
		})
	})

	client, _ := newTestServer(t, mux)
	prs, err := client.ListOpenPullRequests(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prs) != 1 {
		t.Fatalf("pr count = %d, want 1", len(prs))
	}
	if prs[0].Number != 42 {
		t.Errorf("pr number = %d, want 42", prs[0].Number)
	}
	if prs[0].Author != "alice" {
		t.Errorf("author = %q, want %q", prs[0].Author, "alice")
	}
}

func TestListPullRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("state"); got != "closed" {
			t.Fatalf("state query = %q, want %q", got, "closed")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"number": 7,
				"title":  "Closed PR",
				"state":  "closed",
			},
		})
	})

	client, _ := newTestServer(t, mux)
	prs, err := client.ListPullRequests(context.Background(), "owner", "repo", "closed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 || prs[0].GetNumber() != 7 {
		t.Fatalf("prs = %+v", prs)
	}
}

func TestListWorkflowRuns(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/actions/runs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 1,
			"workflow_runs": []map[string]interface{}{
				{
					"id":          321,
					"workflow_id": 123,
					"name":        "CI",
					"status":      "completed",
					"conclusion":  "success",
					"head_branch": "main",
				},
			},
		})
	})

	client, _ := newTestServer(t, mux)
	runs, err := client.ListWorkflowRuns(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("run count = %d, want 1", len(runs))
	}
	if runs[0].Conclusion != "success" {
		t.Errorf("conclusion = %q, want %q", runs[0].Conclusion, "success")
	}
	if runs[0].RunID != 321 {
		t.Errorf("run_id = %d, want %d", runs[0].RunID, 321)
	}
	if runs[0].WorkflowID != 123 {
		t.Errorf("workflow_id = %d, want %d", runs[0].WorkflowID, 123)
	}
}

func TestListDeployments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/deployments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":          1,
				"environment": "production",
				"ref":         "main",
				"url":         "https://api.github.test/deployments/1",
			},
		})
	})
	mux.HandleFunc("GET /api/v3/repos/owner/repo/deployments/1/statuses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"state": "success"},
		})
	})

	client, _ := newTestServer(t, mux)
	deps, err := client.ListDeployments(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("deployment count = %d, want 1", len(deps))
	}
	if deps[0].State != "success" {
		t.Errorf("state = %q, want %q", deps[0].State, "success")
	}
	if deps[0].Environment != "production" {
		t.Errorf("environment = %q, want %q", deps[0].Environment, "production")
	}
	if deps[0].URL != "https://api.github.test/deployments/1" {
		t.Errorf("url = %q, want %q", deps[0].URL, "https://api.github.test/deployments/1")
	}
}

func TestRerunWorkflowRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/actions/runs/321/rerun", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	client, _ := newTestServer(t, mux)
	if err := client.RerunWorkflowRun(context.Background(), "owner", "repo", 321); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelWorkflowRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/actions/runs/654/cancel", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "cancellation requested"})
	})

	client, _ := newTestServer(t, mux)
	if err := client.CancelWorkflowRun(context.Background(), "owner", "repo", 654); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEstimateOpenIssueCount(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<https://api.github.com/repos/owner/repo/issues?page=5>; rel="last"`)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"number": 1, "title": "issue 1"},
		})
	})

	client, _ := newTestServer(t, mux)
	count, err := client.EstimateOpenIssueCount(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count < 0 {
		t.Errorf("count = %d, want >= 0", count)
	}
}

func TestListIssues(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("state"); got != "all" {
			t.Fatalf("state query = %q, want %q", got, "all")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"number": 11, "title": "Issue 11", "state": "open"},
			{"number": 12, "title": "Issue 12", "state": "closed"},
		})
	})

	client, _ := newTestServer(t, mux)
	issues, err := client.ListIssues(context.Background(), "owner", "repo", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 || issues[1].GetNumber() != 12 {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestListCommits(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/commits", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"sha": "abcdef1234567890",
				"author": map[string]interface{}{
					"login": "alice",
				},
				"commit": map[string]interface{}{
					"message": "Initial commit\n\nWith details",
					"author": map[string]interface{}{
						"name": "Alice",
						"date": "2026-03-20T12:00:00Z",
					},
				},
			},
		})
	})

	client, _ := newTestServer(t, mux)
	commits, err := client.ListCommits(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("commit count = %d, want 1", len(commits))
	}
	if commits[0].GetSHA() != "abcdef1234567890" {
		t.Fatalf("sha = %q", commits[0].GetSHA())
	}
}

func TestGetCommit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/commits/abcdef1234567890", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sha": "abcdef1234567890",
			"commit": map[string]interface{}{
				"message": "Detailed commit",
				"author": map[string]interface{}{
					"name":  "Alice",
					"email": "alice@example.com",
					"date":  "2026-03-20T12:00:00Z",
				},
			},
			"files": []map[string]interface{}{
				{"filename": "main.go", "status": "modified", "additions": 2, "deletions": 1},
			},
		})
	})

	client, _ := newTestServer(t, mux)
	commit, err := client.GetCommit(context.Background(), "owner", "repo", "abcdef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if commit.GetSHA() != "abcdef1234567890" {
		t.Fatalf("sha = %q", commit.GetSHA())
	}
	if len(commit.Files) != 1 || commit.Files[0].GetFilename() != "main.go" {
		t.Fatalf("files = %+v", commit.Files)
	}
}

func TestNewClient_NonNil(t *testing.T) {
	c := NewClient(http.DefaultClient)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestWebhookHandler_ValidateSignature_Valid(t *testing.T) {
	secret := "my-webhook-secret"
	body := []byte(`{"action":"opened","number":1}`)
	h := NewWebhookHandler(secret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))

	if err := h.ValidateSignature(req, body); err != nil {
		t.Errorf("ValidateSignature: unexpected error: %v", err)
	}
}

func TestWebhookHandler_ValidateSignature_Invalid(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	h := NewWebhookHandler("secret")

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalidorwrongsignaturehex")

	if err := h.ValidateSignature(req, body); err == nil {
		t.Error("ValidateSignature: expected error for invalid signature")
	}
}

func TestWebhookHandler_ValidateSignature_EmptySecret(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	h := NewWebhookHandler("")

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", "sha256=something")
	if err := h.ValidateSignature(req, body); err == nil {
		t.Error("ValidateSignature: expected error when secret is empty")
	}
}

func TestNewRateLimitBudget_NonNil(t *testing.T) {
	b := NewRateLimitBudget(5000)
	if b == nil {
		t.Fatal("NewRateLimitBudget: expected non-nil")
	}
}

func TestRateLimitBudget_CanProceed_HighRemaining(t *testing.T) {
	b := NewRateLimitBudget(5000)
	if !b.CanProceed() {
		t.Error("CanProceed: expected true when remaining is high")
	}
}
