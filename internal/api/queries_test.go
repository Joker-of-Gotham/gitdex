package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestQueryRequest_JSONContract(t *testing.T) {
	qr := &QueryRequest{
		QueryType:  QueryTaskStatus,
		Filters:    map[string]string{"status": "running"},
		Pagination: Pagination{Page: 1, PerPage: 20},
		SortBy:     "created_at",
		SortOrder:  "desc",
	}
	data, err := json.Marshal(qr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"query_type"`) {
		t.Errorf("JSON missing query_type: %s", raw)
	}
}

func TestQueryResult_JSONContract(t *testing.T) {
	res := &QueryResult{
		QueryType:  QueryTaskStatus,
		Items:      []json.RawMessage{json.RawMessage(`{"id":"t1"}`)},
		TotalCount: 1,
		Page:       1,
		PerPage:    20,
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"total_count"`) {
		t.Errorf("JSON missing total_count: %s", raw)
	}
}

func TestBuildQueryRequest(t *testing.T) {
	qr := BuildQueryRequest("tasks", "status=running")
	if qr.QueryType != QueryTaskStatus {
		t.Errorf("QueryType: got %s, want task_status", qr.QueryType)
	}
	if qr.Filters["status"] != "running" {
		t.Errorf("Filters[status]: got %q", qr.Filters["status"])
	}
}

func TestMemoryAPIRouter_Query(t *testing.T) {
	r := NewMemoryAPIRouter()
	qr := BuildQueryRequest("tasks", "")
	_, err := r.Query(qr)
	if err == nil {
		t.Error("expected error from MemoryAPIRouter.Query without storage")
	}
}

func TestMemoryAPIRouter_GetResource_Task(t *testing.T) {
	r := NewMemoryAPIRouter()
	_, err := r.GetResource("tasks", "task_001")
	if err == nil {
		t.Error("expected error from MemoryAPIRouter.GetResource without storage")
	}
}

func TestMemoryAPIRouter_GetResource_NotFound(t *testing.T) {
	r := NewMemoryAPIRouter()
	_, err := r.GetResource("tasks", "nonexistent")
	if err == nil {
		t.Error("expected error from MemoryAPIRouter.GetResource without storage")
	}
}
