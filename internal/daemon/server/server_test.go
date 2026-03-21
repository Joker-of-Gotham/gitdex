package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/gitdex/internal/daemon/server"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/storage"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = provider.Close() })
	srv := server.New(server.DefaultConfig(), provider)
	return httptest.NewServer(srv.Handler())
}

func TestHealth(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestListPlans_Empty(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/plans")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var plans []*planning.Plan
	if err := json.NewDecoder(resp.Body).Decode(&plans); err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Errorf("plans len = %d, want 0", len(plans))
	}
}

func TestListPlans_AfterSave(t *testing.T) {
	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()

	planStore := provider.PlanStore()
	plan := &planning.Plan{
		PlanID: "test-plan-1",
		Status: planning.PlanDraft,
		Intent: planning.PlanIntent{Source: "test", RawInput: "hello", ActionType: "sync"},
		Steps:  []planning.PlanStep{{Sequence: 1, Action: "sync", Target: "repo", RiskLevel: planning.RiskLow}},
	}
	if err := planStore.Save(plan); err != nil {
		t.Fatal(err)
	}

	srv := server.New(server.DefaultConfig(), provider)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/plans")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var plans []*planning.Plan
	if err := json.NewDecoder(resp.Body).Decode(&plans); err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans len = %d, want 1", len(plans))
	}
	if plans[0].PlanID != "test-plan-1" {
		t.Errorf("PlanID = %q, want test-plan-1", plans[0].PlanID)
	}
}

func TestGetPlan_Valid(t *testing.T) {
	provider, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()

	planStore := provider.PlanStore()
	plan := &planning.Plan{
		PlanID: "test-plan-2",
		Status: planning.PlanDraft,
		Intent: planning.PlanIntent{Source: "test", RawInput: "hello", ActionType: "sync"},
		Steps:  []planning.PlanStep{{Sequence: 1, Action: "sync", Target: "repo", RiskLevel: planning.RiskLow}},
	}
	if err := planStore.Save(plan); err != nil {
		t.Fatal(err)
	}

	srv := server.New(server.DefaultConfig(), provider)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/plans/test-plan-2")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var p planning.Plan
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.PlanID != "test-plan-2" {
		t.Errorf("PlanID = %q, want test-plan-2", p.PlanID)
	}
}

func TestGetPlan_InvalidID(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/plans/nonexistent-plan-id")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetTask_InvalidID(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/tasks/nonexistent-task-id")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestListAuditEntries(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/audit")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var entries []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Empty ledger returns [] or null; both decode as valid JSON
}

func TestListCampaigns(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/v1/campaigns")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var campaigns []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&campaigns); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestCreatePlan(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	body := `{"intent":{"source":"test","raw_input":"hello","action_type":"sync"},"steps":[{"sequence":1,"action":"sync","target":"repo","risk_level":"low"}]}`
	resp, err := ts.Client().Post(ts.URL+"/api/v1/plans", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
}

func TestCreateCampaign(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	body := `{"name":"test campaign","description":"test","status":"draft","target_repos":[],"created_by":"test"}`
	resp, err := ts.Client().Post(ts.URL+"/api/v1/campaigns", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
}
