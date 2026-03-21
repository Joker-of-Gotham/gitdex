package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/storage"
)

type ProviderQueryRouter struct {
	provider storage.StorageProvider
}

func NewProviderQueryRouter(provider storage.StorageProvider) *ProviderQueryRouter {
	return &ProviderQueryRouter{provider: provider}
}

func (r *ProviderQueryRouter) Query(req *QueryRequest) (*QueryResult, error) {
	if req == nil {
		req = &QueryRequest{QueryType: QueryTaskStatus, Pagination: Pagination{Page: 1, PerPage: 20}}
	}
	pagination := req.Pagination
	if pagination.PerPage <= 0 {
		pagination.PerPage = 20
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	items, err := r.loadItems(req.QueryType)
	if err != nil {
		return nil, err
	}
	filtered := filterItems(items, req.Filters)
	total := len(filtered)
	from := (pagination.Page - 1) * pagination.PerPage
	to := from + pagination.PerPage
	if from >= total {
		filtered = []json.RawMessage{}
	} else {
		if to > total {
			to = total
		}
		filtered = filtered[from:to]
	}

	return &QueryResult{
		QueryType:  req.QueryType,
		Items:      filtered,
		TotalCount: total,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
	}, nil
}

func (r *ProviderQueryRouter) GetResource(endpoint, id string) (*APIResponse, error) {
	endpoint = strings.TrimSpace(strings.ToLower(endpoint))
	items, err := r.loadResources(endpoint)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		var payload map[string]any
		if err := json.Unmarshal(item, &payload); err != nil {
			continue
		}
		if matchesResourceID(payload, id) {
			return &APIResponse{StatusCode: 200, Payload: item, Timestamp: time.Now().UTC()}, nil
		}
	}
	return &APIResponse{
		StatusCode: 404,
		Errors:     []APIError{{Code: "not_found", Message: endpoint + " not found: " + id}},
		Timestamp:  time.Now().UTC(),
	}, nil
}

func (r *ProviderQueryRouter) loadItems(queryType QueryType) ([]json.RawMessage, error) {
	switch queryType {
	case QueryTaskStatus:
		return marshalTasks(r.provider)
	case QueryCampaignStatus:
		return marshalCampaigns(r.provider)
	case QueryAuditLog:
		return marshalAuditEntries(r.provider)
	case QueryPlanStatus:
		return marshalPlans(r.provider)
	default:
		return []json.RawMessage{}, nil
	}
}

func (r *ProviderQueryRouter) loadResources(endpoint string) ([]json.RawMessage, error) {
	switch endpoint {
	case "tasks":
		return marshalTasks(r.provider)
	case "campaigns":
		return marshalCampaigns(r.provider)
	case "audit":
		return marshalAuditEntries(r.provider)
	case "plans":
		return marshalPlans(r.provider)
	case "handoffs":
		return marshalHandoffs(r.provider)
	default:
		return []json.RawMessage{}, nil
	}
}

func marshalTasks(provider storage.StorageProvider) ([]json.RawMessage, error) {
	tasks, err := provider.TaskStore().ListTasks()
	if err != nil {
		return nil, err
	}
	return marshalSlice(tasks)
}

func marshalCampaigns(provider storage.StorageProvider) ([]json.RawMessage, error) {
	campaigns, err := provider.CampaignStore().ListCampaigns()
	if err != nil {
		return nil, err
	}
	return marshalSlice(campaigns)
}

func marshalAuditEntries(provider storage.StorageProvider) ([]json.RawMessage, error) {
	entries, err := provider.AuditLedger().Query(audit.AuditFilter{})
	if err != nil {
		return nil, err
	}
	return marshalSlice(entries)
}

func marshalPlans(provider storage.StorageProvider) ([]json.RawMessage, error) {
	plans, err := provider.PlanStore().List()
	if err != nil {
		return nil, err
	}
	return marshalSlice(plans)
}

func marshalHandoffs(provider storage.StorageProvider) ([]json.RawMessage, error) {
	packages, err := provider.HandoffStore().ListPackages()
	if err != nil {
		return nil, err
	}
	return marshalSlice(packages)
}

func marshalSlice[T any](items []*T) ([]json.RawMessage, error) {
	result := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		raw, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		result = append(result, raw)
	}
	return result, nil
}

func filterItems(items []json.RawMessage, filters map[string]string) []json.RawMessage {
	if len(filters) == 0 {
		return items
	}
	filtered := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		if matchFilters(item, filters) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func matchesResourceID(payload map[string]any, id string) bool {
	for _, key := range []string{"id", "task_id", "campaign_id", "plan_id", "entry_id", "package_id"} {
		if value, ok := payload[key]; ok && fmtStr(value) == id {
			return true
		}
	}
	return false
}

func toHandoffArtifactPayload(pkgs []*autonomy.HandoffPackage) any {
	return map[string]any{"packages": pkgs}
}
