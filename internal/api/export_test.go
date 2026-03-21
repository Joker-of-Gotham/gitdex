package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExportResult_JSONContract(t *testing.T) {
	r := &ExportResult{
		ExportType:  ExportPlanReport,
		Format:      "json",
		Data:        `{"plan":"p1"}`,
		FilePath:    "/tmp/out.json",
		GeneratedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(data), `"export_type"`) {
		t.Errorf("JSON missing export_type")
	}
}

func TestDefaultExportEngine_Export_ExportTypes(t *testing.T) {
	engine := &DefaultExportEngine{}
	ctx := context.Background()
	req := &ExportRequest{ExportType: ExportPlanReport, Format: "json"}
	_, err := engine.Export(ctx, req)
	if err == nil {
		t.Error("expected error from DefaultExportEngine without storage")
	}
}

func TestDefaultExportEngine_Export_Formats(t *testing.T) {
	engine := &DefaultExportEngine{}
	ctx := context.Background()
	req := &ExportRequest{ExportType: ExportPlanReport, Format: "json"}
	_, err := engine.Export(ctx, req)
	if err == nil {
		t.Error("expected error from DefaultExportEngine without storage")
	}
}

func TestDefaultExportEngine_Export_NilRequest(t *testing.T) {
	engine := &DefaultExportEngine{}
	ctx := context.Background()
	_, err := engine.Export(ctx, nil)
	if err == nil {
		t.Error("Export(nil): expected error, got nil")
	}
}

func TestListExportTypes_ReturnsAllTypes(t *testing.T) {
	want := []ExportType{
		ExportPlanReport,
		ExportTaskReport,
		ExportCampaignReport,
		ExportAuditReport,
		ExportHandoffArtifact,
	}
	got := ListExportTypes()
	if len(got) != len(want) {
		t.Errorf("ListExportTypes(): got %d types, want %d", len(got), len(want))
	}
	seen := make(map[ExportType]bool)
	for _, et := range got {
		seen[et] = true
	}
	for _, et := range want {
		if !seen[et] {
			t.Errorf("ListExportTypes(): missing %q", et)
		}
	}
}
