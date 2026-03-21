package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/api"
)

func TestExportResult_JSONContract(t *testing.T) {
	result := &api.ExportResult{
		ExportType:  api.ExportPlanReport,
		Format:      "json",
		Data:        `{"simulated":true}`,
		FilePath:    "/tmp/out.json",
		GeneratedAt: time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	fields := []string{
		`"export_type"`,
		`"format"`,
		`"data"`,
		`"generated_at"`,
	}
	raw := string(data)
	for _, f := range fields {
		if !strings.Contains(raw, f) {
			t.Errorf("JSON missing field %s in: %s", f, raw)
		}
	}
}

func TestExportRequest_RoundTrip(t *testing.T) {
	original := &api.ExportRequest{
		ExportType:      api.ExportPlanReport,
		Format:          "json",
		IncludeEvidence: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded api.ExportRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ExportType != original.ExportType {
		t.Errorf("ExportType: got %q, want %q", decoded.ExportType, original.ExportType)
	}
	if decoded.IncludeEvidence != original.IncludeEvidence {
		t.Errorf("IncludeEvidence: got %v, want %v", decoded.IncludeEvidence, original.IncludeEvidence)
	}
}

func TestExportResult_RoundTrip(t *testing.T) {
	original := &api.ExportResult{
		ExportType:  api.ExportCampaignReport,
		Format:      "yaml",
		Data:        `{"campaign":"c1"}`,
		FilePath:    "/tmp/campaign.yaml",
		GeneratedAt: time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded api.ExportResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ExportType != original.ExportType {
		t.Errorf("ExportType: got %q, want %q", decoded.ExportType, original.ExportType)
	}
	if decoded.Format != original.Format {
		t.Errorf("Format: got %q, want %q", decoded.Format, original.Format)
	}
	if decoded.Data != original.Data {
		t.Errorf("Data: got %q, want %q", decoded.Data, original.Data)
	}
	if decoded.FilePath != original.FilePath {
		t.Errorf("FilePath: got %q, want %q", decoded.FilePath, original.FilePath)
	}
	if !decoded.GeneratedAt.Equal(original.GeneratedAt) {
		t.Errorf("GeneratedAt: got %v, want %v", decoded.GeneratedAt, original.GeneratedAt)
	}
}

func TestListExportTypes(t *testing.T) {
	types := api.ListExportTypes()
	if len(types) == 0 {
		t.Error("expected at least one export type")
	}
	seen := make(map[api.ExportType]bool)
	for _, et := range types {
		if et == "" {
			t.Error("export type should not be empty")
		}
		if seen[et] {
			t.Errorf("duplicate export type: %s", et)
		}
		seen[et] = true
	}
}
