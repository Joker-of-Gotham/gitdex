package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/api"
)

func TestAPIRequest_JSONContract(t *testing.T) {
	req := &api.APIRequest{
		RequestID:  "req_abc123",
		Endpoint:   "/api/v1/intents",
		Method:     "POST",
		Payload:    json.RawMessage(`{"intent":"test"}`),
		APIVersion: "v1",
		Timestamp:  time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"request_id"`,
		`"endpoint"`,
		`"method"`,
		`"payload"`,
		`"api_version"`,
		`"timestamp"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestAPIResponse_JSONContract(t *testing.T) {
	resp := &api.APIResponse{
		RequestID:  "req_xyz",
		StatusCode: 201,
		Payload:    json.RawMessage(`{"id":"x"}`),
		Timestamp:  time.Now().UTC(),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"request_id"`,
		`"status_code"`,
		`"timestamp"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestAPIError_JSONContract(t *testing.T) {
	err := &api.APIError{
		Code:    "invalid_payload",
		Message: "payload is required",
		Field:   "payload",
	}

	data, err2 := json.Marshal(err)
	if err2 != nil {
		t.Fatalf("marshal error: %v", err2)
	}

	fields := []string{
		`"code"`,
		`"message"`,
		`"field"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestMemoryAPIRouter_HandleIntent(t *testing.T) {
	router := api.NewMemoryAPIRouter()
	req := &api.APIRequest{
		Endpoint:   "/api/v1/intents",
		Method:     "POST",
		Payload:    json.RawMessage(`{"intent":"add feature"}`),
		APIVersion: api.APIVersion,
		Timestamp:  time.Now().UTC(),
	}
	resp, err := router.Handle(req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("got status %d, want 201", resp.StatusCode)
	}
	if len(resp.Payload) == 0 {
		t.Error("expected non-empty payload")
	}
}
