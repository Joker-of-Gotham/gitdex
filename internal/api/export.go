package api

import (
	"context"
	"fmt"
	"time"
)

// ExportType represents the type of export.
type ExportType string

const (
	ExportPlanReport      ExportType = "plan_report"
	ExportTaskReport      ExportType = "task_report"
	ExportCampaignReport  ExportType = "campaign_report"
	ExportAuditReport     ExportType = "audit_report"
	ExportHandoffArtifact ExportType = "handoff_artifact"
)

// ExportRequest represents an export request.
type ExportRequest struct {
	ExportType      ExportType        `json:"export_type" yaml:"export_type"`
	Filters         map[string]string `json:"filters,omitempty" yaml:"filters,omitempty"`
	Format          string            `json:"format" yaml:"format"` // json, yaml, markdown
	IncludeEvidence bool              `json:"include_evidence" yaml:"include_evidence"`
}

// ExportResult represents the result of an export.
type ExportResult struct {
	ExportType  ExportType `json:"export_type" yaml:"export_type"`
	Format      string     `json:"format" yaml:"format"`
	Data        string     `json:"data" yaml:"data"`
	FilePath    string     `json:"file_path,omitempty" yaml:"file_path,omitempty"`
	GeneratedAt time.Time  `json:"generated_at" yaml:"generated_at"`
}

// ExportEngine performs exports.
type ExportEngine interface {
	Export(ctx context.Context, request *ExportRequest) (*ExportResult, error)
}

// DefaultExportEngine returns an error directing users to configure storage.
type DefaultExportEngine struct{}

func (e *DefaultExportEngine) Export(_ context.Context, req *ExportRequest) (*ExportResult, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	return nil, fmt.Errorf("export requires a configured storage provider; run 'gitdex setup' to configure storage")
}

// ListExportTypes returns all available export types.
func ListExportTypes() []ExportType {
	return []ExportType{
		ExportPlanReport,
		ExportTaskReport,
		ExportCampaignReport,
		ExportAuditReport,
		ExportHandoffArtifact,
	}
}
