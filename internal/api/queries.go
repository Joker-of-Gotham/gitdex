package api

import (
	"encoding/json"
	"strconv"
	"strings"
)

// QueryType represents the type of query.
type QueryType string

const (
	QueryTaskStatus     QueryType = "task_status"
	QueryCampaignStatus QueryType = "campaign_status"
	QueryAuditLog       QueryType = "audit_log"
	QueryPlanStatus     QueryType = "plan_status"
)

// Pagination holds pagination parameters.
type Pagination struct {
	Page    int `json:"page" yaml:"page"`
	PerPage int `json:"per_page" yaml:"per_page"`
}

// QueryRequest represents a query request.
type QueryRequest struct {
	QueryType  QueryType         `json:"query_type" yaml:"query_type"`
	Filters    map[string]string `json:"filters,omitempty" yaml:"filters,omitempty"`
	Pagination Pagination        `json:"pagination,omitempty" yaml:"pagination,omitempty"`
	SortBy     string            `json:"sort_by,omitempty" yaml:"sort_by,omitempty"`
	SortOrder  string            `json:"sort_order,omitempty" yaml:"sort_order,omitempty"`
}

// QueryResult represents a query result.
type QueryResult struct {
	QueryType  QueryType         `json:"query_type" yaml:"query_type"`
	Items      []json.RawMessage `json:"items" yaml:"items"`
	TotalCount int               `json:"total_count" yaml:"total_count"`
	Page       int               `json:"page" yaml:"page"`
	PerPage    int               `json:"per_page" yaml:"per_page"`
}

// QueryRouter provides query and get-by-id capabilities.
type QueryRouter interface {
	Query(request *QueryRequest) (*QueryResult, error)
	GetResource(endpoint, id string) (*APIResponse, error)
}

// BuildQueryRequest builds a QueryRequest from CLI args.
func BuildQueryRequest(queryTypeStr, filterStr string) *QueryRequest {
	qt := QueryType(queryTypeStr)
	if qt == "" {
		qt = QueryTaskStatus
	}
	switch queryTypeStr {
	case "tasks", "task_status":
		qt = QueryTaskStatus
	case "campaigns", "campaign_status":
		qt = QueryCampaignStatus
	case "audit":
		qt = QueryAuditLog
	case "plans", "plan_status":
		qt = QueryPlanStatus
	}

	filters := make(map[string]string)
	if filterStr != "" {
		parts := strings.SplitN(filterStr, "=", 2)
		if len(parts) == 2 {
			filters[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return &QueryRequest{
		QueryType:  qt,
		Filters:    filters,
		Pagination: Pagination{Page: 1, PerPage: 20},
	}
}

func matchFilters(item json.RawMessage, filters map[string]string) bool {
	if len(filters) == 0 {
		return true
	}
	var m map[string]any
	if err := json.Unmarshal(item, &m); err != nil {
		return true
	}
	for k, v := range filters {
		if mv, ok := m[k]; ok {
			ms := fmtStr(mv)
			if ms != v {
				return false
			}
		}
	}
	return true
}

func fmtStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// StoreTask is a no-op retained for interface compatibility.
// Real task storage goes through storage.StorageProvider.TaskStore().
func StoreTask(_ string, _ json.RawMessage) {}
