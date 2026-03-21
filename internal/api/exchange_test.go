package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExchangePayload_JSONContract(t *testing.T) {
	p := &ExchangePayload{
		Format:        ExchangeFormatJSON,
		APIVersion:    "v1",
		SchemaVersion: "1",
		PayloadType:   "plans",
		Data:          json.RawMessage(`{"plan_id":"p1"}`),
		Checksum:      "abc",
		CreatedAt:     time.Now().UTC(),
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"api_version"`) || !strings.Contains(raw, `"payload_type"`) {
		t.Errorf("JSON missing fields: %s", raw)
	}
}

func TestDefaultExchangeValidator_Validate_Valid(t *testing.T) {
	v := NewDefaultExchangeValidator()
	p := &ExchangePayload{
		Format:        ExchangeFormatJSON,
		APIVersion:    "v1",
		SchemaVersion: "1",
		PayloadType:   "plans",
		Data:          json.RawMessage(`{}`),
		CreatedAt:     time.Now().UTC(),
	}
	if err := v.Validate(p); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestDefaultExchangeValidator_Validate_Nil(t *testing.T) {
	v := NewDefaultExchangeValidator()
	if err := v.Validate(nil); err == nil {
		t.Error("expected error for nil payload")
	}
}

func TestDefaultExchangeValidator_Validate_NoAPIVersion(t *testing.T) {
	v := NewDefaultExchangeValidator()
	p := &ExchangePayload{
		Format:        ExchangeFormatJSON,
		APIVersion:    "",
		SchemaVersion: "1",
		PayloadType:   "plans",
		CreatedAt:     time.Now().UTC(),
	}
	if err := v.Validate(p); err == nil {
		t.Error("expected error for empty api_version")
	}
}

func TestParseExchangePayload_JSON(t *testing.T) {
	data := []byte(`{"format":"json","api_version":"v1","schema_version":"1","payload_type":"plans","data":{},"created_at":"2026-03-19T12:00:00Z"}`)
	p, err := ParseExchangePayload(data, ExchangeFormatJSON)
	if err != nil {
		t.Fatalf("ParseExchangePayload failed: %v", err)
	}
	if p.APIVersion != "v1" {
		t.Errorf("APIVersion: got %q", p.APIVersion)
	}
	if p.PayloadType != "plans" {
		t.Errorf("PayloadType: got %q", p.PayloadType)
	}
}
