package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/your-org/gitdex/internal/storage"
	"gopkg.in/yaml.v3"
)

type ProviderExportEngine struct {
	provider storage.StorageProvider
}

func NewProviderExportEngine(provider storage.StorageProvider) *ProviderExportEngine {
	return &ProviderExportEngine{provider: provider}
}

func (e *ProviderExportEngine) Export(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	exportType := req.ExportType
	if exportType == "" {
		exportType = ExportPlanReport
	}
	format := strings.TrimSpace(strings.ToLower(req.Format))
	if format == "" {
		format = "json"
	}
	payload, err := e.buildPayload(exportType, req.Filters, req.IncludeEvidence)
	if err != nil {
		return nil, err
	}
	data, err := marshalExportPayload(payload, format)
	if err != nil {
		return nil, err
	}
	return &ExportResult{
		ExportType:  exportType,
		Format:      format,
		Data:        data,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (e *ProviderExportEngine) buildPayload(exportType ExportType, filters map[string]string, includeEvidence bool) (any, error) {
	router := NewProviderQueryRouter(e.provider)
	switch exportType {
	case ExportPlanReport:
		result, err := router.Query(&QueryRequest{QueryType: QueryPlanStatus, Filters: filters, Pagination: Pagination{Page: 1, PerPage: 1000}})
		if err != nil {
			return nil, err
		}
		return map[string]any{"plans": result.Items, "include_evidence": includeEvidence}, nil
	case ExportTaskReport:
		result, err := router.Query(&QueryRequest{QueryType: QueryTaskStatus, Filters: filters, Pagination: Pagination{Page: 1, PerPage: 1000}})
		if err != nil {
			return nil, err
		}
		return map[string]any{"tasks": result.Items, "include_evidence": includeEvidence}, nil
	case ExportCampaignReport:
		result, err := router.Query(&QueryRequest{QueryType: QueryCampaignStatus, Filters: filters, Pagination: Pagination{Page: 1, PerPage: 1000}})
		if err != nil {
			return nil, err
		}
		return map[string]any{"campaigns": result.Items, "include_evidence": includeEvidence}, nil
	case ExportAuditReport:
		result, err := router.Query(&QueryRequest{QueryType: QueryAuditLog, Filters: filters, Pagination: Pagination{Page: 1, PerPage: 1000}})
		if err != nil {
			return nil, err
		}
		return map[string]any{"audit": result.Items, "include_evidence": includeEvidence}, nil
	case ExportHandoffArtifact:
		pkgs, err := e.provider.HandoffStore().ListPackages()
		if err != nil {
			return nil, err
		}
		return map[string]any{"handoffs": toHandoffArtifactPayload(pkgs), "include_evidence": includeEvidence}, nil
	default:
		return nil, fmt.Errorf("unsupported export type %q", exportType)
	}
}

func marshalExportPayload(payload any, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "yaml", "yml":
		data, err := yaml.Marshal(payload)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "markdown", "md":
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
		return "```json\n" + string(data) + "\n```", nil
	default:
		return "", fmt.Errorf("unsupported export format %q", format)
	}
}
