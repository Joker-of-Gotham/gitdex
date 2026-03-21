package api

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAPIRequest_JSONContract(t *testing.T) {
	req := &APIRequest{
		RequestID:  "req_123",
		Endpoint:   "/api/v1/intents",
		Method:     "POST",
		Payload:    json.RawMessage(`{"intent":"test"}`),
		APIVersion: "v1",
		Timestamp:  time.Now().UTC(),
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty json")
	}
	var decoded APIRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.RequestID != req.RequestID {
		t.Errorf("RequestID: got %q, want %q", decoded.RequestID, req.RequestID)
	}
	if decoded.Endpoint != req.Endpoint {
		t.Errorf("Endpoint: got %q, want %q", decoded.Endpoint, req.Endpoint)
	}
}

func TestAPIResponse_JSONContract(t *testing.T) {
	resp := &APIResponse{
		RequestID:  "req_123",
		StatusCode: 201,
		Payload:    json.RawMessage(`{"id":"abc"}`),
		Errors:     []APIError{{Code: "err", Message: "msg"}},
		Timestamp:  time.Now().UTC(),
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	if !contains(raw, `"request_id"`) || !contains(raw, `"status_code"`) {
		t.Errorf("JSON missing expected fields: %s", raw)
	}
}

func TestMemoryAPIRouter_Handle_SubmitIntent(t *testing.T) {
	r := NewMemoryAPIRouter()
	req := &APIRequest{
		RequestID:  "req_1",
		Endpoint:   "/api/v1/intents",
		Method:     "POST",
		Payload:    json.RawMessage(`{"intent":"deploy to prod"}`),
		APIVersion: APIVersion,
		Timestamp:  time.Now().UTC(),
	}
	resp, err := r.Handle(req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("status: got %d, want 201", resp.StatusCode)
	}
	if len(resp.Payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestMemoryAPIRouter_Handle_IntentBadPayload(t *testing.T) {
	r := NewMemoryAPIRouter()
	req := &APIRequest{
		RequestID:  "req_2",
		Endpoint:   "/api/v1/intents",
		Method:     "POST",
		Payload:    nil,
		APIVersion: APIVersion,
		Timestamp:  time.Now().UTC(),
	}
	resp, err := r.Handle(req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status: got %d, want 400", resp.StatusCode)
	}
	if len(resp.Errors) == 0 {
		t.Error("expected errors")
	}
}

func TestMemoryAPIRouter_ListEndpoints(t *testing.T) {
	r := NewMemoryAPIRouter()
	eps := r.ListEndpoints()
	if len(eps) < 3 {
		t.Errorf("expected at least 3 endpoints, got %d", len(eps))
	}
	var found bool
	for _, ep := range eps {
		if ep.Path == "/api/v1/intents" && ep.Method == "POST" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /api/v1/intents in endpoints")
	}
}

func TestMemoryAPIRouter_Handle_NotFound(t *testing.T) {
	r := NewMemoryAPIRouter()
	req := &APIRequest{
		RequestID:  "req_3",
		Endpoint:   "/api/v1/unknown",
		Method:     "GET",
		APIVersion: APIVersion,
		Timestamp:  time.Now().UTC(),
	}
	resp, err := r.Handle(req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsImpl(s, sub))
}

func containsImpl(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
